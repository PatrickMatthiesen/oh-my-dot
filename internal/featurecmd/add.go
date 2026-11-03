package featurecmd

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/options"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
)

func (state *commandState) runFeatureAdd(cmd *cobra.Command, args []string) error {
	repoPath := state.repoPathProvider()
	alias := state.aliasProvider()

	if state.flagInteractive {
		return state.runInteractiveFeatureAdd(repoPath)
	}

	if len(args) == 0 {
		return fmt.Errorf("feature name required (or use -i for interactive mode)")
	}

	featureName := args[0]
	if err := manifest.ValidateFeatureName(featureName); err != nil {
		return fmt.Errorf("invalid feature name: %w", err)
	}

	metadata, inCatalog := catalog.GetFeature(featureName)
	if !inCatalog {
		fileops.ColorPrintfn(fileops.Yellow, "Warning: feature '%s' not found in catalog, using defaults", featureName)
		metadata = catalog.FeatureMetadata{
			Name:            featureName,
			Description:     "Custom feature",
			DefaultStrategy: "eager",
			SupportedShells: []string{"bash", "zsh", "fish", "posix"},
		}
	}

	var targetShells []string
	var err error

	if state.flagAll {
		targetShells = metadata.SupportedShells
	} else if len(state.flagShell) > 0 {
		targetShells = state.flagShell
	} else {
		targetShells, err = state.selectShells(metadata)
		if err != nil {
			return err
		}
	}

	for _, shellName := range targetShells {
		if !shell.IsShellSupported(shellName) {
			return fmt.Errorf("unsupported shell: %s", shellName)
		}
		if !metadata.SupportsShell(shellName) && inCatalog {
			return fmt.Errorf("feature '%s' does not support shell '%s'", featureName, shellName)
		}
	}

	var optionValues map[string]any
	if inCatalog && len(metadata.Options) > 0 {
		overrides, err := options.ParseOptionOverrides(metadata, state.flagOption)
		if err != nil {
			return fmt.Errorf("failed to parse --option values: %w", err)
		}

		if isInteractive() {
			optionValues, err = options.PromptForOptionsWithOverrides(metadata, overrides)
			if errors.Is(err, interactive.ErrSkip) {
				fileops.ColorPrintfn(fileops.Yellow, "Skipped %s", featureName)
				return nil
			}
			if errors.Is(err, interactive.ErrCancelled) {
				fileops.ColorPrintln("Cancelled", fileops.Yellow)
				return nil
			}
			if err != nil {
				return fmt.Errorf("failed to collect feature options: %w", err)
			}
		} else {
			optionValues, err = options.ResolveOptionsForNonInteractiveWithOverrides(metadata, overrides)
			if err != nil {
				return fmt.Errorf("failed to resolve feature options in non-interactive mode: %w", err)
			}

			if len(optionValues) > 0 {
				fileops.ColorPrintln("Using default values for feature options (non-interactive mode)", fileops.Cyan)
			}
		}
	} else {
		optionValues, err = parseRawOptionPairs(state.flagOption)
		if err != nil {
			return fmt.Errorf("failed to parse --option values: %w", err)
		}
	}

	for _, shellName := range targetShells {
		fileops.ColorPrintfn(fileops.Cyan, "Adding %s to %s...", featureName, shellName)

		err := shell.AddFeatureToShell(repoPath, shellName, featureName, state.flagStrategy, state.flagOnCommand, state.flagDisabled, optionValues)
		if err != nil {
			return fmt.Errorf("failed to add feature to %s: %w", shellName, err)
		}

		fileops.ColorPrintfn(fileops.Green, "  ✓ Feature added")
	}

	if err := autoCommitShellFeatureChanges("Add shell feature: " + featureName); err != nil {
		return fmt.Errorf("failed to commit shell feature changes: %w", err)
	}

	fmt.Println()
	fileops.ColorPrintfn(fileops.Cyan, "Run '%s apply' to activate the feature(s)", alias)
	fileops.ColorPrintfn(fileops.Cyan, "Run '%s push' to push committed changes", alias)

	return nil
}

