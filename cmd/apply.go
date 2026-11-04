package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	applyCommand.Flags().BoolP("verbose", "v", false, "Prints more information about the linking process")
	applyCommand.Flags().Bool("no-shell", false, "Skip shell hook application")
	applyCommand.Flags().Bool("dry-run", false, "Preview links and shell hooks without changing files")
	applyCommand.Flags().Bool("allow-outside-home", false, "Allow link targets outside the current user's home directory")

	rootCmd.AddCommand(applyCommand)
}

func createDotfileLink(sourcePath, targetPath string) error {
	if runtime.GOOS != "windows" {
		if err := os.Symlink(sourcePath, targetPath); err != nil {
			return fmt.Errorf("create symlink: %w", err)
		}
		return nil
	}

	hardLinkErr := os.Link(sourcePath, targetPath)
	if hardLinkErr == nil {
		return nil
	}

	if err := os.Symlink(sourcePath, targetPath); err == nil {
		return nil
	} else {
		return fmt.Errorf("create hard link: %v; create symlink: %w", hardLinkErr, err)
	}
}

func applyTargetWarnings(targetPath string) ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home directory: %w", err)
	}

	var warnings []string
	cleanTarget := filepath.Clean(targetPath)
	cleanHome := filepath.Clean(homeDir)

	if !pathWithinBase(cleanTarget, cleanHome) {
		warnings = append(warnings, "target is outside your home directory")
	}

	if sensitiveApplyTarget(cleanTarget, cleanHome) {
		warnings = append(warnings, "target is a sensitive startup, SSH, autostart, service, or executable path")
	}

	return warnings, nil
}

func pathWithinBase(path, base string) bool {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return false
	}

	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func sensitiveApplyTarget(path, homeDir string) bool {
	rel, err := filepath.Rel(homeDir, path)
	if err != nil {
		return false
	}

	rel = filepath.ToSlash(rel)
	if rel == "." || strings.HasPrefix(rel, "../") || rel == ".." {
		return false
	}

	sensitiveExact := map[string]bool{
		".bashrc":                  true,
		".bash_profile":            true,
		".bash_login":              true,
		".profile":                 true,
		".zshrc":                   true,
		".zprofile":                true,
		".zlogin":                  true,
		".config/fish/config.fish": true,
	}
	if sensitiveExact[rel] {
		return true
	}

	sensitivePrefixes := []string{
		".ssh/",
		".config/autostart/",
		".config/systemd/user/",
		".local/bin/",
		"bin/",
		"AppData/Roaming/Microsoft/Windows/Start Menu/Programs/Startup/",
		"Documents/PowerShell/",
		"Documents/WindowsPowerShell/",
	}
	for _, prefix := range sensitivePrefixes {
		if strings.HasPrefix(rel, prefix) {
			return true
		}
	}

	return false
}

var applyCommand = &cobra.Command{
	Use:          "apply",
	Short:        "Apply the dotfiles and shell hooks to the system",
	Long:         `Applies the dotfiles to the system and installs shell integration hooks.`,
	GroupID:      "dotfiles",
	SilenceUsage: true,
	RunE:         runApply,
}

func containsApplyWarning(warnings []string, warning string) bool {
	for _, item := range warnings {
		if item == warning {
			return true
		}
	}
	return false
}
