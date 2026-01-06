package storage

import (
	"fmt"
	"os"

	"github.com/IBM/ibm-cos-sdk-go/aws"
	"github.com/IBM/ibm-cos-sdk-go/aws/credentials/ibmiam"
	"github.com/IBM/ibm-cos-sdk-go/aws/session"
	"github.com/IBM/ibm-cos-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Config struct {
	APIKey            string
	ServiceInstanceID string
	AuthEndpoint      string
	ServiceEndpoint   string
}

func GetS3ConfigFromEnv() (s3Config, error) {
	switch {
	case os.Getenv("IBM_COS_API_KEY_ID") == "":
		fmt.Println("Environment variable IBM_COS_API_KEY_ID is not set.")
		fallthrough
	case os.Getenv("IBM_COS_SERVICE_INSTANCE_ID") == "":
		fmt.Println("Environment variable IBM_COS_SERVICE_INSTANCE_ID is not set.")
		fallthrough
	case os.Getenv("IBM_COS_AUTH_ENDPOINT") == "":
		fmt.Println("Environment variable IBM_COS_AUTH_ENDPOINT is not set.")
		fallthrough
	case os.Getenv("IBM_COS_SERVICE_ENDPOINT") == "":
		fmt.Println("Environment variable IBM_COS_SERVICE_ENDPOINT is not set.")
		return s3Config{}, fmt.Errorf("one or more required IBM COS environment variables are not set")
	default:
		return s3Config{
			APIKey:            os.Getenv("IBM_COS_API_KEY_ID"),
			ServiceInstanceID: os.Getenv("IBM_COS_SERVICE_INSTANCE_ID"),
			AuthEndpoint:      os.Getenv("IBM_COS_AUTH_ENDPOINT"),
			ServiceEndpoint:   os.Getenv("IBM_COS_SERVICE_ENDPOINT"),
		}, nil
	}

}

func ConnectToS3(config *s3Config) (*s3.S3, error) {

	conf := aws.NewConfig().
		WithEndpoint(config.ServiceEndpoint).
		WithCredentials(ibmiam.NewStaticCredentials(aws.NewConfig(),
			config.AuthEndpoint, config.APIKey, config.ServiceInstanceID)).
		WithS3ForcePathStyle(true)

	sess, err := session.Must(session.NewSession())
	client := s3.New(sess, conf)
	if err != nil {
		return nil, fmt.Errorf("Failed to create S3 client: %v", err)
	}
	return client, nil
}

func ValidateS3Bucket(client *s3.S3, bucketName string) error {
	input := &client.HeadBucketInput{
		Bucket: aws.String(bucketName),
	}

	return nil
}

func SaveCRLToS3(client *s3.S3, bucketName string, objectKey string, crlData []byte) error {
	return nil
}
