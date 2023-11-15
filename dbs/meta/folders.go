package metaDB

import (
	"context"
	"time"

	"github.com/isotiropoulos/storage-api/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	FOLDERSSCOLLECTION = "folders"
)

// InsertOne is to insert an folder in the folders collection
func (folderstore *FolderStore) InsertOne(folder models.Folder) error {

	_, err := db.Collection(FOLDERSSCOLLECTION).InsertOne(context.Background(), folder)
	return err
}

// DeleteOneByID is to delete an file from a particular collection by _id.
func (folderstore *FolderStore) DeleteOneByID(folderID string) error {
	_, err := db.Collection(FOLDERSSCOLLECTION).DeleteOne(context.Background(), bson.M{"_id": folderID})
	return err
}

// DeleteManyWithAncestore is to delete many folders under the same ancestore.
func (folderstore *FolderStore) DeleteManyWithAncestore(ancestore string) error {
	_, err := db.Collection(FOLDERSSCOLLECTION).DeleteMany(context.Background(), bson.M{"ancestors": ancestore})
	return err
}

// GetOneByID is to get a folder by ID.
func (folderstore *FolderStore) GetOneByID(folderID string) (models.Folder, error) {
	var folder models.Folder
	err := db.Collection(FOLDERSSCOLLECTION).FindOne(context.Background(), bson.M{"_id": folderID}).Decode(&folder)
	return folder, err
}

// GetOneByID is to get a folder by ID.
func (folderstore *FolderStore) GetRootByName(folderName string) (models.Folder, error) {
	var folder models.Folder
	err := db.Collection(FOLDERSSCOLLECTION).FindOne(context.Background(), bson.M{"meta.title": folderName, "ancestors": nil}).Decode(&folder)
	return folder, err
}

// GetCursorByParent is to get a cursor with folders in a particular parent folder.
func (folderstore *FolderStore) GetCursorByParent(parentID string) (*mongo.Cursor, error) {

	cursor, err := db.Collection(FOLDERSSCOLLECTION).Find(context.Background(), bson.M{"parent": parentID})
	return cursor, err
}

func (folderstore *FolderStore) UpdateFiles(fileId string, folderID string) error {
	folderstore.mu.Lock()
	_, err := db.Collection(FOLDERSSCOLLECTION).UpdateOne(context.Background(), bson.M{"_id": folderID}, bson.D{{"$push", bson.M{"files": fileId}}})
	folderstore.mu.Unlock()
	return err
}

func (folderstore *FolderStore) UpdateWithId(folder models.Folder) (folderUpdated models.Folder, err error) {
	folderstore.mu.Lock()
	filter := bson.M{"_id": folder.Id}
	// NEVER more than 25
	// if len(folder.Meta.Update) >= 25 {
	// 	folder.Meta.Update = folder.Meta.Update[len(folder.Meta.Update)-24:]
	// }
	update := bson.M{
		"$set": bson.M{
			"meta":      folder.Meta,
			"ancestors": folder.Ancestors,
			"parent":    folder.Parent,
			"files":     folder.Files,
			"folders":   folder.Folders,
			"size":      folder.Size,
		},
	}

	_, erro := db.Collection(FOLDERSSCOLLECTION).UpdateOne(context.TODO(), filter, update)
	folderstore.mu.Unlock()
	return folder, erro
}

func getAncestors(ancestors []string) (*mongo.Cursor, error) {
	cursor, err := db.Collection(FOLDERSSCOLLECTION).Find(context.Background(), bson.M{"_id": bson.M{"$in": ancestors}})
	return cursor, err
}

// UpdateMetaAncestors is a function to add to the []Updated when changes happen to all acestores
func (folderstore *FolderStore) UpdateMetaAncestors(ancestors []string, userID string) error {
	folderstore.mu.Lock()
	//Get the folder
	cursor, err := getAncestors(ancestors)

	if err == nil {
		defer cursor.Close(context.Background())

		for cursor.Next(context.Background()) {

			var result bson.M
			var folder models.Folder
			var newMeta models.Meta

			if err = cursor.Decode(&result); err != nil {
				panic(err)
			}

			bsonBytes, _ := bson.Marshal(result)
			bson.Unmarshal(bsonBytes, &folder)

			// Change the update meta
			// NEVER mere than 25 stored updates
			newMeta = folder.Meta
			// if len(newMeta.Update) == 25 {
			// 	// Remove the first item from the array
			// 	newMeta.Update = newMeta.Update[1:]
			// }
			// newMeta.Update = append(newMeta.Update, models.Updated{
			// 	User: userID,
			// 	Date: time.Now(),
			// })
			newMeta.Update.User = userID
			newMeta.Update.Date = time.Now()

			_, err := db.Collection(FOLDERSSCOLLECTION).UpdateOne(context.Background(), bson.M{"_id": folder.Id}, bson.D{{"$set", bson.M{"meta": newMeta}}})
			if err != nil {
				break
			}
		}
	}
	folderstore.mu.Unlock()
	return err
}

// UpdateAncestorSize is a function to update the size of the folder's ancestors
func (folderstore *FolderStore) UpdateAncestorSize(ancestors []string, size int64, add bool) error {

	folderstore.mu.Lock()

	//Get the folder
	cursor, err := getAncestors(ancestors)

	if err == nil {
		defer cursor.Close(context.Background())

		for cursor.Next(context.Background()) {

			var result bson.M
			var folder models.Folder

			if err = cursor.Decode(&result); err != nil {
				panic(err)
			}

			bsonBytes, _ := bson.Marshal(result)
			bson.Unmarshal(bsonBytes, &folder)

			// Change the size
			var newSize int64

			if add {
				newSize = folder.Size + size
			} else {
				newSize = folder.Size - size
			}

			_, err := db.Collection(FOLDERSSCOLLECTION).UpdateOne(context.Background(), bson.M{"_id": folder.Id}, bson.D{{"$set", bson.M{"size": newSize}}})
			if err != nil {
				break
			}
		}
		folderstore.mu.Unlock()

	}

	return err
}
