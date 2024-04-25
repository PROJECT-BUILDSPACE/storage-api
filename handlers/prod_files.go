package handlers

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"math"
	"path/filepath"
	"time"

	// "math"

	"strconv"

	"github.com/gorilla/mux"
	"github.com/isotiropoulos/storage-api/globals"
	"github.com/isotiropoulos/storage-api/models"
	"github.com/isotiropoulos/storage-api/utils"
	"github.com/minio/minio-go/v7"
	"go.mongodb.org/mongo-driver/bson"

	"encoding/json"
	"net/http"
)

// PostFile handles the /file POST request.
// @Summary Upload a file.
// @Description This is the endopoint to upload files. The files are uploaded using a multipart streaming upload.
// @Description Step 1 is to select the content-type.
// @Description 	- If **application/json** then the request will be sent to initialize the multipart upload. In this case user must pass a **File model as a payload** containing the **folder** and the **original_title** fields. User must also pass the **total** header to specify the number of parts that will be uploaded.
// @Description 	- If **application/octet-stream** user must pass the **binary data** (decoded) in the body and also provide the **file ID** and part number parameters.
// @Tags Files
// @Accept json
// @Accept octet-stream
// @Produce json
// @Param body body interface{}  true "Request body"
// @Param total header string false "Total parts of multipart upload"
// @Param file path string false "File ID"
// @Param part query string false "Number of part"
// @Success 200 {object} models.File "OK"
// @Failure 400 {object} models.ErrorReport "Bad Request"
// @Failure 404 {object} models.ErrorReport "Not Found"
// @Failure 409 {object} models.ErrorReport "Conflict"
// @Failure 500 {object} models.ErrorReport "Internal Server Error"
// @Router /file [post]
// @Security BearerAuth
func PostFile(w http.ResponseWriter, r *http.Request) {

	var content = r.Header.Get("Content-Type")

	if content == "application/json" {
		handleJSON(w, r)
	} else if content == "application/octet-stream" {
		handleOCTET(w, r)
	} else {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not process content.", "", "FIL0001")
		return
	}

}

func handleJSON(w http.ResponseWriter, r *http.Request) {
	// bucketID := r.Header.Get("X-Group-Id")

	// Resolve Claims
	claims, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not resolve user's claims.", err.Error(), "FIL0002")
		return
	}

	var postFile models.File

	// Get new file's ID
	fileID, err := utils.GenerateUUID()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in creating file's ID.", err.Error(), "FIL0003")
		return
	}

	// Get totalPartsCount from headers
	totalPartsCount, err := strconv.Atoi(r.Header.Get("total"))
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not get the parts of the file.", err.Error(), "FIL0004")
		return
	}

	// postFile is the document of the file for DB
	// Contains: Original Title (with Type)
	err = json.NewDecoder(r.Body).Decode(&postFile)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not decode request body.", err.Error(), "FIL0005")
		return
	}
	postFile.Id = fileID

	folder, err := globals.FolderDB.GetOneByID(postFile.FolderID)
	if err != nil || folder.Id == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not find parent folder.", err.Error(), "FIL0008")
		return
	}

	update := models.Updated{
		Date: time.Now(),
		User: claims.Subject,
	}

	// Insert file doc in DB
	base := filepath.Base(postFile.OriginalTitle)
	ext := filepath.Ext(base)
	// title := base[:len(base)-len(ext)]
	postFile.FileType = ext
	// postFile.OriginalTitle = title
	postFile.Size = 0
	postFile.Ancestors = append(folder.Ancestors, postFile.FolderID)
	postFile.Total = totalPartsCount

	meta := postFile.Meta
	meta.DateCreation = time.Now()
	meta.Creator = claims.Subject
	meta.Read = folder.Meta.Read
	meta.Write = folder.Meta.Write
	meta.Update = update
	postFile.Meta = meta

	err = globals.FileDB.InsertOne(postFile)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in creating stream.", err.Error(), "FIL0009")
		return
	}

	// Update parent folder
	err = globals.FolderDB.UpdateFiles(postFile.Id, postFile.FolderID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not update parent folder.", err.Error(), "FIL0010")
		return
	}

	// Update ancestore's meta
	err = globals.FolderDB.UpdateMetaAncestors(postFile.Ancestors, claims.Subject)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not update ancestore's meta.", err.Error(), "FIL0069")
		return
	}

	json.NewEncoder(w).Encode(postFile)
}