func (state *commandState) runInteractiveFeatureAdd(repoPath string) error {
	if !isInteractive() {
		return fmt.Errorf("cannot run interactive mode in non-interactive environment")
	}

	allFeatures := catalog.ListFeatures()
	if len(allFeatures) == 0 {
		return fmt.Errorf("no features available in catalog")
	}

	sortFeaturesByCategory(allFeatures)

	currentShell, _ := shell.DetectCurrentShell()

	shellSet := make(map[string]bool)
	for _, feature := range allFeatures {
		for _, shellName := range feature.SupportedShells {
			shellSet[shellName] = true
		}
	}

	var availableShells []string
	if currentShell != "" && shellSet[currentShell] {
		availableShells = append(availableShells, currentShell)
	}

	var remainingShells []string
	for shellName := range shellSet {
		if shellName != currentShell {
			remainingShells = append(remainingShells, shellName)
		}
	}
	sort.Strings(remainingShells)
	availableShells = append(availableShells, remainingShells...)

	selectedShells, err := interactive.MultiSelect(
		"Select shells to add features to:",
		availableShells,
		func(shellName string) bool { return shellName == currentShell },
	)
	if err != nil {
		if errors.Is(err, interactive.ErrCancelled) {
			fileops.ColorPrintln("Cancelled", fileops.Yellow)
			return nil
		}
		return fmt.Errorf("shell selection cancelled: %w", err)
	}

	if len(selectedShells) == 0 {
		fileops.ColorPrintln("No shells selected", fileops.Yellow)
		return nil
	}

	compatibleFeatures := filterFeaturesByShells(allFeatures, selectedShells)
	if len(compatibleFeatures) == 0 {
		fileops.ColorPrintln("No features available for selected shells", fileops.Yellow)
		return nil
	}

	installedFeaturesByShell := make(map[string]map[string]bool, len(selectedShells))
	for _, shellName := range selectedShells {
		manifestPath := shell.GetManifestPath(repoPath, shellName)
		installedForShell := make(map[string]bool)
		installedFeaturesByShell[shellName] = installedForShell
		m, err := manifest.ParseManifest(manifestPath)
		if err != nil {
			continue
		}

		for _, feature := range m.Features {
			installedForShell[feature.Name] = true
		}
	}

	optionLabels := make([]string, len(compatibleFeatures))
	labelToFeatureName := make(map[string]string, len(compatibleFeatures))
	for i, feature := range compatibleFeatures {
		label := featureInstallLabel(feature, selectedShells, installedFeaturesByShell)
		optionLabels[i] = label
		labelToFeatureName[label] = feature.Name
	}
	selectedLabels, err := interactive.MultiSelect("Select features to add (installed features are marked):", optionLabels, nil)
	if err != nil {
		if errors.Is(err, interactive.ErrCancelled) {
			fileops.ColorPrintln("Cancelled", fileops.Yellow)
			return nil
		}
		return fmt.Errorf("feature selection cancelled: %w", err)
	}

	if len(selectedLabels) == 0 {
		fileops.ColorPrintln("No features selected", fileops.Yellow)
		return nil
	}

	var selectedFeatures []catalog.FeatureMetadata
	for _, selectedLabel := range selectedLabels {
		featureName, exists := labelToFeatureName[selectedLabel]
		if !exists {
			fileops.ColorPrintfn(fileops.Yellow, "Warning: Could not find feature for label: %s", selectedLabel)
			continue
		}
		if metadata, found := catalog.GetFeature(featureName); found {
			selectedFeatures = append(selectedFeatures, metadata)
		} else {
			fileops.ColorPrintfn(fileops.Yellow, "Warning: Feature %s not found in catalog", featureName)
		}
	}

	return state.addSelectedFeatures(repoPath, selectedShells, selectedFeatures, installedFeaturesByShell, options.PromptForOptions)
}

func featureInstallLabel(feature catalog.FeatureMetadata, selectedShells []string, installed map[string]map[string]bool) string {
	label := fmt.Sprintf("%s - %s [%s]", feature.Name, feature.Description, feature.Category)
	var installedShells []string
	for _, shellName := range selectedShells {
		if feature.SupportsShell(shellName) && installed[shellName][feature.Name] {
			installedShells = append(installedShells, shellName)
		}
	}
	if len(installedShells) > 0 {
		label += " (installed: " + strings.Join(installedShells, ", ") + ")"
	}
	return label
}

