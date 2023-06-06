package handlers

import (
	// "context"

	// "context"

	// "fmt"

	// "github.com/gorilla/mux"

	// auth "github.com/isotiropoulos/storage-api/oauth"

	"bytes"
	"context"
	"fmt"
	"strconv"

	// "fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sync"

	"github.com/gorilla/mux"
	"github.com/isotiropoulos/storage-api/models"
	"github.com/isotiropoulos/storage-api/utils"
	"github.com/minio/minio-go/v7"
	"go.mongodb.org/mongo-driver/bson"

	"encoding/json"
	"net/http"
	"time"
	// "strings"
	// "go.mongodb.org/mongo-driver/bson"
)

func PostFileLocal(w http.ResponseWriter, r *http.Request) {

	// Resolve Claims
	claims, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not resolve user's claims.", err.Error(), "FIL0001")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	var postFile models.File

	fileID, err := utils.GenerateUUID()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in creating file's ID.", err.Error(), "FIL0002")
		return
	}
	postFile.Id = fileID
	var content = r.Header.Get("Content-Type")

	// Get bucket ID from headers
	bucketID := r.Header.Get("X-Group-Id")

	if content == "application/json" {
		err = json.NewDecoder(r.Body).Decode(&postFile)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not decode request body.", err.Error(), "FIL0003")
			return
		}

		// Open file
		file, err := os.Open(postFile.OriginalTitle)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not open file.", err.Error(), "FIL0004")
			return
		}
		defer file.Close()

		// Get file info to determine its size
		fileInfo, err := file.Stat()
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not get file's information.", err.Error(), "FIL0005")
			return
		}

		// Calculate the number of parts and the optimal part size
		totalPartsCount := int(math.Ceil(float64(fileInfo.Size()) / float64(partSize)))

		// Initiate multipart upload
		uploadID, err := storage.OpenMultipart(bucketID, postFile.Id)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error in multipart upload.", err.Error(), "FIL0006")
			return
		}

		// Create a slice of channels to hold upload results
		partsCh := make(chan minio.ObjectPart, totalPartsCount)

		// Create a waitgroup to synchronize uploads
		var wg sync.WaitGroup
		// wg.Add(int(totalPartsCount))

		// Iterate over parts and upload them concurrently
		// var parts [totalPartsCount]minio.CompletePart
		parts := make([]minio.CompletePart, totalPartsCount)
		for i := int(1); i <= totalPartsCount; i++ {
			// Read part data into a buffer
			partBuffer := make([]byte, partSize)
			n, err := file.Read(partBuffer)
			if err != nil && err != io.EOF {
				utils.RespondWithError(w, http.StatusBadRequest, "Could not read data into buffer.", err.Error(), "FIL0007")
				return
			}

			// Create a new reader for this part
			partReader := bytes.NewReader(partBuffer[:n])

			// Initiate a new part upload
			opt := models.ObjectPartInfo{
				PartNumber: i,
				// UploadID:     uploadID,
				// ContentLength: int64(n),
			}
			wg.Add(1)

			go func(partReader io.Reader, opt models.ObjectPartInfo) {
				defer wg.Done()
				part, err := storage.PostPart(bucketID, fileID, uploadID, opt.PartNumber, partReader, int64(n), minio.PutObjectPartOptions{})

				if err != nil {
					utils.RespondWithError(w, http.StatusBadRequest, "Could post part of file.", err.Error(), "FIL0008")
					return
				}
				partsCh <- part
			}(partReader, opt)
		}

		// // Wait for all parts to finish uploading
		go func() {
			wg.Wait()
			close(partsCh)
		}()

		// Collect uploaded parts and complete multipart upload
		for part := range partsCh {
			parts[part.PartNumber-1] = minio.CompletePart{
				PartNumber: part.PartNumber,
				ETag:       part.ETag,
			}
		}

		if _, err = storage.CloseMultipart(bucketID, postFile.Id, uploadID, parts); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not close multipart upload.", err.Error(), "FIL0009")
			return
		}

		// Post in Mongo
		base := filepath.Base(fileInfo.Name())
		ext := filepath.Ext(base)
		title := base[:len(base)-len(ext)]
		postFile.Meta.DateCreation = time.Now().Unix()
		postFile.Meta.Title = title
		postFile.FileType = ext
		postFile.OriginalTitle = title
		postFile.Size = fileInfo.Size()
		postFile.Meta.Creator = claims.Subject
		folder, err := folderDB.GetOneByID(postFile.FolderID)
		if err != nil || folder.Id == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not find parent folder.", err.Error(), "FIL0010")
			return
		}
		postFile.Ancestors = append(folder.Ancestors, postFile.FolderID)
		postFile.Meta.Read = folder.Meta.Read
		postFile.Meta.Write = folder.Meta.Write
		postFile.Meta.Update = append(postFile.Meta.Update, models.Updated{
			User: claims.Subject,
			Date: time.Now(),
		})
		err = fileDB.InsertOne(postFile)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Could not insert file.", err.Error(), "FIL0011")
			return
		}

		err = folderDB.UpdateFiles(postFile.Id, postFile.FolderID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Could not update parent folder.", err.Error(), "FIL0012")
			return
		}

		// Update ancestore's meta
		err = folderDB.UpdateMetaAncestors(postFile.Ancestors, claims.Subject)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Could not update ancestore's meta.", err.Error(), "FIL0013")
			return
		}
	}

	json.NewEncoder(w).Encode(postFile)
}

