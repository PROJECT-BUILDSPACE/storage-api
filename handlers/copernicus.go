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

// GetList handles the /copernicus/{service}/getall GET request.
// @Summary Get a list of all available datasets related to a specific service.
// @Description This is the endopoint to get a list of all available datasets of a service.
// @Description Currently supported services are "ads" and "cds"
// @Tags Copernicus
// @Produce json
// @Param service path string true "Service (currently available 'ads' and 'cds')"
// @Success 202 {object} array "Accepted"
// @Failure 400 {object} models.ErrorReport "Bad Request"
// @Failure 500 {object} models.ErrorReport "Internal Server Error"
// @Failure 503 {object} models.ErrorReport "Service Anavailable"
// @Router /copernicus/{service}/getall [get]
func GetList(w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)
	service := params["service"]
	base := ""

	if service != "cds" && service != "ads" {
		utils.RespondWithError(w, http.StatusServiceUnavailable, "Service not supported.", "Provide service 'cds' or 'ads'", "COP0016")
		return
	}

	if service == "cds" {
		base = "https://cds.climate.copernicus.eu/api/v2/resources/"
	}

	if service == "ads" {
		base = "https://ads.atmosphere.copernicus.eu/api/v2/resources/"
	}

	req, err := http.Get(base)

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
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in Response Body", err.Error(), "COP0003")
	}

	//encode response into json
	var a []string
	json.Unmarshal(body, &a)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(a)
}

// GetForm handles the /copernicus/{service}/getform/{id} GET request.
// @Summary Get the form of a dataset that is related to a specific service.
// @Description A form is a set of rules indicating which parameters are neccessary for the dataset to be retrieved.
// @Description Please note that some parameters cannot be used with other. The selection rules are also included in the forms.
// @Tags Copernicus
// @Produce json
// @Param service path string true "Service (currently available 'ads' and 'cds')"
// @Param id path string true "ID of the dataset of interest"
// @Success 200 {object} models.Form "OK"
// @Failure 400 {object} models.ErrorReport "Bad Request"
// @Failure 500 {object} models.ErrorReport "Internal Server Error"
// @Failure 503 {object} models.ErrorReport "Service Anavailable"
// @Router /copernicus/{service}/getform/{id} [get]
func GetForm(w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)
	datasetID := params["id"]

	w.Header().Set("Content-Type", "application/json")
	service := params["service"]
	base := ""

	if service != "cds" && service != "ads" {
		utils.RespondWithError(w, http.StatusServiceUnavailable, "Service not supported.", "Provide service 'cds' or 'ads'", "COP0016")
		return
	}

	if service == "cds" {
		base = "http://datastore.copernicus-climate.eu/c3s/published-forms/c3sprod/"
	}

	//TO BE CHANGED!! ADS CURRENTLY UNAVAILABLE
	if service == "ads" {
		base = "http://datastore.copernicus-climate.eu/c3s/published-forms/c3sprod/"
	}

	req, err := http.Get(base + string(datasetID) + "/form.json")

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
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not handle response", err.Error(), "FORM003")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(myform)
}

// PostDataset handles the /copernicus/{service}/dataset POST request.
// @Summary Post a request for a specific dataset (create a Copernicus task that will make a dataset available for download).
// @Description The request can be specified by the body of the request using the parameters of the dataset.
// @Description Please note that some parameters cannot be used with other. For the dataset parameters' rules consider the dataset's form.
// @Tags Copernicus
// @Accept json
// @Produce json
// @Param service path string true "Service (currently available 'ads' and 'cds')"
// @Param body body models.CopernicusInput  true "Request body"
// @Success 200 {object} models.CopernicusResponse "OK"
// @Failure 400 {object} models.ErrorReport "Bad Request"
// @Failure 500 {object} models.ErrorReport "Internal Server Error"
// @Failure 503 {object} models.ErrorReport "Service Anavailable"
// @Router /copernicus/{service}/dataset [post]
func PostDataset(w http.ResponseWriter, r *http.Request) {

	// forms a request for a single dataset CLIMATE
	// save any VALID requests to database as a file
	// + faulty requests handling
	// also task id is saved

	claims, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve claims.", err.Error(), "COP0004")
		return
	}

	var reqBody models.CopernicusInput
	err = json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve request.", err.Error(), "COP0005")
		return
	}

	name := reqBody.DatasetName
	//encode input to json
	content, err := json.Marshal(reqBody.Body)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve content.", err.Error(), "COP0006")
		return
	}

	params := mux.Vars(r)
	service := params["service"]
	base := ""

	if service != "cds" && service != "ads" {
		utils.RespondWithError(w, http.StatusServiceUnavailable, "Service not supported.", err.Error(), "COP0016")
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
		utils.RespondWithError(w, http.StatusInternalServerError, "Could Read Copernicus Response", err.Error(), "COP0009")
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
	postFile.FileType = reqBody.Body["format"].(string)
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
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(b)
}

// GetStatus handles the /copernicus/{service}/dataset/{id} GET request.
// @Summary Download dataset (if the Copernicus task is completed).
// @Description This endpoint checks the status of a Copernicus task and if it is completed it downloads the file and stores it in MinIO (under the Copernicus public bucket).
// @Description In case of uncompleted task, the response is a json specifying the current status.
// @Tags Copernicus
// @Produce json
// @Param service path string true "Service (currently available 'ads' and 'cds')"
// @Param id path string true "ID of the dataset to be downloaded"
// @Success 200 {object} models.File "OK"
// @Failure 400 {object} models.ErrorReport "Bad Request"
// @Failure 500 {object} models.ErrorReport "Internal Server Error"
// @Failure 503 {object} models.ErrorReport "Service Anavailable"
// @Router /copernicus/{service}/dataset/{id} [get]
func GetStatus(w http.ResponseWriter, r *http.Request) {
	// using task id of a request first checks if task in completed then proceeds to dowload data to db if so
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
		utils.RespondWithError(w, http.StatusServiceUnavailable, "Service not supported.", err.Error(), "COP0016")
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
		utils.RespondWithError(w, http.StatusInternalServerError, "Destination doesn't exist.", err.Error(), "COP0017")
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
		w.WriteHeader(http.StatusOK)
	}

}
