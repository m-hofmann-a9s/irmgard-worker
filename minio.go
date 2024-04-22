package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"
)

func SetupObjectStorage(minioEndpoint string, minioAccessKeyID string, minioSecretAccessKey string, writeBucket string) (*minio.Client, error) {

	// Initialize minio client object.
	minioClient, err := minio.New(minioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioAccessKeyID, minioSecretAccessKey, ""),
		Secure: false,
	})
	failOnError(err, "Failed to setup object storage client")

	// only make write bucket - read bucket is written to and managed by irmgard-webserver
	err = minioClient.MakeBucket(context.Background(), writeBucket, minio.MakeBucketOptions{})
	if err != nil {

		// Check if the bucket already exists (which happens if you run this twice)
		exists, errBucketExists := minioClient.BucketExists(context.Background(), writeBucket)

		if errBucketExists == nil && exists {
			log.Info().Msgf("MinIO: The bucket %s already exists", writeBucket)
			return minioClient, nil
		} else {
			failOnError(err, "Failed to create the bucket")
		}
		return minioClient, err
	} else {
		log.Printf("MinIO: Successfully created the bucket %s", writeBucket)
	}
	return minioClient, nil
}

func uploadResult(minioClient *minio.Client, image *Image) error {
	_, err := minioClient.FPutObject(context.Background(), writeBucket, image.StorageLocation, outputFilePath, minio.PutObjectOptions{})
	return err
}

func downloadFile(minioClient *minio.Client, image *Image) (string, error) {
	filePath := fmt.Sprintf("/tmp/object_recognition/%s", uuid.New())
	err := minioClient.FGetObject(context.Background(), readBucket, image.Name, filePath, minio.GetObjectOptions{})
	if err != nil {
		return "", err
	}
	return filePath, nil
}

func deleteFile(client *minio.Client, image *Image) error {
	return client.RemoveObject(context.Background(), readBucket, image.Name, minio.RemoveObjectOptions{})
}
