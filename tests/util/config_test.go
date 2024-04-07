package util_test

import (
	"path/filepath"

	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/util"
	"github.com/spf13/viper"
)

func Test_EnsureConfigFolder(t *testing.T) {
	err := SetupTestFile(t)
	if err != nil {
		t.Error(err)
	}

	file := filepath.Join(viper.GetString("test-dir"), ".oh-my-dot", "config.json")
	util.EnsureConfigFolder(file)
	if !util.IsDir(filepath.Dir(file)) {
		t.Error("Directory does not exist")
	}
}