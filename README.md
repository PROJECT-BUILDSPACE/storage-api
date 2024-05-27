# Storage API
***A REST API for the BUILDSPACE Core Platform***

## The BUILDSPACE Project
Imagine if buildings used Internet of things platforms and building information modelling solutions to collect data and then paired the data with aerial imaging from drones. This is what the EU-funded BUILDSPACE project plans to do. It will create a platform to allow the integration of these heterogeneous data and offer services at building scale. It will allow the integration of digital twins and provide decision support services for energy demand prediction at the city scale. At building level, digital twin services will be tested during the construction of a new building in Poland. In terms of city services, their link to building digital twins will be tested in three cities in Greece, Latvia and Slovenia.

## Consuming the API
To consume the API it is **highly recomended** to use the [Python Core Platform Client](https://github.com/PROJECT-BUILDSPACE/CorePlatPy "Python Core Platform Client"). 

In case that you need to implement custom code please read read carefully the following instructions as well as the [Swagger documentation](https://api-buildspace.euinno.eu/swagger/index.html#/ "Swagger documentation").

## API Overall Description
This repository contains the source code of the BUILDSPACE Core Platform REST API. The API contains **four namespaces**, namely:

| Namespace  | Description  |
| ------------ | ------------ |
| Buckets  | Contains HTTP endpoints related to S3 buckets  |
| Copernicus  | Contains HTTP endpoints related to the Copernicus Services Integration  |
| Files  |  Contains HTTP endpoints related to file management |
| Folders  | Contains HTTP endpoints related to folder managemet  |

The API is developed in Go and all dependencies can be found in the ```go.mod``` file. The Go version used was v1.20. Bellow follows a short, yet inlightning description of the repository's structure:
+ ```go.mod``` : Dependencies of the project. Run ```go mod init``` to initialize the packages and get all dependencies.

+ ```main.go```: Main file of the API. Run ```go run ./main.go``` to initialize the API.  

+ ```Dockerfile```: The Docker manifest for creating the storage-api image.

+ 📁 **docs**: Docs package contains the Swagger documentation. Online version of the Swagger can also be found [here](https://api-buildspace.euinno.eu/swagger/index.html#/ "here").

+ 📁 **dbs**: Contains the source code of the **metaDB** and **filestorage** packages. The metaDB package manages the meta information of the uploaded files, in cotrast to the filestorage that manages the upload/download/copy/etc. of files in the filesyste
  

  	└── 📁**meta**
  
            └── files.go
  
            └── folders.go
  
            └── mainMeta.go
  
            └── parts.go
  
        └── 📁**objectStorage**
  
            └── objectStorage.go
  

+ 📁 **handlers**: Contains the handlers package source code that includes the HTTP handler functions of the API
  
```
	└── buckets.go: Handler functions for the <strong>Bucket</strong> namespace

	└── copernicus.go: Handler functions for the **Copernicus** namespace

	└── folders.go: Handler functions for the **Folder** namespace

	└── local_files.go: **DEPRECATED** Handler functions for the **Files** namespace (used for local deployment)

	└── prod_files.go: Handler functions for the **Files** namespace
```
+ 📁 **middleware**: Contains the middleware package source code used to identify user (by interpreting the JWT Bearer Token) before perfoming any request and extract useful information regarding the Organizations and permissions of the user.

+ 📁 **oauth**: Contains the oauth package source code used to connect the API with the OpenID Connect Provider.

+ 📁 **utils**: Contains the utils package source code that contains a set of helper functions needed throughout the whole API

+ 📁 **models**: Contains the models package that is a set of all models (structs) of the API. All endpoints and functions of the API interconnect with eachother with specific predefined structures, enhancing the security of the API.

+ 📁 **globals**: Contains the globals package used to initialize global variables for the API.

## Namespace Breakdown
In this section we will describe the API Namespaces and their endpoints in details. We will also provide example requests using ```curl```.

**Note:** For all requests users must provide a JWT Bearer token from the OIDC Provider (same provider as the one in the oauth package).

### Buckets
---
This namespace contains two endpoints one for creating and on for deleting buckets in the S3-compatible file system.

<div>
	<img src="post.svg" alt="css-in-readme" style="vertical-align: middle; width: 80px; height: 80px;">
</div>

| Path | Body | Query Parameters |
| ---- | --------------- | ---------------- |
| /bucket | models.Bucket  | Not applicable   |

   
```curl
curl --location 'https://api-buildspace.euinno.eu/bucket' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer {JWT Token}
--data '{
    "_id": "{ID of Bucket}" ,
    "name": "{Name of the Bucket}"
}'
```


<div>
	<img src="delete.svg" alt="css-in-readme" style="vertical-align: middle; width: 90px; height: 90px;">
</div>


| Path | Body | Query Parameters |
| ---- | --------------- | ---------------- |
| /bucket/{bucket ID} | Not applicable  | Not applicable   |

```
curl --location --request DELETE 'https://api-buildspace.euinno.eu/bucket/{bucket ID}' \
--header 'Authorization: {JWT Token}'
```

### Copernicus
---
This namespace contains four endpoints to manage the Copernicus integrated services.

**Note**: All endpoints need a service path parameter, to specify the service to which the user refers to. Acceptable service parameters are **ads** (for the Atmosphere service) and **cds** (for the Climate Change Service)


<div>
	<img src="get.svg" alt="css-in-readme" style="vertical-align: middle; width: 80px; height: 80px;">
</div>


| Path | Body | Query Parameters |
| ---- | --------------- | ---------------- |
| /copernicus/{service}/getall | Not applicable  | Not applicable   


This endpoint is used to get a list of all available datasets related to a specific service.

```
curl --location 'https://api-buildspace.euinno.eu/copernicus/{service}/getall' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer {JWT Token}'
```

<div>
	<img src="get.svg" alt="css-in-readme" style="vertical-align: middle; width: 80px; height: 80px;">
</div>


| Path | Body | Query Parameters |
| ---- | --------------- | ---------------- |
| /copernicus/{service}/getform/{id} | Not applicable  | Not applicable   

This endpoint is used to get the form of a dataset that is rlated to a specific service. The form is then filled and used as the body of the POST request to get access to this specific dataset. Basically, a form contains all the parameters and the rules they need to follow that need to be specified when asking for a Copernicus dataset.

```
curl --location 'https://api-buildspace.euinno.eu/copernicus/{service}/getform/{dataset}' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer {JWT Token}'
```


<div>
	<img src="post.svg" alt="css-in-readme" style="vertical-align: middle; width: 80px; height: 80px;">
</div>

| Path | Body | Query Parameters |
| ---- | --------------- | ---------------- |
| /copernicus/{service}/dataset | models.CpernicusInput  | Not applicable   |


In this endpoint the user asks for a specific Copernicus resource. The API transforms and forwards the request to the Copernicus APIs and creates a Copernicus Task. As soon as the task finishes the resources are stored in the Core Platform.
```
curl --location 'https://api-buildspace.euinno.eu/copernicus/{service}/dataset' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer {JWT Token}' \
--data '{
		"datasetname" : "{Dataset Name}",
		"body" : "{JSON of filled form}"
}'
```


<div>
	<img src="get.svg" alt="css-in-readme" style="vertical-align: middle; width: 80px; height: 80px;">
</div>


| Path | Body | Query Parameters |
| ---- | --------------- | ---------------- |
| /copernicus/{service}/dataset/{id} | Not applicable  | Not applicable   


This is an extra endpoint to out a Copernicus resource to the Core Platform. It is used only in case the POST request failed to put the resource to the Platform.

```
curl --location 'https://api-buildspace.euinno.eu/copernicus/{service}/dataset/{id}' \
--header 'Authorization: Bearer {JWT Token}'
```


#### Files

The Files namespace contains endpoints related to data management (upload/download/delete/update).

<div>
	<img src="post.svg" alt="css-in-readme" style="vertical-align: middle; width: 80px; height: 80px;">
</div>

This endpoint is used to upload a file. FIles are uploaded using  upload. The usage of this endpoint depends on the Cotent-Type header of the request.

+ **Content-Type: application/json**: Used to initialize the multipart upload. User passes a File model as a payload containing the folder and the original_title fields. User passes also the total header to specify the number of parts that will be uploaded.


| Path | Body | Query Parameters |
| ---- | --------------- | ---------------- |
| /file | models.file   | Not applicable   |


```
curl --location 'https://api-buildspace.euinno.eu/file' \
--header 'total: {number of parts}' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer {JWT Token} \
--data '{
    "meta": {
        "title": "Example Data Initialization"
    },
    "folder": "{Folder ID}",
	"original_title": "{~/someFile.filetype}"
}'
```

+ **Content-Type: application/octet-stream**: Used to upload a part of a file. User passes the binary data (decoded) in the body and also provide the file ID and part number parameters.

| Path | Body | Query Parameters |
| ---- | --------------- | ---------------- |
| /file |  bytes  | part   |



```
curl --location 'https://api-buildspace.euinno.eu/file/{File ID}?part={part_number}' \
--header 'Content-Type: application/octet-stream' \
--header 'Authorization: Bearer {JWT Token} \
--data  {binary data of file part}
```

<div>
	<img src="get.svg" alt="css-in-readme" style="vertical-align: middle; width: 80px; height: 80px;">
</div>




| Path | Body | Query Parameters |
| ---- | --------------- | ---------------- |
| /info/file | Not applicable  | id   |

An endpoint for retrieving the meta data (a File Model) of an uploaded file. Pass the file's ID as a query parameter

```
curl --location 'https://api-buildspace.euinno.eu/info/file?id={id}' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer {JWT Token}'
```


<div>
	<img src="get.svg" alt="css-in-readme" style="vertical-align: middle; width: 80px; height: 80px;">
</div>


| Path | Body | Query Parameters |
| ---- | --------------- | ---------------- |
| /file/{id} | Not applicable  | part   |


This endpoint is used to download files. The files are downloaded using streaming download. User provides the file id as well as the number of the part in interesct and receives the decoded and decrypted bytes.

```
curl --location 'https://api-buildspace.euinno.eu/file/{id}?part={Part Number}' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer {JWT Token}'
```


<div>
	<img src="put.svg" alt="css-in-readme" style="vertical-align: middle; width: 80px; height: 80px;">
</div>


| Path | Body | Query Parameters |
| ---- | --------------- | ---------------- |
| /file | models.File  | Not applicable   


This is the endopoint to update file meta data. Pass a File model of the file that will be updated with the updates included.
**Note**: This endpoint updates the meta data and not the file contents. To update file contents user must delete and re-upload it.

```
curl --location --request PUT 'https://api-buildspace.euinno.eu/file' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer {JWT Token}' \
--data '{Updated File Model}'
```


<div>
	<img src="delete.svg" alt="css-in-readme" style="vertical-align: middle; width: 90px; height: 90px;">
</div>


| Path | Body | Query Parameters |
| ---- | --------------- | ---------------- |
| /file/{id} | Not applicable  | Not applicable   

This is the endopoint to delete files. The files are deleted based on ther id.

```
curl --location --request DELETE 'https://api-buildspace.euinno.eu/file/{id}' \
--header 'Authorization: Bearer {JWT Token}'
```

#### Folders
---

<div>
	<img src="post.svg" alt="css-in-readme" style="vertical-align: middle; width: 80px; height: 80px;">
</div>


| Path | Body | Query Parameters |
| ---- | --------------- | ---------------- |
| /folder | models.Folder  | Not applicable   |

This endpoint is to create a new folder. Essential fields of the body are meta.title (folder's name) and parent (location).

```
curl --location 'https://api-buildspace.euinno.eu/folder' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer {JWT Token}' \
--data '{
    "meta": {"title": "{folder title}"},
    "parent": "{parent_folder_id}"
}'
```


<div>
	<img src="get.svg" alt="css-in-readme" style="vertical-align: middle; width: 80px; height: 80px;">
</div>


| Path | Body | Query Parameters |
| ---- | --------------- | ---------------- |
| /folder | Not Applicable  | id   |

This endpoint is to retrieve a folder. Pass the folder's ID as a query parameter.

```
curl --location 'https://api-buildspace.euinno.eu/folder?id={folder_id}' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer {JWT Token}'
```



<div>
	<img src="get.svg" alt="css-in-readme" style="vertical-align: middle; width: 80px; height: 80px;">
</div>


| Path | Body | Query Parameters |
| ---- | --------------- | ---------------- |
| /folder/list | Not Applicable  | id   |

This endpoint is to list a folder's items. Pass the folder's ID as a query parameter and get as a result a model.FolderList.

```
curl --location 'https://api-buildspace.euinno.eu/folder/list?id={folder_id}' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer {JWT Token}'
```

<div>
	<img src="post.svg" alt="css-in-readme" style="vertical-align: middle; width: 80px; height: 80px;">
</div>


| Path | Body | Query Parameters |
| ---- | --------------- | ---------------- |
| /folder/copy | models.CopyMoveBody  | Not applicable   |

Copy a folder with all nested items. This endpoint is also used to share a folder with another organization.

```
curl --location 'https://api-buildspace.euinno.eu/folder/copy' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer {JWT Token}' \
--data '{
    "_id": "{ID fo the folder that is to be copied}",
    "destination": "{ID of where to paste}",
    "new_name": "{New name of the pasted folder}"
}
'
```


<div>
	<img src="put.svg" alt="css-in-readme" style="vertical-align: middle; width: 80px; height: 80px;">
</div>


| Path | Body | Query Parameters |
| ---- | --------------- | ---------------- |
| /folder | models.Folder  | Not applicable   


Update a folders meta data by the ID. Pass a model.Folder with the updates that are needed.


```
curl --location --request PUT 'https://api-buildspace.euinno.eu/folder' \
--header 'Content-Type: application/json' \
--header 'Authorization: {JWT Token}' \
--data '{Updated Folder Model}'
```



<div>
	<img src="delete.svg" alt="css-in-readme" style="vertical-align: middle; width: 90px; height: 90px;">
</div>


| Path | Body | Query Parameters |
| ---- | --------------- | ---------------- |
| /folder/{id} | Not applicable  | Not applicable   |

This is the endopoint to delete folders with all nested items. The folders are deleted based on ther id.

```
curl --location --request DELETE 'https://api-buildspace.euinno.eu/folder/{id}' \
--header 'Authorization: {JWT Token}'
```


### Run Core Platform
#### In Kubernetes (Recommended)
Core Platform can run in a Kubernetes Cluster. If intrested visit the [Kubernetes manifests repository](https://github.com/PROJECT-BUILDSPACE/kubernetes-manifests "Kubernetes manifests repository") and folow the README.md instructions.

#### From Source Code
To run the Core Platform from Source Code (local deployment), one must have running:
+ a MongoDB at mongodb://localhost:27017
+ a MinIO instance at http://localhost:9000
+ a Keycloak instance at http://localhost:30105

Then run at the directory of main.go 

```go mod init```

and as soon as it finishes:

```go run main.go```

**Note 1: ** These services can be deployed in a local cluster following instructions [here](https://github.com/PROJECT-BUILDSPACE/kubernetes-manifests "here").
**Note 2: ** In the need of customization, one should change the URL's of these services and/or the implementation of the interfaces in the ```dbs``` folder

#### Using Docker
Run the Core Platform using the official Docker image [buildspace/storage-api](https://hub.docker.com/repository/docker/buildspace/storage-api/ "buildspace/storage-api").

**Note: ** The same restrictions as running from source code apply here as well.

### Funding
This Platform was developed in the context of the [BUILDSPACE](https://buildspaceproject.eu/ "BUILDSPACE") project. BUILDSPACE has received funding from European Union Horizon EUSPA 2021 Programme (HORIZON-EUSPA-2021-SPACE) under grant agreement nº [101082575](https://doi.org/10.3030/101082575 "101082575").
