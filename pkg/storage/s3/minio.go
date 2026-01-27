package s3

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"trawler/pkg/logging"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func MinIOGetS3ConfigFromEnv() (*S3Config, error) {
	if os.Getenv("MinIO_S3_ENABLED") != "true" {
		os.Setenv("S3_STORAGE_ENABLED", "false")
		return nil, fmt.Errorf("MinIO_S3_ENABLED is not set to true, MinIO S3 storage is disabled")
	} else {
		os.Setenv("S3_STORAGE_ENABLED", "true")
	}

	var missingVars []string

	if os.Getenv("MINIO_S3_API_KEY_ID") == "" {
		missingVars = append(missingVars, "MINIO_S3_API_KEY_ID")
	}
	if os.Getenv("MINIO_S3_API_KEY_SECRET") == "" {
		missingVars = append(missingVars, "MINIO_S3_API_KEY_SECRET")
	}
	if os.Getenv("MINIO_S3_SERVICE_INSTANCE_ID") == "" {
		logging.LogToConsole(logging.WarningLevel, logging.WarningEvent, "Environment variable MINIO_S3_SERVICE_INSTANCE_ID is not set, but not necessary. Continuing.")
	}
	if os.Getenv("MINIO_S3_SSL_ENABLED") == "" {
		logging.LogToConsole(logging.WarningLevel, logging.WarningEvent, "Environment variable MINIO_S3_SSL_ENABLED is not set, defaulting to false. Continuing.")
		os.Setenv("MINIO_S3_SSL_ENABLED", "false")
	}
	if os.Getenv("MINIO_S3_AUTH_ENDPOINT") == "" {
		missingVars = append(missingVars, "MINIO_S3_AUTH_ENDPOINT")
	}
	if os.Getenv("MINIO_S3_SERVICE_ENDPOINT") == "" {
		missingVars = append(missingVars, "MINIO_S3_SERVICE_ENDPOINT")
	}

	if len(missingVars) > 0 {
		for _, v := range missingVars {
			logging.LogToConsole(logging.WarningLevel, logging.WarningEvent, fmt.Sprintf("Environment variable %s is not set.", v))
		}
		return &S3Config{}, fmt.Errorf("Missing required environment variables: %v", missingVars)
	}

	// Convert MINIO_S3_SSL_ENABLED to bool
	sslEnabled := false
	if os.Getenv("MINIO_S3_SSL_ENABLED") == "true" {
		sslEnabled = true
	}

	return &S3Config{
		APIKey:            os.Getenv("MINIO_S3_API_KEY_ID"),
		APISecret:         []byte(os.Getenv("MINIO_S3_API_KEY_SECRET")),
		SSLEnabled:        sslEnabled,
		ServiceInstanceID: os.Getenv("MINIO_S3_SERVICE_INSTANCE_ID"),
		AuthEndpoint:      os.Getenv("MINIO_S3_AUTH_ENDPOINT"),
		ServiceEndpoint:   os.Getenv("MINIO_S3_SERVICE_ENDPOINT"),
	}, nil
}

// ConnectToS3 creates and returns an IBM S3 client. Remember to zero out the API secret after use.
func MinIOConnectToS3(config *S3Config) (*minio.Client, error) {

	conf := &minio.Options{
		Creds:  credentials.NewStaticV4(config.APIKey, string(config.APISecret), ""),
		Secure: config.SSLEnabled,
		Region: "",
	}

	client, err := minio.New(config.ServiceEndpoint, conf)
	if err != nil {
		return nil, fmt.Errorf("Failed to create MinIO S3 client: %v", err)
	}
	return client, nil
}

func MinIOExistS3Bucket(ctx context.Context, client *minio.Client, bucketName string) (bool, error) {
	exist, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return false, fmt.Errorf("Failed to check status of bucket %s: %v", bucketName, err)
	}

	return exist, nil
}

func MinIOSaveCRLToS3(client *minio.Client, bucketName string, objectKey string, crlData []byte) error {
	// Variables and random content to sample, replace when appropriate
	content := bytes.NewReader(crlData)

	ctx := context.Background()

	uploadInfo, err := client.PutObject(
		ctx,
		bucketName,
		objectKey, // Filename in bucket
		content,
		int64(len(crlData)),
		minio.PutObjectOptions{ContentType: "application/pkix-crl"},
	)
	_ = uploadInfo

	if err != nil {
		return fmt.Errorf("Failed to upload CRL to bucket %s with key %s: %v", bucketName, objectKey, err)
	}

	logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, fmt.Sprintf("Successfully uploaded CRL to bucket %s with key %s", bucketName, objectKey))
	return nil
}

func MinIOZeroS3APISecret(secret *[]byte) {
	for i := range *secret {
		(*secret)[i] = 0
	}
}

func MinIOCreateS3Bucket(ctx context.Context, client *minio.Client, bucketName string) error {
	err := client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	if err != nil {
		return fmt.Errorf("Failed to create bucket %s: %v", bucketName, err)
	}

	return nil
}
