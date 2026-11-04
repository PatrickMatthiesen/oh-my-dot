package featurecmd

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
	"github.com/spf13/cobra"
)

func (state *commandState) validateFeatureUpdateArgs(cmd *cobra.Command, args []string) error {
	if err := cobra.MaximumNArgs(1)(cmd, args); err != nil {
		return err
	}

	if !cmd.Flags().Changed("all") {
		return nil
	}

	if len(args) > 0 {
		return fmt.Errorf("--all cannot be used with a feature name")
	}

	if cmd.Flags().Changed("interactive") {
		return fmt.Errorf("--all cannot be used with --interactive")
	}

	return nil
}

func (state *commandState) runFeatureUpdate(cmd *cobra.Command, args []string) error {
	repoPath := state.repoPathProvider()
	alias := state.aliasProvider()

	if err := interactive.ValidateMode(cmd); err != nil {
		return fmt.Errorf("invalid interaction mode: %w", err)
	}
	if state.flagLocal {
		return state.runLocalFeatureUpdate(cmd, args)
	}
	if state.flagInteractive || (len(args) == 0 && !state.flagAll && interactive.ShouldPrompt(cmd, false)) {
		if len(args) > 0 {
			return fmt.Errorf("interactive mode does not accept a feature name")
		}

		return state.runInteractiveFeatureUpdate(cmd, repoPath)
	}

	if len(args) == 0 {
		if state.flagAll {
			return state.runAllFeatureUpdates(repoPath, alias)
		}

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

	if err := refreshFeatureTargets(repoPath, map[string][]string{featureName: targetShells}, "Refresh shell feature: "+featureName, state.flagForce); err != nil {
		return err
	}

	fmt.Println()
	fileops.ColorPrintfn(fileops.Cyan, "Run '%s push' to push committed changes", alias)
	return nil
}

func (state *commandState) runAllFeatureUpdates(repoPath, alias string) error {
	featureMap, err := state.updatableFeaturesByShell(repoPath)
	if err != nil {
		return err
	}
	if len(featureMap) == 0 {
		fileops.ColorPrintln("No refreshable features found", fileops.Yellow)
		return nil
	}

	if err := refreshFeatureTargets(repoPath, featureMap, "Refresh shell features", state.flagForce); err != nil {
		return err
	}

	fmt.Println()
	fileops.ColorPrintfn(fileops.Cyan, "Run '%s push' to push committed changes", alias)
	return nil
}

func (state *commandState) runInteractiveFeatureUpdate(cmd *cobra.Command, repoPath string) error {
	if !interactive.ShouldPrompt(cmd, false) {
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

	selectedMap := make(map[string][]string, len(selectedFeatureNames))
	for _, featureName := range selectedFeatureNames {
		selectedMap[featureName] = featureMap[featureName]
	}
	if err := refreshFeatureTargets(repoPath, selectedMap, "Refresh shell features (interactive)", state.flagForce); err != nil {
		return err
	}

	fmt.Println()
	fileops.ColorPrintfn(fileops.Cyan, "Run '%s push' to push committed changes", state.aliasProvider())
	return nil
}

func collectUpdatableFeatures(repoPath string, candidateShells []string) (map[string][]string, error) {
	if err := preflightFeatureManifests(repoPath, candidateShells); err != nil {
		return nil, err
	}
	featureMap := make(map[string][]string)

	for _, shellName := range candidateShells {
		base, _, _, err := readFeatureManifest(repoPath, shellName)
		if err != nil {
			return nil, err
		}

		for _, feature := range base.Features {
			if !catalog.HasFeatureTemplate(feature.Name, shellName) {
				continue
			}

			featureMap[feature.Name] = append(featureMap[feature.Name], shellName)
		}
	}

	return featureMap, nil
}

func (state *commandState) updatableFeaturesByShell(repoPath string) (map[string][]string, error) {
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

	return collectUpdatableFeatures(repoPath, candidateShells)
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

	if err := preflightFeatureManifests(repoPath, candidateShells); err != nil {
		return nil, err
	}
	installedShells := make([]string, 0, len(candidateShells))
	for _, shellName := range candidateShells {
		base, _, merged, err := readFeatureManifest(repoPath, shellName)
		if err != nil {
			return nil, err
		}

		for _, feature := range merged.Features {
			if feature.Name == featureName {
				if _, err := base.GetFeature(featureName); err != nil {
					return nil, fmt.Errorf("feature %q is local-only in %s; use feature update %s --shell %s --local", featureName, shellName, featureName, shellName)
				}
				installedShells = append(installedShells, shellName)
				break
			}
		}
	}

	return installedShells, nil
}

// All refresh modes share partial-completion semantics: validate before writing,
// commit completed refreshes, then return the collected failures.
func refreshFeatureTargets(repoPath string, featureMap map[string][]string, commitMessage string, force ...bool) error {
	var featureNames, shellNames []string
	for featureName, shells := range featureMap {
		featureNames = append(featureNames, featureName)
		shellNames = append(shellNames, shells...)
	}
	if len(featureNames) == 0 {
		return nil
	}
	if err := preflightFeatureManifests(repoPath, shellNames); err != nil {
		return err
	}
	if err := preflightFeatureMutation(repoPath); err != nil {
		return err
	}
	sort.Strings(featureNames)
	type target struct {
		name, shell string
		plan        *shell.FeatureConfigurationPlan
	}
	var targets []target
	replace := len(force) > 0 && force[0]
	for _, featureName := range featureNames {
		targetShells := featureMap[featureName]
		sort.Strings(targetShells)
		for _, shellName := range targetShells {
			plan, err := shell.PrepareFeatureRefresh(repoPath, shellName, featureName, replace)
			if err != nil {
				return fmt.Errorf("prepare refresh %s in %s: %w", featureName, shellName, err)
			}
			targets = append(targets, target{featureName, shellName, plan})
		}
	}
	updatedCount := 0
	var failures []error
	for _, target := range targets {
		if !target.plan.Changed {
			continue
		}
		fileops.ColorPrintfn(fileops.Cyan, "Refreshing %s in %s...", target.name, target.shell)
		if err := target.plan.Apply(); err != nil {
			failures = append(failures, fmt.Errorf("refresh %s in %s: %w", target.name, target.shell, err))
			continue
		}
		updatedCount++
		fileops.ColorPrintfn(fileops.Green, "  ✓ Feature file updated")
	}
	if updatedCount > 0 {
		if err := autoCommitShellFeatureChanges(commitMessage); err != nil {
			failures = append(failures, fmt.Errorf("commit successful refreshes: %w", err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("refreshed %d feature installation(s); remaining refreshes failed: %w", updatedCount, errors.Join(failures...))
	}
	return nil
}
