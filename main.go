package main

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
	configParser "trawler/pkg/configYaml"
	crl "trawler/pkg/crl"
	helpers "trawler/pkg/helpers"
	logging "trawler/pkg/logging"
	"trawler/pkg/storage"
	"trawler/pkg/storage/s3"
	// ibms3 "github.com/IBM/ibm-cos-sdk-go/service/s3"
)

type crlTimeStamps struct {
	ThisUpdate     time.Time `json:"thisUpdate"`
	NextUpdate     time.Time `json:"nextUpdate"`
	NextCRLPublish time.Time `json:"nextPublish"`
}

type ErrorReport struct {
	Err         error
	Context     string
	Severity    logging.SeverityLevel
	Criticality logging.CriticalityLevel
}

var wg sync.WaitGroup

var s3Client *s3.Client

func init() {
	// Initialize S3 client if S3 storage is enabled
	if os.Getenv("S3_STORAGE_ENABLED") == "true" {
		var err error
		s3Client, err = s3.AWSCreateS3Client()
		if err != nil {
			logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Failed to create S3 client: %v", err))
			// logging.LogToConsole(logging.DebugLevel, logging.DebugEvent, fmt.Sprintf("s3Client: %v", s3Client))
		}

		// Create S3 bucket if it doesn't exist
		_, err = s3Client.CreateBucket(context.TODO(), s3.AWSCreateBucketInput(
			os.Getenv("S3_BUCKET_NAME"),
		))
		if err != nil {
			logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Error creating S3 bucket: %v", err))
		}
	}
}

func main() {

	////////////////////////////////////////////////
	//////////// INITIALIZATION ////////////////////
	////////////////////////////////////////////////

	// Import configurations from config file
	var configPath string

	if _, exists := os.LookupEnv("CONFIG_PATH"); exists {
		logging.LogToConsole(logging.DebugLevel, logging.DebugEvent, "CONFIG_PATH environment variable found, using that for config path.")
		configPath = os.Getenv("CONFIG_PATH")
	} else {
		logging.LogToConsole(logging.DebugLevel, logging.DebugEvent, "CONFIG_PATH environment variable not found, using default path for config.")
		configPath = "/config/configuration.yaml"
	}

	logging.LogToConsole(logging.DebugLevel, logging.DebugEvent, "Parsing configuration")
	config, err := configParser.ParseConfig(configPath)
	if err != nil {
		logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Failed to parse config: %v", err))
		os.Exit(1)
	}

	////////////////////////////////////////////////
	//////////// MAIN PROGRAM //////////////////////
	////////////////////////////////////////////////

	wg.Add(2)

	serverError := make(chan error, 1)
	errChannel := make(chan ErrorReport, 100)
	stopChannel := make(chan struct{}, 1)

	go func() {
		defer wg.Done()
		handleErrors(errChannel, config)
	}()

	// Start CRL retrieval worker
	go func() {
		crlRetrievalWorker(config, errChannel, stopChannel)
		wg.Done()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	<-quit
	logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, "Termination signal received, shutting down...")

	// Signal goroutines to stop
	close(stopChannel)
	close(serverError)

	wg.Wait()

	logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, "Shutting down gracefully...")
}

func crlRetrievalWorker(config *configParser.Config, errChannel chan<- ErrorReport, stopChan <-chan struct{}) (err error) {

	// Create ticker from config interval
	interval := time.Duration(config.Configurations.Global.PollIntervalMinutes) * time.Minute
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run once immediately, then on interval
	processCRLs(config, errChannel)

	for {
		select {
		case <-stopChan:
			// Clean shutdown signal received
			logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, "Graceful shutdown of Trawler.")
			return
		case <-ticker.C:
			// Execute on interval
			processCRLs(config, errChannel)
		}
	}
}