func GetFileLocal(w http.ResponseWriter, r *http.Request) {

	// Gets params
	params := mux.Vars(r)
	fileId := params["id"]

	// Retrieve file from DB
	getFile, err := fileDB.GetOneByID(fileId)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Error in retrieving file's information.", err.Error(), "FIL0014")
		return
	}

	// Retrieve group
	groupId := r.Header.Get("X-Group-Id")

	// Get object info
	objectInfo, err := storage.StatFiles(getFile.Id, groupId)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in retrieving file's information.", err.Error(), "FIL0015")
		return
	}

	// Calculate the number of parts and the optimal part size
	totalPartsCount := int(math.Ceil(float64(getFile.Size) / float64(partSize)))

	// Create a slice of channels to hold download results
	partsCh := make(chan models.FilePart, totalPartsCount)

	// Create a waitgroup to synchronize downloads
	var wg sync.WaitGroup
	// wg.Add(int(totalPartsCount))

	// Iterate over parts and download them concurrently
	parts := make([]models.FilePart, totalPartsCount)
	for i := int(1); i <= totalPartsCount; i++ {

		wg.Add(1)
		go func(item int) {
			defer wg.Done()

			reader, _, _, err := storage.GetFile(getFile.Id, groupId, minio.GetObjectOptions{
				PartNumber: item,
			})
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, "Could not get file part.", err.Error(), "FIL0016")
				return
			}

			var buf bytes.Buffer
			_, err = io.Copy(&buf, reader)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, "Could not pass file part to buffer.", err.Error(), "FIL0017")
				return
			}

			part := models.FilePart{
				FileID:     getFile.Id,
				PartNumber: item,
				Part:       buf,
			}
			partsCh <- part
		}(i)
	}

	// // Wait for all parts to finish uploading
	go func() {
		wg.Wait()
		close(partsCh)
	}()

	// Collect uploaded parts and complete multipart upload
	for part := range partsCh {
		parts[part.PartNumber-1] = part
	}

	// concatenate bytes of the "Data" field of each object
	var buffer bytes.Buffer
	for _, part := range parts {
		buffer.Write(part.Part.Bytes())
	}

	w.Header().Set("Content-Type", objectInfo.ContentType)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(getFile.Size, 10))
	w.WriteHeader(http.StatusAccepted)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", getFile.Meta.Title+getFile.FileType))
	_, err = io.Copy(w, &buffer)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not send file.", err.Error(), "FIL0018")
		return
	}
}

func DeleteFileLocal(w http.ResponseWriter, r *http.Request) {

	// Resolve Claims
	claims, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not resolve user's claims.", err.Error(), "FIL0019")
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var parentFolder models.Folder

	params := mux.Vars(r) // Gets params

	// Retrive Object from DB
	file, err := fileDB.GetOneByID(params["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Could not find file.", err.Error(), "FIL0020")
		return
	}

	// Delete Object from DB
	err = fileDB.DeleteOneByID(params["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not delete file.", err.Error(), "FIL0021")
		return
	}

	// Update folder containing the file
	// Get the Parent folder
	parentFolder, err = folderDB.GetOneByID(file.FolderID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Could not find parent folder.", err.Error(), "FIL0022")
		return
	}

	// Remove the deleted folder (but keep the order)
	newFiles := utils.RemoveFromSlice(parentFolder.Files, params["id"])

	parentFolder.Files = newFiles
	// Pass new values

	// Update Parent
	_, err = folderDB.UpdateWithId(parentFolder)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Could not update parent folder.", err.Error(), "FIL0023")
		return
	}

	// Update Uncestores
	err = folderDB.UpdateMetaAncestors(file.Ancestors, claims.Subject)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Could not update parent folder.", err.Error(), "FIL0024")
		return
	}

	// Remove Object from MINIO
	groupId := r.Header.Get("X-Group-Id")
	if err = storage.DeleteFile(file.Id, groupId); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in deleting file.", err.Error(), "FIL0025")
		return
	}
	json.NewEncoder(w).Encode(file)
}