func handleOCTET(w http.ResponseWriter, r *http.Request) {
	// Get bucket ID from headers
	bucketID := r.Header.Get("X-Group-Id")

	// Read the request body
	partBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not decode request body.", err.Error(), "FIL0012")
		return
	}

	// Close the request body to prevent resource leaks
	defer r.Body.Close()

	// Get parameters
	params := mux.Vars(r) // Gets params
	fileId := params["id"]
	partNum := r.FormValue("part")

	// Retrive Objects from DB
	file, err := globals.FileDB.GetOneByID(fileId)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Could not find file.", err.Error(), "FIL0013")
		return
	}

	folder, err := globals.FolderDB.GetOneByID(file.FolderID)
	if err != nil || folder.Id == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not find parent folder.", err.Error(), "FIL0064")
		return
	}

	// Make updates in document
	var filePart models.Part
	partId, err := utils.GenerateUUID()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in creating part's ID.", err.Error(), "FIL0065")
		return
	}

	// Convert bytes.Buffer to io.Reader
	partReader := bytes.NewReader(partBytes)
	size := len(partBytes)

	filePart.Id = partId
	filePart.FileID = fileId
	filePart.Size = int64(size)
	filePart.PartNumber, err = strconv.Atoi(partNum)

	// Initiate a new part upload
	// opt := models.ObjectPartInfo{
	// 	PartNumber: filePart.PartNumber,
	// }
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not get the part number.", err.Error(), "FIL0014")
		return
	}

	// Upload part
	uploadInfo, err := globals.Storage.PostPart(bucketID, partId, partReader, int64(size), minio.PutObjectOptions{})
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could post part of file.", err.Error(), "FIL0015")
		return
	}

	filePart.UploadInfo = uploadInfo

	// filePart.Part = objectPart
	err = globals.PartsDB.InsertOne(filePart)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not insert part document.", err.Error(), "FIL0018")
		return
	}

	// Update file size
	_, err = globals.FileDB.UpdateFileSize(file.Id, size)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not update file's size.", err.Error(), "FIL0006")
		return
	}

	// Update Ancestore sizes
	err = globals.FolderDB.UpdateAncestorSize(file.Ancestors, int64(size), true)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not update ancestore's meta.", err.Error(), "FIL0011")
		return
	}

	json.NewEncoder(w).Encode(file)
}

// GetFileInfo handles the /info/file/ get request.
// @Summary Get metadata of file.
// @Description Returns the metadata of a file by it's ID.
// @Tags Files
// @Produce json
// @Param id query string true "File ID"
// @Success 202 {array} byte "Accepted"
// @Failure 400 {object} models.ErrorReport "Not Found"
// @Router /info/file [get]
// @Security BearerAuth
func GetFileInfo(w http.ResponseWriter, r *http.Request) {

	// fileId := r.FormValue("id")
	params := mux.Vars(r) // Gets params
	fileId := params["id"]
	// Retrive Object from DB
	file, err := globals.FileDB.GetOneByID(fileId)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Could not find file.", err.Error(), "FIL0019")
		return
	}

	// Calculate the number of parts and the optimal part size
	totalPartsCount := int(math.Ceil(float64(file.Size) / float64(globals.PartSize)))

	w.Header().Set("parts", strconv.FormatInt(int64(totalPartsCount), 10))
	json.NewEncoder(w).Encode(file)
}

