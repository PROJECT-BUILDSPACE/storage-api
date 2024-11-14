package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/isotiropoulos/storage-api/globals"
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
	foldersCursor, err := globals.FolderDB.GetCursorByParent(folder.Parent)
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
			utils.RespondWithError(w, http.StatusConflict, "Folder Exists.", "Folders in the same path must have different names.", "FOL0005")
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
	folder.Meta.DateCreation = time.Now()
	// folder.Meta.Update = append(folder.Meta.Update, models.Updated{
	// 	User: claims.Subject,
	// 	Date: time.Now(),
	// })
	folder.Meta.Update.User = claims.Subject
	folder.Meta.Update.Date = time.Now()

	var ancestors []string

	if folder.Parent != "" {
		//Get the folder
		object, err := globals.FolderDB.GetOneByID(folder.Parent)
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
	folder.Level = len(ancestors)
	folder.Size = 0
	folder.Folders = make([]string, 0)

	err = globals.FolderDB.InsertOne(folder)
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
		_, err = globals.FolderDB.UpdateWithId(parentFolder)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Could not update parent folder.", err.Error(), "FOL0009")
			return
		}

		// Update Ancestore's Meta
		err = globals.FolderDB.UpdateMetaAncestors(ancestors, claims.Subject)
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
	object, err := globals.FolderDB.GetOneByID(params["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Folder don't exist.", err.Error(), "FOL0012")
		return
	}

	bsonBytes, err := bson.Marshal(object)
	bson.Unmarshal(bsonBytes, &folder)

	// Delete folder from DB
	err = globals.FolderDB.DeleteOneByID(params["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not delete folder.", err.Error(), "FOL0013")
		return
	}

	// Update Parent Folder
	// Get the Parent folder
	object, err = globals.FolderDB.GetOneByID(folder.Parent)
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
	_, err = globals.FolderDB.UpdateWithId(object)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in updating folder.", err.Error(), "FOL0015")
		return
	}

	// Update Ancestore's Meta
	err = globals.FolderDB.UpdateMetaAncestors(folder.Ancestors, claims.Subject)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in updating ancestore's meta.", err.Error(), "FOL0016")
		return
	}

	err = globals.FolderDB.UpdateAncestorSize(folder.Ancestors, folder.Size, false)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Could not update parent folder.", err.Error(), "FIL0037")
		return
	}

	// Delete Nested Items
	// Delete Folders
	err = globals.FolderDB.DeleteManyWithAncestore(params["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in deleting nested folders.", err.Error(), "FOL0017")
		return
	}

	// Delete Files
	fileCursor, err := globals.FileDB.GetCursorByAncestors(params["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error when deleting nested files.", err.Error(), "FOL0018")
		return
	}
	defer fileCursor.Close(context.Background())
	for fileCursor.Next(context.Background()) {
		var result bson.M
		var file models.File
		if err = fileCursor.Decode(&result); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Could not resolve cursor.", err.Error(), "FOL0019")
			return
		}

		// Delete File Document
		bsonBytes, _ := bson.Marshal(result)
		bson.Unmarshal(bsonBytes, &file)
		if err = globals.FileDB.DeleteOneByID(file.Id); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error in deleting file.", err.Error(), "FOL0020")
			return
		}

		// Delete Part Documents
		partCursor, err := globals.PartsDB.GetCursorByFileID(file.Id)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error when deleting nested files.", err.Error(), "FOL0036")
			return
		}

		defer partCursor.Close(context.Background())
		for partCursor.Next(context.Background()) {
			groupId := r.Header.Get("X-Group-Id")

			var result2 bson.M
			var part models.Part
			if err = partCursor.Decode(&result2); err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, "Could not resolve cursor.", err.Error(), "FOL0021")
				return
			}

			// Delete Parts from storage
			bsonBytes2, _ := bson.Marshal(result2)
			bson.Unmarshal(bsonBytes2, &part)
			if err = globals.Storage.DeleteFile(part.Id, groupId); err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, "Error in deleting file.", err.Error(), "FOL0022")
				return
			}
		}

		// Delete parts from collection
		err = globals.PartsDB.DeleteManyWithFile(file.Id)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Could not delete file parts.", err.Error(), "BUC0009")
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
// @Success 200 {object} models.Folder "OK"
// @Failure 400 {object} models.ErrorReport "Bad Request"
// @Failure 404 {object} models.ErrorReport "Not Found"
// @Failure 409 {object} models.ErrorReport "Conflict"
// @Failure 500 {object} models.ErrorReport "Internal Server Error"
// @Router /folder [get]
// @Security BearerAuth
func GetFolder(w http.ResponseWriter, r *http.Request) {

	// folderPath := r.FormValue("folderPath")
	queryValues := r.URL.Query()
	folderID := queryValues.Get("id")

	folderPath := queryValues.Get("path")

	var folder models.Folder
	var err error

	if folderID != "" && folderPath != "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Cannot retrieve folder.", "Cannot process both path and ID.", "FOL0023")
		return

	} else if folderID != "" {
		// Get folder by folder ID
		folder, err = globals.FolderDB.GetOneByID(folderID)
		if err != nil {
			utils.RespondWithError(w, http.StatusNotFound, "Could not get folder.", err.Error(), "FOL0023")
			return
		}

	} else {
		folderPath = strings.TrimSuffix(folderPath, "/")
		folderPath := strings.TrimPrefix(folderPath, "/")

		groupId := r.Header.Get("X-Group-Id")

		path := strings.Split(folderPath, "/")

		if len(path) == 1 {
			folder, err = globals.FolderDB.GetOneByID(groupId)
		} else {
			folderName := path[len(path)-1]

			foldersCursor, err := globals.FolderDB.GetCursorByNameLevel(folderName, groupId, len(path)-1)
			defer foldersCursor.Close(context.Background())

			if err != nil {
				utils.RespondWithError(w, http.StatusNotFound, "Could not get folder.", err.Error(), "FOL0023")
				return
			}

			// Create a context that we can cancel
			ctx2, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Channel to receive the result
			resultChan := make(chan models.Folder, 1)

			// WaitGroup to wait for all goroutines to finish
			var wg sync.WaitGroup

			// Start a goroutine for each folder
			for foldersCursor.Next(context.Background()) {
				var result bson.M
				var inFolder models.Folder
				if err := foldersCursor.Decode(&result); err != nil {
					utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve cursor.", err.Error(), "FOL0028")
					return
				}
				bsonBytes, _ := bson.Marshal(result)
				bson.Unmarshal(bsonBytes, &inFolder)

				wg.Add(1)
				go utils.GrainFolders(ctx2, inFolder, path[len(path)-2], resultChan, &wg)
			}

			// Wait for a result from one of the goroutines
			select {
			case folder = <-resultChan:
				// Condition met, cancel all other goroutines
				// cancel()
				close(resultChan)
			case <-time.After(30 * time.Second): // Timeout to prevent infinite waiting
				fmt.Println("Timeout reached, no condition met")
			}
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
	currentDoc, err := globals.FolderDB.GetOneByID(updateFolder.Id)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Folder trying to get updated doesn't exist.", err.Error(), "FOL0026")
		return
	}

	if updateFolder.Meta.Title != currentDoc.Meta.Title {
		// Check if title is illegal
		foldersCursor, err := globals.FolderDB.GetCursorByParent(parentID)
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
	// updateFolder.Meta.Update = append(currentDoc.Meta.Update, models.Updated{
	// 	User: claims.Subject,
	// 	Date: time.Now(),
	// })
	updateFolder.Meta.Update.User = claims.Subject
	updateFolder.Meta.Update.Date = time.Now()

	folder, err = globals.FolderDB.UpdateWithId(updateFolder)
	if err != nil {
		utils.RespondWithError(w, http.StatusConflict, "Could not update folder.", err.Error(), "FOL0030")
		return
	}

	// Update ancestores meta
	err = globals.FolderDB.UpdateMetaAncestors(folder.Ancestors, claims.Subject)
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
	retObject.Files = []models.File{}
	retObject.Folders = []models.Folder{}

	// Retrieve files from DB
	fileCursor, err := globals.FileDB.GetCursorByFolderID(folderID)
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
		retObject.Files = append(retObject.Files, file)
	}

	// Retrieve folders from DB
	folderCursor, err := globals.FolderDB.GetCursorByParent(folderID)
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

		retObject.Folders = append(retObject.Folders, folder)
	}
	json.NewEncoder(w).Encode(retObject)
}

