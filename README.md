# Storage API
***A REST API for the BUILDSPACE Core Platform***

### The BUILDSPACE Project
Imagine if buildings used Internet of things platforms and building information modelling solutions to collect data and then paired the data with aerial imaging from drones. This is what the EU-funded BUILDSPACE project plans to do. It will create a platform to allow the integration of these heterogeneous data and offer services at building scale. It will allow the integration of digital twins and provide decision support services for energy demand prediction at the city scale. At building level, digital twin services will be tested during the construction of a new building in Poland. In terms of city services, their link to building digital twins will be tested in three cities in Greece, Latvia and Slovenia.

### API Overall Description
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
  

  	└── 📁meta
  
            └── files.go
  
            └── folders.go
  
            └── mainMeta.go
  
            └── parts.go
  
        └── 📁objectStorage
  
            └── objectStorage.go
  

+ 📁 **handlers**: Contains the handlers package source code that includes the HTTP handler functions of the API
  
```
	└── buckets.go: Handler functions for the **Bucket** namespace

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

### Namespace Breakdown
In this section we will describe the API Namespaces and their endpoints in details. We will also provide example requests using ```curl```.

**Note:** For all requests users must provide a JWT Bearer token from the OIDC Provider (same provider as the one in the oauth package).

#### Buckets
---
This namespace contains two endpoints one for creating and on for deleting buckets in the S3-compatible file system.

<div style="display: flex; align-items: center;">
    <img src="post.svg" alt="css-in-readme" style="vertical-align: middle; width: 50px; height: 25px;"> 
    <span style="margin-left: 5px; margin-bottom: 5px;">/bucket</span>
</div>
<table style="border: none;">
  <tr>
    <td valign="top"><img src="post.svg" alt="css-in-readme" style="vertical-align: middle; width: 50px;height: 25px;"> </td>
    <td valign="top">/bucket</td>
  </tr>
</table>


   
```curl
curl --location 'https://api-buildspace.euinno.eu/bucket' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer <JWT Token>
--data '{
    "_id": "< ID of Bucket >" ,
    "name": "< Name of the Bucket >"
}'
```

<div style="background-color: #ffebee; border-left: 5px solid #f44336; padding: 10px; margin: 10px 0; display: inline-flex; align-items: center;">
  <strong style="margin-right: 10px;">DELETE</strong> /bucket/{bucket_id}
</div>

```
curl --location --request DELETE 'https://api-buildspace.euinno.eu/bucket/<bucket_id>' \
--header 'Authorization: <JWT Token> '
```

#### Copernicus
---
This namespace contains four endpoints to manage the Copernicus integrated services.

**Note**: All endpoints need a service path parameter, to specify the service to which the user refers to. Acceptable service parameters are **ads** (for the Atmosphere service) and **cds** (for the Climate Change Service)

<div style="background-color: #e3f2fd; border-left: 5px solid #2196f3; padding: 10px; margin: 10px 0; display: inline-flex; align-items: center;">
  <strong style="margin-right: 10px;">GET</strong> /copernicus/{service}/getall
</div>
This endpoint is used to get a list of all available datasets related to a specific service.

```
curl --location 'https://api-buildspace.euinno.eu/copernicus/{service}/getall' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer <JWT Token>'
```
<div style="background-color: #e3f2fd; border-left: 5px solid #2196f3; padding: 10px; margin: 10px 0; display: inline-flex; align-items: center;">
  <strong style="margin-right: 10px;">GET</strong> /copernicus/{service}/getform/{id}
</div>
This endpoint is used to get the form of a dataset that is rlated to a specific service. The form is then filled and used as the body of the POST request to get access to this specific dataset. Basically, a form contains all the parameters and the rules they need to follow that need to be specified when asking for a Copernicus dataset.

```
curl --location 'https://api-buildspace.euinno.eu/copernicus/{service}/getform/{dataset}' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer <JWT Token>'
```

<div style="background-color: #e8f5e9; border-left: 5px solid #4caf50; padding: 10px; margin: 10px 0; display: inline-flex; align-items: center;">
  <strong style="margin-right: 10px;">POST</strong> /copernicus/{service}/dataset
</div>
In this endpoint the user asks for a specific Copernicus resource. The API transforms and forwards the request to the Copernicus APIs and creates a Copernicus Task. As soon as the task finishes the resources are stored in the Core Platform.
```
curl --location 'https://api-buildspace.euinno.eu/copernicus/{service}/dataset' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer <JWT Token>' \
--data '{
		"datasetname" : "<Dataset Name>",
		"body" : "<JSON of filled form>"
}'
```

<div style="background-color: #e3f2fd; border-left: 5px solid #2196f3; padding: 10px; margin: 10px 0; display: inline-flex; align-items: center;">
  <strong style="margin-right: 10px;">GET</strong> /copernicus/{service}/dataset/{id}
</div>
This is an extra endpoint to out a Copernicus resource to the Core Platform. It is used only in case the POST request failed to put the resource to the Platform.

```
curl --location 'https://api-buildspace.euinno.eu/copernicus/{service}/dataset/{id}' \
--header 'Authorization: Bearer <JWT Token>'
```


#### Files
---
The Files namespace contains endpoints related to data management (upload/download/delete/update).


<div style="background-color: #e8f5e9; border-left: 5px solid #4caf50; padding: 10px; margin: 10px 0; display: inline-flex; align-items: center;">
  <strong style="margin-right: 10px;">POST</strong> /file
</div>
This endpoint is used to upload a file. FIles are uploaded using  upload. The usage of this endpoint depends on the Cotent-Type header of the request.

+ **Content-Type: application/json**: Used to initialize the multipart upload. User passes a File model as a payload containing the folder and the original_title fields. User passes also the total header to specify the number of parts that will be uploaded.
```
curl --location 'https://api-buildspace.euinno.eu/file' \
--header 'total: 2' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer {JWT Token} \
--data '{
    "meta": {
        "title": "Example Data Initialization"
    },
    "folder": "{Folder ID}",
	"original_title": "~/someFile.filetype"
}'
```

+ **Content-Type: application/octet-stream**: Used to upload a part of a file. User passes the binary data (decoded) in the body and also provide the file ID and part number parameters.

```
curl --location 'https://api-buildspace.euinno.eu/file/{File ID}?part={Part Number}' \
--header 'Content-Type: application/octet-stream' \
--header 'Authorization: Bearer {JWT Token} \
--data  {binary data of file part}
```
<div style="background-color: #e3f2fd; border-left: 5px solid #2196f3; padding: 10px; margin: 10px 0; display: inline-flex; align-items: center;">
  <strong style="margin-right: 10px;">GET</strong> /info/file
</div>
An endpoint for retrieving the meta data (a File Model) of an uploaded file

```
curl --location 'https://api-buildspace.euinno.eu/info/file?id={File ID}' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer {JWT Token}'
```


<div style="background-color: #e3f2fd; border-left: 5px solid #2196f3; padding: 10px; margin: 10px 0; display: inline-flex; align-items: center;">
  <strong style="margin-right: 10px;">GET</strong> /file/{id}
</div>
This endpoint is used to download files. The files are downloaded using streaming download. User provides the file id as well as the number of the part in interesct and receives the decoded and decrypted bytes.

```
curl --location 'https://api-buildspace.euinno.eu/file/{File ID}?part={Part Number}' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer {JWT Token}'
```

<div style="background-color: #fff8e1; border-left: 5px solid #ffca28; padding: 10px; margin: 10px 0; display: inline-flex; align-items: center;">
  <strong style="margin-right: 10px;">PUT</strong> /file
</div>
This is the endopoint to update file meta data. Pass a File model of the file that will be updated with the updates included.
**Note**: This endpoint updates the meta data and not the file contents. To update file contents user must delete and re-upload it.

```
curl --location --request PUT 'https://api-buildspace.euinno.eu/file' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer {JWT Token}' \
--data '{Updated File Model}'
```

<div style="background-color: #ffebee; border-left: 5px solid #f44336; padding: 10px; margin: 10px 0; display: inline-flex; align-items: center;">
  <strong style="margin-right: 10px;">DELETE</strong> /file/{id}
</div>
This is the endopoint to delete files. The files are deleted based on ther id.

```
curl --location --request DELETE 'https://api-buildspace.euinno.eu/file/{File ID}' \
--header 'Authorization: Bearer {JWT Token}'
```

#### Folders
---
<div style="background-color: #e8f5e9; border-left: 5px solid #4caf50; padding: 10px; margin: 10px 0; display: inline-flex; align-items: center;">
  <strong style="margin-right: 10px;">POST</strong> /folder
</div>
asc


<div style="background-color: #e3f2fd; border-left: 5px solid #2196f3; padding: 10px; margin: 10px 0; display: inline-flex; align-items: center;">
  <strong style="margin-right: 10px;">GET</strong> /folder
</div>


<div style="background-color: #e3f2fd; border-left: 5px solid #2196f3; padding: 10px; margin: 10px 0; display: inline-flex; align-items: center;">
  <strong style="margin-right: 10px;">GET</strong> /folder/list
</div>

<div style="background-color: #e8f5e9; border-left: 5px solid #4caf50; padding: 10px; margin: 10px 0; display: inline-flex; align-items: center;">
  <strong style="margin-right: 10px;">POST</strong> /folder/copy
</div>
asc

<div style="background-color: #fff8e1; border-left: 5px solid #ffca28; padding: 10px; margin: 10px 0; display: inline-flex; align-items: center;">
  <strong style="margin-right: 10px;">PUT</strong> /folder
</div>


<div style="background-color: #ffebee; border-left: 5px solid #f44336; padding: 10px; margin: 10px 0; display: inline-flex; align-items: center;">
  <strong style="margin-right: 10px;">DELETE</strong> /folder/{id}
</div>



### Run Core Platform 
#### From Source Code
#### Using Docker
#### In Kubernetes

### Funding
This Platform was developed in the context of the [BUILDSPACE](https://buildspaceproject.eu/ "BUILDSPACE") project. BUILDSPACE has received funding from European Union Horizon EUSPA 2021 Programme (HORIZON-EUSPA-2021-SPACE) under grant agreement nº [101082575](https://doi.org/10.3030/101082575 "101082575").
