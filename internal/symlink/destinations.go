package symlink

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/repopath"
)

// CheckDestinationAvailable rejects duplicate keys and destinations before adding a file.
func CheckDestinationAvailable(links Linkings, key, destination string) error {
	wanted, err := destinationID(destination)
	if err != nil {
		return fmt.Errorf("resolve destination: %w", err)
	}
	for _, existing := range sortedLinkKeys(links) {
		if err := repopath.Validate(existing); err != nil {
			return fmt.Errorf("invalid existing key %q: %w", existing, err)
		}
		if strings.EqualFold(existing, key) {
			return fmt.Errorf("repository path %q is already registered as %q", key, existing)
		}
		id, err := destinationID(links[existing])
		if err != nil {
			return fmt.Errorf("resolve destination for %q: %w", existing, err)
		}
		if id == wanted {
			return fmt.Errorf("destination is already managed by %q; remove that entry before adding another", existing)
		}
	}
	return nil
}

// ValidateLinkings checks all repository keys and destination conflicts before apply mutates files.
func ValidateLinkings(repoPath string, links Linkings) error {
	keys := make(map[string]string)
	destinations := make(map[string]string)
	for _, key := range sortedLinkKeys(links) {
		if _, err := repopath.Resolve(repoPath, key); err != nil {
			return fmt.Errorf("invalid repository path %q: %w", key, err)
		}
		folded := strings.ToLower(key)
		if previous, ok := keys[folded]; ok {
			return fmt.Errorf("repository paths %q and %q differ only in case", previous, key)
		}
		keys[folded] = key
		id, err := destinationID(links[key])
		if err != nil {
			return fmt.Errorf("destination for %q: %w", key, err)
		}
		if previous, ok := destinations[id]; ok {
			return fmt.Errorf("entries %q and %q target the same destination; select one before applying", previous, key)
		}
		destinations[id] = key
	}
	return nil
}

func sortedLinkKeys(links Linkings) []string {
	keys := make([]string, 0, len(links))
	for key := range links {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func destinationID(destination string) (string, error) {
	expanded, err := fileops.ExpandPath(destination)
	if err != nil {
		return "", fmt.Errorf("expand path: %w", err)
	}
	if !filepath.IsAbs(expanded) {
		return "", fmt.Errorf("destination %q must be an absolute path or ~/path for this OS", destination)
	}
	// Resolve directory aliases without following the final file: an applied
	// symlink should still identify its destination, not its repository source.
	parent := filepath.Dir(expanded)
	suffix := filepath.Base(expanded)
	for {
		resolved, err := filepath.EvalSymlinks(parent)
		if err == nil {
			id := filepath.Join(resolved, suffix)
			if runtime.GOOS == "windows" {
				id = strings.ToLower(id)
			}
			return id, nil
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("resolve parent directory: %w", err)
		}
		next := filepath.Dir(parent)
		if next == parent {
			return "", fmt.Errorf("cannot resolve destination root %q", parent)
		}
		suffix = filepath.Join(filepath.Base(parent), suffix)
		parent = next
	}
}
