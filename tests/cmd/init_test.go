package cmd_test

import (
	"path/filepath"

	"github.com/PatrickMatthiesen/oh-my-dot/cmd"
	"github.com/PatrickMatthiesen/oh-my-dot/util"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"testing"
)

func Test_Plain_Init_cmd(t *testing.T) {
	fakeGitRepoPath := t.TempDir()
	_, err := git.PlainInit(fakeGitRepoPath, true)
	TBErrorIfNotNil(t, err)

	invokeCommand(t, []string{"init", fakeGitRepoPath})

	isRepo := util.IsGitRepo(viper.GetString("repo-path"))
	if !isRepo {
		t.Error("repo-path is not a git repo")
	}
}

func Test_Existing_Init_cmd(t *testing.T) {
	fakeGitRepoPath := t.TempDir()
	_, err := git.PlainInit(fakeGitRepoPath, true)
	TBErrorIfNotNil(t, err)

	invokeCommand(t, []string{"init", fakeGitRepoPath})

	isRepo := util.IsGitRepo(viper.GetString("repo-path"))
	if !isRepo {
		t.Error("repo-path is not a git repo")
	}
}

func invokeCommand(t *testing.T, args []string) {
	viper.Reset()

	// Set default values for viper
	configFolder := t.TempDir()
	configFile := filepath.Join(configFolder, "config.json")
	viper.SetDefault("dot-home", configFile)

	repoFolder := t.TempDir()
	viper.SetDefault("repo-path", filepath.Join(repoFolder, "dotfiles"))

	viper.SetConfigFile(configFile)

	viper.AutomaticEnv()

	cmd.Execute(func(c *cobra.Command) {
		c.SetArgs(args)
	})
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