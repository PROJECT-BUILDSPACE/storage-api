package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"
	"sync"

	"github.com/isotiropoulos/storage-api/models"
	"github.com/isotiropoulos/storage-api/utils"
	"github.com/minio/minio-go/v7"

	"time"

	"github.com/gorilla/mux"
)

// function to get list of all available datasets form CLIMATE service
func GetCDSList(w http.ResponseWriter, r *http.Request) {

	req, err := http.Get("https://cds.climate.copernicus.eu/api/v2/resources/")

	//form request to destination
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Destination doesn't exist.", err.Error(), "COP0001")
		return
	}

	//do request
	resp, err := http.DefaultClient.Do(req.Request)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not complete request", err.Error(), "COP0002")
	}

	defer resp.Body.Close()

	//read body
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Error in Response Body", err.Error(), "COP0003")
	}

	//encode response into json
	var a []string
	json.Unmarshal(body, &a)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(a)
}

// function to get list of all available datasets form ATMOSPHERE service
// identical function w/ previous func
func GetADSList(w http.ResponseWriter, r *http.Request) {

	req, err := http.Get("https://ads.atmosphere.copernicus.eu/api/v2/resources/")

	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Destination doesn't exist.", err.Error(), "COP0004")
		return
	}

	resp, err := http.DefaultClient.Do(req.Request)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not complete request", err.Error(), "COP0005")
	}

	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Error in Response Body", err.Error(), "COP0003")
	}

	var a []string
	json.Unmarshal(body, &a)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(a)
}

// function to get form for single dataset of CLIMATE service
// user should first get form, then craft the appropirate request
func GetCDSForm(w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)
	datasetID := params["id"]

	w.Header().Set("Content-Type", "application/json")

	//http://datastore.copernicus-climate.eu/c3s/published-forms/c3sprod/monthly_averaged_reanalysis_by_hour_of_day/form.json

	req, err := http.Get("http://datastore.copernicus-climate.eu/c3s/published-forms/c3sprod/" + string(datasetID) + "/form.json")

	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid Dataset Value", err.Error(), "FORM001")
		return
	}

	resp, err := http.DefaultClient.Do(req.Request)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not reach Copernicus endpoint", err.Error(), "FORM002")
		return
	}

	defer resp.Body.Close()

	var myform []models.Form

	err = json.NewDecoder(resp.Body).Decode(&myform)

	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not handel response", err.Error(), "FORM003")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(myform)
}

// forms a request for a single dataset CLIMATE
// save any VALID requests to database as a file
// + faulty requests handling
// also task id is saved
func GetDataset(w http.ResponseWriter, r *http.Request) {

	claims, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve claims.", err.Error(), "COP0004")
		return
	}

	var test models.CopernicusInput
	err = json.NewDecoder(r.Body).Decode(&test)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve request.", err.Error(), "COP0005")
		return
	}

	name := test.DatasetName
	//encode input to json
	content, err := json.Marshal(test.Body)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve content.", err.Error(), "COP0006")
		return
	}

	params := mux.Vars(r)
	service := params["service"]
	base := ""

	if service != "cds" && service != "ads" {
		utils.RespondWithError(w, http.StatusBadRequest, "Destination doesn't exist.", err.Error(), "COP0016")
		return
	}

	if service == "cds" {
		base = "https://cds.climate.copernicus.eu/api/v2/resources/"
	}

	if service == "ads" {
		base = "https://ads.atmosphere.copernicus.eu/api/v2/resources/"
	}

	req, err := http.NewRequest("POST", base+name, bytes.NewBuffer(content))

	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not Reach Copernicus Domain", err.Error(), "COP0007")
		return
	}
	//auth for cds, should be const
	req.Header.Set("Content-Type", "application/json")

	if service == "cds" {
		req.SetBasicAuth(CDS_UID, CDS_KEY)
	}

	if service == "ads" {
		req.SetBasicAuth(ADS_UID, ADS_KEY)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not Reach Copernicus Data", err.Error(), "COP0008")
		return
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could Read Copernicus Response", err.Error(), "COP0009")
	}

	var b models.CopernicusResponse
	json.Unmarshal(body, &b)

	var postFile models.File
	postFile.FolderID = COPERNICUS_BUCKET_ID
	//get bucket
	folder, err := folderDB.GetOneByID(postFile.FolderID)
	if err != nil || folder.Id == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not find parent folder.", err.Error(), "COP0010")
		return
	}
	//create file id
	fileID, err := utils.GenerateUUID()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in creating file's ID.", err.Error(), "COP0011")
		return
	}
	postFile.Id = fileID
	//Create and post file
	update := models.Updated{
		Date: time.Now(),
		User: claims.Subject,
	}

	title := name
	postFile.FileType = test.Body["format"].(string)
	postFile.OriginalTitle = title
	postFile.Size = 0
	postFile.Ancestors = append(folder.Ancestors, postFile.FolderID)

	meta := postFile.Meta
	meta.DateCreation = time.Now()
	meta.Creator = claims.Subject
	meta.Read = folder.Meta.Read
	meta.Write = folder.Meta.Write
	meta.Update = update
	meta.CopTasks = b.RequestID
	postFile.Meta = meta
	err = fileDB.InsertOne(postFile)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in creating stream.", err.Error(), "COP0012")
		return
	}

	// Update parent folder
	err = folderDB.UpdateFiles(postFile.Id, postFile.FolderID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not update parent folder.", err.Error(), "COP0013")
		return
	}

	// Update ancestore's meta
	err = folderDB.UpdateMetaAncestors(postFile.Ancestors, claims.Subject)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not update ancestore's meta.", err.Error(), "COP0014")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(b)
}

