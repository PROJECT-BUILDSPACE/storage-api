package handlers

import (
	db "github.com/isotiropoulos/storage-api/dbs/meta"
	objectstorage "github.com/isotiropoulos/storage-api/dbs/objectStorage"
)

var storage objectstorage.IFileStorage = &objectstorage.FileStorage{}

// var db db.IDatastore = &db.Datastore{}
var fileDB db.IFileStore = &db.FileStore{}
var folderDB db.IFolderStore = &db.FolderStore{}

// var datasetDB db.IDatasetStore = &db.DatasetStore{}
// var streamDB db.IStreamStore = &db.StreamStore{}
// var partDB db.IPartStore = &db.PartStore{}
