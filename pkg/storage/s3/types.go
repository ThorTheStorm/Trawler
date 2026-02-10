package s3

import (
	"time"

	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Config struct {
	APIKey            string
	APISecret         []byte
	SSLEnabled        bool
	ServiceInstanceID string
	AuthEndpoint      string
	ServiceEndpoint   string
	Region            string
}

type S3Authentication struct {
	AccessKey []byte
	SecretKey []byte
}

type S3Request struct {
	URL         string // URL to the S3 service
	Bucket      string // The name of bucket on S3
	ObjectKey   string // Filename or path within the bucket
	Method      S3RequestMethod
	ContentType string
	Body        []byte
	Headers     map[string]string
	Date        time.Time
}

type Client = awsS3.Client

type ListBucketsInput = awsS3.ListBucketsInput

type S3RequestMethod string

const (
	MethodGET    S3RequestMethod = "GET"
	MethodPUT    S3RequestMethod = "PUT"
	MethodPOST   S3RequestMethod = "POST"
	MethodDELETE S3RequestMethod = "DELETE"
)

type S3Service string

const (
	ServiceIBM     S3Service = "IBM"
	ServiceMinIO   S3Service = "MinIO"
	ServiceGeneric S3Service = "Generic"
)
