package models

import (
	"bytes"
	"sync"
	"time"

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
	Groups           []string `json:"groupIDs"`
}

// Bucket struct
type Bucket struct {
	Name string `json:"name"`
	// Location string `json:"location"`
}

// BucketInfo container for bucket metadata.
type BucketInfo struct {
	Name         string    `json:"name"`
	CreationDate time.Time `json:"creation_date"`
}

// ObjectPartInfo bucket metadata.
type ObjectPartInfo struct {
	PartNumber   int       `json:"name"`
	CreationDate time.Time `json:"creation_date"`
}

// Meta contains Metadata of JADS files
type Meta struct {
	Creator      string    `json:"creator" bson:"creator"`             // User's ID that created the file
	Descriptions []string  `json:"descriptions" bson:"descriptions"`   // Array of descriptions for the file
	Title        string    `json:"title" bson:"title"`                 // Title of the file
	DateCreation int64     `json:"date_creation" bson:"date_creation"` // Date and time of creation
	Read         []string  `json:"read" bson:"read"`                   // Array of user ids with reading rights
	Write        []string  `json:"write" bson:"write"`                 // Array of user ids with writing rights
	Tags         []string  `json:"tags" bson:"tags"`                   // Array of tags for the file
	Update       []Updated `json:"update" bson:"update"`               // Array with data that store the updates
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
	// Encrypted     bool     `json:"encrypted" bson:"encrypted"`
}

// Stream contains information about a file's stream.
type Stream struct {
	Id     string               `json:"_id" bson:"_id"`         // Stream's id
	FileID string               `json:"file_id" bson:"file_id"` // Corresponding File ID
	Parts  []minio.CompletePart `json:"parts" bson:"parts"`     // All parts relevant to this stream
	Total  int                  `json:"total" bson:"total"`
	Status string               `json:"status" bson:"status"` // Status of the stream
}

// Part contains information about a file's stream.
type Part struct {
	Id         string `json:"_id" bson:"_id"`                 // Part's id
	PartNumber int    `json:"part_number" bson:"part_number"` // Parts's Number
	StreamID   string `json:"stream_id" bson:"stream_id"`     // Corresponding Stream ID
	FileID     string `json:"file_id" bson:"file_id"`         // Corresponding File ID
	Part       []byte `json:"part" bson:"part"`
}

type FilePart struct {
	PartNumber int          `json:"partNumber" bson:"partNumber"`
	FileID     string       `json:"fileId" bson:"fileId"`
	Part       bytes.Buffer `json:"down_stream" bson:"down_stream"`
}

type MinioPart struct {
	PartNumber int    `json:"part_number" bson:"part_number"`
	ETag       string `json:"etag" bson:"etag"`
}

type Session struct {
	Id         string         `json:"_id" bson:"_id"`
	UploadId   string         `json:"upload_id" bson:"upload_id"`
	TotalParts int            `json:"total_parts" bson:"total_parts"`
	Parts      []MinioPart    `json:"parts" bson:"parts"`
	WGroup     sync.WaitGroup `json:"wait_group" bson:"wait_group"`
	Error      string         `json:"error" bson:"error"`
	Completed  bool           `json:"completed" bson:"completed"`
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
}

// PostFolderBody is the body of a postFolder request.
type PostFolderBody struct {
	FolderName  string `json:"folder_name"` // Folder's name
	Parent      string `json:"parent"`      // Parent's folder id
	Description string `json:"description"` // Description for the folder
}

// UpdateFolderBody is the body of a updateFolder request.
// type UpdateFolderBody struct {
// 	Data map[string]interface{} `json:"data" bson:"data"` // Hashmap with updates (key representes a key of a document and value represents the desirable value)
// }

// FolderList is a list of items in folder.
type FolderList struct {
	Files   map[string]Meta `json:"files"`   // Keys are file ids and values are the files' metadata
	Folders map[string]Meta `json:"folders"` // Keys are folder ids and values are the folders' metadata
}

// CopyMoveBody is the body of an copy or move request
type CopyMoveBody struct {
	Id          string `json:"id"`          // ID of object (file or folder)
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
