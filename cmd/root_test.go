package cmd

import (
	"bytes"
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestDetectInvokedName(t *testing.T) {
	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})

	tests := []struct {
		name     string
		arg0     string
		expected string
	}{
		{
			name:     "uses executable basename",
			arg0:     "/usr/local/bin/oh-my-dot",
			expected: "oh-my-dot",
		},
		{
			name:     "strips executable extension",
			arg0:     "/tools/omdot.exe",
			expected: "omdot",
		},
		{
			name:     "falls back to default when sanitized name is empty",
			arg0:     "/tmp/$$$.exe",
			expected: "oh-my-dot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = []string{tt.arg0}
			if got := detectInvokedName(); got != tt.expected {
				t.Fatalf("detectInvokedName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestAssumedAlias(t *testing.T) {
	originalArgs := os.Args
	originalUse := rootCmd.Use
	t.Cleanup(func() {
		os.Args = originalArgs
		rootCmd.Use = originalUse
	})

	rootCmd.Use = "omdot"
	if got := assumedAlias(); got != "omdot" {
		t.Fatalf("assumedAlias() = %q, want %q", got, "omdot")
	}

	rootCmd.Use = ""
	os.Args = []string{"/tmp/myalias.exe"}
	if got := assumedAlias(); got != "myalias" {
		t.Fatalf("assumedAlias() fallback = %q, want %q", got, "myalias")
	}
}

func TestExecutePrintsRunEErrorWhenUsageIsSilenced(t *testing.T) {
	viper.Reset()
	viper.Set("initialized", true)
	t.Cleanup(viper.Reset)

	failingCmd := &cobra.Command{
		Use:          "failing-test-command",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("boom")
		},
	}
	rootCmd.AddCommand(failingCmd)
	t.Cleanup(func() {
		rootCmd.RemoveCommand(failingCmd)
	})

	output, err := captureExecuteOutput(t, []string{"failing-test-command"})
	if err == nil {
		t.Fatal("expected Execute() to return an error")
	}

	normalized := stripANSICodes(output)
	if !strings.Contains(normalized, "Error: boom") {
		t.Fatalf("expected formatted error output, got %q", normalized)
	}

	if strings.Count(normalized, "Error: boom") != 1 {
		t.Fatalf("expected exactly one formatted error line, got %q", normalized)
	}

	if strings.Contains(normalized, "Usage:\n") {
		t.Fatalf("expected usage output to stay silenced, got %q", normalized)
	}
}

func captureExecuteOutput(t *testing.T, args []string) (string, error) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}

	os.Stdout = w
	os.Stderr = w

	execErr := Execute(func(cmd *cobra.Command) {
		cmd.SetArgs(args)
	})

	_ = w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("ReadFrom() error = %v", err)
	}

	return buf.String(), execErr
}

func stripANSICodes(text string) string {
	ansiPattern := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiPattern.ReplaceAllString(text, "")
}
