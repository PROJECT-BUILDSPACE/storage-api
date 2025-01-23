package utils

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/mitchellh/mapstructure"

	CDSUtils "github.com/SLG-European-Projects/cds-go/utils"
	"github.com/isotiropoulos/storage-api/globals"
	models "github.com/isotiropoulos/storage-api/models"
)

func RespondWithError(w http.ResponseWriter, code int, message string, reason string, internalCode string) {

	err := models.ErrorReport{
		Message:        message + " Please contact the Core Platform Support Team.",
		Reason:         reason,
		Status:         code,
		InternalStatus: internalCode,
	}
	w.Header().Set("Content-Type", "application/json")
	// Set the status code before writing the response body
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(err)
	return
}

func init() {}

func GetPartFromChan(ch <-chan minio.ObjectPart) <-chan minio.ObjectPart {
	firstCh := make(chan minio.ObjectPart)

	go func() {
		firstCh <- <-ch
		close(firstCh)
	}()

	return firstCh
}

// GenerateUUID returns random IDs
func GenerateUUID() (string, error) {
	var uuid [16]byte
	_, err := rand.Read(uuid[:])
	if err != nil {
		return "", err
	}
	// Set the UUID version and variant bits
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0xbf) | 0x80
	return hex.EncodeToString(uuid[:4]) + "-" + hex.EncodeToString(uuid[4:6]) + "-" + hex.EncodeToString(uuid[6:8]) + "-" + hex.EncodeToString(uuid[8:10]) + "-" + hex.EncodeToString(uuid[10:]), nil
}

// CreateFolder is a function to format an item of the folders collection.
func CreateFolder(r models.PostFolderBody, folderID string, ancestors []string, userID string) models.Folder {

	var meta models.Meta
	var dbFolder models.Folder

	meta.Creator = userID
	meta.Description = r.Description
	meta.Title = r.FolderName
	meta.DateCreation = time.Now()
	meta.Read = append(meta.Read, userID)
	meta.Write = append(meta.Write, userID)
	// meta.Update = append(meta.Update, models.Updated{
	// 	Date: time.Now(),
	// 	User: userID,
	// })
	meta.Update.Date = time.Now()
	meta.Update.User = userID

	if r.Parent != "" && r.Parent != ancestors[0] {
		dbFolder = models.Folder{
			Id:        folderID,
			Meta:      meta,
			Parent:    r.Parent,
			Ancestors: append(ancestors, r.Parent),
			Files:     []string{},
			Folders:   []string{},
			Size:      0,
		}
	} else {
		dbFolder = models.Folder{
			Id:        folderID,
			Meta:      meta,
			Parent:    r.Parent,
			Ancestors: ancestors,
			Files:     []string{},
			Folders:   []string{},
			Size:      0,
		}
	}

	return dbFolder
}

func GetClaimsFromContext(ctxClaims interface{}) (*models.OidcClaims, error) {
	// Parse the raw claims into a custom struct
	var claims models.OidcClaims
	err := mapstructure.Decode(ctxClaims, &claims)
	if err != nil {
		return nil, fmt.Errorf("could not parse claims: %v", err)
	}
	return &claims, nil
}

// ItemInArray is a function to check if an array contains an item
func ItemInArray(array []string, item string) bool {
	for _, i := range array {
		if i == item {
			return true
		}
	}
	return false
}

// RemoveFromSlice is a function to remove an item from a slice (Keeps order)
func RemoveFromSlice(slice []string, item string) []string {
	var s int

	for pos, val := range slice {
		if val == item {
			s = pos
			break
		}
	}
	return append(slice[:s], slice[s+1:]...)
}

