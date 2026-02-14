package cmd

import (
	"os"
	"testing"
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
	originalUse := rootCmd.Use
	t.Cleanup(func() {
		rootCmd.Use = originalUse
	})

	rootCmd.Use = "omdot"
	if got := assumedAlias(); got != "omdot" {
		t.Fatalf("assumedAlias() = %q, want %q", got, "omdot")
	}
}
