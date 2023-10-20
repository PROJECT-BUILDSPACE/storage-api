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
// @Security GroupId
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
	bucketID := r.Header.Get("X-Group-Id")

	// Resolve Claims
	claims, err := utils.GetClaimsFromContext(r.Context().Value("claims"))
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not resolve user's claims.", err.Error(), "FIL0002")
		return
	}

	partClose := r.FormValue("part")
	var postFile models.File

	if partClose == "close" {

		params := mux.Vars(r) // Gets params
		fileID := params["id"]

		// Retrive Objects from DB
		postFile, err = fileDB.GetOneByID(fileID)
		if err != nil {
			utils.RespondWithError(w, http.StatusNotFound, "Could not find file.", err.Error(), "FIL0066")
			return
		}

		folder, err := folderDB.GetOneByID(postFile.FolderID)
		if err != nil {
			utils.RespondWithError(w, http.StatusNotFound, "Could not find file.", err.Error(), "FIL0074")
			return
		}

		stream, err := streamDB.GetOneByFileID(fileID)
		if err != nil {
			utils.RespondWithError(w, http.StatusNotFound, "Could not find stream.", err.Error(), "FIL0068")
		}

		// Fetch part documets
		multiParts := make([]minio.CompletePart, stream.Total)

		partCursor, err := partsDB.GetCursorByStreamID(stream.Id)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Could not obtain parts.", err.Error(), "FIL0072")
			return
		}
		defer partCursor.Close(context.Background())

		for partCursor.Next(context.Background()) {
			var result bson.M
			var midPart models.Part
			if err := partCursor.Decode(&result); err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve cursor.", err.Error(), "FIL0073")
				return
			}
			bsonBytes, _ := bson.Marshal(result)
			bson.Unmarshal(bsonBytes, &midPart)
			// Update size of file
			postFile.Size = postFile.Size + midPart.Size
			// Keep in mind indicing!
			multiParts[midPart.PartNumber-1] = midPart.Part
		}

		if _, err = storage.CloseMultipart(bucketID, fileID, stream.Id, multiParts); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not close multipart upload.", err.Error(), "FIL0069")
			return
		}

		// Update size of file and necessary folders
		postFile, err = fileDB.UpdateWithId(postFile)
		if err != nil {
			utils.RespondWithError(w, http.StatusConflict, "Could not update size.", err.Error(), "FIL0016")
			return
		}

		// Update folder size and ancestors' size
		folder.Size = folder.Size + postFile.Size
		folder, err = folderDB.UpdateWithId(folder)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not update folder.", err.Error(), "FIL0017")
			return
		}

		err = folderDB.UpdateAncestorSize(folder.Ancestors, postFile.Size)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not update ancestors' size.", err.Error(), "FIL0067")
			return
		}

		// Final update of stream
		stream.Status = "Completed"
		stream, err = streamDB.UpdateWithId(stream)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not update stream.", err.Error(), "FIL0070")
			return
		}
	} else {
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

		// Initiate multipart upload
		uploadID, err := storage.OpenMultipart(bucketID, postFile.Id)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error in opening multipart upload.", err.Error(), "FIL0006")
			return
		}

		// Create a slice of channels to hold upload results
		// var partsCh []minio.CompletePart
		// partsCh := make([]minio.CompletePart, totalPartsCount)

		stream := models.Stream{
			Id:     uploadID,
			FileID: fileID,
			Total:  totalPartsCount,
			Status: "Pending",
		}

		// Insert Stream in DB
		err = streamDB.InsertOne(stream)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error in creating stream.", err.Error(), "FIL0007")
			return
		}

		// Insert file doc in DB
		base := filepath.Base(postFile.OriginalTitle)
		ext := filepath.Ext(base)
		// title := base[:len(base)-len(ext)]
		postFile.Meta.DateCreation = time.Now()
		// postFile.Meta.Title = title
		postFile.FileType = ext
		// postFile.OriginalTitle = title
		postFile.Size = 0
		postFile.Meta.Creator = claims.Subject
		folder, err := folderDB.GetOneByID(postFile.FolderID)
		if err != nil || folder.Id == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Could not find parent folder.", err.Error(), "FIL0008")
			return
		}
		postFile.Ancestors = append(folder.Ancestors, postFile.FolderID)
		postFile.Meta.Read = folder.Meta.Read
		postFile.Meta.Write = folder.Meta.Write
		// postFile.Meta.Update = append(postFile.Meta.Update, models.Updated{
		// 	User: claims.Subject,
		// 	Date: time.Now(),
		// })

		postFile.Meta.Update.Date = time.Now()
		postFile.Meta.Update.User = claims.Subject

		err = fileDB.InsertOne(postFile)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error in creating stream.", err.Error(), "FIL0009")
			return
		}

		// Update parent folder
		err = folderDB.UpdateFiles(postFile.Id, postFile.FolderID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Could not update parent folder.", err.Error(), "FIL0010")
			return
		}

		// Update ancestore's meta
		err = folderDB.UpdateMetaAncestors(postFile.Ancestors, claims.Subject)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Could not update ancestore's meta.", err.Error(), "FIL0011")
			return
		}

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
	file, err := fileDB.GetOneByID(fileId)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Could not find file.", err.Error(), "FIL0013")
		return
	}

	stream, err := streamDB.GetOneByFileID(fileId)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Could not find stream.", err.Error(), "FIL0064")
	}

	folder, err := folderDB.GetOneByID(file.FolderID)
	if err != nil || folder.Id == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not find parent folder.", err.Error(), "FIL0065")
		return
	}

	// Make updates in document
	var filePart models.Part
	partId, err := utils.GenerateUUID()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in creating part's ID.", err.Error(), "FIL0071")
		return
	}

	// Convert bytes.Buffer to io.Reader
	partReader := bytes.NewReader(partBytes)
	size := len(partBytes)

	filePart.Id = partId
	filePart.FileID = fileId
	filePart.StreamID = stream.Id
	filePart.Size = int64(size)
	filePart.PartNumber, err = strconv.Atoi(partNum)

	// Initiate a new part upload
	opt := models.ObjectPartInfo{
		PartNumber: filePart.PartNumber,
	}
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not get the part number.", err.Error(), "FIL0014")
		return
	}

	// Upload part
	part, err := storage.PostPart(bucketID, stream.FileID, filePart.StreamID, opt.PartNumber, partReader, int64(size), minio.PutObjectPartOptions{})
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could post part of file.", err.Error(), "FIL0015")
		return
	}

	// Insert part document
	objectPart := minio.CompletePart{
		PartNumber: part.PartNumber,
		ETag:       part.ETag,
	}

	filePart.Part = objectPart
	err = partsDB.InsertOne(filePart)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not insert part document.", err.Error(), "FIL0018")
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
// @Success 202 {array} byte "OK"
// @Failure 400 {object} models.ErrorReport "Not Found"
// @Router /info/file [get]
// @Security BearerAuth
// @Security GroupId
func GetFileInfo(w http.ResponseWriter, r *http.Request) {

	// fileId := r.FormValue("id")
	params := mux.Vars(r) // Gets params
	fileId := params["id"]
	// Retrive Object from DB
	file, err := fileDB.GetOneByID(fileId)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Could not find file.", err.Error(), "FIL0019")
		return
	}

	// Calculate the number of parts and the optimal part size
	totalPartsCount := int(math.Ceil(float64(file.Size) / float64(partSize)))

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
// @Security GroupId
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
	getFile, err := fileDB.GetOneByID(fileId)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Error in retrieving file's information.", err.Error(), "FIL0021")
		return
	}

	// Read file part
	reader, _, _, err := storage.GetFile(getFile.Id, groupId, minio.GetObjectOptions{
		PartNumber: partNum,
	})
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
// @Security GroupId
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
	file, err := fileDB.GetOneByID(params["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Could not find file.", err.Error(), "FIL0026")
		return
	}

	// Delete Object from DB
	err = fileDB.DeleteOneByID(params["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not delete file.", err.Error(), "FIL0027")
		return
	}

	// Delete all related streams
	err = streamDB.DeleteManyWithFile(params["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not delete stream.", err.Error(), "FIL0028")
		return
	}

	// Update folder containing the file
	// Get the Parent folder
	parentFolder, err = folderDB.GetOneByID(file.FolderID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Could not find parent folder.", err.Error(), "FIL0029")
		return
	}

	// Remove the deleted file (but keep the order)
	newFiles := utils.RemoveFromSlice(parentFolder.Files, params["id"])

	parentFolder.Files = newFiles
	parentFolder.Size = parentFolder.Size - file.Size
	// Pass new values

	// Update Parent
	_, err = folderDB.UpdateWithId(parentFolder)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Could not update parent folder.", err.Error(), "FIL0030")
		return
	}

	// Update Uncestores
	err = folderDB.UpdateMetaAncestors(file.Ancestors, claims.Subject)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Could not update parent folder.", err.Error(), "FIL0031")
		return
	}

	// Remove Object from MINIO
	groupId := r.Header.Get("X-Group-Id")
	if err = storage.DeleteFile(file.Id, groupId); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in deleting file.", err.Error(), "FIL0032")
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
// @Security GroupId
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
	currentDoc, err := fileDB.GetOneByID(updateFile.Id)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "File trying to get updated doesn't exist.", err.Error(), "FIL0035")
		return
	}

	if updateFile.Meta.Title != currentDoc.Meta.Title {
		// Check if title is illegal
		filesCursor, err := fileDB.GetCursorByFolderID(parentID)
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

	file, err = fileDB.UpdateWithId(updateFile)
	if err != nil {
		utils.RespondWithError(w, http.StatusConflict, "Could not update folder.", err.Error(), "FIL0039")
		return
	}

	// Update ancestores meta
	err = folderDB.UpdateMetaAncestors(file.Ancestors, claims.Subject)
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

	file, err := fileDB.GetOneByID(cmBody.Id)
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

	newParent, err := folderDB.GetOneByID(cmBody.Destination)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Destination doesn't exist.", err.Error(), "FIL0044")
		return
	}

	// Check if title is illegal
	filesCursor, err := fileDB.GetCursorByFolderID(cmBody.Destination)
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

	file.Meta.Title = newName
	file.Id, err = utils.GenerateUUID()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in generating file's ID.", err.Error(), "FIL0048")
		return
	}
	ancestors := append(newParent.Ancestors, file.FolderID)
	file.Ancestors = ancestors
	err = fileDB.InsertOne(file)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not copy file.", err.Error(), "FIL0049")
		return
	}

	bucketID := r.Header.Get("X-Group-Id")
	err = storage.CopyFile(cmBody.Id, file.Id, bucketID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not copy file.", err.Error(), "FIL0050")
		return
	}

	// Update New Parent Folder
	newParent.Files = append(newParent.Files, file.Id)
	// newParent.Meta.Update = append(newParent.Meta.Update, updated)
	newParent.Meta.Update.User = updated.User
	newParent.Meta.Update.Date = updated.Date

	_, err = folderDB.UpdateWithId(newParent)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not update parent folder.", err.Error(), "FIL0051")
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

	var cmBody models.CopyMoveBody
	err = json.NewDecoder(r.Body).Decode(&cmBody)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Could not resolve request.", err.Error(), "FIL0053")
		return
	}

	file, err := fileDB.GetOneByID(cmBody.Id)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Folder doesn't exist.", err.Error(), "FIL0054")
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
		utils.RespondWithError(w, http.StatusBadRequest, "Destination doesn't exist.", err.Error(), "FIL0055")
		return
	}

	// Check if title is illegal
	filesCursor, err := fileDB.GetCursorByFolderID(cmBody.Destination)
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

	oldParent, err := folderDB.GetOneByID(file.FolderID)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Parent folder doesn't exist.", err.Error(), "FIL0059")
		return
	}

	// Remove the deleted file (but keep the order)
	newFiles := utils.RemoveFromSlice(oldParent.Files, file.Id)

	oldParent.Files = newFiles

	// Update old Parent
	oldParent, err = folderDB.UpdateWithId(oldParent)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error in updating old parent folder.", err.Error(), "FIL0060")
		return
	}

	// Update Ancestore's Meta
	err = folderDB.UpdateMetaAncestors(file.Ancestors, claims.Subject)
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
	updatedFile, err := fileDB.UpdateWithId(file)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not file folder.", err.Error(), "FIL0062")
		return
	}

	// Update New Parent Folder
	newParent.Files = append(newParent.Files, file.Id)
	// newParent.Meta.Update = append(newParent.Meta.Update, updated)
	newParent.Meta.Update.User = updated.User
	newParent.Meta.Update.Date = updated.Date
	_, err = folderDB.UpdateWithId(newParent)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not update parent folder.", err.Error(), "FIL0063")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(updatedFile)

}
