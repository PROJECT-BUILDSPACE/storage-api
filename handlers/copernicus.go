package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/isotiropoulos/storage-api/globals"
	"github.com/isotiropoulos/storage-api/models"
	"github.com/isotiropoulos/storage-api/utils"
	"go.mongodb.org/mongo-driver/bson"

	"time"

	"github.com/gorilla/mux"
)

// GetList handles the /copernicus/collections GET request.
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
// @Security BearerAuth
func GetList(w http.ResponseWriter, r *http.Request) {
	// var collections CDSModels.CollectionList

	collections, err := globals.CopernicusClient.GetCollections()
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not retrieve all collections", err.Error(), "COP0001")
		return
	}

	response, err := json.Marshal(collections)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not marshal CollectionList to JSON", err.Error(), "COP0002")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write(response)
}

// GetForm handles the /copernicus/form/{id} GET request.
// @Summary Get the form of a dataset that is related to a specific service.
// @Description A form is a set of rules indicating which parameters are neccessary for the dataset to be retrieved.
// @Description Please note that some parameters cannot be used with other. The selection rules are also included in the forms.
// @Tags Copernicus
// @Produce json
// @Param service path string true "Service (currently available 'ads' and 'cds')"
// @Param id path string true "ID of the dataset of interest"
// @Success 200 {object} formModel "OK"
// @Failure 400 {object} models.ErrorReport "Bad Request"
// @Failure 500 {object} models.ErrorReport "Internal Server Error"
// @Failure 503 {object} models.ErrorReport "Service Anavailable"
// @Router /copernicus/{service}/getform/{id} [get]
// @Security BearerAuth
func GetForm(w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)
	collectionID := params["id"]

	form, err := globals.CopernicusClient.GetCollectionForm(collectionID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not retrieve form", err.Error(), "FORM003")
		return
	}

	response, err := json.Marshal(form)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not marshal list of Forms to JSON", err.Error(), "COP0002")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
	// json.NewEncoder(w).Encode(response)

}

// PostDataset handles the /copernicus/dataset POST request.
// @Summary Post a request for a specific dataset (create a Copernicus task that will make a dataset available for download).
// @Description The request can be specified by the body of the request using the parameters of the dataset.
// @Description Please note that some parameters cannot be used with other. For the dataset parameters' rules consider the dataset's form.
// @Tags Copernicus
// @Accept json
// @Produce json
// @Param service path string true "Service (currently available 'ads' and 'cds')"
// @Param body body models.CopernicusInput  true "Request body"
// @Success 200 {object} models.File "OK"
// @Failure 400 {object} models.ErrorReport "Bad Request"
// @Failure 500 {object} models.ErrorReport "Internal Server Error"
// @Failure 503 {object} models.ErrorReport "Service Anavailable"
// @Router /copernicus/{service}/dataset [post]
// @Security BearerAuth
func PostDataset(w http.ResponseWriter, r *http.Request) {

	// forms a request for a single dataset
	// save any VALID requests to database as a file
	// + faulty requests handling
	// also task id is saved
	var postFile models.File

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

	// Check if already downloaded
	fprint, err := utils.GenerateRequestFingerprint(reqBody)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in creating file's fingerprint.", err.Error(), "COP0011")
		return
	}

	postFile, err = globals.FileDB.GetOneByFingerprint(fprint)
	if err == nil && postFile.Id != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(postFile)
		return
	}

	name := reqBody.DatasetName
	process, err := globals.CopernicusClient.CreateProcess(name, reqBody.Body)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not create Process.", err.Error(), "COP0010")
		return
	}

	postFile.FolderID = globals.COPERNICUS_BUCKET_ID
	//get bucket
	folder, err := globals.FolderDB.GetOneByID(postFile.FolderID)
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
	meta.Title = title
	// postFile.CopernicusDetails = copDetails
	postFile.Meta = meta
	err = globals.FileDB.InsertOne(postFile)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in insterting file.", err.Error(), "COP0012")
		return
	}

	copInput := models.CopernicusRecord{
		Id:            fprint,
		FileId:        fileID,
		DatasetName:   reqBody.DatasetName,
		RequestParams: reqBody.Body,
		Details:       process,
	}

	err = globals.CopernicusDB.InsertOne(copInput)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in storing copernicus input.", err.Error(), "COP0035")
		return
	}

	// Update parent folder
	err = globals.FolderDB.UpdateFiles(postFile.Id, postFile.FolderID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not update parent folder.", err.Error(), "COP0013")
		return
	}

	// Update ancestore's meta
	err = globals.FolderDB.UpdateMetaAncestors(postFile.Ancestors, claims.Subject)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not update ancestore's meta.", err.Error(), "COP0014")
		return
	}

	go utils.CheckCopernicusStatus(copInput, claims.Subject)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(postFile)
}

