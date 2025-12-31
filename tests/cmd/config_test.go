package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/cmd"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// setupTestConfig initializes a test configuration with temporary directories
func setupTestConfig(t *testing.T) (configFile, repoPath string) {
	t.Helper()
	configFolder := t.TempDir()
	configFile = filepath.Join(configFolder, "config.json")
	repoFolder := t.TempDir()
	repoPath = filepath.Join(repoFolder, "dotfiles")

	viper.Reset()
	config.InitializeConfig(configFile)
	viper.SetDefault("repo-path", repoPath)
	viper.SetDefault("dot-home", configFile)
	viper.SetConfigFile(configFile)
	viper.ReadInConfig()

	return configFile, repoPath
}

// captureOutput captures stdout during command execution
func captureOutput(t *testing.T, args []string) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Execute(func(c *cobra.Command) {
		c.SetArgs(args)
	})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func Test_Config_Show_All(t *testing.T) {
	configFile, repoPath := setupTestConfig(t)
	output := captureOutput(t, []string{"config"})

	// Check that output contains key configuration values
	if !strings.Contains(output, "Configuration:") {
		t.Error("Expected output to contain 'Configuration:'")
	}
	if !strings.Contains(output, "location:") {
		t.Error("Expected output to contain 'location:'")
	}
	if !strings.Contains(output, "dotfiles:") {
		t.Error("Expected output to contain 'dotfiles:'")
	}
	if !strings.Contains(output, "initialized:") {
		t.Error("Expected output to contain 'initialized:'")
	}
	if !strings.Contains(output, configFile) {
		t.Errorf("Expected output to contain config file path '%s'", configFile)
	}
	if !strings.Contains(output, repoPath) {
		t.Errorf("Expected output to contain repo path '%s'", repoPath)
	}
}

func Test_Config_Show_Location(t *testing.T) {
	configFile, _ := setupTestConfig(t)
	output := strings.TrimSpace(captureOutput(t, []string{"config", "location"}))

	// Check that output contains the config file path
	if !strings.Contains(output, configFile) {
		t.Errorf("Expected output to contain config file path '%s', got '%s'", configFile, output)
	}
}

func Test_Config_Show_Dotfiles(t *testing.T) {
	_, repoPath := setupTestConfig(t)
	output := strings.TrimSpace(captureOutput(t, []string{"config", "dotfiles"}))

	// Check that output contains the repo path
	if !strings.Contains(output, repoPath) {
		t.Errorf("Expected output to contain repo path '%s', got '%s'", repoPath, output)
	}
}

func Test_Config_Unknown_Key(t *testing.T) {
	setupTestConfig(t)
	output := captureOutput(t, []string{"config", "unknown-key"})

	// Check that output contains error message
	if !strings.Contains(output, "Unknown config key") {
		t.Error("Expected output to contain 'Unknown config key'")
	}
}
