package featurecmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
	"github.com/spf13/cobra"
)

func (state *commandState) runFeatureRemove(cmd *cobra.Command, args []string) error {
	repoPath := state.repoPathProvider()
	alias := state.aliasProvider()

	if state.flagInteractive {
		return state.runInteractiveFeatureRemove(repoPath)
	}

	if len(args) == 0 {
		return fmt.Errorf("feature name required (or use -i for interactive mode)\n\nExamples:\n  %s feature remove git-prompt\n  %s feature remove -i                    # Browse features interactively\n  %s feature remove git-prompt --all      # Remove from all shells", alias, alias, alias)
	}

	featureName := args[0]

	var targetShells []string
	if state.flagAll || len(state.flagShell) > 0 {
		if state.flagAll {
			allShells, err := shell.ListShellsWithFeatures(repoPath)
			if err != nil {
				return fmt.Errorf("failed to list shells: %w", err)
			}
			targetShells = allShells
		} else {
			targetShells = state.flagShell
		}
	} else {
		if !isInteractive() {
			return fmt.Errorf("cannot prompt in non-interactive mode\nPlease specify shell(s) with --shell flag or use --all")
		}

		allShells, err := shell.ListShellsWithFeatures(repoPath)
		if err != nil {
			return fmt.Errorf("failed to list shells: %w", err)
		}
		if len(allShells) == 0 {
			return fmt.Errorf("no shells have been initialized")
		}

		var shellsWithFeature []string
		for _, shellName := range allShells {
			if isFeatureInstalled(repoPath, shellName, featureName) {
				shellsWithFeature = append(shellsWithFeature, shellName)
			}
		}

		if len(shellsWithFeature) == 0 {
			return fmt.Errorf("feature '%s' is not installed in any shell", featureName)
		}

		targetShells, err = interactive.MultiSelect(
			fmt.Sprintf("Remove feature '%s' from which shells?", featureName),
			shellsWithFeature,
			nil,
		)
		if err != nil {
			return err
		}
	}

	for _, shellName := range targetShells {
		fileops.ColorPrintfn(fileops.Cyan, "Removing %s from %s...", featureName, shellName)

		err := shell.RemoveFeatureFromShell(repoPath, shellName, featureName)
		if err != nil {
			return fmt.Errorf("failed to remove feature from %s: %w", shellName, err)
		}

		fileops.ColorPrintfn(fileops.Green, "  ✓ Feature removed")

		needsCleanup, err := shell.NeedsCleanup(repoPath, shellName)
		if err != nil {
			return fmt.Errorf("failed to check if cleanup needed: %w", err)
		}

		if needsCleanup {
			fileops.ColorPrintfn(fileops.Yellow, "\nNo features remaining in %s shell", shellName)

			shouldCleanup := state.flagForce
			if !shouldCleanup && isInteractive() {
				shouldCleanup, err = interactive.Confirm(
					fmt.Sprintf("Remove %s shell integration (hooks and directory)?", shellName),
					true,
				)
				if err != nil {
					return err
				}
			}

			if shouldCleanup {
				if err := shell.CleanupShellDirectory(repoPath, shellName); err != nil {
					return fmt.Errorf("failed to cleanup shell directory: %w", err)
				}
				fileops.ColorPrintfn(fileops.Green, "  ✓ Removed %s shell directory", shellName)
			}
		}
	}

	if err := autoCommitShellFeatureChanges("Remove shell feature: " + featureName); err != nil {
		return fmt.Errorf("failed to commit shell feature changes: %w", err)
	}

	fmt.Println()
	fileops.ColorPrintfn(fileops.Cyan, "Run '%s apply' to sync changes", alias)
	fileops.ColorPrintfn(fileops.Cyan, "Run '%s push' to push committed changes", alias)

	return nil
}

