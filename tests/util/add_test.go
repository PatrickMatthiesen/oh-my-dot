package util_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/util"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/viper"
)

func Test_LinkAndAddFile(t *testing.T) {
	r, err := SetupTestRepo(t)
	TBErrorIfNotNil(t, err)

	// Create a file
	tempSourceDir := t.TempDir()
	file, err := os.CreateTemp(tempSourceDir, "test.txt")
	if err != nil {
		t.Error(err)
	}
	file.WriteString("test")
	defer file.Close()

	// Link the file to the git repo
	err = util.LinkAndAddFile(file.Name())
	if err != nil {
		t.Error(err)
	}

	err = util.Commit("test")
	TBErrorIfNotNil(t, err)

	// Check if the file exists in the git repo
	testFilePath := filepath.Join(viper.GetString("repo-path"), "files", filepath.Base(file.Name()))
	_, err = os.Stat(testFilePath)
	if err != nil {
		t.Error(err)
	}

	commits, err := r.Log(&git.LogOptions{})
	TBErrorIfNotNil(t, err)
	commit, err := commits.Next()
	TBErrorIfNotNil(t, err)
	files, err := commit.Files()
	TBErrorIfNotNil(t, err)
	_, err = files.Next()
	TBErrorIfNotNil(t, err)

	t.Run("Test config push", func(t *testing.T) {
		// Make a bare repo to push to
		_, err := git.PlainInit(viper.GetString("remote-url"), true)
		TBErrorIfNotNil(t, err)

		// Push the repo
		err = util.PushRepo()
		TBErrorIfNotNil(t, err)
	})
}

func Test_CopyAndAddFile(t *testing.T) {
	r, err := SetupTestRepo(t)
	TBErrorIfNotNil(t, err)

	// Create a file
	tempSourceDir := t.TempDir()
	file, err := os.CreateTemp(tempSourceDir, "test.txt")
	if err != nil {
		t.Error(err)
	}
	file.WriteString("test")
	defer file.Close()

	tempDestDir := t.TempDir()

	// Copy the file to the git repo
	err = util.CopyAndAddFile(file.Name(), tempDestDir)
	TBErrorIfNotNil(t, err)

	err = util.Commit("test")
	TBErrorIfNotNil(t, err)

	// Check if the file exists in the git repo
	testFilePath := filepath.Join(viper.GetString("repo-path"), "files", filepath.Base(file.Name()))
	_, err = os.Stat(testFilePath)
	if err != nil {
		t.Error(err)
	}

	commits, err := r.Log(&git.LogOptions{})
	TBErrorIfNotNil(t, err)
	commit, err := commits.Next()
	TBErrorIfNotNil(t, err)
	files, err := commit.Files()
	TBErrorIfNotNil(t, err)
	_, err = files.Next()
	TBErrorIfNotNil(t, err)

	t.Run("Test config push", func(t *testing.T) {
		// Make a bare repo to push to
		_, err := git.PlainInit(viper.GetString("remote-url"), true)
		TBErrorIfNotNil(t, err)

		// Push the repo
		err = util.PushRepo()
		TBErrorIfNotNil(t, err)
	})
}
