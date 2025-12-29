package util_test

import (
	"os"
	"path/filepath"

	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/tests/testutil"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/spf13/viper"
)

func Test_FileExists(t *testing.T) {
	err := testutil.SetupTestFile(t)
	testutil.TBErrorIfNotNil(t, err)

	file := filepath.Join(viper.GetString("test-dir"), "test.txt")
	if !fileops.IsFile(file) {
		t.Error("File does not exist")
	}
}

func Test_FileDoesNotExist(t *testing.T) {
	err := testutil.SetupTestFile(t)
	testutil.TBErrorIfNotNil(t, err)

	file := filepath.Join(viper.GetString("test-dir"), "does-not-exist.txt")
	if fileops.IsFile(file) {
		t.Error("File exists")
	}
}

func Test_IsDir(t *testing.T) {
	err := testutil.SetupTestFile(t)
	testutil.TBErrorIfNotNil(t, err)

	dir := viper.GetString("test-dir")
	if !fileops.IsDir(dir) {
		t.Error("Directory does not exist")
	}
}

func Fuzz_ExpandPath_NonEmptyHomePath(f *testing.F) {
	home, err := os.UserHomeDir()
	testutil.TBErrorIfNotNil(f, err)

	temp := f.TempDir()
	f.Add(temp, temp)

	if os.PathListSeparator == '\\' {
		f.Add("~\\", home)
	}
	f.Add("~/", home)
	f.Add("~", home)

	rel, err := filepath.Rel(home, temp)
	testutil.TBErrorIfNotNil(f, err)

	f.Add("~/"+rel, temp)
	f.Add("~//"+rel, temp)
	f.Add("~"+rel, temp)

	f.Fuzz(func(t *testing.T, testPath string, expected string) {
		result, err := fileops.ExpandPath(testPath)
		testutil.TBErrorIfNotNil(t, err)

		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	})
}