// GenerateRequestFingerprint generates a SHA-256 hash of the JSON-encoded Copernicus request body
func GenerateRequestFingerprint(body models.CopernicusInput) (string, error) {

	// Convert request body to JSON
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	// Sort keys to ensure consistent order
	var m map[string]interface{}
	if err := json.Unmarshal(bodyJSON, &m); err != nil {
		return "", err
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sortedBodyJSON, err := json.Marshal(m)
	if err != nil {
		return "", err
	}

	// Calculate SHA-256 hash
	hash := sha256.Sum256(sortedBodyJSON)

	// Convert hash to hexadecimal string
	fingerprint := fmt.Sprintf("%x", hash)

	return fingerprint, err
}

// CheckCopernicusStatus
func CheckCopernicusStatus(dataset models.CopernicusRecord, subject string) {

	// mark goroutine as running!

	if _, exists := globals.RunningGoroutines.Load(dataset.Id); exists {
		fmt.Printf("Task %s is already running. Skipping...\n", dataset.Id)
		return // Avoid starting a new goroutine if the task is already running
	} else {
		fmt.Printf("No existing task with ID %s. Starting a new one...\n", dataset.Id)
		globals.RunningGoroutines.Store(dataset.Id, true)

	}

	params := map[string]interface{}{
		"log":     true,
		"request": true,
	}
	var dataReader io.ReadCloser
	var size int

InfiniteLoop:
	for {
		task, err := globals.CopernicusClient.GetOneJob(dataset.Details.JobID, params)
		if err != nil {
			// Could not get the task for some reason. Break loop!
			fmt.Println(err.Error())
			break InfiniteLoop
		}

		// Status changed => Update DB Document
		if task.Status != dataset.Details.Status {
			dataset.Details = task
			_, err = globals.CopernicusDB.UpdateWithId(dataset)
			if err != nil {
				fmt.Println(err.Error())
				break InfiniteLoop
			}

			if task.Status == "failed" || task.Status == "dismissed" {
				break InfiniteLoop
			} else if task.Status == "successful" {
				//here we have to save the dataset via the download link to our database
				//split data into parts go routines etc.
				//then we will deal with copernicus data as a normal file in our db
				result, err := globals.CopernicusClient.GetJobResult(dataset.Details.JobID)
				if err != nil {
					fmt.Println(err.Error())
					break InfiniteLoop
				}

				dataReader, size, err = CDSUtils.DownloadFileReader(result.Asset.Value.Href)
				if err != nil {
					dataReader = nil
					fmt.Println(err.Error())
				}
				break InfiniteLoop

			} else {
				time.Sleep(globals.CheckTime)
			}

		}
	}

	if dataReader != nil {

		// Grab reference file
		file, err := globals.FileDB.GetOneByID(dataset.FileId)
		if err != nil {
			fmt.Println("Can't retrieve file document:", err.Error())
			return
		}

		// Calculate the number of parts and the optimal part size
		totalPartsCount := int(math.Ceil(float64(size) / float64(globals.PartSize)))

		// Initialize a slice to hold io.Reader instances for each chunk
		var readers []io.Reader
		var sizes []int64

		checkSize := size

		for checkSize > 0 {
			// Create a buffer for the current chunk
			partBuffer := make([]byte, globals.PartSize)
			n, err := io.ReadFull(dataReader, partBuffer)
			// n, err := downresp.Body.Read(partBuffer)
			if err != nil && !(errors.Is(err, io.ErrUnexpectedEOF)) {
				fmt.Println("Error reading response body:", err)
				break
			}

			// Create a new slice containing only the data read in this iteration
			buf := partBuffer[:n]

			readers = append(readers, bytes.NewReader(buf))
			sizes = append(sizes, int64(n))

			checkSize -= n
		}

		defer dataReader.Close()

		// Create a slice of channels to hold upload results
		partsCh := make(chan models.Part, totalPartsCount)
		errorCh := make(chan error, totalPartsCount)

		// Create a waitgroup to synchronize uploads
		var wg sync.WaitGroup

		for i, reader := range readers {
			wg.Add(1)
			go func(partReader io.Reader, item int) {

				defer wg.Done()

				// Make updates in document
				var filePart models.Part
				partId, err := GenerateUUID()
				if err != nil {
					errorCh <- err
					return
				}

				//get part info
				filePart.Id = partId
				filePart.FileID = dataset.FileId
				filePart.Size = sizes[item]
				filePart.PartNumber = item

				// Upload part
				uploadInfo, err := globals.Storage.PostPart(globals.COPERNICUS_BUCKET_ID, partId, partReader, sizes[item], minio.PutObjectOptions{})
				if err != nil {
					fmt.Println(err.Error())
					errorCh <- err
					return
				}

				filePart.UploadInfo = uploadInfo

				// insert part to db
				err = globals.PartsDB.InsertOne(filePart)
				if err != nil {
					fmt.Println("Error:", err)
					errorCh <- err
					return
				}
				partsCh <- filePart
			}(reader, i)
		}

		// Wait for all parts to finish uploading
		wg.Wait()
		close(partsCh)
		close(errorCh)

		// Collect errors from the error channel
		var hasError bool
		for err := range errorCh {
			fmt.Println("Error:", err)
			hasError = true
		}

		if !hasError {
			//update dataset info
			update := models.Updated{
				Date: time.Now(),
				User: subject,
			}

			file.Total = totalPartsCount
			meta := file.Meta
			meta.Update = update
			file.Meta = meta

			file.Size = int64(size)

			//update file
			file, err = globals.FileDB.UpdateWithId(file)
			if err != nil {
				fmt.Println(err.Error())
			}
		}
		// Unmark goroutine from running!
		globals.RunningGoroutines.Delete(dataset.Id)

	}
}

// GrainFolders helps grab the correct folder from a cursor.
func GrainFolders(ctx context.Context, candidateFolder models.Folder, parentName string, resultChan chan<- models.Folder, wg *sync.WaitGroup) {
	defer wg.Done()

	select {
	case <-ctx.Done():
		// If context is done, exit the goroutine
		return
	// case <-time.After(2 * time.Second): // Simulate processing time
	default:
		parent, _ := globals.FolderDB.GetOneByID(candidateFolder.Parent)
		if parent.Meta.Title == parentName {
			resultChan <- candidateFolder
		}
	}
}