func GetFileInfoLocal(w http.ResponseWriter, r *http.Request) {

	fileId := r.FormValue("id")
	fmt.Println("JASON", fileId)
	// Retrive Object from DB
	file, err := fileDB.GetOneByID(fileId)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Could not find file.", err.Error(), "FIL0026")
		return
	}

	json.NewEncoder(w).Encode(file)
}

func UpdateFileLocal(w http.ResponseWriter, r *http.Request) {

	// Resolve Claims
	claims, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve claims.", err.Error(), "FIL0027")
		return
	}

	var file models.File
	var updateFile models.File

	err = json.NewDecoder(r.Body).Decode(&updateFile)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve request.", err.Error(), "FIL0028")
		return
	}

	// Check title
	parentID := ""
	if updateFile.FolderID != "" {
		parentID = updateFile.FolderID
	} else {
		parentID = r.Header.Get("X-Group-Id")
	}

	// Check if title already exists
	// First get current title
	currentDoc, err := fileDB.GetOneByID(updateFile.Id)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "File trying to get updated doesn't exist.", err.Error(), "FIL0029")
		return
	}

	if updateFile.Meta.Title != currentDoc.Meta.Title {
		// Check if title is illegal
		filesCursor, err := fileDB.GetCursorByFolderID(parentID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Could not obtain siblings.", err.Error(), "FIL0030")
			return
		}
		defer filesCursor.Close(context.Background())

		for filesCursor.Next(context.Background()) {
			var result bson.M
			var inFile models.File
			if err := filesCursor.Decode(&result); err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve cursor.", err.Error(), "FIL0031")
				return
			}
			bsonBytes, _ := bson.Marshal(result)
			bson.Unmarshal(bsonBytes, &inFile)
			if inFile.Meta.Title == updateFile.Meta.Title {
				utils.RespondWithError(w, http.StatusConflict, "File Exists.", "Cannot rename file to this name, since it is already taken.", "FIL0032")
				return
			}
		}
	}

	// Update folder
	updateFile.Meta.Update = append(currentDoc.Meta.Update, models.Updated{
		User: claims.Subject,
		Date: time.Now(),
	})
	file, err = fileDB.UpdateWithId(updateFile)
	if err != nil {
		utils.RespondWithError(w, http.StatusConflict, "Could not update folder.", err.Error(), "FIL0033")
		return
	}

	// Update ancestores meta
	err = folderDB.UpdateMetaAncestors(file.Ancestors, claims.Subject)
	if err != nil {
		utils.RespondWithError(w, http.StatusConflict, "Could not update folder's ancestores.", err.Error(), "FIL0034")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(file)
}

