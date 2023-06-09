package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/isotiropoulos/storage-api/models"
	"github.com/isotiropoulos/storage-api/utils"
	"go.mongodb.org/mongo-driver/bson"
)

// PostFolder handles the /folder post request.
// @Summary Create a new folder.
// @Description Use a Folder model as a payload to create a new folder. Essential fields are meta.title (folder's name) and parent (location).
// @Accept json
// @Produce json
// @Tags Folders
// @Param body body models.Folder true "Folder payload"
// @Param X-Group-Id header string true "Group ID"
// @Success 200 {object} models.Bucket "OK"
// @Failure 400 {object} models.ErrorReport "Bad Request"
// @Failure 409 {object} models.ErrorReport "Conflict"
// @Failure 500 {object} models.ErrorReport "Internal Server Error"
// @Router /folder [post]
// @Security BearerAuth
func PostFolder(w http.ResponseWriter, r *http.Request) {

	// Resolve Claims
	claims, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve claims.", err.Error(), "FOL0001")
		return
	}

	// Resolve Request
	w.Header().Set("Content-Type", "application/json")
	var parentFolder models.Folder
	var folder models.Folder
	err = json.NewDecoder(r.Body).Decode(&folder)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve request.", err.Error(), "FOL0002")
		return
	}

	// Check if name already exists
	foldersCursor, err := folderDB.GetCursorByParent(folder.Parent)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not obtain siblings.", err.Error(), "FOL0003")
		return
	}
	defer foldersCursor.Close(context.Background())

	for foldersCursor.Next(context.Background()) {
		var result bson.M
		var inFolder models.Folder
		if err := foldersCursor.Decode(&result); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve cursor.", err.Error(), "FOL0004")
			return
		}
		bsonBytes, _ := bson.Marshal(result)
		bson.Unmarshal(bsonBytes, &inFolder)
		if inFolder.Meta.Title == folder.Meta.Title {
			utils.RespondWithError(w, http.StatusConflict, "Folder Exists.", err.Error(), "FOL0005")
			return
		}
	}

	// Create folder body
	folder.Files = make([]string, 0)
	folder.Meta.Creator = claims.Subject
	folderID, err := utils.GenerateUUID()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create UUID.", err.Error(), "FOL0006")
		return
	}
	folder.Id = folderID
	folder.Meta.DateCreation = time.Now().Unix()
	folder.Meta.Update = append(folder.Meta.Update, models.Updated{
		User: claims.Subject,
		Date: time.Now(),
	})
	var ancestors []string

	if folder.Parent != "" {
		//Get the folder
		object, err := folderDB.GetOneByID(folder.Parent)
		if err != nil {
			utils.RespondWithError(w, http.StatusNotFound, "Parent folder don't exist.", err.Error(), "FOL0007")
			return
		}
		bsonBytes, _ := bson.Marshal(object)
		bson.Unmarshal(bsonBytes, &parentFolder)
		ancestors = parentFolder.Ancestors

		folder.Meta.Read = object.Meta.Read
		folder.Meta.Write = object.Meta.Write

		ancestors = append(ancestors, folder.Parent)

	} else {
		ancestors = nil
	}

	folder.Ancestors = ancestors

	err = folderDB.InsertOne(folder)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could post folder.", err.Error(), "FOL0008")
		return
	}

	if folder.Parent != "" {

		newFolders := parentFolder.Folders
		if newFolders != nil {
			newFolders = append(newFolders, folderID)
		} else {
			newFolders = []string{folderID}
		}

		parentFolder.Folders = newFolders
		// parentFolder.Children = newFolders

		// Update file
		_, err = folderDB.UpdateWithId(parentFolder)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Could not update parent folder.", err.Error(), "FOL0009")
			return
		}

		// Update Ancestore's Meta
		err = folderDB.UpdateMetaAncestors(ancestors, claims.Subject)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Could not update ancestore's meta.", err.Error(), "FOL0010")
			return
		}

	}
	json.NewEncoder(w).Encode(folder)
}

