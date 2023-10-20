package metaDB

import (
	"context"

	"github.com/isotiropoulos/storage-api/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	PARTSCOLLECTION = "parts"
)

// InsertOne is to insert an part in the parts collection
func (partstore *PartStore) InsertOne(part models.Part) error {
	_, err := db.Collection(PARTSCOLLECTION).InsertOne(context.Background(), part)
	return err
}

// GetOneByID is to get a part by ID.
func (partstore *PartStore) GetOneByID(partID string) (models.Part, error) {

	var part models.Part

	err := db.Collection(PARTSCOLLECTION).FindOne(context.Background(), bson.M{"_id": partID}).Decode(&part)
	return part, err
}

// GetCursorByFileID is to get a cusror of parts part by file ID.
func (partstore *PartStore) GetCursorByFileID(fileID string) (*mongo.Cursor, error) {

	cursor, err := db.Collection(PARTSCOLLECTION).Find(context.Background(), bson.M{"file_id": fileID})
	return cursor, err

}

// GetCursorByFileID is to get a cusror of parts by stream ID.
func (partstore *PartStore) GetCursorByStreamID(streamID string) (*mongo.Cursor, error) {

	cursor, err := db.Collection(PARTSCOLLECTION).Find(context.Background(), bson.M{"stream_id": streamID})
	return cursor, err

}

// DeleteManyWithStream is to delete many parts related to the same stream.
func (partstore *PartStore) DeleteManyWithStream(streamId string) error {
	_, err := db.Collection(PARTSCOLLECTION).DeleteMany(context.Background(), bson.M{"stream_id": streamId})
	return err
}
