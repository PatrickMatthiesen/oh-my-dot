package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyTargetWarningsSensitiveHomePath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir() error = %v", err)
	}

	warnings, err := applyTargetWarnings(filepath.Join(homeDir, ".ssh", "config"))
	if err != nil {
		t.Fatalf("applyTargetWarnings() error = %v", err)
	}

	if !containsApplyWarning(warnings, "target is a sensitive startup, SSH, autostart, service, or executable path") {
		t.Fatalf("expected sensitive target warning, got %v", warnings)
	}
	if containsApplyWarning(warnings, "target is outside your home directory") {
		t.Fatalf("did not expect outside-home warning for home path, got %v", warnings)
	}
}

func TestPathWithinBaseRejectsSiblingPrefix(t *testing.T) {
	base := filepath.Join(string(filepath.Separator), "tmp", "home")
	path := filepath.Join(string(filepath.Separator), "tmp", "home-other", "file")

	if pathWithinBase(path, base) {
		t.Fatalf("expected %s to be outside %s", path, base)
	}
}
