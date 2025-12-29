package main

import (
	"os"
	"path/filepath"

	"github.com/PatrickMatthiesen/oh-my-dot/cmd"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/config"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/spf13/viper"
)

func main() {
	home, err := os.UserHomeDir()
	fileops.CheckIfErrorWithMessage(err, "Error getting home directory")

	configFile := filepath.Join(home, ".oh-my-dot", "config.json")

	config.InitializeConfig(configFile)

	viper.SetDefault("dot-home", configFile)
	viper.SetDefault("repo-path", filepath.Join(home, "dotfiles"))
	// TODO: Set a viper config variable for the files folder in the repo-path, and update the strings in the util/repo.go file to use this variable

	viper.SetConfigFile(configFile)

	viper.ReadInConfig()

	viper.AutomaticEnv()
	cmd.Execute()

	//TODO: make execute return an error. Redirect the error to a log file or print it to the console if env var or flag is set.
	//Update the commands to use RunE and tests to check for the error
}
