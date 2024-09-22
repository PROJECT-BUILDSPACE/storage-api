package metaDB

import (
	"context"

	"github.com/isotiropoulos/storage-api/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	COPERNICUSCOLLECTION = "copernicus"
)

// InsertOne is to insert an part in the parts collection
func (copernicustore *CopernicusStore) InsertOne(copenicus_input models.CopernicusRecord) error {
	_, err := db.Collection(COPERNICUSCOLLECTION).InsertOne(context.Background(), copenicus_input)
	return err
}

// GetOneByID is to get a part by ID.
func (copernicustore *CopernicusStore) GetOneByID(inputId string) (models.CopernicusRecord, error) {

	var coperInput models.CopernicusRecord

	err := db.Collection(COPERNICUSCOLLECTION).FindOne(context.Background(), bson.M{"_id": inputId}).Decode(&coperInput)
	return coperInput, err
}

// GetOneByFileID is to get a part by ID.
func (copernicustore *CopernicusStore) GetOneByFileID(fileId string) (models.CopernicusRecord, error) {

	var coprerRec models.CopernicusRecord

	err := db.Collection(COPERNICUSCOLLECTION).FindOne(context.Background(), bson.M{"file_id": fileId}).Decode(&coprerRec)
	return coprerRec, err
}

// DeleteOneByID is to delete an file from a particular collection by _id.
func (copernicustore *CopernicusStore) DeleteOneByFileID(fileID string) error {
	res, err := db.Collection(COPERNICUSCOLLECTION).DeleteOne(context.Background(), bson.M{"file_id": fileID})
	// If no document matched the deletion filter
	if res.DeletedCount == 0 {
		// You can either log it, return nil, or return a custom error depending on your need
		// For bypassing the error, you can just return nil
		return nil
	}
	return err
}

// UpdateWithId is to update a copernicus' fields.
func (copernicustore *CopernicusStore) UpdateWithId(copenicus_input models.CopernicusRecord) (models.CopernicusRecord, error) {
	copernicustore.mu.Lock()
	// defer filestore.mu.Unlock()

	filter := bson.M{"_id": copenicus_input.Id}
	update := bson.M{
		"$set": bson.M{
			"file_id":            copenicus_input.FileId,
			"dataset_name":       copenicus_input.DatasetName,
			"parameters":         copenicus_input.RequestParams,
			"copernicus_details": copenicus_input.TaskDetails,
		},
	}
	_, erro := db.Collection(COPERNICUSCOLLECTION).UpdateOne(context.TODO(), filter, update)
	copernicustore.mu.Unlock()
	return copenicus_input, erro
}

// GetCursorByService is to get a cursor with datsets by service.
func (copernicustore *CopernicusStore) GetCursorByService(service string) (*mongo.Cursor, error) {

	if service == "all" {
		cursor, err := db.Collection(COPERNICUSCOLLECTION).Find(context.Background(), bson.M{})
		return cursor, err
	}
	cursor, err := db.Collection(COPERNICUSCOLLECTION).Find(context.Background(), bson.M{"task_details.service": service})
	return cursor, err
}
