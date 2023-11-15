package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/gorilla/mux"
	db "github.com/isotiropoulos/storage-api/dbs/meta"
	"github.com/isotiropoulos/storage-api/models"
	auth "github.com/isotiropoulos/storage-api/oauth"
	"github.com/isotiropoulos/storage-api/utils"
)

var fileDB db.IFileStore = &db.FileStore{}
var folderDB db.IFolderStore = &db.FolderStore{}

type IAuth interface {
	AuthMiddleware(h http.HandlerFunc) http.HandlerFunc
	NaiveAuthMiddleware(h http.HandlerFunc) http.HandlerFunc
}

type AuthImplementation struct {
}

func (a *AuthImplementation) AuthMiddleware(h http.HandlerFunc) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiAll := r.Header.Get("Authorization")
		if apiAll == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized.", "No Authorization header.", "MID0001")
			return
		}
		apiKeyAr := strings.Split(apiAll, " ")
		authType := apiKeyAr[0]
		if authType != "Bearer" {
			utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized.", "No Bearer token.", "MID0002")
			return
		}

		apiKey := apiKeyAr[1]
		// log.Println(apiKey)
		claims, err := auth.GetClaims(apiKey)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Unable to resolve claims.", err.Error(), "MID0003")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		ctx = context.WithValue(ctx, "claims", claims)
		_, err = auth.Verifier.Verify(ctx, apiKey)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Unable to initialize verifier.", err.Error(), "MID0004")
			return
		}

		// Find GROUP ID
		// Extract collection
		path := strings.SplitN(r.URL.Path, "?", 2)[0]
		segments := strings.Split(path, "/")
		collection := segments[1]

		// Create query for group ID
		id := ""
		var data interface{}
		var q map[string]string

		vars := mux.Vars(r)

		// Check if id in path params
		id, idExists := vars["id"]

		// If not check if in query params
		if !idExists {
			queryValues := r.URL.Query()
			id = queryValues.Get("id")

			// If not start decoding body
			if id == "" {
				if err := readRequestBody(r, &data); err != nil {
					utils.RespondWithError(w, http.StatusUnauthorized, "Unable to resolve Group.", "Unidentifiable group.", "MID0005")
					return
				}
				q = extractKeysAndValues(data)
			} else {
				q = map[string]string{"_id": id}
			}
		} else {
			q = map[string]string{"_id": id}
		}

		groupID, err := grabGroupId(q, collection)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, "Unable to resolve Group.", err.Error(), "MID0011")
			return
		}
		if !utils.ItemInArray(claims.Groups, groupID) {
			utils.RespondWithError(w, http.StatusForbidden, "Permission Denied.", "No permission rights for user in group.", "MID0006")
			return
		}

		r.Header.Add("X-Group-Id", groupID)

		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *AuthImplementation) NaiveAuthMiddleware(h http.HandlerFunc) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiAll := r.Header.Get("Authorization")
		if apiAll == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized.", "No Authorization header.", "MID0007")
			return
		}
		apiKeyAr := strings.Split(apiAll, " ")
		authType := apiKeyAr[0]
		if authType != "Bearer" {
			utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized.", "No Bearer token.", "MID0008")
			return
		}

		apiKey := apiKeyAr[1]
		claims, err := auth.GetClaims(apiKey)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Unable to resolve claims.", err.Error(), "MID0009")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		ctx = context.WithValue(ctx, "claims", claims)
		_, err = auth.Verifier.Verify(ctx, apiKey)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Unable to initialize verifier.", err.Error(), "MID0010")
			return
		}

		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func readRequestBody(r *http.Request, v interface{}) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	err = r.Body.Close() // Close the original body
	if err != nil {
		return err
	}

	// Create a new ReadCloser from the read bytes and set it as the request body
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	err = json.Unmarshal(body, v)
	if err != nil {
		return err
	}

	return nil
}

func extractKeysAndValues(data interface{}) map[string]string {
	result := make(map[string]string)

	// Use reflection to inspect the keys and values in the data
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Map {
		keys := v.MapKeys()
		for _, key := range keys {
			keyStr := key.Interface().(string)
			value := v.MapIndex(key).Interface()

			if keyStr == "_id" || keyStr == "parent" || keyStr == "folder" {
				result[keyStr] = value.(string)
			}
		}
	}

	return result
}

func grabGroupId(q map[string]string, collection string) (string, error) {

	var group string
	var key string
	var value string
	var err error

	for k, v := range q {
		key = k
		value = v
		break // Only process the first pair
	}

	switch collection {
	case "file":

		if key == "_id" {
			var dbResult models.File
			dbResult, err = fileDB.GetOneByID(value)
			if err != nil {
				return "", err
			}
			group = dbResult.Ancestors[0]
		} else {
			var dbResult models.Folder
			dbResult, err = folderDB.GetOneByID(value)
			if err != nil {
				return "", err
			}
			if dbResult.Level == 0 {
				group = dbResult.Id
			} else {
				group = dbResult.Ancestors[0]
			}
		}

	case "folder":

		dbResult, err := folderDB.GetOneByID(value)
		if err != nil {
			return "", err
		}

		if dbResult.Level == 0 {
			group = dbResult.Id
		} else {
			group = dbResult.Ancestors[0]
		}

	case "bucket":
		dbResult, err := folderDB.GetOneByID(value)
		if err != nil {
			return "", err
		}
		group = dbResult.Id
	}

	return group, err
}
