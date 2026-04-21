package featurecmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
	"github.com/spf13/cobra"
)

func (state *commandState) runFeatureUpdate(cmd *cobra.Command, args []string) error {
	repoPath := state.repoPathProvider()
	alias := state.aliasProvider()

	if state.flagInteractive {
		if len(args) > 0 {
			return fmt.Errorf("interactive mode does not accept a feature name")
		}

		return state.runInteractiveFeatureUpdate(repoPath)
	}

	if len(args) == 0 {
		return fmt.Errorf("feature name required (or use -i for interactive mode)")
	}

	featureName := args[0]

	targetShells, err := state.shellsWithFeature(repoPath, featureName)
	if err != nil {
		return err
	}

	if len(targetShells) == 0 {
		return fmt.Errorf("feature '%s' is not installed in any shell", featureName)
	}

	for _, shellName := range targetShells {
		fileops.ColorPrintfn(fileops.Cyan, "Refreshing %s in %s...", featureName, shellName)

		if err := shell.RefreshFeatureTemplate(repoPath, shellName, featureName); err != nil {
			return fmt.Errorf("failed to refresh feature in %s: %w", shellName, err)
		}

		fileops.ColorPrintfn(fileops.Green, "  ✓ Feature file updated")
	}

	if err := autoCommitShellFeatureChanges("Refresh shell feature: " + featureName); err != nil {
		return fmt.Errorf("failed to commit shell feature changes: %w", err)
	}

	fmt.Println()
	fileops.ColorPrintfn(fileops.Cyan, "Run '%s push' to push committed changes", alias)
	return nil
}

func (state *commandState) runInteractiveFeatureUpdate(repoPath string) error {
	if !isInteractive() {
		return fmt.Errorf("cannot run interactive mode in non-interactive environment")
	}

	var candidateShells []string
	if len(state.flagShell) > 0 {
		candidateShells = append(candidateShells, state.flagShell...)
	} else {
		allShells, err := shell.ListShellsWithFeatures(repoPath)
		if err != nil {
			return fmt.Errorf("failed to list shells: %w", err)
		}
		candidateShells = allShells
	}

	featureMap, err := collectUpdatableFeatures(repoPath, candidateShells)
	if err != nil {
		return err
	}
	if len(featureMap) == 0 {
		fileops.ColorPrintln("No refreshable features found", fileops.Yellow)
		return nil
	}

	labels := make([]string, 0, len(featureMap))
	labelToFeatureName := make(map[string]string, len(featureMap))
	for featureName, shellsWithFeature := range featureMap {
		sort.Strings(shellsWithFeature)
		label := fmt.Sprintf("%s (in %s)", featureName, strings.Join(shellsWithFeature, ", "))
		labels = append(labels, label)
		labelToFeatureName[label] = featureName
	}
	sort.Strings(labels)

	selectedLabels, err := interactive.MultiSelect(
		"Select features to refresh:",
		labels,
		nil,
	)
	if err != nil {
		if err.Error() == "cancelled" {
			fileops.ColorPrintln("Cancelled", fileops.Yellow)
			return nil
		}
		return fmt.Errorf("feature selection cancelled: %w", err)
	}
	if len(selectedLabels) == 0 {
		fileops.ColorPrintln("No features selected", fileops.Yellow)
		return nil
	}

	selectedFeatureNames := make([]string, 0, len(selectedLabels))
	for _, label := range selectedLabels {
		selectedFeatureNames = append(selectedFeatureNames, labelToFeatureName[label])
	}

	for _, featureName := range selectedFeatureNames {
		shellsWithFeature := featureMap[featureName]
		sort.Strings(shellsWithFeature)

		for _, shellName := range shellsWithFeature {
			fileops.ColorPrintfn(fileops.Cyan, "Refreshing %s in %s...", featureName, shellName)

			if err := shell.RefreshFeatureTemplate(repoPath, shellName, featureName); err != nil {
				return fmt.Errorf("failed to refresh feature in %s: %w", shellName, err)
			}

			fileops.ColorPrintfn(fileops.Green, "  ✓ Feature file updated")
		}
	}

	if err := autoCommitShellFeatureChanges("Refresh shell features (interactive)"); err != nil {
		return fmt.Errorf("failed to commit shell feature changes: %w", err)
	}

	fmt.Println()
	fileops.ColorPrintfn(fileops.Cyan, "Run '%s push' to push committed changes", state.aliasProvider())
	return nil
}

func collectUpdatableFeatures(repoPath string, candidateShells []string) (map[string][]string, error) {
	featureMap := make(map[string][]string)

	for _, shellName := range candidateShells {
		manifestPath := shell.GetManifestPath(repoPath, shellName)
		localManifestPath := shell.GetLocalManifestPath(repoPath, shellName)
		merged, err := manifest.ParseManifestWithLocal(manifestPath, localManifestPath)
		if err != nil {
			continue
		}

		for _, feature := range merged.Features {
			if !catalog.HasFeatureTemplate(feature.Name, shellName) {
				continue
			}

			featureMap[feature.Name] = append(featureMap[feature.Name], shellName)
		}
	}

	return featureMap, nil
}

func (state *commandState) shellsWithFeature(repoPath, featureName string) ([]string, error) {
	var candidateShells []string
	if len(state.flagShell) > 0 {
		candidateShells = append(candidateShells, state.flagShell...)
	} else {
		allShells, err := shell.ListShellsWithFeatures(repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to list shells: %w", err)
		}
		candidateShells = allShells
	}

	installedShells := make([]string, 0, len(candidateShells))
	for _, shellName := range candidateShells {
		manifestPath := shell.GetManifestPath(repoPath, shellName)
		localManifestPath := shell.GetLocalManifestPath(repoPath, shellName)
		merged, err := manifest.ParseManifestWithLocal(manifestPath, localManifestPath)
		if err != nil {
			continue
		}

		for _, feature := range merged.Features {
			if feature.Name == featureName {
				installedShells = append(installedShells, shellName)
				break
			}
		}
	}

	return installedShells, nil
}
