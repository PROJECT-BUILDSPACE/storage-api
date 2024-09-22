package globals

import (
	"os"
	"time"

	db "github.com/isotiropoulos/storage-api/dbs/meta"
	objectstorage "github.com/isotiropoulos/storage-api/dbs/objectStorage"
)

const PartSize = 1 * 1024 * 1024

const CheckTime = 5 * time.Second

var Storage objectstorage.IFileStorage = &objectstorage.FileStorage{}

var FileDB db.IFileStore = &db.FileStore{}
var FolderDB db.IFolderStore = &db.FolderStore{}
var PartsDB db.IPartStore = &db.PartStore{}
var CopernicusDB db.ICopernicusStore = &db.CopernicusStore{}

var COPERNICUS_BUCKET_ID = os.Getenv("COP_BUCKET_ID")

// from athos' account
var ADS_UID = os.Getenv("ADS_UID")
var ADS_KEY = os.Getenv("ADS_KEY")
var CDS_UID = os.Getenv("CDS_UID")
var CDS_KEY = os.Getenv("CDS_KEY")

func Init() {

	if COPERNICUS_BUCKET_ID == "" {
		COPERNICUS_BUCKET_ID = "95f8062d-89e2-443e-b750-9174f1a0748a"
	}
	if ADS_UID == "" {
		ADS_UID = "18429"
	}
	if ADS_KEY == "" {
		ADS_KEY = "f534accf-6e3b-4dab-9b83-cd507afd5237"
	}
	if CDS_UID == "" {
		CDS_UID = "287556"
	}
	if CDS_KEY == "" {
		CDS_KEY = "7c81fc53-e2cf-4896-8ba2-86401c5d9501"
	}
}