func processCRLs(config *configParser.Config, errChannel chan<- ErrorReport) error {
	// Loop through all CRL URLs defined in the config file
	for i := 0; i < len(config.Configurations.OnlineCrls); i++ {

		crlUrl := config.Configurations.OnlineCrls[i].URL // Get the URL from the config file
		infoMsgCRL := fmt.Sprintf("Processing CRL from URL: %s", crlUrl)
		logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, strings.Repeat("-", len(infoMsgCRL)))
		logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, infoMsgCRL)
		logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, strings.Repeat("-", len(infoMsgCRL)))

		// Read out the raw CRL data from the crl retrieved from the above URL
		rawCRL, err := crl.RetrieveCertificateRevocationList(crlUrl)
		if err != nil {
			logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Error retrieving CRL: %v", err))
			return nil
		}

		// Parse the raw CRL data into a structured format from ASN.1 DER
		decodedCRL, err := crl.ParseCertificateRevocationList(rawCRL)
		if err != nil {
			logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Error parsing CRL: %v", err))
			return nil
		}

		certData, err := os.ReadFile(config.Configurations.OnlineCrls[i].CertFile)
		if err != nil {
			errChannel <- ErrorReport{
				Err:         err,
				Context:     fmt.Sprintf("Error reading certificate file: %v. Path: %s", err, config.Configurations.OnlineCrls[i].CertFile),
				Severity:    logging.SeverityWarning,
				Criticality: logging.CriticalityLow,
			}
			// logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Error reading certificate file: %v. Path: %s", err, config.Configurations.OnlineCrls[i].CertFile))
			return nil
		}

		// Validate and save the CRL to defined path if valid
		certDataParsed, err := crl.ParseCertificate(certData)
		if err != nil {
			logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Error parsing certificate file: %v", err))
			return nil
		}

		valid, nextPublish, nextPublishTime, err := crl.IsCRLValid(decodedCRL, certDataParsed) // Validate the CRL against the certificate defined in config, and timestamps
		if err != nil {
			logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Error validating CRL: %v", err))
		} else if valid {
			logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, fmt.Sprintf("CRL from %s is valid.", crlUrl))

			var proceedToStore bool = false

			switch nextPublish {
			case false:
				logging.LogToConsole(logging.DebugLevel, logging.DebugEvent, "CRL does not contain NextPublish (ADCS-specific)")
				proceedToStore = true
			case true:
				logging.LogToConsole(logging.DebugLevel, logging.DebugEvent, fmt.Sprintf("CRL contains NextPublish (ADCS-specific). NextPublishTime: %v", nextPublishTime))

				if time.Now().After(nextPublishTime) {
					proceedToStore = true
				}
			}
			if proceedToStore {
				// Store with selected storage backends

				// Local storage
				if localStorageEnabled := config.Configurations.Global.LocalStorageEnabled; localStorageEnabled {
					// Verify and save the CRL to a file
					crlFilePath := fmt.Sprintf("%s%s.crl", config.Configurations.Global.OnlineCrlsPath, config.Configurations.OnlineCrls[i].Name)

					existingFileData, err := os.ReadFile(crlFilePath) // Check if file already exists
					if err == nil && len(existingFileData) > 0 {
						logging.LogToConsole(logging.DebugLevel, logging.DebugEvent, fmt.Sprintf("[Local] Existing CRL file found at %s, comparing hashes.", crlFilePath))

						existingHash := helpers.ComputeHash(existingFileData)
						newHash := helpers.ComputeHash(rawCRL)
						hashMaxLength := 25
						logging.LogToConsole(logging.DebugLevel, logging.DebugEvent, fmt.Sprintf("%-*s %s", hashMaxLength, "Existing CRL Hash:", existingHash))
						logging.LogToConsole(logging.DebugLevel, logging.DebugEvent, fmt.Sprintf("%-*s %s", hashMaxLength, "New CRL Hash:", newHash))

						if existingHash == newHash {
							logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, fmt.Sprintf("[Local] No changes detected in CRL from %s, skipping save.", crlUrl))
							// continue // Skip saving if no changes
						} else {
							logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, fmt.Sprintf("[Local] Changes detected in CRL from %s, updating file.", crlUrl))
							logging.LogToConsole(logging.DebugLevel, logging.DebugEvent, fmt.Sprintf("[Local] CRLFilePath to save to: %s", crlFilePath))
							err = storage.SaveCRLToFile(crlFilePath, rawCRL)
							if err != nil {
								logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("[Local] Error saving CRL to file: %v", err))
							} else {
								logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, fmt.Sprintf("[Local] CRL saved to %s", crlFilePath))
							}
						}
					}
				}
				// S3 storage
				if os.Getenv("S3_STORAGE_ENABLED") == "true" && s3Client != nil {
					// Verify and save the CRL to a file
					crlS3FileName := fmt.Sprintf("%s.crl", config.Configurations.OnlineCrls[i].Name)
					crlS3FullPath := fmt.Sprintf("%s%s.crl", os.Getenv("S3_SERVICE_ENDPOINT"), url.PathEscape(config.Configurations.OnlineCrls[i].Name))

					// Check if file already exists
					s3ObjectOutput, err := s3Client.GetObject(context.TODO(), s3.AWSGetObjectInput(
						os.Getenv("S3_BUCKET_NAME"),
						crlS3FileName,
					))
					if err != nil {
						logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, fmt.Sprintf("[S3] CRL file %s does not exist in S3 bucket, will proceed to upload new file.", crlS3FileName))
					}

					var existingFileData []byte

					if s3ObjectOutput != nil && s3ObjectOutput.Body != nil {
						existingFileData, err = io.ReadAll(s3ObjectOutput.Body)
					}

					// if file exists, continue to compare hashes
					if err == nil && len(existingFileData) > 0 {
						logging.LogToConsole(logging.DebugLevel, logging.DebugEvent, fmt.Sprintf("[S3] Existing CRL file found at %s, comparing hashes.", crlS3FullPath))

						existingHash := helpers.ComputeHash(existingFileData)
						newHash := helpers.ComputeHash(rawCRL)
						hashMaxLength := 25
						logging.LogToConsole(logging.DebugLevel, logging.DebugEvent, fmt.Sprintf("%-*s %s", hashMaxLength, "Existing CRL Hash:", existingHash))
						logging.LogToConsole(logging.DebugLevel, logging.DebugEvent, fmt.Sprintf("%-*s %s", hashMaxLength, "New CRL Hash:", newHash))

						if existingHash == newHash {
							logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, fmt.Sprintf("[S3] No changes detected in CRL from %s, skipping save.", crlUrl))
							// continue // Skip saving if no changes
						} else {
							// Save CRL to S3
							logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, fmt.Sprintf("[S3] Changes detected in CRL from %s compared to version on S3, updating file.", crlUrl))
							logging.LogToConsole(logging.DebugLevel, logging.DebugEvent, fmt.Sprintf("[S3] S3 storage path to save to: %s", crlS3FullPath))
							result, err := s3Client.PutObject(context.TODO(), s3.AWSPutObjectInput(
								os.Getenv("S3_BUCKET_NAME"),
								fmt.Sprintf("%s.crl", config.Configurations.OnlineCrls[i].Name),
								rawCRL,
							))
							if err != nil {
								logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("[S3] Error saving to S3 bucket: %v", err))
							} else {
								logging.LogToConsole(logging.DebugLevel, logging.DebugEvent, fmt.Sprintf("[S3] S3 PutObject result: %v", result))
							}
						}
					}
				} else if s3Client != nil {
					logging.LogToConsole(logging.WarningLevel, logging.WarningEvent, "S3 client not initialized, skipping S3 storage.")
				}
			} // if isNextPublishedPassed
		} else {
			logging.LogToConsole(logging.WarningLevel, logging.WarningEvent, fmt.Sprintf("CRL from %s is NOT valid.", crlUrl))
		}

		// crlTimeStamps := crlTimeStamps{
		// 	ThisUpdate:     crl.ThisUpdate,
		// 	NextUpdate:     crl.NextUpdate,
		// 	NextCRLPublish: nextPublishTime, // This is a ADCS (Microsoft) specific field and not part of the standard x509.RevocationList
		// }

		// log.Printf("crl: %+v\n", crl)
		// pp.Printf("CRL Published Values: %+v\n", crlTimeStamps)                   // Pretty print the CRL timestamps
		// pp.Printf("Is NextCRLPublish Zero Value? %v\n", nextPublishTime.IsZero()) // Check and print if NextCRLPublish is zero value

	} // for i, crlUrl := range config.Configuration.OnlineCrls.URL

	return nil
} // func processCRLs

// handleErrors allows for easy handling of errors throughout the program
func handleErrors(errChannel <-chan ErrorReport, config *configParser.Config) {
	for errReport := range errChannel {
		// Log to console
		logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent,
			fmt.Sprintf("%s: %v", errReport.Context, errReport.Err))

		// Send to external endpoint
		alarm := logging.GenerateAlarm(*config,
			errReport.Context,
			errReport.Criticality,
			errReport.Severity,
			"ThisInstanceAsItWere",
			errReport.Err.Error())

		logging.SendToWebhook(config.Configurations.Alarmathan.WebhookURL, *alarm)
	}
}