// GetFile handles the /file/{id} get request.
// @Summary Download a file.
// @Description This is the endopoint to get files. The files are downloaded using a **multipart streaming download**.
// @Description User provies the file id as well as the part number and receives the decoded and decrypted bytes/
// @Tags Files
// @Produce octet-stream
// @Param id path string true "File ID"
// @Param part query string true "Number of part"
// @Success 202 {object} models.File "Accepted"
// @Failure 400 {object} models.ErrorReport "Bad Request"
// @Router /file/{id} [get]
// @Security BearerAuth
func GetFile(w http.ResponseWriter, r *http.Request) {

	// Retrieve group
	groupId := r.Header.Get("X-Group-Id")

	// Get parameters
	params := mux.Vars(r) // Gets params
	fileId := params["id"]
	partNum, err := strconv.Atoi(r.FormValue("part"))
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not get the part number.", err.Error(), "FIL0020")
		return
	}

	// Retrieve file from DB
	refFile, err := globals.FileDB.GetOneByID(fileId)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Error in retrieving file's information.", err.Error(), "FIL0021")
		return
	}

	getPart, err := globals.PartsDB.GetOneByFileAndPart(refFile.Id, partNum)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Error in retrieving file's part information.", err.Error(), "FIL0028")
		return
	}

	// Read file part
	reader, _, _, err := globals.Storage.GetFile(getPart.Id, groupId, minio.GetObjectOptions{})
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not get file part.", err.Error(), "FIL0022")
		return
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not pass file part to buffer.", err.Error(), "FIL0023")
		return
	}

	// Write bytes
	var buffer bytes.Buffer
	buffer.Write(buf.Bytes())
	size := len(buf.Bytes())

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(int64(size), 10))
	w.WriteHeader(http.StatusAccepted)
	_, err = io.Copy(w, &buffer)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not send file.", err.Error(), "FIL0024")
		return
	}
}

// DeleteFile handles the /file/{id} delete request.
// @Summary Delete file by ID.
// @Description This is the endopoint to delete files. The files are deleted based on ther id.
// @Tags Files
// @Produce json
// @Param id path string true "File ID"
// @Success 202 {object} models.File "Accepted"
// @Failure 400 {object} models.ErrorReport "Bad Request"
// @Router /file/{id} [delete]
// @Security BearerAuth
func DeleteFile(w http.ResponseWriter, r *http.Request) {

	// Resolve Claims
	claims, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not resolve user's claims.", err.Error(), "FIL0025")
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var parentFolder models.Folder

	params := mux.Vars(r) // Gets params

	// Retrive Object from DB
	file, err := globals.FileDB.GetOneByID(params["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Could not find file.", err.Error(), "FIL0026")
		return
	}

	// Delete Object from DB
	err = globals.FileDB.DeleteOneByID(params["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not delete file.", err.Error(), "FIL0027")
		return
	}

	// Update folder containing the file
	// Get the Parent folder
	parentFolder, err = globals.FolderDB.GetOneByID(file.FolderID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Could not find parent folder.", err.Error(), "FIL0029")
		return
	}

	// Remove the deleted file (but keep the order)
	newFiles := utils.RemoveFromSlice(parentFolder.Files, params["id"])

	parentFolder.Files = newFiles
	// parentFolder.Size = parentFolder.Size - file.Size
	// Pass new values

	// Update Parent
	_, err = globals.FolderDB.UpdateWithId(parentFolder)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Could not update parent folder.", err.Error(), "FIL0030")
		return
	}

	// Update Uncestores
	err = globals.FolderDB.UpdateMetaAncestors(file.Ancestors, claims.Subject)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Could not update parent folder.", err.Error(), "FIL0071")
		return
	}

	err = globals.FolderDB.UpdateAncestorSize(file.Ancestors, file.Size, false)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Could not update parent folder.", err.Error(), "FIL0031")
		return
	}

	// Remove Object from MINIO
	groupId := r.Header.Get("X-Group-Id")
	partsCursor, err := globals.PartsDB.GetCursorByFileID(file.Id)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in retrieving parts.", err.Error(), "FIL0070")
		return
	}
	defer partsCursor.Close(context.Background())

	for partsCursor.Next(context.Background()) {
		var result bson.M
		var part models.Part
		if err := partsCursor.Decode(&result); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve cursor.", err.Error(), "FIL0007")
			return
		}
		bsonBytes, _ := bson.Marshal(result)
		bson.Unmarshal(bsonBytes, &part)
		if err = globals.Storage.DeleteFile(part.Id, groupId); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error in deleting file.", err.Error(), "FIL0032")
			return
		}
	}

	err = globals.PartsDB.DeleteManyWithFile(params["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not delete file parts.", err.Error(), "FIL0016")
		return
	}

	json.NewEncoder(w).Encode(file)
}

