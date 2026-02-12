package shell

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
)

const ohMyPoshThemeFileName = "oh-my-posh.omp.json"

// HelpersFileContent is the template content for the helpers.sh file
const HelpersFileContent = `#!/usr/bin/env sh
# oh-my-dot shell framework - helper functions
# Shared utilities for all shells

# Helper function to check if a command exists
omd_command_exists() {
    command -v "$1" >/dev/null 2>&1
}
`

// InitializeShellDirectory creates the directory structure for a shell
func InitializeShellDirectory(repoPath, shellName string) error {
	shellDir := GetShellDirectory(repoPath, shellName)

	// Create shell directory
	if err := os.MkdirAll(shellDir, 0755); err != nil {
		return fmt.Errorf("failed to create shell directory: %w", err)
	}

	// Create features subdirectory
	featuresDir := GetFeaturesDirectory(repoPath, shellName)
	if err := os.MkdirAll(featuresDir, 0755); err != nil {
		return fmt.Errorf("failed to create features directory: %w", err)
	}

	// Create lib directory (shared)
	libDir := filepath.Join(repoPath, "omd-shells", "lib")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		return fmt.Errorf("failed to create lib directory: %w", err)
	}

	// Create helpers.sh if it doesn't exist
	helpersPath := filepath.Join(libDir, "helpers.sh")
	if _, err := os.Stat(helpersPath); os.IsNotExist(err) {
		if err := os.WriteFile(helpersPath, []byte(HelpersFileContent), 0644); err != nil {
			return fmt.Errorf("failed to create helpers.sh: %w", err)
		}
	}

	// Create empty enabled.json
	manifestPath := GetManifestPath(repoPath, shellName)
	emptyManifest := &manifest.FeatureManifest{
		Features: []manifest.FeatureConfig{},
	}
	if err := manifest.WriteManifest(manifestPath, emptyManifest); err != nil {
		return fmt.Errorf("failed to create manifest: %w", err)
	}

	// Create init script using the generator
	if err := RegenerateInitScript(repoPath, shellName); err != nil {
		return fmt.Errorf("failed to generate init script: %w", err)
	}

	return nil
}

// AddFeatureToShell adds a feature to a specific shell
func AddFeatureToShell(repoPath, shellName, featureName string, strategy string, onCommand []string, disabled bool, options map[string]any) error {
	// Check if shell directory exists, if not initialize it
	if !ShellDirectoryExists(repoPath, shellName) {
		if err := InitializeShellDirectory(repoPath, shellName); err != nil {
			return fmt.Errorf("failed to initialize shell directory: %w", err)
		}
	}

	// Load manifest
	manifestPath := GetManifestPath(repoPath, shellName)
	m, err := manifest.ParseManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Get feature metadata from catalog
	metadata, inCatalog := catalog.GetFeature(featureName)

	// Determine strategy (use provided or default from catalog)
	if strategy == "" && inCatalog {
		strategy = metadata.DefaultStrategy
	}

	// Determine onCommand (use provided or default from catalog)
	if len(onCommand) == 0 && inCatalog && metadata.DefaultStrategy == "on-command" {
		onCommand = metadata.DefaultCommands
	}

	// Create feature config
	featureConfig := manifest.FeatureConfig{
		Name:      featureName,
		Strategy:  strategy,
		OnCommand: onCommand,
		Disabled:  disabled,
		Options:   options,
	}

	// Add to manifest
	if err := m.AddFeature(featureConfig); err != nil {
		return fmt.Errorf("failed to add feature to manifest: %w", err)
	}

	// Write manifest
	if err := manifest.WriteManifest(manifestPath, m); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	// Create feature file template
	featurePath, err := GetFeatureFilePath(repoPath, shellName, featureName)
	if err != nil {
		return err
	}

	// Try catalog template first, then generic template
	var featureContent string
	if catalog.HasFeatureTemplate(featureName, shellName) {
		if err := catalog.WriteFeatureTemplate(repoPath, shellName, featureName, options); err != nil {
			return fmt.Errorf("failed to write feature template: %w", err)
		}
		// Template written successfully, no need to write again
	} else {
		// No catalog template, use generic template
		featureContent = generateFeatureTemplate(shellName, featureName, metadata, options)
		if err := os.WriteFile(featurePath, []byte(featureContent), 0644); err != nil {
			return fmt.Errorf("failed to create feature file: %w", err)
		}
	}

	// Regenerate init script to include the new feature
	if err := RegenerateInitScript(repoPath, shellName); err != nil {
		return fmt.Errorf("failed to regenerate init script: %w", err)
	}

	return nil
}

