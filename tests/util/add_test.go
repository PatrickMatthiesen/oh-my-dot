package util_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/tests/testutil"
	internalgit "github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/viper"
)

func Test_LinkAndAddFile(t *testing.T) {
	r, err := testutil.SetupTestRepo(t)
	testutil.TBErrorIfNotNil(t, err)

	// Create a file
	tempSourceDir := t.TempDir()
	file, err := os.CreateTemp(tempSourceDir, "test.txt")
	if err != nil {
		t.Error(err)
	}
	file.WriteString("test")
	defer file.Close()

	// Link the file to the git repo
	err = internalgit.LinkAndAddFile(file.Name())
	if err != nil {
		t.Error(err)
	}

	err = internalgit.Commit("test")
	testutil.TBErrorIfNotNil(t, err)

	// Check if the file exists in the git repo
	testFilePath := filepath.Join(viper.GetString("repo-path"), "files", filepath.Base(file.Name()))
	_, err = os.Stat(testFilePath)
	if err != nil {
		t.Error(err)
	}

	commits, err := r.Log(&git.LogOptions{})
	testutil.TBErrorIfNotNil(t, err)
	commit, err := commits.Next()
	testutil.TBErrorIfNotNil(t, err)
	files, err := commit.Files()
	testutil.TBErrorIfNotNil(t, err)
	_, err = files.Next()
	testutil.TBErrorIfNotNil(t, err)

	t.Run("Test config push", func(t *testing.T) {
		// Push the repo (remote is already set up by SetupTestRepo)
		err := internalgit.PushRepo()
		testutil.TBErrorIfNotNil(t, err)
	})
}

func Test_CopyAndAddFile(t *testing.T) {
	r, err := testutil.SetupTestRepo(t)
	testutil.TBErrorIfNotNil(t, err)

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
	err = internalgit.CopyAndAddFile(file.Name(), tempDestDir)
	testutil.TBErrorIfNotNil(t, err)

	err = internalgit.Commit("test")
	testutil.TBErrorIfNotNil(t, err)

	// Check if the file exists in the git repo
	testFilePath := filepath.Join(viper.GetString("repo-path"), "files", filepath.Base(file.Name()))
	_, err = os.Stat(testFilePath)
	if err != nil {
		t.Error(err)
	}

	commits, err := r.Log(&git.LogOptions{})
	testutil.TBErrorIfNotNil(t, err)
	commit, err := commits.Next()
	testutil.TBErrorIfNotNil(t, err)
	files, err := commit.Files()
	testutil.TBErrorIfNotNil(t, err)
	_, err = files.Next()
	testutil.TBErrorIfNotNil(t, err)

	t.Run("Test config push", func(t *testing.T) {
		// Push the repo (remote is already set up by SetupTestRepo)
		err := internalgit.PushRepo()
		testutil.TBErrorIfNotNil(t, err)
	})
}
