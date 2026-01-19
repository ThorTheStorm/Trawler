package s3

import (
	"bytes"
	"fmt"
	"os"
	"trawler/pkg/logging"

	"github.com/IBM/ibm-cos-sdk-go/aws"
	"github.com/IBM/ibm-cos-sdk-go/aws/credentials/ibmiam"
	"github.com/IBM/ibm-cos-sdk-go/aws/session"

	"github.com/IBM/ibm-cos-sdk-go/service/s3"
	//"github.com/aws/aws-sdk-go-v2/service/s3"
)

func IBMGetS3ConfigFromEnv() (*S3Config, error) {
	if os.Getenv("IBM_COS_ENABLED") != "true" {
		os.Setenv("S3_STORAGE_ENABLED", "false")
		return nil, fmt.Errorf("IBM_COS_ENABLED is not set to true, S3 storage is disabled")
	} else {
		os.Setenv("S3_STORAGE_ENABLED", "true")
	}

	var missingVars []string

	if os.Getenv("IBM_COS_API_KEY_ID") == "" {
		missingVars = append(missingVars, "IBM_COS_API_KEY_ID")
	}
	if os.Getenv("IBM_COS_API_KEY_SECRET") == "" {
		missingVars = append(missingVars, "IBM_COS_API_KEY_SECRET")
	}
	if os.Getenv("IBM_COS_SERVICE_INSTANCE_ID") == "" {
		logging.LogToConsole(logging.WarningLevel, logging.WarningEvent, "Environment variable IBM_COS_SERVICE_INSTANCE_ID is not set, but not necessary. Continuing.")
	}
	if os.Getenv("IBM_COS_AUTH_ENDPOINT") == "" {
		missingVars = append(missingVars, "IBM_COS_AUTH_ENDPOINT")
	}
	if os.Getenv("IBM_COS_SERVICE_ENDPOINT") == "" {
		missingVars = append(missingVars, "IBM_COS_SERVICE_ENDPOINT")
	}

	if len(missingVars) > 0 {
		for _, v := range missingVars {
			logging.LogToConsole(logging.WarningLevel, logging.WarningEvent, fmt.Sprintf("Environment variable %s is not set.", v))
		}
		return &S3Config{}, fmt.Errorf("Missing required environment variables: %v", missingVars)
	}

	return &S3Config{
		APIKey:            os.Getenv("IBM_COS_API_KEY_ID"),
		APISecret:         []byte(os.Getenv("IBM_COS_API_KEY_SECRET")),
		ServiceInstanceID: os.Getenv("IBM_COS_SERVICE_INSTANCE_ID"),
		AuthEndpoint:      os.Getenv("IBM_COS_AUTH_ENDPOINT"),
		ServiceEndpoint:   os.Getenv("IBM_COS_SERVICE_ENDPOINT"),
	}, nil
}

// ConnectToS3 creates and returns an IBM S3 client. Remember to zero out the API secret after use.
func IBMConnectToS3(config *S3Config) *s3.S3 {

	conf := aws.NewConfig().
		WithEndpoint(config.ServiceEndpoint).
		WithCredentials(ibmiam.NewStaticCredentials(aws.NewConfig(),
			config.AuthEndpoint, config.APIKey, config.ServiceInstanceID)).
		WithS3ForcePathStyle(true)

	sess := session.Must(session.NewSession())
	client := s3.New(sess, conf)
	return client
}

func IBMExistS3Bucket(client *s3.S3, bucketName string) (bool, error) {
	input := &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err := client.HeadBucket(input)
	if err != nil {
		return false, fmt.Errorf("Failed to check status of bucket %s: %v", bucketName, err)
	}

	return true, nil
}

func IBMSaveCRLToS3(client *s3.S3, bucketName string, objectKey string, crlData []byte) error {
	// Variables and random content to sample, replace when appropriate
	content := bytes.NewReader(crlData)

	input := s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   content,
	}
	_, err := client.PutObject(&input)
	if err != nil {
		return fmt.Errorf("Failed to upload CRL to bucket %s with key %s: %v", bucketName, objectKey, err)
	}

	logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, fmt.Sprintf("Successfully uploaded CRL to bucket %s with key %s", bucketName, objectKey))
	return nil
}

func IBMZeroS3APISecret(secret *[]byte) {
	for i := range *secret {
		(*secret)[i] = 0
	}
}

func IBMCreateS3Bucket(client *s3.S3, bucketName string) error {
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err := client.CreateBucket(input)
	if err != nil {
		return fmt.Errorf("Failed to create bucket %s: %v", bucketName, err)
	}

	return nil
}
