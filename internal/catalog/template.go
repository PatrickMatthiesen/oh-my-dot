package catalog

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed features/**/*
var featureTemplates embed.FS

// posixCompatibleShells lists shells that can fall back to posix implementation
var posixCompatibleShells = map[string]bool{
	"bash": true,
	"zsh":  true,
	"sh":   true,
}

// GetFeatureTemplate retrieves the template content for a feature and shell
// Falls back to posix.sh for POSIX-compatible shells if shell-specific template doesn't exist
func GetFeatureTemplate(featureName, shellName string) (string, error) {
	ext := GetShellExtension(shellName)
	templatePath := fmt.Sprintf("features/%s/%s%s", featureName, shellName, ext)

	content, err := featureTemplates.ReadFile(templatePath)
	if err != nil {
		// If shell-specific template not found and shell is POSIX-compatible, try posix fallback
		if posixCompatibleShells[shellName] {
			posixPath := fmt.Sprintf("features/%s/posix.sh", featureName)
			content, posixErr := featureTemplates.ReadFile(posixPath)
			if posixErr == nil {
				return string(content), nil
			}
		}
		return "", fmt.Errorf("template not found for feature %s and shell %s: %w", featureName, shellName, err)
	}

	return string(content), nil
}

// HasFeatureTemplate checks if a template exists for a feature and shell
// Returns true if either shell-specific or posix fallback exists
func HasFeatureTemplate(featureName, shellName string) bool {
	ext := GetShellExtension(shellName)
	templatePath := fmt.Sprintf("features/%s/%s%s", featureName, shellName, ext)

	_, err := featureTemplates.ReadFile(templatePath)
	if err == nil {
		return true
	}

	// Check for posix fallback if shell is POSIX-compatible
	if posixCompatibleShells[shellName] {
		posixPath := fmt.Sprintf("features/%s/posix.sh", featureName)
		_, posixErr := featureTemplates.ReadFile(posixPath)
		return posixErr == nil
	}

	return false
}

// WriteFeatureTemplate writes a feature template to the user's repository
func WriteFeatureTemplate(repoPath, shellName, featureName string) error {
	content, err := GetFeatureTemplate(featureName, shellName)
	if err != nil {
		return err
	}

	ext := GetShellExtension(shellName)
	featurePath := filepath.Join(repoPath, "omd-shells", shellName, "features", featureName+ext)

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(featurePath), 0755); err != nil {
		return fmt.Errorf("failed to create feature directory: %w", err)
	}

	// Write template
	if err := os.WriteFile(featurePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write feature template: %w", err)
	}

	return nil
}

// GetShellExtension returns the file extension for a given shell
func GetShellExtension(shellName string) string {
	switch shellName {
	case "fish":
		return ".fish"
	case "powershell":
		return ".ps1"
	default:
		return ".sh"
	}
}
