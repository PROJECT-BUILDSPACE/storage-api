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
	_ "github.com/isotiropoulos/storage-api/docs"
	handle "github.com/isotiropoulos/storage-api/handlers"
	"github.com/isotiropoulos/storage-api/middleware"
	auth "github.com/isotiropoulos/storage-api/oauth"
	httpSwagger "github.com/swaggo/http-swagger"
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
	// headers.Add("Vary", "Origin")
	headers.Add("Vary", "Access-Control-Request-Method")
	headers.Add("Vary", "Access-Control-Request-Headers")
	headers.Add("Access-Control-Allow-Headers", "Content-Type, Origin, Accept, token, Authorization, X-Group-Id, Total, total")
	headers.Add("Access-Control-Allow-Methods", "GET, PUT, DELETE, POST, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	json.NewEncoder(w)
	// return w
}

// @title BUILSPACE Core Platform Swagger API
// @version 1.0
// @description This is a swagger for the API that was developed as a core platform of the BUILDSPACE project.
// @termsOfService http://swagger.io/terms/

// @SecurityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

// @contact.name BUILDSPACE Core Platform Support
// @contact.url http://www.swagger.io/support
// @contact.email isotiropoulos@singularlogic.eu

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

func main() {
	fmt.Println("Starting the application...")
	objectstorage.Init()
	db.NewDB()
	auth.Init()
	r := mux.NewRouter()
	r.Methods("OPTIONS").HandlerFunc(optionsHandler)

	deployment := os.Getenv("DEPLOYMENT")
	if deployment == "" {
		deployment = "PROD" // "LOCAL", "PROD"
	}
	// Route handles & endpoints
	var mid middleware.IAuth = &middleware.AuthImplementation{}

	r.HandleFunc("/bucket", mid.NaiveAuthMiddleware(handle.MakeBucket)).Methods("POST")
	r.HandleFunc("/bucket/{id}", mid.AuthMiddleware(handle.DeleteBucket)).Methods("DELETE")

	// File-wise
	if deployment == "PROD" {
		r.HandleFunc("/file", mid.AuthMiddleware(handle.PostFile)).Methods("POST")

		r.HandleFunc("/file/copy", mid.AuthMiddleware(handle.CopyFile)).Methods("POST")
		r.HandleFunc("/file/move", mid.AuthMiddleware(handle.MoveFile)).Methods("PUT")
		r.HandleFunc("/file/info/{id}", mid.AuthMiddleware(handle.GetFileInfo)).Methods("GET")
		r.HandleFunc("/file/{id}", mid.AuthMiddleware(handle.PostFile)).Queries("part", "{partNum}").Methods("POST")
		r.HandleFunc("/file/{id}", mid.AuthMiddleware(handle.GetFile)).Queries("part", "{partNum}").Methods("GET")
		r.HandleFunc("/file/{id}", mid.AuthMiddleware(handle.DeleteFile)).Methods("DELETE")
		r.HandleFunc("/file", mid.AuthMiddleware(handle.UpdateFile)).Methods("PUT")
	} else if deployment == "LOCAL" {
		// r.HandleFunc("/file", mid.AuthMiddleware(handle.PostFileLocal)).Methods("POST")
		// r.HandleFunc("/file/{id}", mid.AuthMiddleware(handle.GetFileLocal)).Methods("GET")
		// r.HandleFunc("/info/file", mid.AuthMiddleware(handle.GetFileInfoLocal)).Queries("id", "{fileId}").Methods("GET")
		// r.HandleFunc("/file/{id}", mid.AuthMiddleware(handle.DeleteFileLocal)).Methods("DELETE")
		// r.HandleFunc("/file", mid.AuthMiddleware(handle.UpdateFileLocal)).Methods("PUT")
		// r.HandleFunc("/copy/file", mid.AuthMiddleware(handle.CopyFileLocal)).Methods("POST")
		// r.HandleFunc("/move/file", mid.AuthMiddleware(handle.MoveFileLocal)).Methods("PUT")
	} else {
		log.Panicln("Deployment " + deployment + " not supprted. Please select PROD or LOCAL. If still in doubt contact the BUILDSPACE Support Team.")
	}

	// Folder-wise
	r.HandleFunc("/folder", mid.AuthMiddleware(handle.PostFolder)).Methods("POST")
	r.HandleFunc("/folder/{id}", mid.AuthMiddleware(handle.DeleteFolder)).Methods("DELETE")
	r.HandleFunc("/folder", mid.AuthMiddleware(handle.UpdateFolder)).Methods("PUT")
	r.HandleFunc("/folder", mid.AuthMiddleware(handle.GetFolder)).Queries("id", "{folderId}").Methods("GET")
	r.HandleFunc("/folder/list", mid.AuthMiddleware(handle.GetFolderItems)).Queries("id", "{folderId}").Methods("GET")

	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	loggedRouter := handlers.LoggingHandler(os.Stdout, r)
	allowedOrigins := handlers.AllowedOrigins([]string{"*"})
	allowedMethods := handlers.AllowedMethods([]string{"GET", "POST", "DELETE", "PUT", "OPTIONS"})
	allowedHeaders := handlers.AllowedHeaders([]string{"*"})
	exposedHeaders := handlers.ExposedHeaders([]string{"*"})
	log.Fatal(http.ListenAndServe(":30000", handlers.CORS(allowedOrigins, allowedHeaders, allowedMethods, exposedHeaders, handlers.IgnoreOptions())(loggedRouter)))
}
