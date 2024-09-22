package metaDB

import (
	"context"

	"github.com/isotiropoulos/storage-api/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	FILESCOLLECTION = "files"
)

// InsertOne is to insert an file in the files collection
func (filestore *FileStore) InsertOne(file models.File) error {
	_, err := db.Collection(FILESCOLLECTION).InsertOne(context.Background(), file)
	return err
}

// DeleteOneByID is to delete an file from a particular collection by _id.
func (filestore *FileStore) DeleteOneByID(fileID string) error {

	_, err := db.Collection(FILESCOLLECTION).DeleteOne(context.Background(), bson.M{"_id": fileID})
	return err
}

// GetOneByID is to get a file by ID.
func (filestore *FileStore) GetOneByID(fileID string) (models.File, error) {

	var file models.File

	err := db.Collection(FILESCOLLECTION).FindOne(context.Background(), bson.M{"_id": fileID}).Decode(&file)
	return file, err
}

// GetOneByFingerprint is to get a file by its taskID.
func (filestore *FileStore) GetOneByFingerprint(fingerprint string) (models.File, error) {

	var file models.File

	err := db.Collection(FILESCOLLECTION).FindOne(context.Background(), bson.M{"copernicus_details.fingerprint": fingerprint}).Decode(&file)
	return file, err
}

// GetCursorByFolderID is to get a cursor with files from a particular folder.
func (filestore *FileStore) GetCursorByFolderID(folderID string) (*mongo.Cursor, error) {

	cursor, err := db.Collection(FILESCOLLECTION).Find(context.Background(), bson.M{"folder": folderID})
	return cursor, err
}

// GetCursorByAncestors is to get a cursor with files ancestore.
func (filestore *FileStore) GetCursorByAncestors(ancestors string) (*mongo.Cursor, error) {

	cursor, err := db.Collection(FILESCOLLECTION).Find(context.Background(), bson.M{"ancestors": ancestors})
	return cursor, err
}

// UpdateWithId is to update a file's fields.
func (filestore *FileStore) UpdateWithId(file models.File) (objUpdated models.File, err error) {
	filestore.mu.Lock()
	// defer filestore.mu.Unlock()

	filter := bson.M{"_id": file.Id}
	update := bson.M{
		"$set": bson.M{
			"meta":           file.Meta,
			"folder":         file.FolderID,
			"original_title": file.OriginalTitle,
			"ancestors":      file.Ancestors,
			"file_type":      file.FileType,
			"size":           file.Size,
			"total":          file.Total,
		},
	}
	_, erro := db.Collection(FILESCOLLECTION).UpdateOne(context.TODO(), filter, update)
	filestore.mu.Unlock()
	return file, erro
}

func (filestore *FileStore) UpdateFileSize(fileID string, size int) (objUpdated models.File, err error) {
	filestore.mu.Lock()

	var file models.File
	err = db.Collection(FILESCOLLECTION).FindOne(context.Background(), bson.M{"_id": fileID}).Decode(&file)
	if err != nil {
		return file, err
	}

	file.Size = file.Size + int64(size)
	filter := bson.M{"_id": fileID}
	update := bson.M{
		"$set": bson.M{
			"size": file.Size,
		},
	}
	_, erro := db.Collection(FILESCOLLECTION).UpdateOne(context.TODO(), filter, update)
	filestore.mu.Unlock()
	return file, erro
}

// DeleteManyWithAncestore is to delete many folders under the same ancestore.
func (filestore *FileStore) DeleteManyWithAncestore(ancestore string) error {
	_, err := db.Collection(FILESCOLLECTION).DeleteMany(context.Background(), bson.M{"ancestors": ancestore})
	return err
}
