package utils

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/mitchellh/mapstructure"

	models "github.com/isotiropoulos/storage-api/models"
)

func RespondWithError(w http.ResponseWriter, code int, message string, reason string, internalCode string) {

	err := models.ErrorReport{
		Message:        message + " Please contact the BUILDSPACE Support Team.",
		Reason:         reason,
		Status:         code,
		InternalStatus: internalCode,
	}
	w.Header().Set("Content-Type", "application/json")
	// Set the status code before writing the response body
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(err)
	return
}

func init() {}

func GetPartFromChan(ch <-chan minio.ObjectPart) <-chan minio.ObjectPart {
	firstCh := make(chan minio.ObjectPart)

	go func() {
		firstCh <- <-ch
		close(firstCh)
	}()

	return firstCh
}

// GenerateUUID returns random IDs
func GenerateUUID() (string, error) {
	var uuid [16]byte
	_, err := rand.Read(uuid[:])
	if err != nil {
		return "", err
	}
	// Set the UUID version and variant bits
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0xbf) | 0x80
	return hex.EncodeToString(uuid[:4]) + "-" + hex.EncodeToString(uuid[4:6]) + "-" + hex.EncodeToString(uuid[6:8]) + "-" + hex.EncodeToString(uuid[8:10]) + "-" + hex.EncodeToString(uuid[10:]), nil
}

// CreateFolder is a function to format an item of the folders collection.
func CreateFolder(r models.PostFolderBody, folderID string, ancestors []string, userID string) models.Folder {

	var meta models.Meta
	var dbFolder models.Folder

	meta.Creator = userID
	meta.Descriptions = append(meta.Descriptions, r.Description)
	meta.Title = r.FolderName
	meta.DateCreation = time.Now().UnixNano()
	meta.Read = append(meta.Read, userID)
	meta.Write = append(meta.Write, userID)
	meta.Update = append(meta.Update, models.Updated{
		Date: time.Now(),
		User: userID,
	})

	if r.Parent != "" && r.Parent != ancestors[0] {
		dbFolder = models.Folder{
			Id:        folderID,
			Meta:      meta,
			Parent:    r.Parent,
			Ancestors: append(ancestors, r.Parent),
			Files:     []string{},
			Folders:   []string{},
		}
	} else {
		dbFolder = models.Folder{
			Id:        folderID,
			Meta:      meta,
			Parent:    r.Parent,
			Ancestors: ancestors,
			Files:     []string{},
			Folders:   []string{},
		}
	}

	return dbFolder
}

func GetClaimsFromContext(ctxClaims interface{}) (*models.OidcClaims, error) {
	// Parse the raw claims into a custom struct
	var claims models.OidcClaims
	err := mapstructure.Decode(ctxClaims, &claims)
	if err != nil {
		return nil, fmt.Errorf("could not parse claims: %v", err)
	}
	return &claims, nil
}

// ItemInArray is a function to check if an array contains an item
func ItemInArray(array []string, item string) bool {
	for _, i := range array {
		if i == item {
			return true
		}
	}
	return false
}

// RemoveFromSlice is a function to remove an item from a slice (Keeps order)
func RemoveFromSlice(slice []string, item string) []string {
	var s int

	for pos, val := range slice {
		if val == item {
			s = pos
			break
		}
	}
	return append(slice[:s], slice[s+1:]...)
}
