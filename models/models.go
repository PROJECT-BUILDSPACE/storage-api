package models

import (
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

// CopernicusTask contains information on the response of a Copericus task
type CopernicusTask struct {
	State     string              `json:"state"`    //State of files to be downloaded
	Location  string              `json:"location"` // Download link
	RequestID string              `json:"request_id"`
	Message   string              `json:"message,omitempty"` //If state=denied then this gives the reason the request was not accepted. Otherwise this is not present.
	Error     CopernicusTaskError `json:"error,omitempty"`   // Only in case of task failed

}

// CopernicusTaskError contains information in cas state of task is "failed"
type CopernicusTaskError struct {
	Reason    string      `json:"reason"`              //Reason of Failure
	Message   string      `json:"message"`             // Message to the user
	Url       string      `json:"url,omitempty"`       // A URI which is unique to a particular class of error
	Context   interface{} `json:"context,omitempty"`   // An arbitrary JSON object containing additional information for debugging
	Permanent bool        `json:"permanent,omitempty"` // If this is true then the request should not be retried unchanged
	Who       string      `json:"who,omitempty"`       // Is this a problem in the server or the request (server or client)?
}

// CopernicusDetails contains information related to the Copernicus files
type CopernicusDetails struct {
	TaskID      string              `json:"task_id" bson:"task_id"`         // Copernicus Task
	Service     string              `json:"sercice" bson:"service"`         // Dataset's related service
	Fingerprint string              `json:"fingerprint" bson:"fingerprint"` // Fingerprint of the dataset (used to identify datasets based on request parameters)
	Status      string              `json:"status" bson:"status"`           // Copernicus task status
	Error       CopernicusTaskError `json:"error,omitempty" bson:"error"`   // Error details in case of status failed
}

// Updated contains information about the versions of an file
type Updated struct {
	Date time.Time `json:"date" bson:"date"` // Date and time of update
	User string    `json:"user" bson:"user"` // User's id that updated
}

// File contains information about a file.
type File struct {
	Id                string            `json:"_id" bson:"_id"`                       // File's id
	Meta              Meta              `json:"meta" bson:"meta"`                     // File's Metadata
	FolderID          string            `json:"folder" bson:"folder"`                 // Parent folder of the file
	Ancestors         []string          `json:"ancestors" bson:"ancestors"`           // All ancestor folders
	OriginalTitle     string            `json:"original_title" bson:"original_title"` // The file's title before uploading
	FileType          string            `json:"file_type" bson:"file_type"`           // The file's extention
	Size              int64             `json:"size" bson:"size"`
	Total             int               `json:"total" bson:"total"`
	CopernicusDetails CopernicusDetails `json:"copernicus_details,omitempty" bson:"copernicus_details"` // Details related to Copernicus datasets
}

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
	DatasetName string                 `json:"datasetname"` //Name of specific Copernicus API
	Body        map[string]interface{} `json:"body"`        // Request body
}

type Form struct {
	Css      string  `json:"css,omitempty" bson:"css"`
	Details  Details `json:"details,omitempty" bson:"description"`
	Help     string  `json:"help,omitempty" bson:"help"`
	Label    string  `json:"label,omitempty" bson:"label"`
	Name     string  `json:"name,omitempty" bson:"name"`
	Required bool    `json:"required,omitempty" bson:"required"`
	Type     string  `json:"type,omitempty" bson:"type"`
}

type Details struct {
	Columns           int               `json:"columns,omitempty" bson:"columns"`
	ID                int               `json:"id,omitempty" bson:"id"`
	Labels            map[string]string `json:"labels,omitempty" bson:"labels"`
	Values            []string          `json:"values,omitempty" bson:"values"`
	Accordion         bool              `json:"accordion,omitempty" bson:"accordion"`
	AccordionGroups   bool              `json:"accordionGroups,omitempty" bson:"accordionGroups"`
	Displayaslist     bool              `json:"displayaslist,omitempty" bson:"displayaslist"`
	Fullheight        bool              `json:"fullheight,omitempty" bson:"fullheight"`
	Withmap           bool              `json:"withmap,omitempty" bson:"withmap"`
	Wrapping          bool              `json:"wrapping,omitempty" bson:"wrapping"`
	Precision         int               `json:"precision,omitempty" bson:"precision"`
	MaximumSelections int               `json:"maximumSelections,omitempty" bson:"maximumSelections"`
	TextFile          string            `json:"text:file,omitempty" bson:"text:file"`
	Information       string            `json:"information,omitempty" bson:"information"`
	AccordionOptions  *AccordionOpts    `json:"accordionOptions,omitempty" bson:"accordionOptions"`
	Default           []interface{}     `json:"default,omitempty" bson:"default"`
	Extentlabels      []string          `json:"extentlabels,omitempty" bson:"extentlabels"`
	Groups            []interface{}     `json:"groups,omitempty" bson:"groups"`
	Range             *RangeLocal       `json:"range,omitempty" bson:"range"`
	ChangeVisible     bool              `json:"changevisible,omitempty" bson:"changevisible"`
	ConCat            string            `json:"concat,omitempty" bson:"concat"`
	Latidude          Coords            `json:"latitude,omitempty" bson:"latitude"`
	Longitude         Coords            `json:"longitude,omitempty" bson:"longidude"`
	Projection        Projection        `json:"projection,omitempty" bson:"projrction"`
	Text              string            `json:"text,omitempty" bson:"text"`
	Fields            []Fields          `json:"fields,omitempty" bson:"fields"`
}

type AccordionOpts struct {
	OpenGroups interface{} `json:"openGroups,omitempty" bson:"openGroups"`
	Searchable bool        `json:"searchable,omitempty" bson:"searchable"`
}

type RangeLocal struct {
	E float32 `json:"e,omitempty" bson:"e"`
	N float32 `json:"n,omitempty" bson:"n"`
	W float32 `json:"w,omitempty" bson:"w"`
	S float32 `json:"s,omitempty" bson:"s"`
}

type FormRespLocal struct {
	Name     string `json:"name,omitempty" bson:"name"`
	Required bool   `json:"required,omitempty" bson:"required"`
	Type     string `json:"type,omitempty" bson:"type"`
}

type Coords struct {
	Default   int        `json:"default,omitempty" bson:"default"`
	Precision int        `json:"precision,omitempty" bson:"precision"`
	Range     CoordRange `json:"range,omitempty" bson:"range"`
}

type CoordRange struct {
	Min int `json:"min,omitempty" bson:"min"`
	Max int `json:"max,omitempty" bson:"max"`
}

type Projection struct {
	ID      int  `json:"id,omitempty" bson:"id"`
	Overlay bool `json:"overlay,omitempty" bson:"overlay"`
	Use     bool `json:"use,omitempty" bson:"use"`
}

type Fields struct {
	Comments    string `json:"comments,omitempty" bson:"comments"`
	MaxLength   int    `json:"maxlength,omitempty" bson:"maxlength"`
	Placeholder string `json:"placeholder,omitempty" bson:"placeholder"`
	Required    bool   `json:"required,omitempty" bson:"required"`
	Type        string `json:"type,omitempty" bson:"type"`
}
