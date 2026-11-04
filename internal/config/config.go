package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/spf13/viper"
)

// var homeDir, _ = os.UserHomeDir()
// var DefaultRepoPath = filepath.Join(homeDir, "dotfiles")

// GetDefaultRepoPath returns the default dotfiles repository directory.
func GetDefaultRepoPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", "dotfiles")
	}
	return filepath.Join(homeDir, "dotfiles")
}

// InitializeConfig reads configuration, creating a new file only when absent.
// Invalid existing configuration is never replaced.
func InitializeConfig(configFile string) error {
	return LoadConfig(configFile, true)
}

// LoadConfig reads configuration and only creates missing files when requested.
// Read-only commands can inspect an uninitialized installation without writes.
func LoadConfig(configFile string, create bool) error {
	if !create {
		viper.SetConfigFile(configFile)
		if err := viper.ReadInConfig(); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("read configuration %q: %w; repair this file before retrying", configFile, err)
		}
		return nil
	}
	// Ensure the directory exists
	configDir := filepath.Dir(configFile)
	if err := fileops.EnsureDir(configDir); err != nil {
		return fmt.Errorf("create configuration directory %q: %w", configDir, err)
	}

	// Set viper config file
	viper.SetConfigFile(configFile)

	// Try to read existing config, create if it doesn't exist
	if err := viper.ReadInConfig(); err != nil {
		if os.IsNotExist(err) {
			// Config file not found, create it
			if err := viper.SafeWriteConfigAs(configFile); err != nil {
				return fmt.Errorf("create configuration %q: %w", configFile, err)
			}
		} else {
			return fmt.Errorf("read configuration %q: %w; repair this file before retrying", configFile, err)
		}
	}

	return nil
}