// UpdateFile handles the /file put request.
// @Summary Update a file.
// @Description This is the endopoint to update file meta data. Pass a models.File of the file that will be updated with the updates included.
// @Description **Note** that this endpoint updates the meta data and not the file contents. To update file contents user must delete int and re-upload it.
// @Tags Files
// @Accept json
// @Produce json
// @Param body body models.File  true "Request body"
// @Success 200 {object} models.File "OK"
// @Failure 400 {object} models.ErrorReport "Bad Request"
// @Failure 404 {object} models.ErrorReport "Not Found"
// @Failure 409 {object} models.ErrorReport "Conflict"
// @Failure 500 {object} models.ErrorReport "Internal Server Error"
// @Router /file [put]
// @Security BearerAuth
func UpdateFile(w http.ResponseWriter, r *http.Request) {

	// Resolve Claims
	claims, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve claims.", err.Error(), "FIL0033")
		return
	}

	var file models.File
	var updateFile models.File

	err = json.NewDecoder(r.Body).Decode(&updateFile)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve request.", err.Error(), "FIL0034")
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
	currentDoc, err := globals.FileDB.GetOneByID(updateFile.Id)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "File trying to get updated doesn't exist.", err.Error(), "FIL0035")
		return
	}

	if updateFile.Meta.Title != currentDoc.Meta.Title {
		// Check if title is illegal
		filesCursor, err := globals.FileDB.GetCursorByFolderID(parentID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Could not obtain siblings.", err.Error(), "FIL0036")
			return
		}
		defer filesCursor.Close(context.Background())

		for filesCursor.Next(context.Background()) {
			var result bson.M
			var inFile models.File
			if err := filesCursor.Decode(&result); err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve cursor.", err.Error(), "FIL0037")
				return
			}
			bsonBytes, _ := bson.Marshal(result)
			bson.Unmarshal(bsonBytes, &inFile)
			if inFile.Meta.Title == updateFile.Meta.Title {
				utils.RespondWithError(w, http.StatusConflict, "File Exists.", "Cannot rename file to this name, since it is already taken.", "FIL0038")
				return
			}
		}
	}

	// Update folder
	// updateFile.Meta.Update = append(currentDoc.Meta.Update, models.Updated{
	// 	User: claims.Subject,
	// 	Date: time.Now(),
	// })
	updateFile.Meta.Update.Date = time.Now()
	updateFile.Meta.Update.User = claims.Subject

	file, err = globals.FileDB.UpdateWithId(updateFile)
	if err != nil {
		utils.RespondWithError(w, http.StatusConflict, "Could not update folder.", err.Error(), "FIL0039")
		return
	}

	// Update ancestores meta
	err = globals.FolderDB.UpdateMetaAncestors(file.Ancestors, claims.Subject)
	if err != nil {
		utils.RespondWithError(w, http.StatusConflict, "Could not update folder's ancestores.", err.Error(), "FIL0040")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(file)
}

