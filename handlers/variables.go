package handlers

import (
	db "github.com/isotiropoulos/storage-api/dbs/meta"
	objectstorage "github.com/isotiropoulos/storage-api/dbs/objectStorage"
)

const partSize = 5 * 1024 * 1024

var storage objectstorage.IFileStorage = &objectstorage.FileStorage{}

var fileDB db.IFileStore = &db.FileStore{}
var folderDB db.IFolderStore = &db.FolderStore{}
var partsDB db.IPartStore = &db.PartStore{}
