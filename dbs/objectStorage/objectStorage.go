package filestorage

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"

	models "github.com/isotiropoulos/storage-api/models"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// IFileStorage is the Jaqpot Client Interface
type IFileStorage interface {
	// Make a new bucket
	MakeBucket(r models.Bucket) (models.Bucket, error)

	// Delete bucket
	DeleteBucket(bucketId string) error
	// Open Multipart Upload
	OpenMultipart(bucket string, fileID string) (string, error)

	PostPart(bucket string, fileID string, data io.Reader,
		size int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)

	// Close Multipart Upload
	CloseMultipart(bucket string, fileID string, uploadID string, parts []minio.CompletePart) (minio.UploadInfo, error)
	// PostFile(userID string, byteString, fileID string) error

	// // Delete a file
	DeleteFile(fileID string, userID string) error

	// Get file information
	StatFiles(fileID string, bucket string) (minio.ObjectInfo, error)

	// Gets a file stream
	GetFile(fileID string, bucket string, opts minio.GetObjectOptions) (io.ReadCloser, minio.ObjectInfo, http.Header, error)

	// // Copy an file with new name
	CopyFile(originalName string, newName string, bucket string) error
}

// FileStorage ...
type FileStorage struct{}

// MinioClient is a Client of Minio
var minioCore *minio.Core
var minioClient *minio.Client

// Init is a function to create a minio Client.
func Init() {

	_minioURL := os.Getenv("MINIO_URL")
	if _minioURL == "" {
		_minioURL = "localhost:9000"
	}

	_accessKeyID := os.Getenv("ACCESS_KEY")
	if _accessKeyID == "" {
		_accessKeyID = "YHVd2SlQUBNs0xmE"
	}

	_secretAccessKey := os.Getenv("SECRET_ACCESS_KEY")
	if _secretAccessKey == "" {
		_secretAccessKey = "GNw0Mzrawq2pFABP7VdV10Zzcdixohe7"
	}

	// Initialize minio client file.
	// change minio.New to minio.NewCore to have multiparts
	minioCoreLoc, err := minio.NewCore(_minioURL, &minio.Options{
		Creds:  credentials.NewStaticV4(_accessKeyID, _secretAccessKey, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalln(err)
	}
	minioClient, err = minio.New(_minioURL, &minio.Options{
		Creds:  credentials.NewStaticV4(_accessKeyID, _secretAccessKey, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalln(err)
	}
	minioCore = minioCoreLoc
}

// MakeBucket is a function to make a new Bucket.
func (fileStorage *FileStorage) MakeBucket(bucket models.Bucket) (models.Bucket, error) {

	existance, err := minioCore.BucketExists(context.Background(), bucket.Id)
	if err != nil {
		return models.Bucket{}, err
	}

	if !existance {
		err = minioCore.MakeBucket(context.Background(), bucket.Id, minio.MakeBucketOptions{Region: bucket.Id, ObjectLocking: true})
		if err != nil {
			return models.Bucket{}, err
		}
	}

	// Get Bucket Info
	buckets, err := getBuckets()
	if err != nil {
		return models.Bucket{}, err
	}
	for _, b := range buckets {
		if b.Name == bucket.Id {
			return models.Bucket{
				Id:           b.Name,
				Name:         bucket.Name,
				CreationDate: b.CreationDate,
			}, err
		}
	}
	return models.Bucket{}, err

}

// DeleteBucket is a function to delete a Bucket.
func (fileStorage *FileStorage) DeleteBucket(bucketID string) error {

	// List all objects in the bucket.
	objects := minioClient.ListObjects(context.Background(), bucketID, minio.ListObjectsOptions{
		WithVersions: true,
	})

	// Delete all objects and versions in the bucket.
	for object := range objects {
		if object.Err != nil {
			return object.Err
		}

		err := minioCore.RemoveObject(context.Background(), bucketID, object.Key, minio.RemoveObjectOptions{
			GovernanceBypass: true,
			VersionID:        object.VersionID,
		})
		if err != nil {
			return err
		}
	}

	err := minioClient.RemoveBucket(context.Background(), bucketID)
	if err != nil {
		return err
	}
	return err

}

// OpenMultipart is a function to create a Multipart Upload Stream.
func (fileStorage *FileStorage) OpenMultipart(bucket string, fileID string) (string, error) {
	return minioCore.NewMultipartUpload(context.Background(), bucket, fileID, minio.PutObjectOptions{})
}

// PostPart is a function to create a Multipart Upload Stream.
func (fileStorage *FileStorage) PostPart(bucket string, fileID string,
	data io.Reader, size int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	return minioCore.PutObject(context.Background(), bucket, fileID, data, size, "", "", opts)
}

// OpenMultipart is a function to create a Multipart Upload Stream.
func (fileStorage *FileStorage) CloseMultipart(bucket string, fileID string, uploadID string, parts []minio.CompletePart) (minio.UploadInfo, error) {
	return minioCore.CompleteMultipartUpload(context.Background(), bucket, fileID, uploadID, parts, minio.PutObjectOptions{})
}

// DeleteFile deletes a file.
func (fileStorage *FileStorage) DeleteFile(fileID string, bucket string) error {

	err := minioCore.RemoveObject(context.Background(), bucket, fileID, minio.RemoveObjectOptions{
		GovernanceBypass: true,
	})
	if err != nil {
		panic(err)
	}
	return err
}

// StatFiles returns file information.
func (fileStorage *FileStorage) StatFiles(fileID string, bucket string) (minio.ObjectInfo, error) {

	stat, err := minioCore.StatObject(context.Background(), bucket, fileID, minio.GetObjectOptions{})
	if err != nil {
		panic(err)
	}

	return stat, err
}

// StatFiles returns file information.
func (fileStorage *FileStorage) GetFile(fileID string, bucket string, opts minio.GetObjectOptions) (io.ReadCloser, minio.ObjectInfo, http.Header, error) {

	reader, info, header, err := minioCore.GetObject(context.Background(), bucket, fileID, opts)
	if err != nil {
		panic(err)
	}
	return reader, info, header, err
}

// CopyFile is to Copy a file with new name
func (fileStorage *FileStorage) CopyFile(originalName string, newName string, bucket string) error {

	// Source object
	srcOpts := minio.CopySrcOptions{
		Bucket: bucket,
		Object: originalName,
	}

	// Destination object
	dstOpts := minio.CopyDestOptions{
		Bucket: bucket,
		Object: newName,
	}

	// Copy object call
	_, err := minioClient.CopyObject(context.Background(), dstOpts, srcOpts)

	return err

}

// getBuckets lists all buckets.
func getBuckets() ([]minio.BucketInfo, error) {

	buckets, err := minioCore.ListBuckets(context.Background())
	if err != nil {
		return []minio.BucketInfo{}, err
	}

	return buckets, err
}
