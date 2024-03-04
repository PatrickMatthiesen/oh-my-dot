package main

import (
	"os"

	"github.com/PatrickMatthiesen/oh-my-dot/cmd"
	"github.com/PatrickMatthiesen/oh-my-dot/util"
	"github.com/spf13/viper"
	"path/filepath"
)

func main() {
	home, err := os.UserHomeDir()
	util.CheckIfErrorWithMessage(err, "Error getting home directory")

	configFile := filepath.Join(home, ".oh-my-dot", "config.json")

	go util.EnsureConfigFolder(configFile)

	viper.SetDefault("dot-home", configFile)
	viper.SetDefault("repo-path", filepath.Join(home, "dotfiles"))
	viper.SetConfigFile(configFile)

	viper.AutomaticEnv()
	cmd.Execute()
}
