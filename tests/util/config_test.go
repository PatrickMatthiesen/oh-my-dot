package util_test

import (
	"path/filepath"

	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/tests/testutil"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/config"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/spf13/viper"
)

func Test_InitializeConfig(t *testing.T) {
	err := testutil.SetupTestFile(t)
	if err != nil {
		t.Error(err)
	}

	file := filepath.Join(viper.GetString("test-dir"), ".oh-my-dot", "config.json")
	err = config.InitializeConfig(file)
	if err != nil {
		t.Error(err)
	}

	if !fileops.IsDir(filepath.Dir(file)) {
		t.Error("Directory does not exist")
	}

	if !fileops.IsFile(file) {
		t.Error("Config file was not created")
	}
}
