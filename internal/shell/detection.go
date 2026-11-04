package shell

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	manifestFileName      = "enabled.json"
	localManifestFileName = "enabled.local.json"
)

var (
	currentGOOS = runtime.GOOS
	lookPath    = exec.LookPath
	runCommand  = func(name string, args ...string) ([]byte, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, name, args...)
		command.WaitDelay = time.Second
		output, err := command.Output()
		if ctx.Err() != nil {
			return nil, fmt.Errorf("PowerShell profile query exceeded its five-second limit: %w", ctx.Err())
		}
		return output, err
	}
)

// DetectCurrentShell attempts to detect the current shell
func DetectCurrentShell() (string, error) {
	// PowerShell detection (Windows-specific)
	// PowerShell doesn't set $SHELL, but may set PowerShell-specific environment variables
	if runtime.GOOS == "windows" {
		// POWERSHELL_DISTRIBUTION_CHANNEL is set by PowerShell but not by cmd.exe
		if os.Getenv("PSModulePath") != "" || os.Getenv("POWERSHELL_DISTRIBUTION_CHANNEL") != "" {
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

// IsShellExecutableAvailable reports whether the current environment can resolve a shell executable.
func IsShellExecutableAvailable(shellName string) bool {
	var candidates []string

	switch shellName {
	case "bash":
		candidates = []string{"bash", "bash.exe"}
	case "zsh":
		candidates = []string{"zsh", "zsh.exe"}
	case "fish":
		candidates = []string{"fish", "fish.exe"}
	case "powershell":
		candidates = powershellExecutableCandidates()
	case "posix":
		candidates = []string{"sh", "sh.exe"}
	default:
		return false
	}

	for _, candidate := range candidates {
		if resolvedPath, err := lookPath(candidate); err == nil {
			if shellName == "bash" && isWslBashLauncher(resolvedPath) {
				continue
			}
			return true
		}
	}

	return false
}

func isWslBashLauncher(path string) bool {
	if currentGOOS != "windows" {
		return false
	}

	normalizedPath := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	return strings.HasSuffix(normalizedPath, "/windows/system32/bash.exe")
}

func powershellExecutableCandidates() []string {
	if currentGOOS == "windows" {
		return []string{"pwsh", "powershell", "powershell.exe"}
	}

	// On non-Windows hosts, prefer native pwsh and avoid picking up Windows PowerShell via interop.
	return []string{"pwsh"}
}

// ResolveProfilePath resolves the profile path for a shell, expanding ~ to home directory
func ResolveProfilePath(shellConfig ShellConfig) (string, error) {
	path := shellConfig.ProfilePath

	// Handle PowerShell $PROFILE variable
	if path == "$PROFILE" {
		profilePath, err := detectPowerShellProfilePath()
		if err != nil {
			return "", fmt.Errorf("PowerShell profile discovery failed: %w; run pwsh -NoProfile -Command '$PROFILE.CurrentUserCurrentHost' in your terminal, resolve the error, then retry", err)
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

func detectPowerShellProfilePath() (string, error) {
	for _, executable := range powershellExecutableCandidates() {
		resolvedPath, err := lookPath(executable)
		if err != nil {
			continue
		}
		output, err := runCommand(resolvedPath, "-NoProfile", "-NonInteractive", "-Command", "$PROFILE.CurrentUserCurrentHost")
		if err != nil {
			return "", fmt.Errorf("query profile with %s: %w", executable, err)
		}
		profilePath := strings.TrimSpace(string(output))
		if profilePath == "" {
			return "", fmt.Errorf("%s returned an empty profile path", executable)
		}
		if strings.ContainsAny(profilePath, "\r\n\x00") {
			return "", fmt.Errorf("%s returned an invalid profile path", executable)
		}
		absolute := filepath.IsAbs(profilePath)
		if currentGOOS == "windows" {
			absolute = strings.HasPrefix(profilePath, `\\`) || len(profilePath) > 2 && profilePath[1] == ':' && (profilePath[2] == '\\' || profilePath[2] == '/')
		}
		if !absolute {
			return "", fmt.Errorf("%s returned a relative profile path", executable)
		}
		return profilePath, nil
	}
	return "", fmt.Errorf("no PowerShell executable found on PATH")
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
	return filepath.Join(GetShellDirectory(repoPath, shellName), manifestFileName)
}

// GetLocalManifestPath returns the path to enabled.local.json for a shell
func GetLocalManifestPath(repoPath, shellName string) string {
	return filepath.Join(GetShellDirectory(repoPath, shellName), localManifestFileName)
}

// LocalManifestFileName returns the filename used for device-local shell overrides.
func LocalManifestFileName() string {
	return localManifestFileName
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