// CopyFile is to copy a file.
func CopyFileLocal(w http.ResponseWriter, r *http.Request) {

	// Resolve Claims
	claims, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve claims.", err.Error(), "FIL0035")
		return
	}

	var cmBody models.CopyMoveBody
	err = json.NewDecoder(r.Body).Decode(&cmBody)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve request.", err.Error(), "FIL0036")
		return
	}

	file, err := fileDB.GetOneByID(cmBody.Id)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "File doesn't exist.", err.Error(), "FIL0037")
		return
	}

	var newName string
	if cmBody.NewName == "" {
		newName = file.Meta.Title
	} else {
		newName = cmBody.NewName
	}

	newParent, err := folderDB.GetOneByID(cmBody.Destination)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Destination doesn't exist.", err.Error(), "FIL0038")
		return
	}

	// Check if title is illegal
	filesCursor, err := fileDB.GetCursorByFolderID(cmBody.Destination)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not obtain siblings.", err.Error(), "FIL0039")
		return
	}
	defer filesCursor.Close(context.Background())

	for filesCursor.Next(context.Background()) {
		var result bson.M
		var inFile models.File
		if err := filesCursor.Decode(&result); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve cursor.", err.Error(), "FIL0040")
			return
		}
		bsonBytes, _ := bson.Marshal(result)
		bson.Unmarshal(bsonBytes, &inFile)
		if inFile.Meta.Title == newName {
			utils.RespondWithError(w, http.StatusConflict, "File Exists.", "Cannot copy file to destination with this name since it is already taken.", "FIL0041")
			return
		}
	}

	// Insert file
	file.FolderID = cmBody.Destination
	// Create a new `Updated` struct
	updated := models.Updated{
		User: claims.Subject,
		Date: time.Now(),
	}
	file.Meta.Update = []models.Updated{updated}
	file.Meta.Title = newName
	file.Id, err = utils.GenerateUUID()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in generating file's ID.", err.Error(), "FIL0042")
		return
	}
	ancestors := append(newParent.Ancestors, file.FolderID)
	file.Ancestors = ancestors
	err = fileDB.InsertOne(file)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not copy file.", err.Error(), "FIL0043")
		return
	}

	bucketID := r.Header.Get("X-Group-Id")
	err = storage.CopyFile(cmBody.Id, file.Id, bucketID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not copy file.", err.Error(), "FIL0044")
		return
	}

	// Update New Parent Folder
	newParent.Files = append(newParent.Files, file.Id)
	newParent.Meta.Update = append(newParent.Meta.Update, updated)
	_, err = folderDB.UpdateWithId(newParent)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not update parent folder.", err.Error(), "FIL0045")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(file)

}

// MoveFile is to move a folder.
func MoveFileLocal(w http.ResponseWriter, r *http.Request) {

	// Resolve Claims
	claims, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve claims.", err.Error(), "FIL0046")
		return
	}

	var cmBody models.CopyMoveBody
	err = json.NewDecoder(r.Body).Decode(&cmBody)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve request.", err.Error(), "FIL0047")
		return
	}

	file, err := fileDB.GetOneByID(cmBody.Id)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Folder doesn't exist.", err.Error(), "FIL0048")
		return
	}

	var newName string
	if cmBody.NewName == "" {
		newName = file.Meta.Title
	} else {
		newName = cmBody.NewName
	}

	newParent, err := folderDB.GetOneByID(cmBody.Destination)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Destination doesn't exist.", err.Error(), "FIL0049")
		return
	}

	// Check if title is illegal
	filesCursor, err := fileDB.GetCursorByFolderID(cmBody.Destination)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not obtain siblings.", err.Error(), "FIL0050")
		return
	}
	defer filesCursor.Close(context.Background())

	for filesCursor.Next(context.Background()) {
		var result bson.M
		var inFile models.File
		if err := filesCursor.Decode(&result); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve cursor.", err.Error(), "FIL0051")
			return
		}
		bsonBytes, _ := bson.Marshal(result)
		bson.Unmarshal(bsonBytes, &inFile)
		if inFile.Meta.Title == newName {
			utils.RespondWithError(w, http.StatusConflict, "File Exists.", "Cannot move file to destination with this name since it is already taken.", "FIL0052")
			return
		}
	}

	oldParent, err := folderDB.GetOneByID(file.FolderID)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Parent folder doesn't exist.", err.Error(), "FIL0053")
		return
	}

	// Remove the deleted file (but keep the order)
	newFiles := utils.RemoveFromSlice(oldParent.Files, file.Id)

	oldParent.Files = newFiles

	// Update old Parent
	oldParent, err = folderDB.UpdateWithId(oldParent)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in updating old parent folder.", err.Error(), "FIL0054")
		return
	}

	// Update Ancestore's Meta
	err = folderDB.UpdateMetaAncestors(file.Ancestors, claims.Subject)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in updating ancestore's meta.", err.Error(), "FIL0055")
		return
	}

	file.FolderID = cmBody.Destination
	// Create a new `Updated` struct
	updated := models.Updated{
		User: claims.Subject,
		Date: time.Now(),
	}
	file.Meta.Update = []models.Updated{updated}
	file.Meta.Title = newName
	ancestors := append(newParent.Ancestors, file.FolderID)
	file.Ancestors = ancestors
	updatedFile, err := fileDB.UpdateWithId(file)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not file folder.", err.Error(), "FIL0056")
		return
	}

	// Update New Parent Folder
	newParent.Files = append(newParent.Files, file.Id)
	newParent.Meta.Update = append(newParent.Meta.Update, updated)
	_, err = folderDB.UpdateWithId(newParent)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not update parent folder.", err.Error(), "FIL0057")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(updatedFile)

}