func (state *commandState) addSelectedFeatures(repoPath string, selectedShells []string, selectedFeatures []catalog.FeatureMetadata, installedFeaturesByShell map[string]map[string]bool, promptOptions func(catalog.FeatureMetadata) (map[string]any, error)) error {
	alias := state.aliasProvider()
	addedCount := 0
	skippedCount := 0
	failedCount := 0
	cancelled := false
	remainingCount := 0

	for index, feature := range selectedFeatures {
		if !hasPendingFeatureInstall(feature, selectedShells, installedFeaturesByShell) {
			fileops.ColorPrintfn(fileops.Yellow, "Skipping %s (already installed in selected shell(s))", feature.Name)
			skippedCount++
			continue
		}

		var optionValues map[string]any
		if len(feature.Options) > 0 {
			fileops.ColorPrintfn(fileops.Cyan, "\nConfiguring %s:", feature.Name)
			var err error
			optionValues, err = promptOptions(feature)
			if errors.Is(err, interactive.ErrCancelled) {
				cancelled = true
				remainingCount = len(selectedFeatures) - index
				break
			}
			if errors.Is(err, interactive.ErrSkip) {
				fileops.ColorPrintfn(fileops.Yellow, "Skipped %s", feature.Name)
				skippedCount++
				continue
			}
			if err != nil {
				fileops.ColorPrintfn(fileops.Red, "  ✗ Failed to collect options: %s", err)
				failedCount++
				continue
			}
		} else {
			optionValues = make(map[string]any)
		}

		for _, shellName := range selectedShells {
			if !feature.SupportsShell(shellName) {
				if len(selectedShells) == 1 || len(selectedFeatures) == 1 {
					fileops.ColorPrintfn(fileops.Yellow, "Skipping %s in %s (not supported)", feature.Name, shellName)
				}
				skippedCount++
				continue
			}

			if installedFeaturesByShell[shellName][feature.Name] {
				fileops.ColorPrintfn(fileops.Yellow, "Skipping %s in %s (already installed)", feature.Name, shellName)
				skippedCount++
				continue
			}

			fileops.ColorPrintfn(fileops.Cyan, "Adding %s to %s...", feature.Name, shellName)

			err := shell.AddFeatureToShell(repoPath, shellName, feature.Name, state.flagStrategy, state.flagOnCommand, state.flagDisabled, optionValues)
			if err != nil {
				fileops.ColorPrintfn(fileops.Red, "  ✗ Failed: %s", err)
				failedCount++
				continue
			}

			fileops.ColorPrintfn(fileops.Green, "  ✓ Feature added")
			installedFeaturesByShell[shellName][feature.Name] = true
			addedCount++
		}
	}

	fmt.Println()
	if cancelled {
		fileops.ColorPrintfn(fileops.Yellow, "Cancelled. %d selected feature(s) were not added. Previously added features are kept.", remainingCount)
	}
	if failedCount > 0 {
		fileops.ColorPrintfn(fileops.Red, "Failed to add %d feature(s)", failedCount)
	}
	if addedCount > 0 {
		fileops.ColorPrintfn(fileops.Green, "Successfully added %d feature(s)", addedCount)
	}
	if skippedCount > 0 {
		fileops.ColorPrintfn(fileops.Yellow, "Skipped %d feature(s)", skippedCount)
	}

	if addedCount > 0 {
		if err := autoCommitShellFeatureChanges("Add shell features (interactive)"); err != nil {
			return fmt.Errorf("failed to commit shell feature changes: %w", err)
		}

		fmt.Println()
		fileops.ColorPrintfn(fileops.Cyan, "Run '%s apply' to activate the feature(s)", alias)
		fileops.ColorPrintfn(fileops.Cyan, "Run '%s push' to push committed changes", alias)
	}

	if failedCount > 0 {
		return fmt.Errorf("failed to add %d feature(s); successfully added features were kept", failedCount)
	}
	return nil
}

func (state *commandState) selectShells(metadata catalog.FeatureMetadata) ([]string, error) {
	currentShell, err := shell.DetectCurrentShell()
	if err != nil {
		currentShell = ""
	}

	supportedShells := metadata.SupportedShells
	if len(supportedShells) == 0 {
		supportedShells = []string{"bash", "zsh", "fish", "posix"}
	}

	if len(supportedShells) == 1 && supportedShells[0] == currentShell {
		return supportedShells, nil
	}

	currentShellSupported := false
	for _, shellName := range supportedShells {
		if shellName == currentShell {
			currentShellSupported = true
			break
		}
	}

	if currentShellSupported && len(supportedShells) > 1 {
		if !isInteractive() {
			return nil, fmt.Errorf(
				"cannot prompt for shell selection in non-interactive mode\nPlease specify target shell(s):\n  --shell bash          Add to specific shell\n  --all                 Add to all supported shells\n\nExample: %s feature add %s --shell bash",
				state.aliasProvider(),
				metadata.Name,
			)
		}

		return interactive.MultiSelect(
			fmt.Sprintf("Select shells to add feature '%s' to:", metadata.Name),
			supportedShells,
			func(shellName string) bool { return shellName == currentShell },
		)
	}

	if !isInteractive() {
		return nil, fmt.Errorf("feature does not support current shell, cannot prompt in non-interactive mode\nPlease specify target shell(s) with --shell flag\n\nSupported shells: %v", supportedShells)
	}

	return interactive.MultiSelect(
		fmt.Sprintf("Select shells to add feature '%s' to:", metadata.Name),
		supportedShells,
		nil,
	)
}
