package symlink

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/git"
)

type Linkings map[string]string

func AddLinking(name, link string) error {
	links, err := GetLinkings()
	if err != nil {
		return fmt.Errorf("error getting links: %w", err)
	}
	links[name] = link
	return SaveLinkings(links)
}

func RemoveLinking(name string) error {
	links, err := GetLinkings()
	if err != nil {
		return fmt.Errorf("error getting links for removal: %w", err)
	}

	delete(links, name)

	return SaveLinkings(links)
}

// GetLinkings reads registrations for the configured repository.
func GetLinkings() (Linkings, error) { return LoadLinkings(viper.GetString("repo-path")) }

// LoadLinkings reads a repository's registrations without changing configuration.
// Missing metadata means no registrations; invalid or unreadable metadata fails.
func LoadLinkings(repoPath string) (Linkings, error) {
	path := filepath.Join(repoPath, "linkings.json")
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return Linkings{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect links file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("links file %s must be a regular file", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read links file: %w", err)
	}
	var links Linkings
	if err := json.Unmarshal(data, &links); err != nil {
		return nil, fmt.Errorf("parse links file %s: %w", path, err)
	}
	if links == nil {
		return nil, fmt.Errorf("links file %s must contain an object, not null", path)
	}
	return links, nil
}

func SaveLinkings(links Linkings) error {
	linkFile := filepath.Join(viper.GetString("repo-path"), "linkings.json")
	// Ensure repository directory exists
	if err := fileops.EnsureDir(filepath.Dir(linkFile)); err != nil {
		return fmt.Errorf("error ensuring repo path: %w", err)
	}

	file, err := json.MarshalIndent(links, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling links: %w", err)
	}

	err = os.WriteFile(linkFile, file, 0644)
	if err != nil {
		return fmt.Errorf("error writing links to file: %w", err)
	}

	// Use slash path for git regardless of OS
	err = git.StageChange("linkings.json")
	if err != nil {
		return fmt.Errorf("error staging changes: %w", err)
	}

	return nil
}

func BuildLinkPath(file string) (string, error) {
	absPath, err := filepath.Abs(file)
	if err != nil {
		return "", fmt.Errorf("error getting absolute path: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error getting home directory: %w", err)
	}

	// Normalize both paths for comparison
	home = filepath.Clean(home)
	absPath = filepath.Clean(absPath)
	rel, err := filepath.Rel(home, absPath)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel) {
		absPath = "~/" + filepath.ToSlash(rel)
	}

	return filepath.ToSlash(absPath), nil
}
