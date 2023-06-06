package handlers

// import (
// 	"github.com/isotiropoulos/storage-api/models"
// 	"github.com/isotiropoulos/storage-api/utils"
// 	"go.mongodb.org/mongo-driver/bson"

// 	"encoding/json"
// 	"net/http"
// 	"time"
// )

// func OpenStream(w http.ResponseWriter, r *http.Request) {
// 	// Resolve Claims
// 	claims, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
// 	if err != nil {
// 		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve claims.", err.Error(), "STR0001")
// 		return
// 	}

// 	// Resolve Request
// 	w.Header().Set("Content-Type", "application/json")
// 	var stream models.Stream
// 	err = json.NewDecoder(r.Body).Decode(&stream)
// 	if err != nil {
// 		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve request.", err.Error(), "STR0002")
// 		return
// 	}

// 	var relevantFile models.File
// 	folderID, err := utils.GenerateUUID()
// 	if err != nil {
// 		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create UUID.", err.Error(), "STR0006")
// 		return
// 	}

// 	str.Files = make([]string, 0)
// 	folder.Meta.Creator = claims.Subject
// 	folderID, err := utils.GenerateUUID()
// 	if err != nil {
// 		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create UUID.", err.Error(), "STR0006")
// 		return
// 	}
// 	folder.Id = folderID
// 	folder.Meta.DateCreation = time.Now().Unix()
// 	folder.Meta.Update = append(folder.Meta.Update, models.Updated{
// 		User: claims.Subject,
// 		Date: time.Now(),
// 	})
// 	var ancestors []string

// 	if folder.Parent != "" {
// 		//Get the folder
// 		object, err := folderDB.GetOneByID(folder.Parent)
// 		if err != nil {
// 			utils.RespondWithError(w, http.StatusNotFound, "Parent folder don't exist.", err.Error(), "STR0007")
// 			return
// 		}
// 		bsonBytes, _ := bson.Marshal(object)
// 		bson.Unmarshal(bsonBytes, &parentFolder)
// 		ancestors = parentFolder.Ancestors

// 		folder.Meta.Read = object.Meta.Read
// 		folder.Meta.Write = object.Meta.Write

// 		ancestors = append(ancestors, folder.Parent)

// 	} else {
// 		ancestors = nil
// 	}

// 	folder.Ancestors = ancestors

// 	err = folderDB.InsertOne(folder)
// 	if err != nil {
// 		utils.RespondWithError(w, http.StatusInternalServerError, "Could post folder.", err.Error(), "STR0008")
// 		return
// 	}

// 	if folder.Parent != "" {

// 		newFolders := parentFolder.Folders
// 		if newFolders != nil {
// 			newFolders = append(newFolders, folderID)
// 		} else {
// 			newFolders = []string{folderID}
// 		}

// 		parentFolder.Folders = newFolders
// 		// parentFolder.Children = newFolders

// 		// Update file
// 		_, err = folderDB.UpdateWithId(parentFolder)
// 		if err != nil {
// 			utils.RespondWithError(w, http.StatusInternalServerError, "Could not update parent folder.", err.Error(), "STR0009")
// 			return
// 		}

// 		// Update Ancestore's Meta
// 		err = folderDB.UpdateMetaAncestors(ancestors, claims.Subject)
// 		if err != nil {
// 			utils.RespondWithError(w, http.StatusInternalServerError, "Could not update ancestore's meta.", err.Error(), "STR0010")
// 			return
// 		}

// 	}
// 	json.NewEncoder(w).Encode(folder)
// }
