package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
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

func captureCommandOutput(t *testing.T, args []string) (string, error) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	err := cmd.Execute(func(c *cobra.Command) {
		c.SetArgs(args)
	})

	w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), err
}

func stripANSI(text string) string {
	ansiPattern := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiPattern.ReplaceAllString(text, "")
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
	if !strings.Contains(output, "allow-gh-auth:") {
		t.Error("Expected output to contain 'allow-gh-auth:'")
	}
	if !strings.Contains(output, "push-compact:") {
		t.Error("Expected output to contain 'push-compact:'")
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

func Test_Config_Show_AllowGHAuth(t *testing.T) {
	setupTestConfig(t)
	viper.Set("update.allow-gh-auth", true)

	output := strings.TrimSpace(captureOutput(t, []string{"config", "allow-gh-auth"}))
	if output != "true" {
		t.Fatalf("expected allow-gh-auth output to be true, got %q", output)
	}
}

func Test_Config_Show_PushCompactDefault(t *testing.T) {
	setupTestConfig(t)

	output := strings.TrimSpace(captureOutput(t, []string{"config", "push-compact"}))
	if output != "auto" {
		t.Fatalf("expected push-compact output to be auto, got %q", output)
	}
}

func Test_Config_Set_PushCompact(t *testing.T) {
	setupTestConfig(t)

	captureOutput(t, []string{"config", "set", "push-compact", "ask"})

	output := strings.TrimSpace(captureOutput(t, []string{"config", "push-compact"}))
	if output != "ask" {
		t.Fatalf("expected push-compact output to be ask, got %q", output)
	}
}

func Test_Config_Update_Alias_PushCompact(t *testing.T) {
	setupTestConfig(t)

	captureOutput(t, []string{"config", "update", "push-compact", "off"})

	output := strings.TrimSpace(captureOutput(t, []string{"config", "push-compact"}))
	if output != "off" {
		t.Fatalf("expected push-compact output to be off, got %q", output)
	}
}

func Test_Config_Completion_ConfigKeys(t *testing.T) {
	setupTestConfig(t)

	output, err := captureCommandOutput(t, []string{"__complete", "config", ""})
	if err != nil {
		t.Fatalf("completion error: %v", err)
	}

	if !strings.Contains(output, "push-compact") {
		t.Fatalf("expected completion output to contain push-compact, got %q", output)
	}
	if !strings.Contains(output, "allow-gh-auth") {
		t.Fatalf("expected completion output to contain allow-gh-auth, got %q", output)
	}
}

func Test_Config_Completion_SetKeysAndValues(t *testing.T) {
	setupTestConfig(t)

	output, err := captureCommandOutput(t, []string{"__complete", "config", "set", ""})
	if err != nil {
		t.Fatalf("completion error: %v", err)
	}

	if !strings.Contains(output, "push-compact") {
		t.Fatalf("expected set key completion output to contain push-compact, got %q", output)
	}
	if strings.Contains(output, "remote-url") {
		t.Fatalf("expected set key completion output to omit read-only remote-url, got %q", output)
	}

	output, err = captureCommandOutput(t, []string{"__complete", "config", "set", "push-compact", ""})
	if err != nil {
		t.Fatalf("completion error: %v", err)
	}

	for _, value := range []string{"auto", "ask", "off"} {
		if !strings.Contains(output, value) {
			t.Fatalf("expected value completion output to contain %s, got %q", value, output)
		}
	}
}

func Test_FeatureUpdate_ErrorFormatting(t *testing.T) {
	setupTestConfig(t)
	viper.Set("initialized", true)

	output, err := captureCommandOutput(t, []string{"feature", "update"})
	if err == nil {
		t.Fatal("expected error for missing feature name")
	}

	output = stripANSI(output)

	if !strings.Contains(output, "Error: feature name required (or use -i for interactive mode)\n\nUsage:\n") {
		t.Fatalf("expected error followed by blank line and usage, got %q", output)
	}

	if strings.Count(output, "Error: feature name required (or use -i for interactive mode)") != 1 {
		t.Fatalf("expected exactly one formatted error line, got %q", output)
	}

	if !strings.Contains(output, "--all             Refresh all installed catalog features") {
		t.Fatalf("expected update usage to include --all flag, got %q", output)
	}
}
