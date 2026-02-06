package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	cfg "trawler/pkg/config"
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

	// Setup file-structure on local storage
	if config.Configurations.Global.LocalStorageEnabled {
		logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, "Local storage enabled, setting up local storage structure.")

		// Create online CRL storage folder
		onlineCrlsPath := config.Configurations.Global.OnlineCrlsPath
		err := storage.CreateFolderIfNotExists(onlineCrlsPath)
		if err != nil {
			logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Failed to create online CRL storage folder: %v", err))
		} else {
			logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, fmt.Sprintf("Online CRL storage folder ensured at: %s", onlineCrlsPath))
		}

		// Create CA storage folder
		gitStoragePath := config.Configurations.Global.GitStoragePath
		err = storage.CreateFolderIfNotExists(gitStoragePath)
		if err != nil {
			logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Failed to create CA storage folder: %v", err))
		} else {
			logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, fmt.Sprintf("CA storage folder ensured at: %s", gitStoragePath))
		}
	}

	// Initialize S3 client if S3 storage is enabled
	if os.Getenv("S3_STORAGE_ENABLED") == "true" {
		var err error
		s3Client, err = s3.AWSCreateS3Client()
		if err != nil {
			logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Failed to create S3 client: %v", err))
		}
	}

	// Get vault client
	if os.Getenv("VAULT_ENABLED") == "true" {
		vaultClient = vault.GetVaultClient()
	}
	// // Retrieve certificates for CA validation
	// if config.Configurations.Global.GitStoragePath != "" && config.Configurations.Global.GitRepoURL != "" {
	// 	repoExist := false // Set default to false and only set to true if we successfully open or clone the repository
	// 	repo, err := gitops.OpenRepository(config.Configurations.Global.GitStoragePath)
	// 	if err != nil && err == gitops.ErrRepositoryNotExists {
	// 		logging.LogToConsole(logging.WarningLevel, logging.WarningEvent, "Couldn't find repository in given directory. Cloning from remote.")

	// 		repo, err = gitops.CloneRepository(config.Configurations.Global.GitRepoURL, config.Configurations.Global.GitStoragePath)
	// 		if err != nil {
	// 			logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Failed to clone CA storage repository: %v", err))
	// 		} else {
	// 			repoExist = true
	// 		}
	// 	} else if err != nil {
	// 		logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Failed to open CA storage repository: %v", err))
	// 	} else {
	// 		repoExist = true
	// 	}

	// 	// Only attempt to pull changes if repository exists (either opened or cloned successfully)
	// 	if repoExist == true {
	// 		logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, "Pulling latest changes from CA storage repository.")
	// 		err = gitops.PullRepository(repo)
	// 		if err != nil {
	// 			logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Failed to pull latest changes from CA storage repository: %v", err))
	// 		}
	// 	}
	// } else {
	// 	logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, "CA storage path or Git repository URL not set in configuration.")
	// }
}

func main() {
	////////////////////////////////////////////////
	//////////// MAIN PROGRAM //////////////////////
	////////////////////////////////////////////////

	// Define how many processes to wait for
	wg.Add(2)

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
		crlRetrievalWorker(config, errChannel, stopChannel)
		wg.Done()
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
