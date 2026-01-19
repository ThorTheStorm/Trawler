package s3

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
	"trawler/pkg/logging"
)



func GetS3ConfigFromEnv() (*S3Config, error) {
	if os.Getenv("S3_STORAGE_ENABLED") != "true" {
		os.Setenv("S3_STORAGE_ENABLED", "false")
		return nil, fmt.Errorf("S3_STORAGE_ENABLED is not set to true, S3 storage is disabled")
	}

	var missingVars []string

	if os.Getenv("S3_API_KEY_ID") == "" {
		missingVars = append(missingVars, "S3_API_KEY_ID")
	}
	if os.Getenv("S3_API_KEY_SECRET") == "" {
		missingVars = append(missingVars, "S3_API_KEY_SECRET")
	}
	if os.Getenv("S3_SERVICE_INSTANCE_ID") == "" {
		logging.LogToConsole(logging.WarningLevel, logging.WarningEvent, "Environment variable S3_SERVICE_INSTANCE_ID is not set, but not necessary. Continuing.")
	}
	if os.Getenv("S3_SSL_ENABLED") == "" {
		logging.LogToConsole(logging.WarningLevel, logging.WarningEvent, "Environment variable S3_SSL_ENABLED is not set, defaulting to false. Continuing.")
		os.Setenv("S3_SSL_ENABLED", "false")
	}
	if os.Getenv("S3_AUTH_ENDPOINT") == "" {
		missingVars = append(missingVars, "S3_AUTH_ENDPOINT")
	}
	if os.Getenv("S3_SERVICE_ENDPOINT") == "" {
		missingVars = append(missingVars, "S3_SERVICE_ENDPOINT")
	}

	if len(missingVars) > 0 {
		for _, v := range missingVars {
			logging.LogToConsole(logging.WarningLevel, logging.WarningEvent, fmt.Sprintf("Environment variable %s is not set.", v))
		}
		return &S3Config{}, fmt.Errorf("Missing required environment variables: %v", missingVars)
	}

	// Convert S3_SSL_ENABLED to bool
	sslEnabled := false
	if os.Getenv("S3_SSL_ENABLED") == "true" {
		sslEnabled = true
	}

	return &S3Config{
		APIKey:            os.Getenv("S3_API_KEY_ID"),
		APISecret:         []byte(os.Getenv("S3_API_KEY_SECRET")),
		SSLEnabled:        sslEnabled,
		ServiceInstanceID: os.Getenv("S3_SERVICE_INSTANCE_ID"),
		AuthEndpoint:      os.Getenv("S3_AUTH_ENDPOINT"),
		ServiceEndpoint:   os.Getenv("S3_SERVICE_ENDPOINT"),
	}, nil
}

func CreateS3Request(request *S3Request) (*http.Request, error) {
	// Create the HTTP request based on the S3Request struct
	// Populate httpRequest fields based on request
	httpRequest, err := http.NewRequest(
		string(request.Method),
		request.URL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to create HTTP request: %v", err)
	}
	// Set headers
	for key, value := range request.Headers {
		httpRequest.Header.Set(key, value)
	}
	httpRequest.Header.Set("Content-Type", request.ContentType)
	httpRequest.Header.Set("Date", request.Date.Format(time.RFC1123))

	// Set the body if present
	if len(request.Body) > 0 {
		httpRequest.Body = io.NopCloser(bytes.NewReader(request.Body))
	}
	return httpRequest, nil
}
func CreateHTTPClient() (*http.Client, error) {
	client := &http.Client{}
	return client, nil
}

func CloseHTTPIdleConnections(client *http.Client) error {
	client.CloseIdleConnections()
	return nil
}

func SendS3Request(client *http.Client, request *http.Request) (*http.Response, error) {
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("Failed to send S3 request: %v", err)
	}
	return response, nil
}

func createUrlPath(request *S3Request) string {
	urlString := "/"
	if request.Bucket != "" {
		urlString += request.Bucket
		if request.ObjectKey != "" && request.ContentType != "" {
			urlString += fmt.Sprintf("/%s/%s", request.ObjectKey, request.ContentType)
		}
	}
	return urlString
}

// Implements signing in accordance to AWS Signature Version 4: https://docs.aws.amazon.com/AmazonS3/latest/API/RESTAuthentication.html
func CreateS3RequestSignature(request *S3Request, auth *S3Authentication) {
	stringToSign := []byte(fmt.Sprintf("%s\n\n\n%s\n%s", request.Method, request.Date.Format(time.RFC1123), createUrlPath(request))) // Create the string to sign, and make sure it is UTF-8 encoded
	hmac := hmac.New(sha256.New, auth.SecretKey)
	hmac.Write(stringToSign)
	signature := base64.StdEncoding.EncodeToString(hmac.Sum(nil))

	if request.Headers == nil {
		request.Headers = make(map[string]string)
	}
	request.Headers["Authorization"] = fmt.Sprintf("AWS %s:%s", string(auth.AccessKey), signature)

	logging.LogToConsole(logging.DebugLevel, logging.DebugEvent, fmt.Sprintf("Request.Headers: %v", request.Headers))
}

// ExistS3Bucket checks if the specified S3 bucket exists. Needs the input of an HTTP client and a prepared HTTP request.
func ExistS3Bucket(client *http.Client, request *http.Request) (bool, error) {
	resp, err := client.Do(request)
	if err != nil {
		return false, fmt.Errorf("Failed to check status of bucket: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return false, nil
	} else {
		return false, fmt.Errorf("Unexpected response status code: %d", resp.StatusCode)
	}
}

func CreateS3Bucket(client *http.Client, request *http.Request) error {
	resp, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("Failed to create bucket: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Failed to create bucket, unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func SaveCRLToS3(client *http.Client, request *S3Request) error {
	// Variables and random content to sample, replace when appropriate

	httpRequest, err := CreateS3Request(request)
	if err != nil {
		return fmt.Errorf("Failed to create S3 request: %v", err)
	}

	uploadInfo, err := client.Do(httpRequest)
	_ = uploadInfo

	if err != nil {
		return fmt.Errorf("Failed to upload CRL to bucket %s with key %s: %v", request.Bucket, request.ObjectKey, err)
	}

	logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, fmt.Sprintf("Successfully uploaded CRL to bucket %s with key %s", request.Bucket, request.ObjectKey))
	return nil
}

// func CheckS3Type(client interface{}, bucketName string) (S3Service, error) {
// 	switch client.(type) {
// 	case *s3.S3:
// 		return ServiceIBM, nil
// 	case *minio.Client:
// 		return ServiceMinIO, nil
// 	case *http.Client:
// 		return ServiceGeneric, nil
// 	default:
// 		return "", fmt.Errorf("Unsupported S3 client type")
// 	}
// }

func ZeroS3APIAuthentication(s3Authentication *S3Authentication) {
	zeroBytes(&s3Authentication.AccessKey)
	zeroBytes(&s3Authentication.SecretKey)
}

func zeroBytes(data *[]byte) {
	for i := range *data {
		(*data)[i] = 0
	}
}
