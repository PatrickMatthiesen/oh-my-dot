package util_test

import (
	"path/filepath"

	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/tests/testutil"
	"github.com/PatrickMatthiesen/oh-my-dot/util"
	"github.com/spf13/viper"
)

func Test_InitializeConfig(t *testing.T) {
	err := testutil.SetupTestFile(t)
	if err != nil {
		t.Error(err)
	}

	file := filepath.Join(viper.GetString("test-dir"), ".oh-my-dot", "config.json")
	err = util.InitializeConfig(file)
	if err != nil {
		t.Error(err)
	}

	if !util.IsDir(filepath.Dir(file)) {
		t.Error("Directory does not exist")
	}

	if !util.IsFile(file) {
		t.Error("Config file was not created")
	}
}