// GetStatus handles the/copernicus/status/dataset/{fileId} GET request.
// @Summary Download dataset (if the Copernicus task is completed).
// @Description This endpoint checks the status of a Copernicus task and if it is completed it downloads the file and stores it in MinIO (under the Copernicus public bucket).
// @Description In case of uncompleted task, the response is a json specifying the current status.
// @Tags Copernicus
// @Produce json
// @Param fileId path string true "ID of the dataset to be downloaded"
// @Success 200 {object} models.File "OK"
// @Failure 400 {object} models.ErrorReport "Bad Request"
// @Failure 500 {object} models.ErrorReport "Internal Server Error"
// @Failure 503 {object} models.ErrorReport "Service Anavailable"
// @Router /copernicus/{service}/dataset/{id} [get]
// @Security BearerAuth
func CheckStatus(w http.ResponseWriter, r *http.Request) {
	// using task id of a request first checks if task in completed then proceeds to dowload data to db if so
	//get claims
	// claims, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
	// if err != nil {
	// 	utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve claims.", err.Error(), "COP0015")
	// 	return
	// }

	params := mux.Vars(r)
	fileID := params["fileId"]

	copRec, err := globals.CopernicusDB.GetOneByFileID(fileID)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve claims.", err.Error(), "COP0038")
		return
	}

	//get relevant file from db
	postFile, err := globals.FileDB.GetOneByID(copRec.FileId)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in retrieving file from database.", err.Error(), "COP0020")
		return
	}

	if postFile.Size == 0 {
		// Start downloading
		claims, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve claims.", err.Error(), "COP0004")
			return
		}
		utils.CheckCopernicusStatus(copRec, claims.Subject)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(postFile)

}

// GetAvailable handles the /copernicus/{service}/available GET request.
// @Summary Get a list of available Copernicus datasets, based on services.
// @Description This endpoint returns all available-for-download datasets
// @Tags Copernicus
// @Produce json
// @Param service path string true "Service (currently available 'ads' and 'cds', or 'all' for no specific service)"
// @Success 200 {object} []models.CopernicusRecord "OK"
// @Failure 400 {object} models.ErrorReport "Bad Request"
// @Failure 500 {object} models.ErrorReport "Internal Server Error"
// @Failure 503 {object} models.ErrorReport "Service Anavailable"
// @Router /copernicus/{service}/available [get]
// @Security BearerAuth
func GetAvailable(w http.ResponseWriter, r *http.Request) {

	var response []models.CopernicusRecord

	dataCursor, err := globals.CopernicusDB.GetCursorAll()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not open cursor.", err.Error(), "COP0036")
		return
	}
	defer dataCursor.Close(context.Background())

	for dataCursor.Next(context.Background()) {
		var result bson.M
		var record models.CopernicusRecord
		if err := dataCursor.Decode(&result); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve cursor.", err.Error(), "FIL0037")
			return
		}
		bsonBytes, _ := bson.Marshal(result)
		bson.Unmarshal(bsonBytes, &record)
		response = append(response, record)
	}

	json.NewEncoder(w).Encode(response)
	w.WriteHeader(http.StatusOK)
}
