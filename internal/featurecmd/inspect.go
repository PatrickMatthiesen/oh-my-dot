package featurecmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
	"github.com/spf13/cobra"
)

type featureRecord struct {
	Shell      string         `json:"shell"`
	Name       string         `json:"name"`
	Strategy   string         `json:"strategy"`
	OnCommand  []string       `json:"onCommand,omitempty"`
	Disabled   bool           `json:"disabled"`
	Options    map[string]any `json:"options,omitempty"`
	Origin     string         `json:"origin"`
	Overridden bool           `json:"overridden"`
	Path       string         `json:"path"`
}

type featureInventory struct {
	SchemaVersion int             `json:"schemaVersion"`
	Features      []featureRecord `json:"features"`
}

// Inspection is strict: a broken private override cannot be reported as healthy
// shared configuration. Runtime startup retains its separate warn-and-ignore policy.
func readFeatureManifest(repoPath, shellName string) (*manifest.FeatureManifest, *manifest.FeatureManifest, *manifest.MergedManifest, error) {
	if !shell.IsShellSupported(shellName) {
		return nil, nil, nil, fmt.Errorf("unsupported shell: %s", shellName)
	}
	base, err := manifest.ParseManifest(shell.GetManifestPath(repoPath, shellName))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("read %s shared configuration: %w", shellName, err)
	}
	localFile, err := manifest.OpenValidatedLocalManifest(shell.GetLocalManifestPath(repoPath, shellName))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("read %s local configuration: %w", shellName, err)
	}
	var local *manifest.FeatureManifest
	if localFile != nil {
		defer localFile.Close()
		local, err = manifest.ParseManifestReader(localFile)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("parse local configuration %s: %w", shell.GetLocalManifestPath(repoPath, shellName), err)
		}
	}
	return base, local, manifest.MergeManifests(base, local), nil
}

func featureRecords(repoPath string, selected []string) ([]featureRecord, error) {
	shells := append([]string(nil), selected...)
	if len(shells) == 0 {
		var err error
		shells, err = shell.ListShellsWithFeatures(repoPath)
		if err != nil {
			return nil, fmt.Errorf("list configured shells: %w", err)
		}
	}
	sort.Strings(shells)
	records := make([]featureRecord, 0)
	for _, shellName := range shells {
		_, _, merged, err := readFeatureManifest(repoPath, shellName)
		if err != nil {
			return nil, err
		}
		for _, feature := range merged.Features {
			path, err := shell.GetEffectiveFeatureFilePath(repoPath, shellName, feature.Name)
			if err != nil {
				return nil, fmt.Errorf("resolve feature implementation: %w", err)
			}
			origin := "shared"
			if feature.Override.IsFromLocal {
				origin = "local"
			}
			config := feature.FeatureConfig
			if config.Strategy == "" {
				if metadata, ok := catalog.GetFeature(config.Name); ok {
					config.Strategy = metadata.DefaultStrategy
				} else {
					config.Strategy = "eager"
				}
			}
			records = append(records, featureRecord{Shell: shellName, Name: config.Name, Strategy: config.Strategy, OnCommand: config.OnCommand, Disabled: config.Disabled, Options: config.Options, Origin: origin, Overridden: feature.Override.IsOverridden, Path: path})
		}
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].Shell == records[j].Shell {
			return records[i].Name < records[j].Name
		}
		return records[i].Shell < records[j].Shell
	})
	return records, nil
}

func writeFeatureJSON(cmd *cobra.Command, value any) error {
	if err := json.NewEncoder(cmd.OutOrStdout()).Encode(value); err != nil {
		return fmt.Errorf("write feature JSON: %w", err)
	}
	return nil
}

func configuredRecordsOrEmpty(repoPath string, selected []string) ([]featureRecord, error) {
	for _, shellName := range selected {
		if !shell.IsShellSupported(shellName) {
			return nil, fmt.Errorf("unsupported shell: %s", shellName)
		}
	}
	records, err := featureRecords(repoPath, selected)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		if _, statErr := os.Stat(shell.GetShellDirectory(repoPath, "")); os.IsNotExist(statErr) {
			return []featureRecord{}, nil
		}
	}
	return records, err
}

func featurePresent(repoPath, shellName, featureName string) (bool, error) {
	if _, err := os.Stat(shell.GetManifestPath(repoPath, shellName)); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	_, _, merged, err := readFeatureManifest(repoPath, shellName)
	if err != nil {
		return false, err
	}
	for _, feature := range merged.Features {
		if feature.Name == featureName {
			return true, nil
		}
	}
	return false, nil
}

func (state *commandState) installedSettingsMatch(repoPath, name string, shells []string, metadata catalog.FeatureMetadata, overrides map[string]any, cmd *cobra.Command) (bool, error) {
	same := func(left, right any) bool {
		a, aErr := json.Marshal(left)
		b, bErr := json.Marshal(right)
		return aErr == nil && bErr == nil && string(a) == string(b)
	}
	for _, shellName := range shells {
		base, _, merged, err := readFeatureManifest(repoPath, shellName)
		if err != nil {
			return false, err
		}
		var config *manifest.FeatureConfig
		if !state.flagLocal {
			config, _ = base.GetFeature(name)
		} else {
			for _, feature := range merged.Features {
				if feature.Name == name {
					value := feature.FeatureConfig
					config = &value
					break
				}
			}
		}
		if config == nil {
			return false, nil
		}
		for key, wanted := range overrides {
			actual, exists := config.Options[key]
			if !exists {
				for _, opt := range metadata.Options {
					if opt.Name == key {
						actual = opt.Default
						break
					}
				}
			}
			if !same(actual, wanted) {
				return false, nil
			}
		}
		actualStrategy := config.Strategy
		if actualStrategy == "" {
			actualStrategy = metadata.DefaultStrategy
		}
		if state.flagStrategy != "" && actualStrategy != state.flagStrategy {
			return false, nil
		}
		if len(state.flagOnCommand) > 0 || cmd.Flags().Changed("on-command") {
			if !same(config.OnCommand, state.flagOnCommand) {
				return false, nil
			}
		}
		if state.flagDisabled || cmd.Flags().Changed("disabled") {
			if config.Disabled != state.flagDisabled {
				return false, nil
			}
		}
	}
	return true, nil
}
