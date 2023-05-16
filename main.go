package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	db "github.com/isotiropoulos/storage-api/dbs/meta"
	objectstorage "github.com/isotiropoulos/storage-api/dbs/objectStorage"
	handle "github.com/isotiropoulos/storage-api/handlers"
	middleware "github.com/isotiropoulos/storage-api/middleware"
	auth "github.com/isotiropoulos/storage-api/oauth"
	"honnef.co/go/tools/config"
)

type IApp interface {
	GetConfig()
}

type App struct {
	config *config.Config
}

//docker run -it -p 8000:8000 --name quo --env MONGO_URL="mongodb://host.docker.internal:27017" quots

func optionsHandler(w http.ResponseWriter, r *http.Request) {

	headers := w.Header()
	headers.Add("Access-Control-Allow-Origin", "*")
	headers.Add("Vary", "Origin")
	headers.Add("Vary", "Access-Control-Request-Method")
	headers.Add("Vary", "Access-Control-Request-Headers")
	headers.Add("Access-Control-Allow-Headers", "Content-Type, Origin, Accept, token, Authorization, Client-id, Client-secret, Total, total")
	headers.Add("Access-Control-Allow-Methods", "GET, PUT, DELETE, POST,OPTIONS")
	json.NewEncoder(w)
	// return w
}

func main() {
	fmt.Println("Starting the application...")
	objectstorage.Init()
	db.NewDB()
	auth.Init()
	r := mux.NewRouter()
	r.Methods("OPTIONS").HandlerFunc(optionsHandler)

	// Route handles & endpoints
	var mid middleware.IAuth = &middleware.AuthImplementation{}

	// Bucket-wise
	r.HandleFunc("/bucket", mid.AuthMiddleware(handle.MakeBucket)).Methods("POST")
	r.HandleFunc("/bucket/{id}", mid.AuthMiddleware(handle.DeleteBucket)).Methods("DELETE")

	// File-wise
	r.HandleFunc("/file", mid.AuthMiddleware(handle.PostFile)).Methods("POST")
	r.HandleFunc("/file", mid.AuthMiddleware(handle.GetFile)).Queries("id", "{fileId}").Methods("GET")
	r.HandleFunc("/file/info", mid.AuthMiddleware(handle.GetFileInfo)).Queries("id", "{fileId}").Methods("GET")
	r.HandleFunc("/file/{id}", mid.AuthMiddleware(handle.DeleteFile)).Methods("DELETE")
	r.HandleFunc("/file", mid.AuthMiddleware(handle.UpdateFile)).Methods("PUT")
	r.HandleFunc("/copy/file", mid.AuthMiddleware(handle.CopyFile)).Methods("POST")
	r.HandleFunc("/move/file", mid.AuthMiddleware(handle.MoveFile)).Methods("PUT")

	// Folder-wise
	r.HandleFunc("/folder", mid.AuthMiddleware(handle.PostFolder)).Methods("POST")
	r.HandleFunc("/folder/{id}", mid.AuthMiddleware(handle.DeleteFolder)).Methods("DELETE")
	r.HandleFunc("/folder", mid.AuthMiddleware(handle.UpdateFolder)).Methods("PUT")
	r.HandleFunc("/folder", mid.AuthMiddleware(handle.GetFolder)).Queries("id", "{folderId}").Methods("GET")
	r.HandleFunc("/folder/list", mid.AuthMiddleware(handle.GetFolderItems)).Queries("id", "{folderId}").Methods("GET")

	loggedRouter := handlers.LoggingHandler(os.Stdout, r)
	allowedOrigins := handlers.AllowedOrigins([]string{"*"})
	allowedMethods := handlers.AllowedMethods([]string{"GET", "POST", "DELETE", "PUT", "OPTIONS"})
	allowedHeaders := handlers.AllowedHeaders([]string{"*"})
	exposedHeaders := handlers.ExposedHeaders([]string{"*"})
	log.Fatal(http.ListenAndServe(":30000", handlers.CORS(allowedOrigins, allowedHeaders, allowedMethods, exposedHeaders, handlers.IgnoreOptions())(loggedRouter)))
}
