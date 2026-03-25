package testutil

import (
	"os"
	"path/filepath"
	"time"

	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	internalgit "github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/viper"
)

func SetupTestFile(t testing.TB) error {
	viper.Reset()

	// Set the test dir path
	temp := t.TempDir()
	viper.Set("test-dir", temp)

	// add an empty test file
	file, err := os.Create(filepath.Join(temp, "test.txt"))
	if err != nil {
		t.Error(err)
	}
	file.Close()

	return nil
}

// CreateRemoteRepo creates a fake remote repository with a default branch and an initial commit.
func CreateRemoteRepo(t testing.TB, branchName string) string {
	return CreateRemoteRepoAt(t, t.TempDir(), t.TempDir(), branchName)
}

// CreateRemoteRepoAt creates a fake remote repository with a default branch and an initial commit
// using the provided paths for the bare remote and temporary working repository.
func CreateRemoteRepoAt(t testing.TB, remoteRepoPath, tempRepoPath, branchName string) string {
	err := os.MkdirAll(remoteRepoPath, os.ModePerm)
	TBErrorIfNotNil(t, err)

	err = os.MkdirAll(tempRepoPath, os.ModePerm)
	TBErrorIfNotNil(t, err)

	_, err = git.PlainInit(remoteRepoPath, true)
	TBErrorIfNotNil(t, err)

	tempRepo, err := git.PlainInit(tempRepoPath, false)
	TBErrorIfNotNil(t, err)

	// Add the remote to the temporary repo
	_, err = tempRepo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{remoteRepoPath},
	})
	TBErrorIfNotNil(t, err)

	// Set the initial branch name (defaults to master, we want to control it)
	if branchName != "" {
		// Create the branch reference before any commits
		headRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.ReferenceName("refs/heads/"+branchName))
		err = tempRepo.Storer.SetReference(headRef)
		TBErrorIfNotNil(t, err)
	}

	// Create a file and commit to establish the branch
	testFilePath := filepath.Join(tempRepoPath, "test.txt")
	err = os.WriteFile(testFilePath, []byte("test content"), 0644)
	TBErrorIfNotNil(t, err)

	wt, err := tempRepo.Worktree()
	TBErrorIfNotNil(t, err)

	_, err = wt.Add("test.txt")
	TBErrorIfNotNil(t, err)

	// Commit with author information
	_, err = wt.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	TBErrorIfNotNil(t, err)

	// Push to the bare repository
	err = tempRepo.Push(&git.PushOptions{
		RemoteName: "origin",
	})
	TBErrorIfNotNil(t, err)

	// -- The temporary repo is no longer needed --
	// the testing framework will clean it up for us

	// Set HEAD in the bare repository to point to the correct branch
	remoteRepo, err := git.PlainOpen(remoteRepoPath)
	TBErrorIfNotNil(t, err)

	if branchName != "" {
		headRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.ReferenceName("refs/heads/"+branchName))
		err = remoteRepo.Storer.SetReference(headRef)
		TBErrorIfNotNil(t, err)
	}

	return remoteRepoPath
}

// CreateBareRemoteRepo creates an empty bare repository that can be used as a push target.
func CreateBareRemoteRepo(t testing.TB) string {
	remoteRepoPath := t.TempDir()
	_, err := git.PlainInit(remoteRepoPath, true)
	TBErrorIfNotNil(t, err)

	return remoteRepoPath
}

func SetupTestRepo(t testing.TB) (*git.Repository, error) {
	viper.Reset()

	// Set the test repo path
	temp := t.TempDir()
	viper.Set("repo-path", temp)

	// Set the remote url
	remote := CreateRemoteRepo(t, "main")
	viper.Set("remote-url", remote)

	// Create a git repo
	err := os.MkdirAll(temp, os.ModePerm)
	fileops.CheckIfError(err)

	// Initialize the git repo
	return internalgit.InitGitRepo(temp, remote, false)
}

// SetupPushableTestRepo creates a local repository with an empty bare origin remote.
// This is cheaper than SetupTestRepo when tests only need a configured remote for push.
func SetupPushableTestRepo(t testing.TB) (*git.Repository, error) {
	viper.Reset()

	temp := t.TempDir()
	viper.Set("repo-path", temp)

	remote := CreateBareRemoteRepo(t)
	viper.Set("remote-url", remote)

	err := os.MkdirAll(temp, os.ModePerm)
	TBErrorIfNotNil(t, err)

	return internalgit.InitGitRepo(temp, remote, false)
}

func TBErrorIfNotNil(t testing.TB, err error) {
	if err != nil {
		t.Error(err)
	}
}
