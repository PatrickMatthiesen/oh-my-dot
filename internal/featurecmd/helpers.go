package featurecmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
)

func autoCommitShellFeatureChanges(commitMessage string) error {
	committed, err := git.StageAndCommitShellFeatureChanges(commitMessage)
	if err != nil {
		return err
	}

	if committed {
		fileops.ColorPrintln("Shell feature changes committed.", fileops.Green)
	} else {
		fileops.ColorPrintln("No committable shell changes detected (device-local overrides are never committed).", fileops.Yellow)
	}

	return nil
}

func sortFeaturesByCategory(features []catalog.FeatureMetadata) {
	categoryOrder := map[string]int{
		"alias":      1,
		"completion": 2,
		"prompt":     3,
		"tool":       4,
	}

	sort.Slice(features, func(i, j int) bool {
		catI := categoryOrder[features[i].Category]
		catJ := categoryOrder[features[j].Category]

		if catI == 0 {
			catI = 999
		}
		if catJ == 0 {
			catJ = 999
		}

		if catI != catJ {
			return catI < catJ
		}

		return features[i].Name < features[j].Name
	})
}

func isFeatureInstalled(repoPath, shellName, featureName string) bool {
	manifestPath := shell.GetManifestPath(repoPath, shellName)
	m, err := manifest.ParseManifest(manifestPath)
	if err != nil {
		return false
	}
	_, err = m.GetFeature(featureName)
	return err == nil
}

func hasPendingFeatureInstall(feature catalog.FeatureMetadata, selectedShells []string, installedFeaturesByShell map[string]map[string]bool) bool {
	for _, shellName := range selectedShells {
		if !feature.SupportsShell(shellName) {
			continue
		}

		if !installedFeaturesByShell[shellName][feature.Name] {
			return true
		}
	}

	return false
}

func filterFeaturesByShells(features []catalog.FeatureMetadata, shells []string) []catalog.FeatureMetadata {
	if len(shells) == 0 {
		return features
	}

	filtered := []catalog.FeatureMetadata{}
	for _, feature := range features {
		for _, shellName := range shells {
			if feature.SupportsShell(shellName) {
				filtered = append(filtered, feature)
				break
			}
		}
	}

	return filtered
}

func parseRawOptionPairs(rawOptions []string) (map[string]any, error) {
	values := make(map[string]any)
	for _, raw := range rawOptions {
		parts := strings.SplitN(raw, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			return nil, fmt.Errorf("invalid --option format '%s' (expected key=value)", raw)
		}

		key := strings.TrimSpace(parts[0])
		values[key] = parts[1]
	}

	return values, nil
}

func preflightFeatureMutation(repoPath string) error {
	if err := git.CheckShellFeatureCommitSafety(repoPath); err != nil {
		return fmt.Errorf("cannot modify features: %w", err)
	}
	return nil
}

// Parse every selected shared manifest before a batch can write or stage files.
// Missing manifests are expected for new shells. Malformed existing files must
// never be mistaken for an unconfigured shell and included in a later commit.
func preflightFeatureManifests(repoPath string, shellNames []string) error {
	for _, shellName := range shellNames {
		if !shell.IsShellSupported(shellName) {
			return fmt.Errorf("unsupported shell: %s", shellName)
		}
		// Check ancestors themselves. On Windows a child of a regular file
		// can report ErrNotExist, which must not mean an unconfigured shell.
		for _, directory := range []string{filepath.Join(repoPath, "omd-shells"), shell.GetShellDirectory(repoPath, shellName)} {
			info, err := os.Stat(directory)
			if err == nil && !info.IsDir() {
				return fmt.Errorf("cannot change features in %s; %s is not a directory", shellName, directory)
			}
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("inspect shell directory %s: %w", directory, err)
			}
		}
		path := shell.GetManifestPath(repoPath, shellName)
		if _, err := manifest.ParseManifest(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("cannot change features in %s; repair %s before retrying: %w", shellName, path, err)
		}
	}
	return nil
}
