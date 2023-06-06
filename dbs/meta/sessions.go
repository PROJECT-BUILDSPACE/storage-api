package metaDB

// import (
// 	"context"

// 	"github.com/isotiropoulos/storage-api/models"

// 	"go.mongodb.org/mongo-driver/bson"
// )

// const (
// 	SESSIONCOLLECTION = "sessions"
// )

// // InsertOne is to insert an session in the sessions collection
// func (sessionstore *SessionStore) InsertOne(session models.Session) error {

// 	_, err := db.Collection(SESSIONCOLLECTION).InsertOne(context.Background(), session)
// 	return err
// }

// // DeleteOneByID is to delete an file from a particular collection by _id.
// func (sessionstore *SessionStore) DeleteOneByID(sessionID string) error {
// 	_, err := db.Collection(SESSIONCOLLECTION).DeleteOne(context.Background(), bson.M{"_id": sessionID})
// 	return err
// }

// // GetOneByID is to get a session by ID.
// func (sessionstore *SessionStore) GetOneByID(sessionID string) (models.Session, error) {

// 	var session models.Session

// 	err := db.Collection(FILESCOLLECTION).FindOne(context.Background(), bson.M{"_id": sessionID}).Decode(&session)
// 	return session, err
// }

// // UpdateWithId is to update a sessions's fields.
// func (sessionstore *SessionStore) UpdateWithId(session models.Session) (objSession models.Session, err error) {
// 	filter := bson.M{"_id": session.Id}
// 	update := bson.M{
// 		"$set": bson.M{
// 			"upload_id":   session.UploadId,
// 			"total_parts": session.TotalParts,
// 			"parts":       session.Parts,
// 			"wait_group":  session.WGroup,
// 			"error":       session.Error,
// 			"completed":   session.Completed,
// 		},
// 	}
// 	_, erro := db.Collection(SESSIONCOLLECTION).UpdateOne(context.TODO(), filter, update)
// 	return session, erro
// }
