package models

import (
	"time"

	CDSModels "github.com/SLG-European-Projects/cds-go/models"
	"github.com/minio/minio-go/v7"
	"gopkg.in/square/go-jose.v2/jwt"
)

// OIDC Claims struct
type OidcClaims struct {
	*jwt.Claims
	Name             string   `json:"name,omitEmpty"`
	PreferedUsername string   `json:"preferred_username"`
	GivenName        string   `json:"given_name"`
	FamilyName       string   `json:"family_name,omitEmpty"`
	Email            string   `json:"email"`
	Groups           []string `json:"group_names"`
	EditorIn         []string `json:"editor_in,omitempty"`
	ViewerIn         []string `json:"viewer_in,omitempty"`
}

// Bucket struct
type Bucket struct {
	Id           string    `json:"_id"`
	Name         string    `json:"name"`
	CreationDate time.Time `json:"creation_date"`
}

// Meta contains Metadata of BUILDSPACE files
type Meta struct {
	Creator      string    `json:"creator" bson:"creator"`             // User's ID that created the file
	Description  string    `json:"description" bson:"description"`     // Array of descriptions for the file
	Title        string    `json:"title" bson:"title"`                 // Title of the file
	DateCreation time.Time `json:"date_creation" bson:"date_creation"` // Date and time of creation
	Read         []string  `json:"read" bson:"read"`                   // Array of user ids with reading rights
	Write        []string  `json:"write" bson:"write"`                 // Array of user ids with writing rights
	Tags         []string  `json:"tags" bson:"tags"`                   // Array of tags for the file
	Update       Updated   `json:"update" bson:"update"`               // Array with data that store the updates
}

// Updated contains information about the versions of an file
type Updated struct {
	Date time.Time `json:"date" bson:"date"` // Date and time of update
	User string    `json:"user" bson:"user"` // User's id that updated
}

// File contains information about a file.
type File struct {
	Id            string   `json:"_id" bson:"_id"`                       // File's id
	Meta          Meta     `json:"meta" bson:"meta"`                     // File's Metadata
	FolderID      string   `json:"folder" bson:"folder"`                 // Parent folder of the file
	Ancestors     []string `json:"ancestors" bson:"ancestors"`           // All ancestor folders
	OriginalTitle string   `json:"original_title" bson:"original_title"` // The file's title before uploading
	FileType      string   `json:"file_type" bson:"file_type"`           // The file's extention
	Size          int64    `json:"size" bson:"size"`
	Total         int      `json:"total" bson:"total"`
}

//	CopernicusDetails CopernicusDetails `json:"copernicus_details,omitempty" bson:"copernicus_details"` // Details related to Copernicus datasets

// Part contains information about a file's stream.
type Part struct {
	Id         string           `json:"_id" bson:"_id"`                 // Part's id
	PartNumber int              `json:"part_number" bson:"part_number"` // Parts's Number
	FileID     string           `json:"file_id" bson:"file_id"`         // Corresponding File ID
	Size       int64            `json:"size" bson:"size"`               // Corresponding Part's size
	UploadInfo minio.UploadInfo `json:"upload_info" bson:"upload_info"` // Corresponding Part's upload info
}

// UpdateFileBody is the body of a postFile request.
type UpdateFileBody struct {
	FileStream string                 `json:"file_stream" bson:"file_stream"` // File's bytes (as a string)
	Data       map[string]interface{} `json:"data" bson:"data"`               // Hashmap with updates (key representes a key of a document and value represents the desirable value)
}

// Folder contains information about a file.
type Folder struct {
	Id        string   `json:"_id" bson:"_id"`             // Folder's id
	Meta      Meta     `json:"meta" bson:"meta"`           // Folder's Metadata
	Parent    string   `json:"parent" bson:"parent"`       // Parent's folder id
	Ancestors []string `json:"ancestors" bson:"ancestors"` // Array of ancestors' ids
	Files     []string `json:"files" bson:"files"`         // Array of files' ids included
	Folders   []string `json:"folders" bson:"folders"`     // Array of folders' ids included
	Level     int      `json:"level" bson:"level"`         // Level of the folder (root is level 0 etc..)
	Size      int64    `json:"size" bson:"size"`           // Size of a folder (cumulative size of folder's items)
}

// PostFolderBody is the body of a postFolder request.
type PostFolderBody struct {
	FolderName  string `json:"folder_name"` // Folder's name
	Parent      string `json:"parent"`      // Parent's folder id
	Description string `json:"description"` // Description for the folder
}

// FolderList is a list of items in folder.
type FolderList struct {
	Files   []File   `json:"files"`   // Keys are file ids and values are the files' metadata
	Folders []Folder `json:"folders"` // Keys are folder ids and values are the folders' metadata
}

// CopyMoveBody is the body of an copy or move request
type CopyMoveBody struct {
	Id          string `json:"_id"`         // ID of object (file or folder)
	Destination string `json:"destination"` // ID of destination
	NewName     string `json:"new_name"`    // ID of destination
}

// ErrorReport is to report an error
type ErrorReport struct {
	Message        string `json:"message"`         // Message of the error
	Reason         string `json:"reason"`          // Reason of the error
	Status         int    `json:"status"`          // Status of the error
	InternalStatus string `json:"internal_status"` // Status of the error
}

// Input for the request to send to Copericus API
type CopernicusInput struct {
	DatasetName string                 `json:"dataset_name"` //Name of specific Copernicus API
	Body        map[string]interface{} `json:"body"`         // Request body
}

// Input for the request to send to Copericus API
type CopernicusRecord struct {
	Id            string                         `json:"_id" bson:"_id"`                   // Copernicus body fingerprint as id
	FileId        string                         `json:"file_id" bson:"file_id"`           // Reference File ID
	DatasetName   string                         `json:"dataset_name"`                     //Name of specific Copernicus API
	RequestParams map[string]interface{}         `json:"parameters" bson:"parameters"`     // Request body
	Details       CDSModels.PostProcessExecution `json:"details,omitempty" bson:"details"` // Details related to Copernicus datasets
}
