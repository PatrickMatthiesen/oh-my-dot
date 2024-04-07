package util_test

import (
	"os"
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

func Fuzz_ExpandPath_NonEmptyHomePath(f *testing.F) {
	home, err := os.UserHomeDir()
	if err != nil {
		f.Error(err)
	}
	temp := f.TempDir()
	f.Add(temp, temp)

	f.Add("~\\", home)
	f.Add("~/", home)
	f.Add("~", home)

	rel, err := filepath.Rel(home, temp)
	if err != nil {
		f.Error(err)
	}
	f.Add("~/" + rel, temp)
	f.Add("~//" + rel, temp)
	f.Add("~" + rel, temp)

	f.Fuzz(func(t *testing.T, testPath string, expected string) {
		result, err := util.ExpandPath(testPath)
		if err != nil {
			t.Error(err)
		}
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	})
}