// CopyFile is to copy a file.
func CopyFile(w http.ResponseWriter, r *http.Request) {

	// Resolve Claims
	claims, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve claims.", err.Error(), "FIL0041")
		return
	}

	var cmBody models.CopyMoveBody
	err = json.NewDecoder(r.Body).Decode(&cmBody)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve request.", err.Error(), "FIL0042")
		return
	}

	file, err := globals.FileDB.GetOneByID(cmBody.Id)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "File doesn't exist.", err.Error(), "FIL0043")
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
		utils.RespondWithError(w, http.StatusBadRequest, "Destination doesn't exist.", err.Error(), "FIL0044")
		return
	}

	// Check if title is illegal
	filesCursor, err := globals.FileDB.GetCursorByFolderID(cmBody.Destination)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not obtain siblings.", err.Error(), "FIL0045")
		return
	}
	defer filesCursor.Close(context.Background())

	for filesCursor.Next(context.Background()) {
		var result bson.M
		var inFile models.File
		if err := filesCursor.Decode(&result); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve cursor.", err.Error(), "FIL0046")
			return
		}
		bsonBytes, _ := bson.Marshal(result)
		bson.Unmarshal(bsonBytes, &inFile)
		if inFile.Meta.Title == newName {
			utils.RespondWithError(w, http.StatusConflict, "File Exists.", "Cannot copy file to destination with this name since it is already taken.", "FIL0047")
			return
		}
	}

	bucketID := r.Header.Get("X-Group-Id")
	newFileId, err := utils.GenerateUUID()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in generating file's ID.", err.Error(), "FIL0048")
		return
	}
	partsCursor, err := globals.PartsDB.GetCursorByFileID(cmBody.Id)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in retrieving parts.", err.Error(), "FIL0017")
		return
	}
	defer partsCursor.Close(context.Background())

	for partsCursor.Next(context.Background()) {
		var result bson.M
		var part models.Part
		if err := partsCursor.Decode(&result); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve cursor.", err.Error(), "FIL0068")
			return
		}
		bsonBytes, _ := bson.Marshal(result)
		bson.Unmarshal(bsonBytes, &part)

		newPartID, err := utils.GenerateUUID()
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error in generating part's ID.", err.Error(), "FIL0073")
			return
		}

		err = globals.Storage.CopyFile(part.Id, newPartID, bucketID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Could not copy file.", err.Error(), "FIL0050")
			return
		}

		part.Id = newPartID
		part.FileID = newFileId
		err = globals.PartsDB.InsertOne(part)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Could not copy file.", err.Error(), "FIL0072")
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
	// file.Meta.Update = []models.Updated{updated}
	file.Meta.Update.User = updated.User
	file.Meta.Update.Date = updated.Date
	file.Id = newFileId
	file.Meta.Title = newName

	ancestors := append(newParent.Ancestors, file.FolderID)
	file.Ancestors = ancestors
	err = globals.FileDB.InsertOne(file)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not copy file.", err.Error(), "FIL0049")
		return
	}

	// Update New Parent Folder
	newParent.Files = append(newParent.Files, file.Id)
	// newParent.Meta.Update = append(newParent.Meta.Update, updated)
	newParent.Meta.Update.User = updated.User
	newParent.Meta.Update.Date = updated.Date

	_, err = globals.FolderDB.UpdateWithId(newParent)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not update parent folder.", err.Error(), "FIL0051")
		return
	}

	// Update Ancestores Size
	err = globals.FolderDB.UpdateAncestorSize(ancestors, file.Size, true)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not update parent folder.", err.Error(), "FIL0074")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(file)

}

