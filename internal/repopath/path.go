// Package repopath handles portable paths relative to a repository's files directory.
package repopath

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// Validate checks that key is a portable, forward-slash relative file path.
func Validate(key string) error {
	if key == "" || !utf8.ValidString(key) {
		return fmt.Errorf("repository file path must be nonempty valid UTF-8")
	}
	for _, part := range strings.Split(key, "/") {
		if part == "" || part == "." || part == ".." || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") {
			return fmt.Errorf("invalid repository path component %q", part)
		}
		for _, char := range part {
			if char < 32 || strings.ContainsRune(`\<>:"|?*`, char) {
				return fmt.Errorf("nonportable character in repository path component %q", part)
			}
		}
		if strings.EqualFold(part, ".git") || strings.EqualFold(part, "git~1") {
			return fmt.Errorf("reserved Git metadata name in repository path component %q", part)
		}
		base := strings.ToUpper(strings.TrimRight(strings.SplitN(part, ".", 2)[0], " "))
		deviceNumber := strings.TrimPrefix(strings.TrimPrefix(base, "COM"), "LPT")
		if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || base == "CONIN$" || base == "CONOUT$" ||
			((strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && len([]rune(deviceNumber)) == 1 && strings.Contains("123456789¹²³", deviceNumber)) {
			return fmt.Errorf("reserved Windows name in repository path component %q", part)
		}
	}
	return nil
}

// Resolve returns the absolute path for key and rejects existing symlink components.
// Missing directories and files are allowed so callers can plan new additions.
func Resolve(repoPath, key string) (string, error) {
	if err := Validate(key); err != nil {
		return "", fmt.Errorf("invalid repository file path: %w", err)
	}
	if strings.TrimSpace(repoPath) == "" {
		return "", fmt.Errorf("repository path is not set")
	}
	root, err := filepath.Abs(repoPath)
	if err != nil {
		return "", fmt.Errorf("resolve repository directory: %w", err)
	}
	parts := append([]string{"files"}, strings.Split(key, "/")...)
	current := root
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("inspect repository path %s: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("repository path %s is a symlink", current)
		}
		if i < len(parts)-1 && !info.IsDir() {
			return "", fmt.Errorf("repository path %s is not a directory", current)
		}
	}
	return current, nil
}

// CheckAvailable rejects existing targets, prefix conflicts, and differently cased
// components so additions behave consistently on case-sensitive and Windows filesystems.
func CheckAvailable(repoPath, key string) error {
	if _, err := Resolve(repoPath, key); err != nil {
		return fmt.Errorf("cannot add repository file %q: %w", key, err)
	}
	parts := append([]string{"files"}, strings.Split(key, "/")...)
	current := repoPath
	for i, part := range parts {
		entries, err := os.ReadDir(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect repository directory %s: %w", current, err)
		}
		for _, entry := range entries {
			if !strings.EqualFold(entry.Name(), part) {
				continue
			}
			if entry.Name() != part {
				return fmt.Errorf("repository path %q conflicts with existing casing %q", key, filepath.Join(current, entry.Name()))
			}
			if i == len(parts)-1 {
				return fmt.Errorf("repository path %q already exists; choose a different repository file path", key)
			}
			if !entry.IsDir() {
				return fmt.Errorf("repository path %q conflicts with existing file %s", key, filepath.Join(current, part))
			}
		}
		current = filepath.Join(current, part)
	}
	return nil
}

// List returns sorted portable file keys recursively, rejecting symlinks and unsafe paths.
// A missing files directory is an empty collection.
func List(repoPath string) ([]string, error) {
	if strings.TrimSpace(repoPath) == "" {
		return nil, fmt.Errorf("repository path is not set")
	}
	root, err := filepath.Abs(filepath.Join(repoPath, "files"))
	if err != nil {
		return nil, fmt.Errorf("inspect repository files directory: %w", err)
	}
	info, err := os.Lstat(root)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect repository files directory: %w", err)
	}
	if err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
		return nil, fmt.Errorf("repository files path must be a directory without symlinks")
	}
	keys := []string{}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path == root && os.IsNotExist(walkErr) {
				return nil
			}
			return fmt.Errorf("walk repository files: %w", walkErr)
		}
		if path == root {
			return nil
		}
		key, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("resolve repository file key: %w", err)
		}
		key = filepath.ToSlash(key)
		if err := Validate(key); err != nil {
			return fmt.Errorf("invalid stored file %q: %w", key, err)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("repository path %s is a symlink", path)
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("repository path %s is not a regular file", path)
		}
		keys = append(keys, key)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list repository files: %w", err)
	}
	sort.Strings(keys)
	return keys, nil
}
