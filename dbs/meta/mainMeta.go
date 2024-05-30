package metaDB

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/isotiropoulos/storage-api/models"
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

	// Get file with params
	GetOneByTaskID(taskID string) (models.File, error)

	// Get many with params
	GetCursorByFolderID(folderID string) (*mongo.Cursor, error)

	// Get many with params
	GetCursorByAncestors(ancestors string) (*mongo.Cursor, error)

	// Update item with Params
	UpdateWithId(file models.File) (models.File, error)

	// Delete many with common ancestore
	DeleteManyWithAncestore(ancestore string) error

	// Add to the file size the parts size
	UpdateFileSize(fileID string, size int) (objUpdated models.File, err error)

	// Return Copernicus file by Fingerprint
	GetOneByFingerprint(fingerprint string) (models.File, error)
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
	UpdateAncestorSize(ancestors []string, size int64, add bool) error

	// UpdateMetaAncestors is a function to add to the []Updated when changes happen to all acestores
	UpdateMetaAncestors(ancestors []string, userID string) error

	// UpdateWithId is a function to update a Folder
	UpdateWithId(folder models.Folder) (folderUpdated models.Folder, err error)

	// GetCursorByNameLevel is to get a cursor with folders given Folder Name, Group ID and Folder Level.
	GetCursorByNameLevel(name string, group string, level int) (*mongo.Cursor, error)
}

// IPartStore is a Database Interface for the Sessions
type IPartStore interface {

	// Insert a new stream
	InsertOne(part models.Part) error

	// Get a part by ID
	GetOneByID(partID string) (models.Part, error)

	// Get a cusror of parts by file ID
	GetCursorByFileID(fileID string) (*mongo.Cursor, error)

	// Get part based on file ID and part number
	GetOneByFileAndPart(fileID string, partNum int) (models.Part, error)

	// Delete parts related to certain file
	DeleteManyWithFile(fileId string) error

	// Delete parts related to certain bucket
	DeleteManyWithBucket(bucketId string) error
}

// FileStore ...
type FileStore struct {
	mu sync.RWMutex
}

// FolderStore ...
type FolderStore struct {
	mu sync.RWMutex
}

// PartStore ...
type PartStore struct {
	// mu sync.Mutex
}

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
		mongoURL = "mongodb://minikube.local:30000"
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
