package util_test

import (
	"os"
	"path/filepath"

	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/util"
	"github.com/go-git/go-git/v5"
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

func SetupTestRepo(t testing.TB) (*git.Repository, error) {
	viper.Reset()

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

func TBErrorIfNotNil(t testing.TB, err error) {
	if err != nil {
		t.Error(err)
	}
}

func FErrorIfNotNil(f *testing.F, err error) {
	if err != nil {
		f.Error(err)
	}
}
