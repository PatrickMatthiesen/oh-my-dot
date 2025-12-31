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

func Test_Config_Show_All(t *testing.T) {
	// Setup
	configFolder := t.TempDir()
	configFile := filepath.Join(configFolder, "config.json")
	repoFolder := t.TempDir()
	repoPath := filepath.Join(repoFolder, "dotfiles")

	viper.Reset()
	config.InitializeConfig(configFile)
	viper.SetDefault("repo-path", repoPath)
	viper.SetDefault("dot-home", configFile)
	viper.SetConfigFile(configFile)
	viper.ReadInConfig()

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Execute(func(c *cobra.Command) {
		c.SetArgs([]string{"config"})
	})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

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
	// Setup
	configFolder := t.TempDir()
	configFile := filepath.Join(configFolder, "config.json")
	repoFolder := t.TempDir()
	repoPath := filepath.Join(repoFolder, "dotfiles")

	viper.Reset()
	config.InitializeConfig(configFile)
	viper.SetDefault("repo-path", repoPath)
	viper.SetDefault("dot-home", configFile)
	viper.SetConfigFile(configFile)
	viper.ReadInConfig()

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Execute(func(c *cobra.Command) {
		c.SetArgs([]string{"config", "location"})
	})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := strings.TrimSpace(buf.String())

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that output contains the config file path
	if !strings.Contains(output, configFile) {
		t.Errorf("Expected output to contain config file path '%s', got '%s'", configFile, output)
	}
}

func Test_Config_Show_Dotfiles(t *testing.T) {
	// Setup
	configFolder := t.TempDir()
	configFile := filepath.Join(configFolder, "config.json")
	repoFolder := t.TempDir()
	repoPath := filepath.Join(repoFolder, "dotfiles")

	viper.Reset()
	config.InitializeConfig(configFile)
	viper.SetDefault("repo-path", repoPath)
	viper.SetDefault("dot-home", configFile)
	viper.SetConfigFile(configFile)
	viper.ReadInConfig()

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Execute(func(c *cobra.Command) {
		c.SetArgs([]string{"config", "dotfiles"})
	})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := strings.TrimSpace(buf.String())

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that output contains the repo path
	if !strings.Contains(output, repoPath) {
		t.Errorf("Expected output to contain repo path '%s', got '%s'", repoPath, output)
	}
}

func Test_Config_Unknown_Key(t *testing.T) {
	// Setup
	configFolder := t.TempDir()
	configFile := filepath.Join(configFolder, "config.json")
	repoFolder := t.TempDir()
	repoPath := filepath.Join(repoFolder, "dotfiles")

	viper.Reset()
	config.InitializeConfig(configFile)
	viper.SetDefault("repo-path", repoPath)
	viper.SetDefault("dot-home", configFile)
	viper.SetConfigFile(configFile)
	viper.ReadInConfig()

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Execute(func(c *cobra.Command) {
		c.SetArgs([]string{"config", "unknown-key"})
	})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that output contains error message
	if !strings.Contains(output, "Unknown config key") {
		t.Error("Expected output to contain 'Unknown config key'")
	}
}
