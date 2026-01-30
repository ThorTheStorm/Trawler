package gitops

import (
	"os"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

var ErrRepositoryNotExists = git.ErrRepositoryNotExists

// Cloines a Git repository from the given URL to the specified destination path
func CloneRepository(repoURL string, destinationPath string) (*git.Repository, error) {
	repo, err := git.PlainClone(destinationPath, false, &git.CloneOptions{
		URL: repoURL,
		Auth: &http.BasicAuth{
			Username: "git",                         // This can be anything except an empty string
			Password: os.Getenv("GIT_ACCESS_TOKEN"), // If using SSH keys or other auth methods, adjust accordingly
		},
	})
	if err != nil {
		return nil, err
	}
	return repo, nil
}

// Pulls the latest changes from the remote repository
func PullRepository(repo *git.Repository) error {
	w, err := repo.Worktree()
	if err != nil {
		return err
	}

	err = w.Pull(&git.PullOptions{RemoteName: "origin"})
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
