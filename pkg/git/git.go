package gitops

import (
	"fmt"
	"os"
	"reflect"
	cfg "trawler/pkg/config"
	"trawler/pkg/logging"
	"trawler/pkg/storage"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

var ErrRepositoryNotExists = git.ErrRepositoryNotExists
var gitConfig GitConfig

type GitConfig struct {
	Enabled       bool
	RepositoryURL string
	Branch        string
	Username      string
	AccessToken   string
}

type DirectoriesFromTo struct {
	From string
	To   string
}

var ErrRepoAlreadyExists = git.ErrRepositoryAlreadyExists

func ValidateGitConfig() (*GitConfig, error) {
	var config = GitConfig{
		Enabled:       os.Getenv("GIT_ENABLED") == "true",
		RepositoryURL: os.Getenv("GIT_REPO_URL"),
		Branch:        os.Getenv("GIT_BRANCH"),
		Username:      os.Getenv("GIT_USERNAME"),
		AccessToken:   os.Getenv("GIT_ACCESS_TOKEN"),
	}

	var missingVars []string

	value := reflect.ValueOf(gitConfig)
	typ := value.Type()

	for i := 0; i < value.NumField(); i++ {
		field := typ.Field(i)
		varName := field.Name
		varValue := value.Field(i).Interface()

		switch varName {
		case "GIT_ENABLED":
			if varValue != "true" && varValue != "false" {
				missingVars = append(missingVars, "GIT_ENABLED must be 'true' or 'false'")
			}
		case "GIT_REPO_URL", "GIT_BRANCH", "GIT_USERNAME", "GIT_ACCESS_TOKEN":
			if varValue == "" {
				missingVars = append(missingVars, fmt.Sprintf("%s is required", varName))
			}
		case "GIT_STORAGE_PATH":
			if varValue == "" {
				logging.LogToConsole(logging.WarningLevel, logging.WarningEvent, fmt.Sprintf("%s is missing in environment, but possibly registered in Trawler config.", varName))
			}
		}
	}

	if len(missingVars) > 0 {
		for _, v := range missingVars {
			logging.LogToConsole(logging.WarningLevel, logging.WarningEvent, fmt.Sprintf("Environment variable %s is not set.", v))
		}
		return &gitConfig, fmt.Errorf("missing required environment variables: %v", missingVars)
	}
	gitConfig = config
	return &gitConfig, nil
}

// Cloines a Git repository from the given URL to the specified destination path
func CloneRepository(path string) (*git.Repository, error) {
	repo, err := git.PlainClone(path, false, &git.CloneOptions{
		URL: gitConfig.RepositoryURL,
		Auth: &http.BasicAuth{
			Username: gitConfig.Username,    // This can be anything except an empty string
			Password: gitConfig.AccessToken, // If using SSH keys or other auth methods, adjust accordingly
		},
	})
	if err != nil {
		return nil, err
	}
	return repo, nil
}

// Pulls the latest changes from the remote repository
func PullRepository(path string) error {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return err
	}
	w, err := repo.Worktree()
	if err != nil {
		return err
	}

	err = w.Pull(&git.PullOptions{RemoteName: gitConfig.Branch, Auth: &http.BasicAuth{
		Username: gitConfig.Username,
		Password: gitConfig.AccessToken,
	}})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}
	return nil
}

// Opens an existing Git repository at the specified path
func OpenRepository(path string) (*git.Repository, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func CopyItemsToLocalStorage(config *cfg.Config) error {

	err := syncGitRepository(config)
	if err != nil {
		logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Error syncing Git repository: %v", err))
	} else {
		logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, "Successfully synced Git repository.")
	}

	var directoriesFromTo = []DirectoriesFromTo{
		{From: config.Configurations.Global.OnlineCAStoragePath, To: config.Configurations.Global.GitStoragePath + "/cas-online"},
		{From: config.Configurations.Global.OfflineCAStoragePath, To: config.Configurations.Global.GitStoragePath + "/cas-offline"},
		{From: config.Configurations.Global.OfflineCrlsPath, To: config.Configurations.Global.GitStoragePath + "/crls-offline"},
	}

	value := reflect.ValueOf(directoriesFromTo)
	typ := value.Type()

	for i := 0; i < value.NumField(); i++ {
		field := typ.Field(i)
		dirFrom := field.Name
		dirTo := value.Field(i).Interface().(string)

		if dirFrom != "" {
			err := storage.CopyFile(dirFrom, dirTo)
			if err != nil {
				logging.LogToConsole(logging.ErrorLevel, logging.ErrorEvent, fmt.Sprintf("Error copying %s to local storage: %v", dirFrom, err))
				return err
			} else {
				logging.LogToConsole(logging.InfoLevel, logging.InfoEvent, fmt.Sprintf("Successfully copied %s to local storage.", dirFrom))
			}
		}
	}

	return nil
}

func syncGitRepository(config *cfg.Config) error {
	// Do a git-sync if git storage is enabled
	if gitConfig.Enabled == true {
		err := PullRepository(config.Configurations.Global.GitStoragePath)
		if err != nil {
			return err
		}
	}
	return nil
}
