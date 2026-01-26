package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DetectCurrentShell attempts to detect the current shell
func DetectCurrentShell() (string, error) {
	// PowerShell detection (Windows-specific)
	// PowerShell doesn't set $SHELL, but may set PowerShell-specific environment variables
	if runtime.GOOS == "windows" {
		// POWERSHELL_DISTRIBUTION_CHANNEL is set by PowerShell but not by cmd.exe
		if os.Getenv("POWERSHELL_DISTRIBUTION_CHANNEL") != "" {
			return "powershell", nil
		}
	}

	// Try $SHELL environment variable first
	shellPath := os.Getenv("SHELL")
	if shellPath != "" {
		shellName := filepath.Base(shellPath)
		// Normalize shell name
		shellName = normalizeShellName(shellName)
		if IsShellSupported(shellName) {
			return shellName, nil
		}
	}

	// Try $0 (current shell process)
	// This is less reliable but can work in some cases
	shell0 := os.Getenv("0")
	if shell0 != "" {
		shellName := filepath.Base(shell0)
		shellName = normalizeShellName(shellName)
		if IsShellSupported(shellName) {
			return shellName, nil
		}
	}

	return "", fmt.Errorf("could not detect current shell")
}

// normalizeShellName normalizes shell names (e.g., "bash" from "/bin/bash" or "bash.exe")
func normalizeShellName(name string) string {
	// Remove path separators
	name = filepath.Base(name)

	// Remove common extensions
	name = strings.TrimSuffix(name, ".exe")
	name = strings.TrimSuffix(name, ".bat")
	name = strings.TrimSuffix(name, ".cmd")

	// Handle some common variations
	switch strings.ToLower(name) {
	case "pwsh", "powershell.exe", "powershell":
		return "powershell"
	case "sh", "dash":
		return "posix"
	default:
		return strings.ToLower(name)
	}
}

// ResolveProfilePath resolves the profile path for a shell, expanding ~ to home directory
func ResolveProfilePath(shellConfig ShellConfig) (string, error) {
	path := shellConfig.ProfilePath

	// Handle PowerShell $PROFILE variable
	if path == "$PROFILE" {
		profilePath := os.Getenv("PROFILE")
		if profilePath == "" {
			// Fallback to default PowerShell profile location
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("failed to get user home directory: %w", err)
			}
			profilePath = filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
		}
		return profilePath, nil
	}

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	return path, nil
}

// GetShellDirectory returns the path to the shell directory within omd-shells
func GetShellDirectory(repoPath, shellName string) string {
	return filepath.Join(repoPath, "omd-shells", shellName)
}

// GetInitScriptPath returns the path to the init script for a shell
func GetInitScriptPath(repoPath, shellName string) (string, error) {
	config, ok := GetShellConfig(shellName)
	if !ok {
		return "", fmt.Errorf("unsupported shell: %s", shellName)
	}

	return filepath.Join(GetShellDirectory(repoPath, shellName), config.InitScript), nil
}

// GetManifestPath returns the path to enabled.json for a shell
func GetManifestPath(repoPath, shellName string) string {
	return filepath.Join(GetShellDirectory(repoPath, shellName), "enabled.json")
}

// GetLocalManifestPath returns the path to enabled.local.json for a shell
func GetLocalManifestPath(repoPath, shellName string) string {
	return filepath.Join(GetShellDirectory(repoPath, shellName), "enabled.local.json")
}

// GetFeaturesDirectory returns the path to the features directory for a shell
func GetFeaturesDirectory(repoPath, shellName string) string {
	return filepath.Join(GetShellDirectory(repoPath, shellName), "features")
}

// GetFeatureFilePath returns the path to a specific feature file
func GetFeatureFilePath(repoPath, shellName, featureName string) (string, error) {
	config, ok := GetShellConfig(shellName)
	if !ok {
		return "", fmt.Errorf("unsupported shell: %s", shellName)
	}

	fileName := featureName + config.Extension
	return filepath.Join(GetFeaturesDirectory(repoPath, shellName), fileName), nil
}

// ShellDirectoryExists checks if a shell directory has been initialized
func ShellDirectoryExists(repoPath, shellName string) bool {
	shellDir := GetShellDirectory(repoPath, shellName)
	info, err := os.Stat(shellDir)
	return err == nil && info.IsDir()
}