func copySubFile(fileID string, newDest string, bucketFrom string, bucketTo string, nName string, user string) {

	cmBody := models.CopyMoveBody{

		Id:          fileID,
		Destination: newDest,
		NewName:     nName,
	}

	file, err := globals.FileDB.GetOneByID(cmBody.Id)
	if err != nil {
		//utils.RespondWithError(w, http.StatusBadRequest, "File doesn't exist.", err.Error(), "FIL0043")
		return
	}

	var newName string
	if cmBody.NewName == "" {
		newName = file.Meta.Title
	} else {
		newName = cmBody.NewName
	}

	newParent, err := globals.FolderDB.GetOneByID(cmBody.Destination)
	if err != nil {
		//utils.RespondWithError(w, http.StatusBadRequest, "Destination doesn't exist.", err.Error(), "FIL0044")
		return
	}

	// Check if title is illegal
	filesCursor, err := globals.FileDB.GetCursorByFolderID(cmBody.Destination)
	if err != nil {
		//utils.RespondWithError(w, http.StatusInternalServerError, "Could not obtain siblings.", err.Error(), "FIL0045")
		return
	}
	defer filesCursor.Close(context.Background())

	for filesCursor.Next(context.Background()) {
		var result bson.M
		var inFile models.File
		if err := filesCursor.Decode(&result); err != nil {
			//utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve cursor.", err.Error(), "FIL0046")
			return
		}
		bsonBytes, _ := bson.Marshal(result)
		bson.Unmarshal(bsonBytes, &inFile)
		if inFile.Meta.Title == newName {
			//utils.RespondWithError(w, http.StatusConflict, "File Exists.", "Cannot copy file to destination with this name since it is already taken.", "FIL0047")
			return
		}
	}

	//bucketID := r.Header.Get("X-Group-Id")
	newFileId, err := utils.GenerateUUID()
	if err != nil {
		//utils.RespondWithError(w, http.StatusInternalServerError, "Error in generating file's ID.", err.Error(), "FIL0048")
		return
	}
	partsCursor, err := globals.PartsDB.GetCursorByFileID(cmBody.Id)
	if err != nil {
		//utils.RespondWithError(w, http.StatusInternalServerError, "Error in retrieving parts.", err.Error(), "FIL0017")
		return
	}
	defer partsCursor.Close(context.Background())

	for partsCursor.Next(context.Background()) {
		var result bson.M
		var part models.Part
		if err := partsCursor.Decode(&result); err != nil {
			//utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve cursor.", err.Error(), "FIL0068")
			return
		}
		bsonBytes, _ := bson.Marshal(result)
		bson.Unmarshal(bsonBytes, &part)

		newPartID, err := utils.GenerateUUID()
		if err != nil {
			//utils.RespondWithError(w, http.StatusInternalServerError, "Error in generating part's ID.", err.Error(), "FIL0073")
			return
		}

		err = globals.Storage.CopyFile(part.Id, newPartID, bucketFrom, bucketTo)
		if err != nil {
			//utils.RespondWithError(w, http.StatusInternalServerError, "Could not copy file.", err.Error(), "FIL0050")
			return
		}

		part.Id = newPartID
		part.FileID = newFileId
		err = globals.PartsDB.InsertOne(part)
		if err != nil {
			//utils.RespondWithError(w, http.StatusInternalServerError, "Could not copy file.", err.Error(), "FIL0072")
			return
		}
	}

	// Insert file
	file.FolderID = cmBody.Destination
	// Create a new `Updated` struct
	updated := models.Updated{
		User: user,
		Date: time.Now(),
	}
	// file.Meta.Update = []models.Updated{updated}
	file.Meta.Update.User = updated.User
	file.Meta.Update.Date = updated.Date
	file.Id = newFileId
	file.Meta.Title = newName

	ancestors := append(newParent.Ancestors, file.FolderID)
	file.Ancestors = ancestors
	err = globals.FileDB.InsertOne(file)
	if err != nil {
		//utils.RespondWithError(w, http.StatusInternalServerError, "Could not copy file.", err.Error(), "FIL0049")
		return
	}

	// Update New Parent Folder
	newParent.Files = append(newParent.Files, file.Id)
	// newParent.Meta.Update = append(newParent.Meta.Update, updated)
	newParent.Meta.Update.User = updated.User
	newParent.Meta.Update.Date = updated.Date

	_, err = globals.FolderDB.UpdateWithId(newParent)
	if err != nil {
		//utils.RespondWithError(w, http.StatusInternalServerError, "Could not update parent folder.", err.Error(), "FIL0051")
		return
	}

	// Update Ancestores Size
	err = globals.FolderDB.UpdateAncestorSize(ancestors, file.Size, true)
	if err != nil {
		//utils.RespondWithError(w, http.StatusInternalServerError, "Could not update parent folder.", err.Error(), "FIL0074")
		return
	}

}

