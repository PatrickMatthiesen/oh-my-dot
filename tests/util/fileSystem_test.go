package util_test

import (
	"path/filepath"

	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/util"
	"github.com/spf13/viper"
)

func Test_FileExists(t *testing.T) {
	err := SetupTestFile(t)
	if err != nil {
		t.Error(err)
	}

	file := filepath.Join(viper.GetString("test-dir"), "test.txt")
	if !util.IsFile(file) {
		t.Error("File does not exist")
	}
}

func Test_FileDoesNotExist(t *testing.T) {
	err := SetupTestFile(t)
	if err != nil {
		t.Error(err)
	}

	file := filepath.Join(viper.GetString("test-dir"), "does-not-exist.txt")
	if util.IsFile(file) {
		t.Error("File exists")
	}
}

func Test_IsDir(t *testing.T) {
	err := SetupTestFile(t)
	if err != nil {
		t.Error(err)
	}

	dir := viper.GetString("test-dir")
	if !util.IsDir(dir) {
		t.Error("Directory does not exist")
	}
}