// using task id of a request first checks if task in completed then proceeds to dowload data to db if so
func GetStatus(w http.ResponseWriter, r *http.Request) {

	//get claims
	claims, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve claims.", err.Error(), "COP0015")
		return
	}

	params := mux.Vars(r)
	taskID := params["id"]
	service := params["service"]
	base := ""

	if service != "cds" && service != "ads" {
		utils.RespondWithError(w, http.StatusBadRequest, "Destination doesn't exist.", err.Error(), "COP0016")
		return
	}

	if service == "cds" {
		base = "https://cds.climate.copernicus.eu/api/v2/tasks/"
	}

	if service == "ads" {
		base = "https://ads.atmosphere.copernicus.eu/api/v2/tasks/"
	}
	req, err := http.Get(base + string(taskID))

	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Destination doesn't exist.", err.Error(), "COP0017")
		return
	}

	//basic auth should be refactored into consts
	req.Header.Set("Content-Type", "application/json")

	if service == "cds" {
		req.Request.SetBasicAuth(CDS_UID, CDS_KEY)
	}

	if service == "ads" {
		req.Request.SetBasicAuth(ADS_UID, ADS_KEY)
	}

	resp, err := http.DefaultClient.Do(req.Request)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not complete request", err.Error(), "COP0018")
	}

	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Error in Copernicus Response", err.Error(), "COP0019")
	}

	var b models.CopernicusResponse

	json.Unmarshal(body, &b)

	if b.State != "completed" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(b)
	} else {
		//here we have to save the dataset via the download link to our database
		//split data into parts go routines etc.
		//then we will deal with copernicus data as a normal file in our db
		w.Header().Set("Content-Type", "application/json")

		var postFile models.File

		//get relevant file from db
		postFile, err = fileDB.GetOneByTaskID(taskID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error in retrieving file from database.", err.Error(), "COP0020")
			return
		}
		// Set bucket ID; this is the "copernicus bucket" where all the cop data is saved

		postFile.FolderID = COPERNICUS_BUCKET_ID

		downreq, err := http.Get(b.Location)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Destination doesn't exist.", err.Error(), "COP0021")
			return
		}

		downresp, err := http.DefaultClient.Do(downreq.Request)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Error in Dataset location", err.Error(), "COP0022")
		}

		defer downresp.Body.Close()

		//calculate download size of complete file
		downloadSize, err := strconv.Atoi(downresp.Header.Get("Content-Length"))
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not calculate size of dataset.", err.Error(), "COP0023")
			return
		}

		// Calculate the number of parts and the optimal part size
		totalPartsCount := int(math.Ceil(float64(downloadSize) / float64(partSize)))

		//reach copernicus bucket in db
		folder, err := folderDB.GetOneByID(postFile.FolderID)
		if err != nil || folder.Id == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not find parent folder.", err.Error(), "COP0024")
			return
		}

		//update files' missing info and meta
		update := models.Updated{
			Date: time.Now(),
			User: claims.Subject,
		}

		postFile.Total = totalPartsCount
		meta := postFile.Meta
		meta.Update = update
		postFile.Meta = meta

		//update file
		_, err = fileDB.UpdateWithId(postFile)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Could update file entry.", err.Error(), "COP0025")
		}
		// Create a slice of channels to hold upload results
		partsCh := make(chan models.Part, totalPartsCount)

		// Create a waitgroup to synchronize uploads
		var wg sync.WaitGroup

		//iterate through the parts
		for i := int(1); i <= totalPartsCount; i++ {

			// Read part data into a buffer
			partBuffer := make([]byte, partSize)
			n, err := io.ReadFull(downresp.Body, partBuffer)
			//check if error is not end of stream
			if err != nil && !(errors.Is(err, io.ErrUnexpectedEOF)) {
				break
			}

			// Create a new reader for this part
			partReader := bytes.NewReader(partBuffer[:n])
			//partBytes := partBuffer[:n]
			size := len(partBuffer[:n])
			if size == 0 {
				return
			}
			//fmt.Println(size, partSize)
			wg.Add(1)
			go func(partReader io.Reader, item int) {
				//post file

				defer wg.Done()
				// Make updates in document
				var filePart models.Part
				partId, err := utils.GenerateUUID()
				if err != nil {
					utils.RespondWithError(w, http.StatusInternalServerError, "Error in creating part's ID.", err.Error(), "COP0026")
					return
				}

				//get part info
				filePart.Id = partId
				filePart.FileID = postFile.Id
				filePart.Size = int64(size)
				filePart.PartNumber = item
				// Upload part
				uploadInfo, err := storage.PostPart(COPERNICUS_BUCKET_ID, partId, partReader, int64(size), minio.PutObjectOptions{})
				if err != nil {
					utils.RespondWithError(w, http.StatusBadRequest, "Could post part of file.", err.Error(), "COP0027")
					return
				}

				filePart.UploadInfo = uploadInfo

				// insert part to db
				err = partsDB.InsertOne(filePart)
				if err != nil {
					utils.RespondWithError(w, http.StatusBadRequest, "Could not insert part document.", err.Error(), "COP0028")
					return
				}
				//add part to parts channel
				partsCh <- filePart

			}(partReader, i)

		}

		// Wait for all parts to finish uploading
		go func() {
			wg.Wait()
			close(partsCh)
		}()

		//Update file size
		_, err = fileDB.UpdateFileSize(postFile.Id, downloadSize)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Could not update file's size.", err.Error(), "COP0029")
			return
		}

		// Update Ancestor sizes
		err = folderDB.UpdateAncestorSize(postFile.Ancestors, int64(downloadSize), true)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Could not update ancestore's meta.", err.Error(), "COP0030")
			return
		}

		json.NewEncoder(w).Encode(postFile)
		w.WriteHeader(http.StatusAccepted)
	}

}
