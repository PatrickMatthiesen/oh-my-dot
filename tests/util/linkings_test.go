package util_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/symlink"
)

func Test_BuildLinkPath_ReplacesHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "home directory file",
			input:    filepath.Join(home, "test.txt"),
			expected: "~/test.txt",
		},
		{
			name:     "home subdirectory file",
			input:    filepath.Join(home, "Documents", "file.txt"),
			expected: "~/Documents/file.txt",
		},
		{
			name:     "nested home subdirectory",
			input:    filepath.Join(home, "repos", "project", "config.yaml"),
			expected: "~/repos/project/config.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := symlink.BuildLinkPath(tt.input)
			if err != nil {
				t.Fatalf("BuildLinkPath returned error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func Test_BuildLinkPath_UsesPosixFormat(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	testPath := filepath.Join(home, "Documents", "test.txt")
	result, err := symlink.BuildLinkPath(testPath)
	if err != nil {
		t.Fatalf("BuildLinkPath returned error: %v", err)
	}

	// Result should always use forward slashes, never backslashes
	if strings.Contains(result, "\\") {
		t.Errorf("Result contains backslashes: %s", result)
	}

	// Result should use forward slashes
	if !strings.Contains(result, "/") && len(result) > 2 {
		t.Errorf("Result does not contain forward slashes: %s", result)
	}
}

func Test_BuildLinkPath_NonHomeDirectory(t *testing.T) {
	// Test that paths outside home directory are still normalized to POSIX format
	testPath := filepath.Join(string(filepath.Separator), "etc", "config", "app.conf")
	result, err := symlink.BuildLinkPath(testPath)
	if err != nil {
		t.Fatalf("BuildLinkPath returned error: %v", err)
	}

	// Should not contain ~ since it's not under home
	if strings.HasPrefix(result, "~") {
		t.Errorf("Non-home path should not start with ~: %s", result)
	}

	// Should still use forward slashes
	if strings.Contains(result, "\\") {
		t.Errorf("Result contains backslashes: %s", result)
	}
}

func Fuzz_BuildLinkPath(f *testing.F) {
	home, err := os.UserHomeDir()
	if err != nil {
		f.Error(err)
	}

	f.Add(filepath.Join(home, "test.txt"), "~/test.txt")
	f.Add(filepath.Join(home, "test/test.txt"), "~/test/test.txt")
	f.Add(filepath.Join(home, "test/test/test.txt"), "~/test/test/test.txt")

	f.Fuzz(func(t *testing.T, path string, expected string) {
		link, err := symlink.BuildLinkPath(path)
		if err != nil {
			t.Error(err)
		}
		if link != expected {
			t.Errorf("Expected %s, got %s", expected, link)
		}
	})
}