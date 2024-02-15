package handlers

import (
	"os"

	db "github.com/isotiropoulos/storage-api/dbs/meta"
	objectstorage "github.com/isotiropoulos/storage-api/dbs/objectStorage"
)

const partSize = 5 * 1024 * 1024

var storage objectstorage.IFileStorage = &objectstorage.FileStorage{}

var fileDB db.IFileStore = &db.FileStore{}
var folderDB db.IFolderStore = &db.FolderStore{}
var partsDB db.IPartStore = &db.PartStore{}

var COPERNICUS_BUCKET_ID = os.Getenv("COP_BUCKET_ID")

// from athos' account
var ADS_UID = os.Getenv("ADS_UID")
var ADS_KEY = os.Getenv("ADS_KEY")
var CDS_UID = os.Getenv("CDS_UID")
var CDS_KEY = os.Getenv("CDS_KEY")

func Init() {

	if COPERNICUS_BUCKET_ID == "" {
		COPERNICUS_BUCKET_ID = "d75cbf55-a6a7-49d2-ab3c-7d6a31a09337"
	}
	if ADS_UID == "" {
		ADS_UID = "17432"
	}
	if ADS_KEY == "" {
		ADS_KEY = "12466f67-666d-4448-b937-71b757b4705c"
	}
	if CDS_UID == "" {
		CDS_UID = "264046"
	}
	if CDS_KEY == "" {
		CDS_KEY = "c3d5a56a-361e-4899-b663-ba1e7ff1697d"
	}
}