// generateFeatureTemplate creates a template feature file
func generateFeatureTemplate(shellName, featureName string, metadata catalog.FeatureMetadata, options map[string]any) string {
	var shebang string
	switch shellName {
	case "bash":
		shebang = "#!/usr/bin/env bash"
	case "zsh":
		shebang = "#!/usr/bin/env zsh"
	case "fish":
		shebang = "#!/usr/bin/env fish"
	case "posix":
		shebang = "#!/usr/bin/env sh"
	case "powershell":
		shebang = "# PowerShell"
	}

	description := featureName
	if metadata.Description != "" {
		description = metadata.Description
	}

	template := fmt.Sprintf(`%s
# oh-my-dot feature: %s
# %s
# 
# Add your shell configuration below

`, shebang, featureName, description)

	// Add option comments if provided
	if len(options) > 0 {
		template += "# Configured options:\n"
		for key, value := range options {
			template += fmt.Sprintf("#   %s: %v\n", key, value)
		}
		template += "\n"
	}

	return template
}

// RemoveFeatureFromShell removes a feature from a shell
func RemoveFeatureFromShell(repoPath, shellName, featureName string) error {
	// Load manifest
	manifestPath := GetManifestPath(repoPath, shellName)
	m, err := manifest.ParseManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Remove from manifest
	if err := m.RemoveFeature(featureName); err != nil {
		return fmt.Errorf("failed to remove feature from manifest: %w", err)
	}

	// Write manifest
	if err := manifest.WriteManifest(manifestPath, m); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	// Delete feature file
	featurePath, err := GetFeatureFilePath(repoPath, shellName, featureName)
	if err != nil {
		return err
	}

	if err := os.Remove(featurePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete feature file: %w", err)
	}

	if featureName == "oh-my-posh" {
		themePath := filepath.Join(GetFeaturesDirectory(repoPath, shellName), ohMyPoshThemeFileName)
		if err := os.Remove(themePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete oh-my-posh theme file: %w", err)
		}
	}

	// Regenerate init script to remove the feature
	if err := RegenerateInitScript(repoPath, shellName); err != nil {
		return fmt.Errorf("failed to regenerate init script: %w", err)
	}

	return nil
}

// NeedsCleanup checks if a shell directory should be cleaned up (no features left)
func NeedsCleanup(repoPath, shellName string) (bool, error) {
	manifestPath := GetManifestPath(repoPath, shellName)
	m, err := manifest.ParseManifest(manifestPath)
	if err != nil {
		return false, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return !m.HasFeatures(), nil
}

// CleanupShellDirectory removes a shell directory and its contents
func CleanupShellDirectory(repoPath, shellName string) error {
	shellDir := GetShellDirectory(repoPath, shellName)
	return os.RemoveAll(shellDir)
}

// EnableFeatureWithOptions enables a disabled feature and optionally updates its configuration
func EnableFeatureWithOptions(repoPath, shellName, featureName, strategy string, onCommand []string) error {
	manifestPath := GetManifestPath(repoPath, shellName)
	m, err := manifest.ParseManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	err = m.UpdateFeature(featureName, func(f *manifest.FeatureConfig) {
		f.Disabled = false
		if strategy != "" {
			f.Strategy = strategy
		}
		if len(onCommand) > 0 {
			f.OnCommand = onCommand
		}
	})
	if err != nil {
		return err
	}

	if err := manifest.WriteManifest(manifestPath, m); err != nil {
		return err
	}

	// Regenerate init script to include the enabled feature
	return RegenerateInitScript(repoPath, shellName)
}

// EnableFeature enables a disabled feature
func EnableFeature(repoPath, shellName, featureName string) error {
	manifestPath := GetManifestPath(repoPath, shellName)
	m, err := manifest.ParseManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	err = m.UpdateFeature(featureName, func(f *manifest.FeatureConfig) {
		f.Disabled = false
	})
	if err != nil {
		return err
	}

	if err := manifest.WriteManifest(manifestPath, m); err != nil {
		return err
	}

	// Regenerate init script to include the enabled feature
	return RegenerateInitScript(repoPath, shellName)
}

// DisableFeature disables a feature
func DisableFeature(repoPath, shellName, featureName string) error {
	manifestPath := GetManifestPath(repoPath, shellName)
	m, err := manifest.ParseManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	err = m.UpdateFeature(featureName, func(f *manifest.FeatureConfig) {
		f.Disabled = true
	})
	if err != nil {
		return err
	}

	if err := manifest.WriteManifest(manifestPath, m); err != nil {
		return err
	}

	// Regenerate init script to exclude the disabled feature
	return RegenerateInitScript(repoPath, shellName)
}

// ListShellsWithFeatures returns a list of shells that have been initialized
func ListShellsWithFeatures(repoPath string) ([]string, error) {
	omdShellsDir := filepath.Join(repoPath, "omd-shells")

	// Check if omd-shells directory exists
	if _, err := os.Stat(omdShellsDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	// Read directory
	entries, err := os.ReadDir(omdShellsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read omd-shells directory: %w", err)
	}

	var shells []string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "lib" {
			// Verify it's a valid shell by checking if enabled.json exists
			manifestPath := GetManifestPath(repoPath, entry.Name())
			if _, err := os.Stat(manifestPath); err == nil {
				shells = append(shells, entry.Name())
			}
		}
	}

	return shells, nil
}
