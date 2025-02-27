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

			// If not check if path in query params
			if id == "" {

				pathQuery := queryValues.Get("path")
				// Remove trailing slash to handle cases like "/s1/s2/s3/s4/"
				pathQuery = strings.TrimSuffix(pathQuery, "/")
				pathQuery = strings.TrimPrefix(pathQuery, "/")

				// If not start decoding body
				if pathQuery == "" {

					if err := readRequestBody(r, &data); err != nil {
						utils.RespondWithError(w, http.StatusUnauthorized, "Unable to resolve Group.", "Unidentifiable group.", "MID0005")
						return
					}
					q = extractKeysAndValues(data)
				} else {
					pathSegments := strings.Split(pathQuery, "/")
					q = map[string]string{"name": pathSegments[0]}
					collection = "folder"
				}
			} else {
				q = map[string]string{"_id": id}
			}
		} else {
			q = map[string]string{"_id": id}
		}

		groupID, groupName, folderIds, err := grabGroupId(q, collection)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, "Unable to resolve Group.", err.Error(), "MID0011")
			return
		}

		allow := false

		if utils.ItemInArray(claims.Groups, groupName) {
			r.Header.Add("X-Mode", "normal")
			allow = true
		}

		// Second chance: Check if user has access to shared content
		if !allow {
			if folderIds != nil {
				if checkSharedContent(claims.EditorIn, folderIds) {
					r.Header.Add("X-Mode", "editor")
					allow = true
				}

				if checkSharedContent(claims.ViewerIn, folderIds) {
					r.Header.Add("X-Mode", "viewer")
					allow = true
				}

			}
		}

		if !allow {
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
		// r.Header.Add("X-Mode", "normal")
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

	var middleInterface interface{}

	// Convert the data interface to a string
	dataStr, ok := data.(string)
	if ok {
		if json.Valid([]byte(dataStr)) {
			// Unmarshal the JSON string into a map
			err := json.Unmarshal([]byte(dataStr), &middleInterface)
			if err != nil {
				return result
			}
		}
	} else {
		middleInterface = data
	}

	// Use reflection to inspect the keys and values in the data
	v := reflect.ValueOf(middleInterface)
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

func grabGroupId(q map[string]string, collection string) (string, string, []string, error) {

	var groupID string
	var groupName string
	var key string
	var value string
	var err error

	var folderIds []string

	for k, v := range q {
		key = k
		value = v
		break // Only process the first pair
	}

	switch collection {
	case "file":

		if key == "_id" {
			var dbResult models.File
			var dbResult2 models.Folder
			dbResult, err = fileDB.GetOneByID(value)
			if err != nil {
				return "", "", nil, err
			}
			dbResult2, err = folderDB.GetOneByID(dbResult.Ancestors[0])
			if err != nil {
				return "", "", nil, err
			}
			groupID = dbResult2.Id
			groupName = dbResult2.Meta.Title
			folderIds = append(dbResult.Ancestors, dbResult.FolderID)
		} else {
			var dbResult models.Folder
			dbResult, err = folderDB.GetOneByID(value)
			if err != nil {
				return "", "", nil, err
			}
			if dbResult.Level == 0 {
				groupID = dbResult.Id
				groupName = dbResult.Meta.Title
				folderIds = []string{dbResult.Id}
			} else {
				folderIds = append(dbResult.Ancestors, dbResult.Id)
				id := dbResult.Ancestors[0]
				dbResult, err = folderDB.GetOneByID(id)
				if err != nil {
					return "", "", nil, err
				}
				groupID = dbResult.Id
				groupName = dbResult.Meta.Title
			}

		}

	case "folder":
		if key == "name" {
			dbResult, err := folderDB.GetRootByName(value)
			if err != nil {
				return "", "", nil, err
			}
			groupID = dbResult.Id
			groupName = dbResult.Meta.Title
			folderIds = []string{dbResult.Id}
		} else {
			dbResult, err := folderDB.GetOneByID(value)
			if err != nil {
				return "", "", nil, err
			}

			if dbResult.Level == 0 {
				groupID = dbResult.Id
				groupName = dbResult.Meta.Title
				folderIds = []string{dbResult.Id}
			} else {
				folderIds = append(dbResult.Ancestors, dbResult.Id)
				id := dbResult.Ancestors[0]
				dbResult, err = folderDB.GetOneByID(id)
				if err != nil {
					return "", "", nil, err
				}
				groupID = dbResult.Id
				groupName = dbResult.Meta.Title
			}
		}

	case "bucket":
		dbResult, err := folderDB.GetOneByID(value)
		if err != nil {
			return "", "", nil, err
		}
		groupID = dbResult.Id
		groupName = dbResult.Meta.Title
		folderIds = nil
	}

	return groupID, groupName, folderIds, err
}

func checkSharedContent(arr1 []string, arr2 []string) bool {
	// Create a map to store unique elements of the first array
	lookup := make(map[string]struct{})

	// Populate the map with unique elements from the first array
	for _, item := range arr1 {
		lookup[item] = struct{}{}
	}

	// Check if any element of the second array exists in the map
	for _, item := range arr2 {
		if _, found := lookup[item]; found {
			return true
		}
	}

	return false
}
