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

	configName := "config.env"
	configFolder := filepath.Join(home, ".oh-my-dot")

	viper.SetDefault("dot-home", filepath.Join(configFolder, configName))
	viper.AddConfigPath(configFolder)
	viper.SetConfigName(configName)

	viper.AutomaticEnv()
	cmd.Execute()
}