package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"trawler/pkg/logging"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

// func (c *Client) ListBuckets(ctx context.Context, input *awsS3.ListBucketsInput) (*awsS3.ListBucketsOutput, error) {
// 	return c.ListBuckets(ctx, input)
// }

// Init S3 variable from AWS S3 environment variables
func init() {
	_, err := AWSValidateS3ConfigFromEnv()
	if err != nil {
		logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("AWS S3 configuration validation failed: %v", err))
	}
}

// AWSGetS3ConfigFromEnv retrieves S3 configuration from the environment variables for AWS S3.
func AWSValidateS3ConfigFromEnv() (*S3Config, error) {
	if os.Getenv("AWS_S3_STORAGE_ENABLED") != "true" {
		os.Setenv("S3_STORAGE_ENABLED", "false")
		return nil, fmt.Errorf("AWS_S3_ENABLED is not set to true, AWS S3 storage is disabled")
	} else {
		os.Setenv("S3_STORAGE_ENABLED", "true")
	}

	var missingVars []string

	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		missingVars = append(missingVars, "AWS_S3_API_KEY_ID")
	}
	if os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		missingVars = append(missingVars, "AWS_S3_API_KEY_SECRET")
	}
	if os.Getenv("AWS_S3_SERVICE_INSTANCE_ID") == "" {
		logging.LogToConsole(logging.WarningLevel, logging.WarningEvent, "Environment variable AWS_S3_SERVICE_INSTANCE_ID is not set, but not necessary. Continuing.")
	}
	if os.Getenv("AWS_S3_AUTH_ENDPOINT") == "" {
		missingVars = append(missingVars, "AWS_S3_AUTH_ENDPOINT")
	}
	if os.Getenv("AWS_S3_SERVICE_ENDPOINT") == "" {
		missingVars = append(missingVars, "AWS_S3_SERVICE_ENDPOINT")
	} else {
		os.Setenv("S3_SERVICE_ENDPOINT", os.Getenv("AWS_S3_SERVICE_ENDPOINT"))
	}
	if os.Getenv("AWS_S3_BUCKET_NAME") == "" {
		missingVars = append(missingVars, "AWS_S3_BUCKET_NAME")
	} else {
		os.Setenv("S3_BUCKET_NAME", os.Getenv("AWS_S3_BUCKET_NAME"))
	}

	if len(missingVars) > 0 {
		for _, v := range missingVars {
			logging.LogToConsole(logging.WarningLevel, logging.WarningEvent, fmt.Sprintf("Environment variable %s is not set.", v))
		}
		return &S3Config{}, fmt.Errorf("Missing required environment variables: %v", missingVars)
	}

	return &S3Config{
		ServiceInstanceID: os.Getenv("AWS_S3_SERVICE_INSTANCE_ID"),
		AuthEndpoint:      os.Getenv("AWS_S3_AUTH_ENDPOINT"),
		ServiceEndpoint:   os.Getenv("AWS_S3_SERVICE_ENDPOINT"),
	}, nil
}

func AWSCreateS3Client() (*Client, error) {
	// Get AWS S3 configuration
	s3Config, err := awsConfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS SDK config, %v", err)
	}

	// Create S3 service client
	client := awsS3.NewFromConfig(s3Config, func(o *awsS3.Options) {
		o.BaseEndpoint = aws.String(os.Getenv("AWS_S3_SERVICE_ENDPOINT"))
		o.UsePathStyle = true
		if os.Getenv("AWS_S3_SSL_ENABLED") == "false" {
			o.EndpointOptions.DisableHTTPS = true
		}
	})
	return client, nil
}

func StringPtr(s string) *string {
	return &s
}

func AWSListBucketsInput() *awsS3.ListBucketsInput {
	return &awsS3.ListBucketsInput{}
}

func AWSCreateBucketInput(bucketName string) *awsS3.CreateBucketInput {
	return &awsS3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}
}

func AWSPutObjectInput(bucketName, objectKey string, data []byte) *awsS3.PutObjectInput {
	return &awsS3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   io.Reader(bytes.NewReader(data)),
	}
}

func AWSGetObjectInput(bucketName, objectKey string) *awsS3.GetObjectInput {
	return &awsS3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}
}
