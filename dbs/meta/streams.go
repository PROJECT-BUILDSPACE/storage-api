package metaDB

import (
	"context"

	"github.com/isotiropoulos/storage-api/models"

	"go.mongodb.org/mongo-driver/bson"
)

const (
	STREAMSCOLLECTION = "streams"
)

// InsertOne is to insert an stream in the streams collection
func (streamstore *StreamStore) InsertOne(stream models.Stream) error {
	_, err := db.Collection(STREAMSCOLLECTION).InsertOne(context.Background(), stream)
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

// DeleteManyWithFile is to delete many streams related to the same file.
func (streamstore *StreamStore) DeleteManyWithFile(fileId string) error {
	_, err := db.Collection(STREAMSCOLLECTION).DeleteMany(context.Background(), bson.M{"file_id": fileId})
	return err
}
