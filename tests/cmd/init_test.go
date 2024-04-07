package cmd

import (
	"os"
	"path/filepath"

	"github.com/PatrickMatthiesen/oh-my-dot/cmd"
	"github.com/PatrickMatthiesen/oh-my-dot/util"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/viper"

	"testing"
)

func Test_Init_cmd(t *testing.T) {
	fakeGitRepoPath := t.TempDir()
	_, err := git.PlainInit(fakeGitRepoPath, true)
	if err != nil {
		t.Error(err)
	}

	invokeCommand(t, []string{"init", fakeGitRepoPath})

	isRepo := util.IsGitRepo(viper.GetString("repo-path"))
	if !isRepo {
		t.Error("repo-path is not a git repo")
	}
}

func invokeCommand(t *testing.T, args []string) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = append([]string{os.Args[0]}, args...)

	// Set default values for viper
	configFolder := t.TempDir()
	configFile := filepath.Join(configFolder, "config.json")
	viper.SetDefault("dot-home", configFile)

	repoFolder := t.TempDir()
	viper.SetDefault("repo-path", filepath.Join(repoFolder, "dotfiles"))

	viper.SetConfigFile(configFile)

	viper.ReadInConfig()

	viper.AutomaticEnv()

	rootCmd := cmd.Execute()
	if rootCmd == nil {
		t.Error("rootCmd is nil")
	}
}