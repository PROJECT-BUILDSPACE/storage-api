package globals

import (
	"os"
	"sync"
	"time"

	goCDS "github.com/SLG-European-Projects/cds-go"
	db "github.com/isotiropoulos/storage-api/dbs/meta"
	objectstorage "github.com/isotiropoulos/storage-api/dbs/objectStorage"
)

const PartSize = 5 * 1024 * 1024

const CheckTime = 5 * time.Second

var Storage objectstorage.IFileStorage = &objectstorage.FileStorage{}

var FileDB db.IFileStore = &db.FileStore{}
var FolderDB db.IFolderStore = &db.FolderStore{}
var PartsDB db.IPartStore = &db.PartStore{}
var CopernicusDB db.ICopernicusStore = &db.CopernicusStore{}

var COPERNICUS_BUCKET_ID = os.Getenv("COP_BUCKET_ID")

var CDS_URL = os.Getenv("CDS_URL")
var CDS_KEY = os.Getenv("CDS_KEY")

var RunningGoroutines sync.Map // Map to track goroutine IDs

var CopernicusClient goCDS.Client

func Init() {

	if COPERNICUS_BUCKET_ID == "" {
		COPERNICUS_BUCKET_ID = "ee7d2834-b7be-4008-8b6a-edd55729b893"
	}

	if CDS_URL == "" {
		CDS_URL = "https://cds.climate.copernicus.eu/api"
	}
	if CDS_KEY == "" {
		CDS_KEY = "b08c3645-2a03-4a3f-a9bb-00059dab9c98"
	}

	CopernicusClient = *goCDS.InitClient(CDS_URL, CDS_KEY)
}