func copySubFolder(folderID string, newDest string, bucketID string, nName string, user string) string {

	cmBody := models.CopyMoveBody{

		Id:          folderID,
		Destination: newDest,
		NewName:     nName,
	}

	folder, err := globals.FolderDB.GetOneByID(cmBody.Id)
	if err != nil {
		//utils.RespondWithError(w, http.StatusBadRequest, "File doesn't exist.", err.Error(), "FIL0043")
		return ""
	}

	subFolders := folder.Folders
	subFiles := folder.Files

	folder.Folders = make([]string, 0)
	folder.Files = make([]string, 0)

	var newName string
	if cmBody.NewName == "" {
		newName = folder.Meta.Title
	} else {
		newName = cmBody.NewName
	}

	//Get Children and Files lists
	//Get Destination of New Folder
	newParent, err := globals.FolderDB.GetOneByID(cmBody.Destination)
	if err != nil {
		//fmt.Errorf("Destination doesn't exist.: %v", err, "FOL0036")
		return ""
	}

	newFolderId, err := utils.GenerateUUID()
	if err != nil {
		//utils.RespondWithError(w, http.StatusInternalServerError, "Error in generating file's ID.", err.Error(), "FIL0048")
		return ""
	}

	folder.Ancestors = newParent.Ancestors

	// Insert file
	folder.Parent = cmBody.Destination
	// Create a new `Updated` struct
	updated := models.Updated{
		User: user,
		Date: time.Now(),
	}
	// file.Meta.Update = []models.Updated{updated}
	folder.Meta.Update.User = updated.User
	folder.Meta.Update.Date = updated.Date
	folder.Id = newFolderId
	folder.Meta.Title = newName

	ancestors := append(newParent.Ancestors, folder.Parent)
	folder.Ancestors = ancestors
	err = globals.FolderDB.InsertOne(folder)
	if err != nil {
		//utils.RespondWithError(w, http.StatusInternalServerError, "Could not copy file.", err.Error(), "FIL0049")
		return ""
	}

	// Update New Parent Folder
	newParent.Folders = append(newParent.Folders, folder.Id)
	// newParent.Meta.Update = append(newParent.Meta.Update, updated)
	newParent.Meta.Update.User = updated.User
	newParent.Meta.Update.Date = updated.Date

	_, err = globals.FolderDB.UpdateWithId(newParent)
	if err != nil {
		//utils.RespondWithError(w, http.StatusInternalServerError, "Could not update parent folder.", err.Error(), "FIL0051")
		return ""
	}

	// Update Ancestores Size
	err = globals.FolderDB.UpdateAncestorSize(ancestors, folder.Size, true)
	if err != nil {
		//utils.RespondWithError(w, http.StatusInternalServerError, "Could not update parent folder.", err.Error(), "FIL0074")
		return ""
	}

	//COPY SUBFILES
	for _, element := range subFiles {
		if len(newParent.Ancestors) == 0 {
			copySubFile(element, newFolderId, bucketID, newParent.Id, "", user)
		} else {
			copySubFile(element, newFolderId, bucketID, newParent.Ancestors[0], "", user)
		}
	}
	//COPY SUBFOLDERS
	for _, element := range subFolders {
		copySubFolder(element, newFolderId, bucketID, "", user)
	}

	return newFolderId
}

