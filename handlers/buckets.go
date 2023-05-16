package handlers

import (
	"encoding/json"
	"net/http"

	// "strings"

	"github.com/gorilla/mux"
	models "github.com/isotiropoulos/storage-api/models"
	"github.com/isotiropoulos/storage-api/utils"
	// "BUILDSPACE-api/utils"
	// "github.com/gorilla/mux"
)

// MakeBucket is to make a new bucket
func MakeBucket(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Accept", "application/json")

	decoder := json.NewDecoder(r.Body)
	var req models.Bucket

	// Resolve Request
	err := decoder.Decode(&req)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve request.", err.Error(), "BUC0001")
		return
	}

	// Make Bucket
	info, err := storage.MakeBucket(req)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not create bucket.", err.Error(), "BUC0002")
		return
	}

	// Check if bucket folder exists
	_, err = folderDB.GetOneByID(req.Name)
	// if error then folder id doesn't exist!!
	if err != nil {
		// Make bucket's main folder
		folderData := models.PostFolderBody{
			FolderName:  req.Name,
			Parent:      "",
			Description: "Main folder.",
		}

		postFolder := utils.CreateFolder(folderData, req.Name, nil, req.Name)

		err = folderDB.InsertOne(postFolder)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not create bucket.", err.Error(), "BUC0003")
			return
		}
	}
	json.NewEncoder(w).Encode(info)
}

// DeleteBucket is to delete a bucket
func DeleteBucket(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	// Gets params
	params := mux.Vars(r)
	bucketId := params["id"]

	// Delete Bucket
	err := storage.DeleteBucket(bucketId)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not delete bucket.", err.Error(), "BUC0004")
		return
	}

	// Delete nested folders
	err = folderDB.DeleteManyWithAncestore(bucketId)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not delete bucket's root folder.", err.Error(), "BUC0005")
		return
	}

	// Delete root folder
	err = folderDB.DeleteOneByID(bucketId)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not delete bucket's root folder.", err.Error(), "BUC0005")
		return
	}

	// Delete nested files
	err = fileDB.DeleteManyWithAncestore(bucketId)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not delete bucket's files.", err.Error(), "BUC0006")
		return
	}
	json.NewEncoder(w).Encode(models.Bucket{
		Name: bucketId,
	})
}
