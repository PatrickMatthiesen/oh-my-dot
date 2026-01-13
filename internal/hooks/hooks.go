package hooks

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	// Marker patterns for hook blocks
	HookStartMarker = "# >>> oh-my-dot shell >>>"
	HookEndMarker   = "# <<< oh-my-dot shell <<<"

	// Bash login shim markers (separate from main hook)
	LoginShimStartMarker = "# >>> oh-my-dot bash login >>>"
	LoginShimEndMarker   = "# <<< oh-my-dot bash login <<<"
)

// HookContent represents the content to be inserted into a shell profile
type HookContent struct {
	Shell    string
	InitPath string // Path to the init script (e.g., "$HOME/dotfiles/omd-shells/bash/init.sh")
}

// GenerateHook generates the hook content for a specific shell
func GenerateHook(shell, initPath string) string {
	switch shell {
	case "bash", "posix":
		return fmt.Sprintf(`%s
if [ -r "%s" ]; then
  . "%s"
fi
%s`, HookStartMarker, initPath, initPath, HookEndMarker)

	case "zsh":
		return fmt.Sprintf(`%s
if [ -r "%s" ]; then
  source "%s"
fi
%s`, HookStartMarker, initPath, initPath, HookEndMarker)

	case "fish":
		return fmt.Sprintf(`%s
if test -r "%s"
  source "%s"
end
%s`, HookStartMarker, initPath, initPath, HookEndMarker)

	case "powershell":
		return fmt.Sprintf(`%s
$omdInit = "%s"
if (Test-Path $omdInit) {
  . $omdInit
}
%s`, HookStartMarker, initPath, HookEndMarker)

	default:
		return ""
	}
}

// GenerateBashLoginShim generates the bash login shim for .bash_profile
func GenerateBashLoginShim(bashrcPath string) string {
	return fmt.Sprintf(`%s
if [ -r "%s" ]; then
  . "%s"
fi
%s`, LoginShimStartMarker, bashrcPath, bashrcPath, LoginShimEndMarker)
}

// HasHook checks if a profile file already contains the hook
func HasHook(profilePath string) (bool, error) {
	file, err := os.Open(profilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == HookStartMarker {
			return true, nil
		}
	}

	return false, scanner.Err()
}

// InsertHook inserts the hook into a profile file (idempotent)
// Returns (added, error) where added=true if hook was inserted, false if already existed
func InsertHook(profilePath, hookContent string) (bool, error) {
	// Check if hook already exists
	hasHook, err := HasHook(profilePath)
	if err != nil {
		return false, fmt.Errorf("failed to check for existing hook: %w", err)
	}
	if hasHook {
		// Hook already exists, nothing to do
		return false, nil
	}

	// Read existing content if file exists
	var existingContent string
	data, err := os.ReadFile(profilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, fmt.Errorf("failed to read profile: %w", err)
		}
		// File doesn't exist, create it
	} else {
		existingContent = string(data)
	}

	// Append hook to the end
	var newContent string
	if existingContent != "" && !strings.HasSuffix(existingContent, "\n") {
		newContent = existingContent + "\n\n" + hookContent + "\n"
	} else if existingContent != "" {
		newContent = existingContent + "\n" + hookContent + "\n"
	} else {
		newContent = hookContent + "\n"
	}

	// Write back to file
	if err := os.WriteFile(profilePath, []byte(newContent), 0644); err != nil {
		return false, fmt.Errorf("failed to write profile: %w", err)
	}

	return true, nil
}

// RemoveHook removes the hook block from a profile file
func RemoveHook(profilePath string) error {
	file, err := os.Open(profilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, nothing to remove
			return nil
		}
		return fmt.Errorf("failed to open profile: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	inHookBlock := false
	hookFound := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == HookStartMarker {
			inHookBlock = true
			hookFound = true
			continue
		}

		if trimmed == HookEndMarker {
			inHookBlock = false
			continue
		}

		if !inHookBlock {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read profile: %w", err)
	}

	if !hookFound {
		// No hook found, nothing to do
		return nil
	}

	// Write back without the hook block
	content := strings.Join(lines, "\n")
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	if err := os.WriteFile(profilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write profile: %w", err)
	}

	return nil
}

// NeedsBashLoginShim checks if .bash_profile needs a shim to source .bashrc
func NeedsBashLoginShim(bashProfilePath string) (bool, error) {
	file, err := os.Open(bashProfilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, so we'll need to create it with the shim
			return true, nil
		}
		return false, err
	}
	defer file.Close()

	// Check if .bashrc is already sourced or if our shim is present
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check if .bashrc is already being sourced
		if strings.Contains(line, ".bashrc") && (strings.Contains(line, "source") || strings.Contains(line, ". ")) {
			return false, nil
		}

		// Check if our shim is present
		if line == LoginShimStartMarker {
			return false, nil
		}
	}

	return true, scanner.Err()
}
