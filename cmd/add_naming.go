package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/repopath"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/symlink"
)

// addNameCandidates preserves short names and adds bounded directory context.
// It never includes the home directory itself, a filesystem root, or a volume.
func addNameCandidates(destination, home string) []string {
	key := filepath.Base(destination)
	candidates := []string{key}
	parent := filepath.Dir(destination)
	for depth := 0; depth < 3; depth++ {
		if parent == filepath.Dir(parent) || sameNativePath(parent, home) {
			break
		}
		key = filepath.Base(parent) + "/" + key
		candidates = append(candidates, key)
		parent = filepath.Dir(parent)
	}
	return candidates
}

func sameNativePath(a, b string) bool {
	rel, err := filepath.Rel(a, b)
	return err == nil && rel == "."
}

func availableAddKey(repoPath, key string, links symlink.Linkings) error {
	if err := repopath.CheckAvailable(repoPath, key); err != nil {
		return fmt.Errorf("repository path %q: %w", key, err)
	}
	keys := make([]string, 0, len(links))
	for existing := range links {
		keys = append(keys, existing)
	}
	sort.Strings(keys)
	parts := strings.Split(key, "/")
	for _, existing := range keys {
		other := strings.Split(existing, "/")
		for i := 0; i < len(parts) && i < len(other); i++ {
			if !strings.EqualFold(parts[i], other[i]) {
				break
			}
			if parts[i] != other[i] || i == len(parts)-1 || i == len(other)-1 {
				return fmt.Errorf("repository path %q conflicts with registered entry %q", key, existing)
			}
		}
	}
	return nil
}

func selectAddKey(cmd *cobra.Command, repoPath, destination string, links symlink.Linkings) (string, error) {
	explicit, _ := cmd.Flags().GetString("as")
	if cmd.Flags().Changed("as") || explicit != "" {
		if err := availableAddKey(repoPath, explicit, links); err != nil {
			return "", fmt.Errorf("%w; choose a different --as path", err)
		}
		return explicit, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory for naming: %w", err)
	}
	candidates := addNameCandidates(destination, home)
	var lastErr error
	for _, candidate := range candidates {
		if err := availableAddKey(repoPath, candidate, links); err == nil {
			return candidate, nil
		} else {
			lastErr = err
		}
	}
	if !interactive.ShouldPrompt(cmd, false) {
		return "", fmt.Errorf("cannot choose an available portable name for %s: %w; rerun with --as <folder/filename> to choose another name", destination, lastErr)
	}
	fileops.ColorPrintfn(fileops.Yellow, "Could not choose a repository name for %s: %v", destination, lastErr)
	defaultName := candidates[len(candidates)-1]
	if repopath.Validate(defaultName) != nil {
		defaultName = ""
	}
	for {
		key, err := interactive.PromptInput("Store as (path inside files/)", defaultName)
		if err != nil {
			return "", fmt.Errorf("choose repository name: %w", err)
		}
		if err := availableAddKey(repoPath, key, links); err != nil {
			fileops.ColorPrintfn(fileops.Yellow, "%v", err)
			defaultName = key
			continue
		}
		return key, nil
	}
}
