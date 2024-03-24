package repo_test

import (
	"os"
	"path/filepath"

	// "path/filepath"
	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/util"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/viper"
)

func Setup(t testing.TB) (*git.Repository, error) {
	// Set the test repo path
	temp := t.TempDir()
	viper.Set("repo-path", temp)

	// Set the remote url
	remote := t.TempDir()
	viper.Set("remote-url", remote)

	// Create a git repo
	err := os.MkdirAll(temp, os.ModePerm)
	util.CheckIfError(err)

	// Initialize the git repo
	return util.InitGitRepo(temp, remote, false)
}

func TestMoveAndAddFile(t *testing.T) {
	r, err := Setup(t)
	util.CheckIfError(err)

	// Create a file
	tempSourceDir := t.TempDir()
	file, err := os.CreateTemp(tempSourceDir, "test.txt")
	if err != nil {
		t.Error(err)
	}
	file.WriteString("test")
	defer file.Close()

	// Move the file to the git repo
	err = util.LinkAndAddFile(file.Name())
	util.CheckIfError(err)

	// Check if the file exists in the git repo
	testFilePath := filepath.Join(viper.GetString("repo-path"), "files", filepath.Base(file.Name()))
	_, err = os.Stat(testFilePath)
	if err != nil {
		t.Error(err)
	}

	commits, err := r.Log(&git.LogOptions{})
	util.CheckIfError(err)
	commit, err := commits.Next()
	util.CheckIfError(err)
	files, err := commit.Files()
	util.CheckIfError(err)
	_, err = files.Next()
	util.CheckIfError(err)

	t.Run("Test config push", func(t *testing.T) {
		// Make a bare repo to push to
		_, err := git.PlainInit(viper.GetString("remote-url"), true)
		util.CheckIfError(err)

		// Push the repo
		err = util.PushRepo()
		util.CheckIfError(err)
	})
}