// MoveFile is to move a folder.
func MoveFile(w http.ResponseWriter, r *http.Request) {

	// Resolve Claims
	claims, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve claims.", err.Error(), "FIL0052")
		return
	}

	// Decode body
	var cmBody models.CopyMoveBody
	err = json.NewDecoder(r.Body).Decode(&cmBody)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve request.", err.Error(), "FIL0053")
		return
	}

	// Get file document
	file, err := globals.FileDB.GetOneByID(cmBody.Id)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Folder doesn't exist.", err.Error(), "FIL0054")
		return
	}

	// Get new folder document
	newParent, err := globals.FolderDB.GetOneByID(cmBody.Destination)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Destination doesn't exist.", err.Error(), "FIL0055")
		return
	}

	// Get old folder document
	oldParent, err := globals.FolderDB.GetOneByID(file.FolderID)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Parent folder doesn't exist.", err.Error(), "FIL0059")
		return
	}

	// Check if title is illegal
	var newName string
	if cmBody.NewName == "" {
		newName = file.Meta.Title
	} else {
		newName = cmBody.NewName
	}

	filesCursor, err := globals.FileDB.GetCursorByFolderID(cmBody.Destination)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not obtain siblings.", err.Error(), "FIL0056")
		return
	}
	defer filesCursor.Close(context.Background())

	for filesCursor.Next(context.Background()) {
		var result bson.M
		var inFile models.File
		if err := filesCursor.Decode(&result); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve cursor.", err.Error(), "FIL0057")
			return
		}
		bsonBytes, _ := bson.Marshal(result)
		bson.Unmarshal(bsonBytes, &inFile)
		if inFile.Meta.Title == newName {
			utils.RespondWithError(w, http.StatusConflict, "File Exists.", "Cannot move file to destination with this name since it is already taken.", "FIL0058")
			return
		}
	}

	// Remove the deleted file (but keep the order)
	newFiles := utils.RemoveFromSlice(oldParent.Files, file.Id)
	oldAncestores := file.Ancestors

	oldParent.Files = newFiles

	// Update old Parent
	oldParent, err = globals.FolderDB.UpdateWithId(oldParent)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in updating old parent folder.", err.Error(), "FIL0060")
		return
	}

	// Update OLD Parent Ancestore's Meta
	err = globals.FolderDB.UpdateMetaAncestors(file.Ancestors, claims.Subject)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in updating ancestore's meta.", err.Error(), "FIL0061")
		return
	}

	file.FolderID = cmBody.Destination
	// Create a new `Updated` struct
	updated := models.Updated{
		User: claims.Subject,
		Date: time.Now(),
	}
	// file.Meta.Update = []models.Updated{updated}
	file.Meta.Update.User = updated.User
	file.Meta.Update.Date = updated.Date

	file.Meta.Title = newName
	ancestors := append(newParent.Ancestors, file.FolderID)
	file.Ancestors = ancestors
	updatedFile, err := globals.FileDB.UpdateWithId(file)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not file folder.", err.Error(), "FIL0062")
		return
	}

	// Update New Parent Folder
	newParent.Files = append(newParent.Files, file.Id)
	// newParent.Meta.Update = append(newParent.Meta.Update, updated)
	newParent.Meta.Update.User = updated.User
	newParent.Meta.Update.Date = updated.Date
	_, err = globals.FolderDB.UpdateWithId(newParent)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not update parent folder.", err.Error(), "FIL0063")
		return
	}

	// Create an array with updatable ancestores
	// Create maps to store the presence of items
	mapOld := make(map[string]bool)
	mapNew := make(map[string]bool)

	// Populate mapA with items from array a
	for _, item := range oldAncestores {
		mapOld[item] = true
	}

	// Populate mapB with items from array b
	for _, item := range ancestors {
		mapNew[item] = true
	}

	// Create an array with items that are in array b but not in array a
	var newUpdatable []string

	// Create an array with items that are in array a but not in array b
	var oldUpdatable []string

	// Check items in array b
	for itemB := range mapNew {
		if !mapOld[itemB] {
			newUpdatable = append(newUpdatable, itemB)
		}
	}

	// Check items in array a
	for itemA := range mapOld {
		if !mapNew[itemA] {
			oldUpdatable = append(oldUpdatable, itemA)
		}
	}

	// Update NEW Parent Ancestore's Size
	if len(newUpdatable) > 0 {
		err = globals.FolderDB.UpdateAncestorSize(newUpdatable, file.Size, true)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error in updating ancestore's size.", err.Error(), "FIL0066")
			return
		}
	}

	// Update OLD Parent Ancestore's Size

	if len(newUpdatable) > 0 {
		err = globals.FolderDB.UpdateAncestorSize(oldAncestores, file.Size, false)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error in updating ancestore's size.", err.Error(), "FIL0067")
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(updatedFile)

}
