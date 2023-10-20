package metaDB

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/isotiropoulos/storage-api/models"
	"github.com/minio/minio-go/v7"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// IFileStore is a Database Interface for the Files
type IFileStore interface {
	// Insert a new file
	InsertOne(file models.File) error

	// Delete by _id
	DeleteOneByID(fileID string) error

	// Get file with params
	GetOneByID(fileID string) (models.File, error)

	// Get many with params
	GetCursorByFolderID(folderID string) (*mongo.Cursor, error)

	// Get many with params
	GetCursorByAncestors(ancestors string) (*mongo.Cursor, error)

	// Update item with Params
	UpdateWithId(file models.File) (models.File, error)

	// Delete many with common ancestore
	DeleteManyWithAncestore(ancestore string) error
}

// IFolderStore is a Database Interface for the Folders
type IFolderStore interface {
	// Insert a new file
	InsertOne(folder models.Folder) error

	// Delete by _id
	DeleteOneByID(folderID string) error

	// Delete by ancestore
	DeleteManyWithAncestore(acnsetore string) error

	// Get file with params
	GetOneByID(folderID string) (models.Folder, error)

	// Get root by name
	GetRootByName(folderName string) (models.Folder, error)

	// Get many with DatasetID
	GetCursorByParent(parentID string) (*mongo.Cursor, error)

	// Update Files field of a folder
	UpdateFiles(fileId string, folderID string) error

	// UpdateAncestorSize is a function to update the size of the folder's ancestors
	UpdateAncestorSize(ancestors []string, size int64) error

	// UpdateMetaAncestors is a function to add to the []Updated when changes happen to all acestores
	UpdateMetaAncestors(ancestors []string, userID string) error

	// UpdateWithId is a function to update a Folder
	UpdateWithId(folder models.Folder) (folderUpdated models.Folder, err error)
}

// IStreamStore is a Database Interface for the Sessions
type IStreamStore interface {
	// Insert a new stream
	InsertOne(stream models.Stream) error

	// Get a stream by ID
	GetOneByID(streamID string) (models.Stream, error)

	// Get a stream by the related file ID
	GetOneByFileID(fileID string) (models.Stream, error)

	// Update a stream by ID
	UpdateWithId(stream models.Stream) (objectStream models.Stream, err error)

	// Delete streams related to a file
	DeleteManyWithFile(fileId string) error

	// UpdateInArrayByIdAndIndex is to update a particular stream's part.
	UpdateInArrayByIdAndIndex(streamId string, index string, part minio.CompletePart) (err error)
}

// IPartStore is a Database Interface for the Sessions
type IPartStore interface {

	// Insert a new stream
	InsertOne(part models.Part) error

	// Get a part by ID
	GetOneByID(partID string) (models.Part, error)

	// Get a cusror of parts by file ID
	GetCursorByFileID(fileID string) (*mongo.Cursor, error)

	// Get a cusror of parts by stream ID
	GetCursorByStreamID(streamID string) (*mongo.Cursor, error)

	// Delete streams related to a stream
	DeleteManyWithStream(streamId string) error
}

// FileStore ...
type FileStore struct{}

// FolderStore ...
type FolderStore struct{}

// StreamStore ...
type StreamStore struct{}

// PartStore ...
type PartStore struct{}

// db is a Client of mongoDB
var db *mongo.Database

// NewDB is a function to create a minio Client.
func NewDB() {
	log.Println("Starting DB")
	mongoURL := os.Getenv("MONGO_URL")
	database := os.Getenv("DATABASE")
	if database == "" {
		database = "minio"
	}
	if mongoURL == "" {
		mongoURL = "mongodb://localhost:27017"
	}
	log.Println("Starting at " + mongoURL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))

	db = client.Database(database)
	if err != nil {
		log.Panicln(err.Error())
	}
}