// CopyFolder handles the /folder/copy post request.
// @Summary Copy a folder and all nested files/folders.
// @Description Copy a folder with all nested items.
// @Description This endpoint is also used to share a folder with another organization.
// @Accept json
// @Produce json
// @Tags Folders
// @Param body body models.CopyMoveBody true "Body with Copy details"
// @Success 202 {object} models.FolderList "Accepted"
// @Failure 400 {object} models.ErrorReport "Bad Request"
// @Failure 409 {object} models.ErrorReport "Conflict"
// @Failure 500 {object} models.ErrorReport "Internal Server Error"
// @Router /folder/copy [post]
// @Security BearerAuth
func CopyFolder(w http.ResponseWriter, r *http.Request) {

	// Resolve Claims
	claims, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve claims.", err.Error(), "FOL0036")
		return
	}

	var cmBody models.CopyMoveBody
	err = json.NewDecoder(r.Body).Decode(&cmBody)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve request.", err.Error(), "FOL0037")
		return
	}

	// Chech if title is legal

	folderCursor, err := globals.FolderDB.GetCursorByParent(cmBody.Destination)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not obtain siblings.", err.Error(), "FIL0039")
		return
	}
	defer folderCursor.Close(context.Background())

	for folderCursor.Next(context.Background()) {
		var result bson.M
		var inFile models.File
		if err := folderCursor.Decode(&result); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Could not resolve cursor.", err.Error(), "FOL0040")
			return
		}
		bsonBytes, _ := bson.Marshal(result)
		bson.Unmarshal(bsonBytes, &inFile)
		if inFile.Meta.Title == cmBody.NewName {
			utils.RespondWithError(w, http.StatusConflict, "Folder Exists.", "Cannot copy file to destination with this name since it is already taken.", "FOL0041")
			return
		}
	}

	// Call helper function to recursivelly copy target folder and all sub files and folders

	var newFLID = copySubFolder(cmBody.Id, cmBody.Destination, r.Header.Get("X-Group-Id"), cmBody.NewName, claims.Subject)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	// Load copied folder to generate response

	folder, err := globals.FolderDB.GetOneByID(newFLID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not locate copied folder.", err.Error(), "FOL0042")
		return
	}
	json.NewEncoder(w).Encode(folder) //response should be mongo obj or resp w/ error
}

