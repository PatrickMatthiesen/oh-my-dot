package util

import (
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// var homeDir, _ = os.UserHomeDir()
// var DefaultRepoPath = filepath.Join(homeDir, "dotfiles")

func GetDefaultRepoPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", "dotfiles")
	}
	return filepath.Join(homeDir, "dotfiles")
}

func InitializeConfig(configFile string) error {
	// Ensure the directory exists
	configDir := filepath.Dir(configFile)
	if err := EnsureDir(configDir); err != nil {
		log.Println("No config directory found")
		log.Println("Making a new one")
		return err
	}

	// Set viper config file
	viper.SetConfigFile(configFile)

	// Try to read existing config, create if it doesn't exist
	if err := viper.ReadInConfig(); err != nil {
		if os.IsNotExist(err) {
			log.Println("No config file found")
			log.Println("Making a new one")
			// Config file not found, create it
			if err := viper.SafeWriteConfigAs(configFile); err != nil {
				return err
			}
			log.Println("Config file created at " + configFile)
		} else {
			return err
		}
	}

	return nil
}