func (state *commandState) runInteractiveFeatureRemove(repoPath string) error {
	alias := state.aliasProvider()

	if !isInteractive() {
		return fmt.Errorf("cannot run interactive mode in non-interactive environment")
	}

	allShells, err := shell.ListShellsWithFeatures(repoPath)
	if err != nil {
		return fmt.Errorf("failed to list shells: %w", err)
	}
	if len(allShells) == 0 {
		fileops.ColorPrintln("No shell features configured", fileops.Yellow)
		return nil
	}

	selectedShells, err := interactive.MultiSelect(
		"Select shells to remove features from:",
		allShells,
		nil,
	)
	if err != nil {
		if err.Error() == "cancelled" {
			fileops.ColorPrintln("Cancelled", fileops.Yellow)
			return nil
		}
		return fmt.Errorf("shell selection cancelled: %w", err)
	}
	if len(selectedShells) == 0 {
		fileops.ColorPrintln("No shells selected", fileops.Yellow)
		return nil
	}

	type featureInfo struct {
		name   string
		shells []string
	}

	featureMap := make(map[string][]string)
	for _, shellName := range selectedShells {
		manifestPath := shell.GetManifestPath(repoPath, shellName)
		localManifestPath := shell.GetLocalManifestPath(repoPath, shellName)
		merged, err := manifest.ParseManifestWithLocal(manifestPath, localManifestPath)
		if err != nil {
			continue
		}

		for _, feature := range merged.Features {
			featureMap[feature.Name] = append(featureMap[feature.Name], shellName)
		}
	}

	if len(featureMap) == 0 {
		fileops.ColorPrintln("No features found in selected shells", fileops.Yellow)
		return nil
	}

	var features []featureInfo
	var featureLabels []string
	for name, shellsWithFeature := range featureMap {
		features = append(features, featureInfo{name: name, shells: shellsWithFeature})
		label := fmt.Sprintf("%s (in %s)", name, strings.Join(shellsWithFeature, ", "))
		featureLabels = append(featureLabels, label)
	}

	sort.Slice(features, func(i, j int) bool {
		return features[i].name < features[j].name
	})
	sort.Strings(featureLabels)

	selectedLabels, err := interactive.MultiSelect(
		"Select features to remove:",
		featureLabels,
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

	var selectedFeatures []string
	for _, label := range selectedLabels {
		name := strings.Split(label, " (in ")[0]
		selectedFeatures = append(selectedFeatures, name)
	}

	removedCount := 0
	for _, featureName := range selectedFeatures {
		shellsWithFeature := featureMap[featureName]
		for _, shellName := range shellsWithFeature {
			fileops.ColorPrintfn(fileops.Cyan, "Removing %s from %s...", featureName, shellName)

			err := shell.RemoveFeatureFromShell(repoPath, shellName, featureName)
			if err != nil {
				fileops.ColorPrintfn(fileops.Red, "  ✗ Failed: %s", err)
				continue
			}

			fileops.ColorPrintfn(fileops.Green, "  ✓ Feature removed")
			removedCount++

			needsCleanup, err := shell.NeedsCleanup(repoPath, shellName)
			if err != nil {
				continue
			}

			if needsCleanup {
				fileops.ColorPrintfn(fileops.Yellow, "\nNo features remaining in %s shell", shellName)

				shouldCleanup, err := interactive.Confirm(
					fmt.Sprintf("Remove %s shell integration (hooks and directory)?", shellName),
					true,
				)
				if err != nil || !shouldCleanup {
					continue
				}

				if err := shell.CleanupShellDirectory(repoPath, shellName); err != nil {
					fileops.ColorPrintfn(fileops.Red, "  ✗ Failed to cleanup: %s", err)
				} else {
					fileops.ColorPrintfn(fileops.Green, "  ✓ Removed %s shell directory", shellName)
				}
			}
		}
	}

	fmt.Println()
	if removedCount > 0 {
		if err := autoCommitShellFeatureChanges("Remove shell features (interactive)"); err != nil {
			return fmt.Errorf("failed to commit shell feature changes: %w", err)
		}

		fileops.ColorPrintfn(fileops.Green, "Successfully removed %d feature(s)", removedCount)
		fmt.Println()
		fileops.ColorPrintfn(fileops.Cyan, "Run '%s apply' to sync changes", alias)
		fileops.ColorPrintfn(fileops.Cyan, "Run '%s push' to push committed changes", alias)
	} else {
		fileops.ColorPrintln("No features were removed", fileops.Yellow)
	}

	return nil
}