func moveFolder(w http.ResponseWriter, r *http.Request) {

	// Resolve Claims
	_, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve claims.", err.Error(), "FOL0043")
		return
	}

	var cmBody models.CopyMoveBody
	err = json.NewDecoder(r.Body).Decode(&cmBody)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve request.", err.Error(), "FOL0043")
		return
	}

	// Chech if title is legal

	folderCursor, err := globals.FolderDB.GetCursorByParent(cmBody.Destination)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not obtain siblings.", err.Error(), "FOL0045")
		return
	}
	defer folderCursor.Close(context.Background())

	for folderCursor.Next(context.Background()) {
		var result bson.M
		var inFile models.File
		if err := folderCursor.Decode(&result); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve cursor.", err.Error(), "FOL0046")
			return
		}
		bsonBytes, _ := bson.Marshal(result)
		bson.Unmarshal(bsonBytes, &inFile)
		if inFile.Meta.Title == cmBody.NewName {
			utils.RespondWithError(w, http.StatusConflict, "Folder Exists.", "Cannot copy file to destination with this name since it is already taken.", "FOL0047")
			return
		}
	}

	//Get folder document
	folder, err := globals.FolderDB.GetOneByID(cmBody.Id)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Folder doesn't exist.", err.Error(), "FOL0048")
		return
	}

	//Get new parent folder
	newParent, err := globals.FolderDB.GetOneByID(cmBody.Destination)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Destination doesn't exist.", err.Error(), "FOL0049")
		return
	}

	//Check if destination is child (or descendant) of target file
	for _, ans := range newParent.Ancestors {
		if ans == cmBody.Id {
			utils.RespondWithError(w, http.StatusBadRequest, "Destination can't be child of target folder", err.Error(), "FOL0050")
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(folder)
}

func GetMyFolders(w http.ResponseWriter, r *http.Request) {

	// folderPath := r.FormValue("folderPath")
	var folders []models.Folder

	claims, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not resolve claims.", err.Error(), "FOL0053")
		return
	}

	// Retrieve files from DB
	folderCursor, err := globals.FolderDB.GetCursorByUserID(claims.Subject)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not get cursor of folders.", err.Error(), "FOL0054")
		return
	}
	defer folderCursor.Close(context.Background())

	for folderCursor.Next(context.Background()) {

		var result bson.M
		var folder models.Folder
		if err = folderCursor.Decode(&result); err != nil {
			utils.RespondWithError(w, http.StatusConflict, "Could not resolve cursor.", err.Error(), "FOL0055")
			return
		}

		bsonBytes, _ := bson.Marshal(result)
		bson.Unmarshal(bsonBytes, &folder)
		folders = append(folders, folder)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(folders)

}
