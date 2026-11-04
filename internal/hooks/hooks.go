package hooks

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
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
	initPath = normalizeHookPath(shell, initPath)

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
	bashrcPath = filepath.ToSlash(bashrcPath)

	return fmt.Sprintf(`%s
if [ -r "%s" ]; then
  . "%s"
fi
%s`, LoginShimStartMarker, bashrcPath, bashrcPath, LoginShimEndMarker)
}

func normalizeHookPath(shell, path string) string {
	switch shell {
	case "bash", "posix", "zsh", "fish":
		return strings.ReplaceAll(path, "\\", "/")
	default:
		return path
	}
}

// HasHook checks if a profile file already contains the hook
func HasHook(profilePath string) (bool, error) {
	content, err := os.ReadFile(profilePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read profile: %w", err)
	}
	ranges, err := hookRanges(string(content))
	return len(ranges) > 0, err
}

// hookRanges validates balanced, non-nested markers before any profile mutation.
// Ranges address SplitAfter lines, preserving all bytes outside valid blocks.
func hookRanges(content string) ([][2]int, error) {
	var ranges [][2]int
	start := -1
	for index, line := range strings.SplitAfter(content, "\n") {
		switch strings.TrimSpace(line) {
		case HookStartMarker:
			if start >= 0 {
				return nil, fmt.Errorf("nested oh-my-dot hook markers; repair the profile block manually")
			}
			start = index
		case HookEndMarker:
			if start < 0 {
				return nil, fmt.Errorf("oh-my-dot hook end marker has no start; repair the profile block manually")
			}
			ranges = append(ranges, [2]int{start, index + 1})
			start = -1
		}
	}
	if start >= 0 {
		return nil, fmt.Errorf("oh-my-dot hook start marker has no end; repair the profile block manually")
	}
	return ranges, nil
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
	if err := os.MkdirAll(filepath.Dir(profilePath), 0700); err != nil {
		return false, fmt.Errorf("failed to create profile directory: %w", err)
	}
	if err := fileops.WriteTextFileLF(profilePath, newContent, 0644); err != nil {
		return false, fmt.Errorf("failed to write profile: %w", err)
	}

	return true, nil
}

// RemoveHook removes the hook block from a profile file
func RemoveHook(profilePath string) error {
	content, err := os.ReadFile(profilePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read profile: %w", err)
	}
	ranges, err := hookRanges(string(content))
	if err != nil {
		return err
	}
	if len(ranges) == 0 {
		return nil
	}
	lines := strings.SplitAfter(string(content), "\n")
	var kept strings.Builder
	previous := 0
	for _, block := range ranges {
		for _, line := range lines[previous:block[0]] {
			kept.WriteString(line)
		}
		previous = block[1]
	}
	for _, line := range lines[previous:] {
		kept.WriteString(line)
	}
	// Preserve the user's line endings, final newline, and existing permissions.
	if err := os.WriteFile(profilePath, []byte(kept.String()), 0644); err != nil {
		return fmt.Errorf("write profile: %w", err)
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

// HookMatches checks that exactly one well-formed managed block matches the
// expected hook. Line ending differences alone do not make a hook stale.
func HookMatches(profilePath, expected string) (bool, error) {
	data, err := os.ReadFile(profilePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read profile: %w", err)
	}
	blocks, err := hookRanges(string(data))
	if err != nil {
		return false, err
	}
	if len(blocks) != 1 {
		return false, fmt.Errorf("expected one managed hook block in %s; repair duplicate or missing markers", profilePath)
	}
	lines := strings.SplitAfter(string(data), "\n")
	actual := strings.Join(lines[blocks[0][0]:blocks[0][1]], "")
	normalize := func(s string) string { return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n")) }
	return normalize(actual) == normalize(expected), nil
}

// ReplaceHook updates only one validated managed block, preserving every byte
// outside it. Malformed or ambiguous blocks are never rewritten.
func ReplaceHook(profilePath, expected string) error {
	data, err := os.ReadFile(profilePath)
	if err != nil {
		return fmt.Errorf("read profile: %w", err)
	}
	blocks, err := hookRanges(string(data))
	if err != nil {
		return err
	}
	if len(blocks) != 1 {
		return fmt.Errorf("cannot replace ambiguous managed hook in %s", profilePath)
	}
	lines := strings.SplitAfter(string(data), "\n")
	block := blocks[0]
	replacement := strings.TrimRight(expected, "\r\n") + "\n"
	if strings.Contains(string(data), "\r\n") {
		replacement = strings.ReplaceAll(replacement, "\n", "\r\n")
	}
	updated := strings.Join(lines[:block[0]], "") + replacement + strings.Join(lines[block[1]:], "")
	if err := os.WriteFile(profilePath, []byte(updated), 0644); err != nil {
		return fmt.Errorf("update managed hook: %w", err)
	}
	return nil
}
