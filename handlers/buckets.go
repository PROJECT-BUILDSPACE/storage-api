package handlers

import (
	"encoding/json"
	"net/http"

	// "github.com/swaggo/http-swagger"

	// "strings"

	"github.com/gorilla/mux"
	"github.com/isotiropoulos/storage-api/globals"
	models "github.com/isotiropoulos/storage-api/models"
	"github.com/isotiropoulos/storage-api/utils"
	// "BUILDSPACE-api/utils"
	// "github.com/gorilla/mux"
)

// MakeBucket handles the /bucket POST request.
// @Summary Create bucket.
// @Description Use a Bucket model to create a new bucket.
// @Tags Buckets
// @Accept json
// @Produce json
// @Param body body models.Bucket true "Bucket payload"
// @Success 200 {object} models.Bucket "OK"
// @Failure 400 {object} models.ErrorReport "Bad Request"
// @Failure 500 {object} models.ErrorReport "Internal Server Error"
// @Router /bucket [post]
// @Security BearerAuth
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
	info, err := globals.Storage.MakeBucket(req)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not create bucket.", err.Error(), "BUC0002")
		return
	}

	// Check if bucket folder exists
	_, err = globals.FolderDB.GetOneByID(req.Id)
	// if error then folder id doesn't exist!!
	if err != nil {
		// Make bucket's main folder
		folderData := models.PostFolderBody{
			FolderName:  req.Name,
			Parent:      "",
			Description: "Main folder.",
		}

		postFolder := utils.CreateFolder(folderData, req.Id, []string{}, req.Id)

		err = globals.FolderDB.InsertOne(postFolder)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not create bucket.", err.Error(), "BUC0003")
			return
		}
	}

	json.NewEncoder(w).Encode(info)
}

// DeleteBucket handles the /bucket/{id} delete request.
// @Summary Delete bucket with all contents.
// @Description Delete a bucket based on it's ID.
// @Accept json
// @Produce json
// @Tags Buckets
// @Param id path string true "Bucket Id"
// @Success 200 {object} models.Bucket "OK"
// @Failure 400 {object} models.ErrorReport "Bad Request"
// @Failure 500 {object} models.ErrorReport "Internal Server Error"
// @Router /bucket/{id} [delete]
// @Security BearerAuth
func DeleteBucket(w http.ResponseWriter, r *http.Request) {
	mode := r.Header.Get("X-Mode")
	if mode != "normal" {
		utils.RespondWithError(w, http.StatusForbidden, "User not allowed", "User with shared rights can't perform this action", "BUC0001")
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Gets params
	params := mux.Vars(r)
	bucketId := params["id"]

	// Delete Bucket
	err := globals.Storage.DeleteBucket(bucketId)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not delete bucket.", err.Error(), "BUC0004")
		return
	}

	// Delete nested folders
	err = globals.FolderDB.DeleteManyWithAncestore(bucketId)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not delete bucket's root folder.", err.Error(), "BUC0005")
		return
	}

	// Delete root folder
	err = globals.FolderDB.DeleteOneByID(bucketId)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not delete bucket's root folder.", err.Error(), "BUC0008")
		return
	}

	// Delete parts
	err = globals.PartsDB.DeleteManyWithBucket(bucketId)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not delete bucket's parts.", err.Error(), "BUC0006")
		return
	}

	// Delete files
	err = globals.FileDB.DeleteManyWithAncestore(bucketId)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not delete bucket's files.", err.Error(), "BUC0007")
		return
	}

	json.NewEncoder(w).Encode(models.Bucket{
		Id: bucketId,
	})
}