// DeleteFolder handles the /folder/{id} delete request.
// @Summary Delete folder by id.
// @Description Pass folder's id to delete it. Nested items (either files or folders) will be deleted as well.
// @Accept json
// @Produce json
// @Tags Folders
// @Param id path string true "Folder payload"
// @Param X-Group-Id header string true "Group ID"
// @Success 200 {object} models.Folder "OK"
// @Failure 400 {object} models.ErrorReport "Bad Request"
// @Failure 404 {object} models.ErrorReport "Not Found"
// @Failure 409 {object} models.ErrorReport "Conflict"
// @Failure 500 {object} models.ErrorReport "Internal Server Error"
// @Router /folder/{id} [delete]
// @Security BearerAuth
func DeleteFolder(w http.ResponseWriter, r *http.Request) {

	// Resolve Claims
	claims, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve claims.", err.Error(), "FOL0011")
		return
	}

	// Get params
	w.Header().Set("Content-Type", "application/json")
	var folder models.Folder
	var parentFolder models.Folder
	params := mux.Vars(r) // Gets params

	// Retrieve folder from DB
	object, err := folderDB.GetOneByID(params["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Folder don't exist.", err.Error(), "FOL0012")
		return
	}

	bsonBytes, err := bson.Marshal(object)
	bson.Unmarshal(bsonBytes, &folder)

	// Delete folder from DB
	err = folderDB.DeleteOneByID(params["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not delete folder.", err.Error(), "FOL0013")
		return
	}

	// Update Parent Folder
	// Get the Parent folder
	object, err = folderDB.GetOneByID(folder.Parent)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Could not find parent folder.", err.Error(), "FOL0014")
		return
	}
	bsonBytes, err = bson.Marshal(object)
	bson.Unmarshal(bsonBytes, &parentFolder)

	// Remove the deleted folder (but keep the order)
	newFolders := utils.RemoveFromSlice(parentFolder.Folders, params["id"])

	object.Folders = newFolders

	// Update Parent
	_, err = folderDB.UpdateWithId(object)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in updating folder.", err.Error(), "FOL0015")
		return
	}

	// Update Ancestore's Meta
	err = folderDB.UpdateMetaAncestors(folder.Ancestors, claims.Subject)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in updating ancestore's meta.", err.Error(), "FOL0016")
		return
	}

	// Delete Nested Items
	// Delete Folders
	err = folderDB.DeleteManyWithAncestore(params["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in deleting nested folders.", err.Error(), "FOL0017")
		return
	}

	// Delete Files
	cursor, err := fileDB.GetCursorByAncestors(params["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error when deleting nested files.", err.Error(), "FOL0018")
		return
	}

	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {

		var result bson.M
		var file models.File
		if err = cursor.Decode(&result); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Could not resolve cursor.", err.Error(), "FOL0019")
			return
		}

		bsonBytes, _ := bson.Marshal(result)
		bson.Unmarshal(bsonBytes, &file)
		if err = fileDB.DeleteOneByID(file.Id); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error in deleting file.", err.Error(), "FOL0020")
			return
		}

		err = streamDB.DeleteManyWithFile(file.Id)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Could not delete stream.", err.Error(), "FOL0036")
			return
		}

		// Remove Object from MINIO
		groupId := r.Header.Get("X-Group-Id")
		if err = storage.DeleteFile(file.Id, groupId); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error in deleting file.", err.Error(), "FOL0021")
			return
		}
	}
	json.NewEncoder(w).Encode(folder)
}

