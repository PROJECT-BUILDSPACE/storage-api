package metaDB

import (
	"context"

	"github.com/isotiropoulos/storage-api/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	STREAMSCOLLECTION = "streams"
)

// InsertOne is to insert an stream in the streams collection
func (streamstore *StreamStore) InsertOne(stream models.Stream) error {
	_, err := db.Collection(STREAMSCOLLECTION).InsertOne(context.Background(), stream)
	return err
}

// DeleteOneByID is to delete an stream from a particular collection by _id.
func (streamstore *StreamStore) DeleteOneByID(streamID string) error {

	_, err := db.Collection(STREAMSCOLLECTION).DeleteOne(context.Background(), bson.M{"_id": streamID})
	return err
}

// GetOneByID is to get a stream by ID.
func (streamstore *StreamStore) GetOneByID(streamID string) (models.Stream, error) {

	var stream models.Stream

	err := db.Collection(STREAMSCOLLECTION).FindOne(context.Background(), bson.M{"_id": streamID}).Decode(&stream)
	return stream, err
}

// GetOneByID is to get a stream by ID.
func (streamstore *StreamStore) GetOneByFileID(fileID string) (models.Stream, error) {

	var stream models.Stream

	err := db.Collection(STREAMSCOLLECTION).FindOne(context.Background(), bson.M{"file_id": fileID}).Decode(&stream)
	return stream, err
}

// GetCursorByFolderID is to get a cursor with streams from a particular folder.
func (streamstore *StreamStore) GetCursorByFolderID(folderID string) (*mongo.Cursor, error) {

	cursor, err := db.Collection(STREAMSCOLLECTION).Find(context.Background(), bson.M{"folder": folderID})
	return cursor, err
}

// GetCursorByAncestors is to get a cursor with streams ancestore.
func (streamstore *StreamStore) GetCursorByAncestors(ancestors string) (*mongo.Cursor, error) {

	cursor, err := db.Collection(STREAMSCOLLECTION).Find(context.Background(), bson.M{"ancestors": ancestors})
	return cursor, err
}

// UpdateWithId is to update a stream's fields.
func (streamstore *StreamStore) UpdateWithId(stream models.Stream) (objUpdated models.Stream, err error) {
	filter := bson.M{"_id": stream.Id}
	update := bson.M{
		"$set": bson.M{
			"file_id": stream.FileID,
			"parts":   stream.Parts,
			"status":  stream.Status,
		},
	}
	_, erro := db.Collection(STREAMSCOLLECTION).UpdateOne(context.TODO(), filter, update)
	return stream, erro
}

// DeleteManyWithAncestore is to delete many folders under the same ancestore.
func (streamstore *StreamStore) DeleteManyWithAncestore(ancestore string) error {
	_, err := db.Collection(STREAMSCOLLECTION).DeleteMany(context.Background(), bson.M{"ancestors": ancestore})
	return err
}
