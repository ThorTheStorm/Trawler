package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	api "trawler/pkg/api/health"
	cfg "trawler/pkg/config"
	git "trawler/pkg/git"
	health "trawler/pkg/health"
	logging "trawler/pkg/logging"
	"trawler/pkg/storage"
	"trawler/pkg/storage/s3"
	"trawler/pkg/vault"
)

var wg sync.WaitGroup              // WaitGroup for goroutines
var s3Client *s3.Client            // S3 client variable
var configPath string              // Configuration variables
var config *cfg.Config             // Global configuration variable
var vaultClient *vault.VaultClient // Vault client variable
var gitConfig *git.GitConfig

// Variables for health
var s3HealthStatus = health.HealthStatusUnknown
var vaultHealthStatus = health.HealthStatusUnknown
var gitHealthStatus = health.HealthStatusUnknown

func init() {

	////////////////////////////////////////////////
	//////////// INITIALIZATION ////////////////////
	////////////////////////////////////////////////

	// Retrieve and save config for further use
	if _, exists := os.LookupEnv("CONFIG_PATH"); exists {
		logging.LogToConsole(logging.DebugLevel, logging.DebugEvent, "CONFIG_PATH environment variable found, using that for config path.")
		configPath = os.Getenv("CONFIG_PATH")
	} else {
		logging.LogToConsole(logging.DebugLevel, logging.DebugEvent, "CONFIG_PATH environment variable not found, using default path for config.")
		configPath = "/config/configuration.yaml"
	}
	logging.LogToConsole(logging.DebugLevel, logging.DebugEvent, "Parsing configuration")
	var err error
	config, err = cfg.ParseConfig(configPath)
	if err != nil && config == nil {
		logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Failed to parse config: %v", err))
		os.Exit(1)
	}

	// Validate file-structure on local storage
	//syscall.Umask(0022) // Set umask to ensure created directories are writable
	if config.Configurations.Global.LocalStorageEnabled {
		err := storage.ValidateLocalStoragePaths(config.Configurations.Global.DataPath, config.Configurations.Global.OnlineCrlsPath, config.Configurations.Global.OfflineCrlsPath, config.Configurations.Global.GitStoragePath)
		if err != nil {
			logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Local storage path validation failed: %v", err))
			os.Exit(1)
		}
		logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, "Local storage paths validated successfully.")
	}

	// Initialize S3 client if S3 storage is enabled and configuration is valid
	s3Config, err := s3.GetS3Config()

	if err != nil && s3Config == nil {
		logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("S3 configuration validation failed: %v", err))
	} else if s3Config != nil {
		logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, "S3 storage enabled and configuration validated successfully.")
		s3Client, err = s3.AWSCreateS3Client()
		if err != nil {
			logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Failed to create S3 client: %v", err))
		}
	}
	// Validate access to the S3 bucket
	_, err = s3Client.HeadBucket(context.TODO(), s3.AWSHeadBucketInput(
		//TODO: consider moving this to a separate health check function that can be called periodically instead of just at startup
		//TODO: Fix this error: Failed to access S3 bucket: operation error S3: HeadBucket, https response error StatusCode: 301, RequestID: , HostID: , api error MovedPermanently: Moved Permanently
		os.Getenv("AWS_S3_BUCKET_NAME"),
	))
	if err != nil {
		logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Failed to access S3 bucket: %v", err))
		s3HealthStatus = health.HealthStatusUnhealthy
	} else {
		logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, "Successfully accessed S3 bucket.")
		s3HealthStatus = health.HealthStatusOK

	}

	// Get vault client
	if os.Getenv("VAULT_ENABLED") == "true" {
		vaultClient = vault.GetVaultClient()
	}

	// Validate Git configuration
	gitConfig, err = git.ValidateGitConfig()
	if err != nil {
		logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Git configuration validation failed: %v", err))
	} else {
		logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, "Git configuration validated successfully.")
	}

	if gitConfig.Enabled == true {
		// Validate access to Git repository
		_, err := git.CloneRepository(config.Configurations.Global.GitStoragePath)
		if err != nil && err != git.ErrRepoAlreadyExists {
			logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Failed to access Git repository: %v", err))
			gitHealthStatus = health.HealthStatusUnhealthy
		} else if err != nil && err == git.ErrRepoAlreadyExists {
			logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, "Git repository already exists locally.")
			gitHealthStatus = health.HealthStatusOK
		} else {
			logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, "Successfully accessed Git repository.")
			gitHealthStatus = health.HealthStatusOK
		}
	} else {
		logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, "Git storage not enabled, skipping Git repository access validation.")
		gitHealthStatus = health.HealthStatusUnknown
	}
}

func main() {
	////////////////////////////////////////////////
	//////////// MAIN PROGRAM //////////////////////
	////////////////////////////////////////////////

	// Define how many processes to wait for
	wg.Add(3)

	// All channels for error-handling and graceful shutdown
	serverError := make(chan error, 1)                // Channel to capture server errors
	errChannel := make(chan logging.ErrorReport, 100) // Channel for all errors occurring in the program
	stopChannel := make(chan struct{}, 1)             // Channel to signal goroutines to stop

	// Start error handling goroutine
	go func() {
		defer wg.Done()
		logging.HandleErrors(errChannel, config)
	}()

	// Start CRL retrieval worker
	go func() {
		defer wg.Done()
		crlRetrievalWorker(config, errChannel, stopChannel)
	}()

	// Start health API server
	go func() {
		defer wg.Done()
		if err := api.StartHealthServer(8080, stopChannel); err != nil {
			logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Health server error: %v", err))
		}
	}()

	// Setup signal capturing for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	<-quit
	logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, "Termination signal received, shutting down...")

	// Signal goroutines to stop
	close(stopChannel)
	close(serverError)

	// Wait for all goroutines to finish
	wg.Wait()

	logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, "Shutting down gracefully...")
}