// GetFolder handles the /folder?id={id} get request.
// @Summary Get folder by id.
// @Description Get a folders meta data by the ID. Pass the ID in a query parameter.
// @Accept json
// @Produce json
// @Tags Folders
// @Param id query string true "Folder ID"
// @Param X-Group-Id header string true "Group ID"
// @Success 200 {object} models.Folder "OK"
// @Failure 400 {object} models.ErrorReport "Bad Request"
// @Failure 404 {object} models.ErrorReport "Not Found"
// @Failure 409 {object} models.ErrorReport "Conflict"
// @Failure 500 {object} models.ErrorReport "Internal Server Error"
// @Router /folder [get]
// @Security BearerAuth
func GetFolder(w http.ResponseWriter, r *http.Request) {

	folderID := r.FormValue("id")

	var folder models.Folder
	var err error

	if folderID == "" {
		// Get root
		groupId := r.Header.Get("X-Group-Id")
		folder, err = folderDB.GetOneByID(groupId)
		if err != nil {
			utils.RespondWithError(w, http.StatusNotFound, "Could not get folder.", err.Error(), "FOL0022")
			return
		}
	} else {
		// Get folder by folder ID
		folder, err = folderDB.GetOneByID(folderID)
		if err != nil {
			utils.RespondWithError(w, http.StatusNotFound, "Could not get folder.", err.Error(), "FOL0023")
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(folder)
}

// UpdateFolder handles the /folder put request.
// @Summary Update folder by ID.
// @Description Update a folders meta data by the ID. Pass the Folder model with the updates that are needed.
// @Accept json
// @Produce json
// @Tags Folders
// @Param body body models.Folder true "Update body"
// @Param X-Group-Id header string true "Group ID"
// @Success 200 {object} models.Folder "OK"
// @Failure 400 {object} models.ErrorReport "Bad Request"
// @Failure 404 {object} models.ErrorReport "Not Found"
// @Failure 409 {object} models.ErrorReport "Conflict"
// @Failure 500 {object} models.ErrorReport "Internal Server Error"
// @Router /folder [put]
// @Security BearerAuth
func UpdateFolder(w http.ResponseWriter, r *http.Request) {

	// Resolve Claims
	claims, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve claims.", err.Error(), "FOL0024")
		return
	}

	var folder models.Folder

	var updateFolder models.Folder
	err = json.NewDecoder(r.Body).Decode(&updateFolder)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve request.", err.Error(), "FOL0025")
		return
	}

	// Check title
	parentID := ""
	if updateFolder.Parent != "" {
		parentID = updateFolder.Parent
	} else {
		parentID = r.Header.Get("X-Group-Id")
	}

	// Check if title already exists
	// First get current title
	currentDoc, err := folderDB.GetOneByID(updateFolder.Id)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Folder trying to get updated doesn't exist.", err.Error(), "FOL0026")
		return
	}

	if updateFolder.Meta.Title != currentDoc.Meta.Title {
		// Check if title is illegal
		foldersCursor, err := folderDB.GetCursorByParent(parentID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Could not obtain siblings.", err.Error(), "FOL0027")
			return
		}
		defer foldersCursor.Close(context.Background())

		for foldersCursor.Next(context.Background()) {
			var result bson.M
			var inFolder models.Folder
			if err := foldersCursor.Decode(&result); err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve cursor.", err.Error(), "FOL0028")
				return
			}
			bsonBytes, _ := bson.Marshal(result)
			bson.Unmarshal(bsonBytes, &inFolder)
			if inFolder.Meta.Title == updateFolder.Meta.Title {
				utils.RespondWithError(w, http.StatusConflict, "Folder Exists.", "Cannot rename folder to this name, since it is already taken.", "FOL0029")
				return
			}
		}
	}

	// Update folder
	updateFolder.Meta.Update = append(currentDoc.Meta.Update, models.Updated{
		User: claims.Subject,
		Date: time.Now(),
	})
	folder, err = folderDB.UpdateWithId(updateFolder)
	if err != nil {
		utils.RespondWithError(w, http.StatusConflict, "Could not update folder.", err.Error(), "FOL0030")
		return
	}

	// Update ancestores meta
	err = folderDB.UpdateMetaAncestors(folder.Ancestors, claims.Subject)
	if err != nil {
		utils.RespondWithError(w, http.StatusConflict, "Could not update folder's ancestores.", err.Error(), "FOL0031")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(folder)
}

// GetFolderItems handles the /folder/list?id={id} get request.
// @Summary List folder's items.
// @Description Get lists of files and folders in a specific folder, by id. Result is a FolderList model
// @Accept json
// @Produce json
// @Tags Folders
// @Param id query string true "Folder ID"
// @Param X-Group-Id header string true "Group ID"
// @Success 200 {object} models.FolderList "OK"
// @Failure 409 {object} models.ErrorReport "Conflict"
// @Failure 500 {object} models.ErrorReport "Internal Server Error"
// @Router /folder/list [get]
// @Security BearerAuth
func GetFolderItems(w http.ResponseWriter, r *http.Request) {

	folderID := r.FormValue("id")

	if folderID == "" {
		// Get root
		folderID = r.Header.Get("X-Group-Id")
	}

	w.Header().Set("Content-Type", "application/json")

	var retObject models.FolderList
	retObject.Files = make(map[string]models.Meta)
	retObject.Folders = make(map[string]models.Meta)

	// Retrieve files from DB
	fileCursor, err := fileDB.GetCursorByFolderID(folderID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not get files in folder.", err.Error(), "FOL0032")
		return
	}
	defer fileCursor.Close(context.Background())

	for fileCursor.Next(context.Background()) {

		var result bson.M
		var file models.File
		if err = fileCursor.Decode(&result); err != nil {
			utils.RespondWithError(w, http.StatusConflict, "Could not resolve cursor.", err.Error(), "FOL0033")
			return
		}

		bsonBytes, _ := bson.Marshal(result)
		bson.Unmarshal(bsonBytes, &file)
		retObject.Files[file.Id] = file.Meta
	}

	// Retrieve folders from DB
	folderCursor, err := folderDB.GetCursorByParent(folderID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not get children folders in folder.", err.Error(), "FOL0034")
		return
	}

	defer folderCursor.Close(context.Background())

	for folderCursor.Next(context.Background()) {

		var result bson.M
		var folder models.Folder
		if err = folderCursor.Decode(&result); err != nil {
			utils.RespondWithError(w, http.StatusConflict, "Could not resolve cursor.", err.Error(), "FOL0035")
			return
		}

		bsonBytes, _ := bson.Marshal(result)
		bson.Unmarshal(bsonBytes, &folder)

		retObject.Folders[folder.Id] = folder.Meta
	}

	json.NewEncoder(w).Encode(retObject)
}
