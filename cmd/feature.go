package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

var featureCmd = &cobra.Command{
	Use:     "feature",
	Short:   "Manage shell features",
	Long:    `Add, remove, enable, disable, and list shell features`,
	GroupID: "dotfiles",
}

var featureAddCmd = &cobra.Command{
	Use:   "add [feature]",
	Short: "Add a shell feature",
	Long: `Add a shell feature to one or more shells.

Interactive mode (-i): Browse and select features from the catalog
Non-interactive: Specify feature name directly

The command will intelligently select which shell(s) to add the feature to based on:
- Feature compatibility (which shells support this feature)
- Your current shell
- Interactive prompts (if multiple options available)

Examples:
  omdot feature add -i                    # Browse catalog interactively
  omdot feature add git-prompt
  omdot feature add kubectl-completion --shell bash
  omdot feature add core-aliases --all`,
	Args: cobra.MaximumNArgs(1),
	RunE: runFeatureAdd,
}

var featureRemoveCmd = &cobra.Command{
	Use:   "remove [feature]",
	Short: "Remove a shell feature",
	Long: `Remove a shell feature from one or more shells.

If this is the last feature in a shell, the shell integration will be automatically
cleaned up (hooks removed from profile, directory deleted).

Interactive mode (-i): Browse and select features to remove
Non-interactive: Specify feature name directly

Examples:
  omdot feature remove -i                     # Browse features interactively
  omdot feature remove git-prompt
  omdot feature remove kubectl-completion --shell bash
  omdot feature remove core-aliases --all`,
	Args: cobra.MaximumNArgs(1),
	RunE: runFeatureRemove,
}

var featureListCmd = &cobra.Command{
	Use:   "list",
	Short: "List shell features",
	Long: `List all enabled shell features, optionally filtered by shell.

Examples:
  omdot feature list
  omdot feature list --shell bash`,
	Args: cobra.NoArgs,
	RunE: runFeatureList,
}

var featureEnableCmd = &cobra.Command{
	Use:   "enable <feature>",
	Short: "Enable a disabled feature",
	Long: `Enable a previously disabled feature without re-adding it.

Examples:
  omdot feature enable git-prompt
  omdot feature enable kubectl-completion --shell bash`,
	Args: cobra.ExactArgs(1),
	RunE: runFeatureEnable,
}

var featureDisableCmd = &cobra.Command{
	Use:   "disable <feature>",
	Short: "Disable a feature without removing it",
	Long: `Disable a feature without deleting its configuration file.
This allows you to temporarily turn off a feature.

Examples:
  omdot feature disable git-prompt
  omdot feature disable kubectl-completion --shell bash`,
	Args: cobra.ExactArgs(1),
	RunE: runFeatureDisable,
}

var featureInfoCmd = &cobra.Command{
	Use:   "info <feature>",
	Short: "Show detailed information about a feature",
	Long: `Display metadata about a feature from the catalog, including:
- Description
- Default load strategy
- Supported shells
- Current configuration (if installed)

Examples:
  omdot feature info git-prompt
  omdot feature info kubectl-completion`,
	Args: cobra.ExactArgs(1),
	RunE: runFeatureInfo,
}

var (
	flagShell       []string
	flagAll         bool
	flagStrategy    string
	flagOnCommand   []string
	flagDisabled    bool
	flagForce       bool
	flagInteractive bool
)

func init() {
	rootCmd.AddCommand(featureCmd)
	featureCmd.AddCommand(featureAddCmd)
	featureCmd.AddCommand(featureRemoveCmd)
	featureCmd.AddCommand(featureListCmd)
	featureCmd.AddCommand(featureEnableCmd)
	featureCmd.AddCommand(featureDisableCmd)
	featureCmd.AddCommand(featureInfoCmd)

	// Add flags
	featureAddCmd.Flags().BoolVarP(&flagInteractive, "interactive", "i", false, "Browse and select features from catalog")
	featureAddCmd.Flags().StringSliceVar(&flagShell, "shell", nil, "Target specific shell(s)")
	featureAddCmd.Flags().BoolVar(&flagAll, "all", false, "Add to all supported shells")
	featureAddCmd.Flags().StringVar(&flagStrategy, "strategy", "", "Override load strategy (eager, defer, on-command)")
	featureAddCmd.Flags().StringSliceVar(&flagOnCommand, "on-command", nil, "Set trigger commands for on-command strategy")
	featureAddCmd.Flags().BoolVar(&flagDisabled, "disabled", false, "Add feature but keep it disabled")

	featureRemoveCmd.Flags().StringSliceVar(&flagShell, "shell", nil, "Target specific shell(s)")
	featureRemoveCmd.Flags().BoolVar(&flagAll, "all", false, "Remove from all shells")
	featureRemoveCmd.Flags().BoolVar(&flagForce, "force", false, "Skip confirmation prompts")
	featureRemoveCmd.Flags().BoolVarP(&flagInteractive, "interactive", "i", false, "Browse and select features to remove")

	featureListCmd.Flags().StringSliceVar(&flagShell, "shell", nil, "Filter by specific shell(s)")

	featureEnableCmd.Flags().StringSliceVar(&flagShell, "shell", nil, "Target specific shell(s)")
	featureEnableCmd.Flags().StringVar(&flagStrategy, "strategy", "", "Override load strategy")
	featureEnableCmd.Flags().StringSliceVar(&flagOnCommand, "on-command", nil, "Set trigger commands")

	featureDisableCmd.Flags().StringSliceVar(&flagShell, "shell", nil, "Target specific shell(s)")
	featureDisableCmd.Flags().BoolVar(&flagAll, "all", false, "Disable in all shells")
}

func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// sortFeaturesByCategory sorts features by category first, then alphabetically by name
func sortFeaturesByCategory(features []catalog.FeatureMetadata) {
	// Define category order
	categoryOrder := map[string]int{
		"alias":      1,
		"completion": 2,
		"prompt":     3,
		"tool":       4,
	}

	sort.Slice(features, func(i, j int) bool {
		catI := categoryOrder[features[i].Category]
		catJ := categoryOrder[features[j].Category]

		// If category order not defined, put at end
		if catI == 0 {
			catI = 999
		}
		if catJ == 0 {
			catJ = 999
		}

		// Sort by category first
		if catI != catJ {
			return catI < catJ
		}

		// Within same category, sort alphabetically by name
		return features[i].Name < features[j].Name
	})
}

// isFeatureInstalled checks if a feature is installed in a specific shell
func isFeatureInstalled(repoPath, shellName, featureName string) bool {
	manifestPath := shell.GetManifestPath(repoPath, shellName)
	m, err := manifest.ParseManifest(manifestPath)
	if err != nil {
		return false
	}
	_, err = m.GetFeature(featureName)
	return err == nil
}

func runFeatureAdd(cmd *cobra.Command, args []string) error {
	repoPath := viper.GetString("repo-path")

	// Interactive mode: browse catalog
	if flagInteractive {
		return runInteractiveFeatureAdd(repoPath)
	}

	// Non-interactive mode: require feature name
	if len(args) == 0 {
		return fmt.Errorf("feature name required (or use -i for interactive mode)")
	}

	featureName := args[0]

	// Validate feature name
	if err := manifest.ValidateFeatureName(featureName); err != nil {
		return fmt.Errorf("invalid feature name: %w", err)
	}

	// Get feature metadata from catalog
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

	// Determine target shells
	var targetShells []string
	var err error

	if flagAll {
		targetShells = metadata.SupportedShells
	} else if len(flagShell) > 0 {
		targetShells = flagShell
	} else {
		// Smart shell selection
		targetShells, err = selectShells(metadata)
		if err != nil {
			return err
		}
	}

	// Validate all target shells are supported
	for _, shellName := range targetShells {
		if !shell.IsShellSupported(shellName) {
			return fmt.Errorf("unsupported shell: %s", shellName)
		}
		if !metadata.SupportsShell(shellName) && inCatalog {
			return fmt.Errorf("feature '%s' does not support shell '%s'", featureName, shellName)
		}
	}

	// Add feature to each target shell
	for _, shellName := range targetShells {
		fileops.ColorPrintfn(fileops.Cyan, "Adding %s to %s...", featureName, shellName)

		err := shell.AddFeatureToShell(repoPath, shellName, featureName, flagStrategy, flagOnCommand, flagDisabled)
		if err != nil {
			return fmt.Errorf("failed to add feature to %s: %w", shellName, err)
		}

		fileops.ColorPrintfn(fileops.Green, "  ✓ Feature added")
	}

	fmt.Println()
	fileops.ColorPrintln("Changes staged for commit.", fileops.Green)
	fileops.ColorPrintln("Run 'omdot apply' to activate the feature(s)", fileops.Cyan)
	fileops.ColorPrintln("Run 'omdot push' to commit and push changes", fileops.Cyan)

	return nil
}

func runInteractiveFeatureAdd(repoPath string) error {
	if !isInteractive() {
		return fmt.Errorf("cannot run interactive mode in non-interactive environment")
	}

	// Get all features from catalog
	allFeatures := catalog.ListFeatures()
	if len(allFeatures) == 0 {
		return fmt.Errorf("no features available in catalog")
	}

	// Sort features by category first, then alphabetically by name
	sortFeaturesByCategory(allFeatures)

	// Detect current shell first to pre-select it
	currentShell, _ := shell.DetectCurrentShell()

	// Collect all supported shells from all features
	shellSet := make(map[string]bool)
	for _, f := range allFeatures {
		for _, s := range f.SupportedShells {
			shellSet[s] = true
		}
	}

	// Convert to sorted list
	var availableShells []string
	for s := range shellSet {
		availableShells = append(availableShells, s)
	}

	// Prompt for shell selection FIRST
	selectedShells, err := interactive.MultiSelect(
		"Select shells to add features to:",
		availableShells,
		func(s string) bool { return s == currentShell },
	)
	if err != nil {
		return fmt.Errorf("shell selection cancelled: %w", err)
	}

	if len(selectedShells) == 0 {
		fileops.ColorPrintln("No shells selected", fileops.Yellow)
		return nil
	}

	// Build a map of already installed features across selected shells
	installedFeatures := make(map[string]bool)
	for _, shellName := range selectedShells {
		// Check all features in catalog against this shell
		for _, f := range allFeatures {
			if isFeatureInstalled(repoPath, shellName, f.Name) {
				installedFeatures[f.Name] = true
			}
		}
	}

	// Create feature options with descriptions
	type featureOption struct {
		feature catalog.FeatureMetadata
		label   string
	}

	options := make([]featureOption, len(allFeatures))
	optionLabels := make([]string, len(allFeatures))

	for i, f := range allFeatures {
		label := fmt.Sprintf("%s - %s [%s]", f.Name, f.Description, f.Category)
		options[i] = featureOption{feature: f, label: label}
		optionLabels[i] = label
	}

	// Prompt user to select features with already installed ones pre-selected
	selectedLabels, err := interactive.MultiSelect(
		"Select features to add:",
		optionLabels,
		func(label string) bool {
			// Extract feature name from label (before " - ")
			for _, opt := range options {
				if opt.label == label {
					return installedFeatures[opt.feature.Name]
				}
			}
			return false
		},
	)
	if err != nil {
		return fmt.Errorf("feature selection cancelled: %w", err)
	}

	if len(selectedLabels) == 0 {
		fileops.ColorPrintln("No features selected", fileops.Yellow)
		return nil
	}

	// Map selected labels back to features
	var selectedFeatures []catalog.FeatureMetadata
	for _, selectedLabel := range selectedLabels {
		for _, opt := range options {
			if opt.label == selectedLabel {
				selectedFeatures = append(selectedFeatures, opt.feature)
				break
			}
		}
	}

	// Add each feature to each selected shell (if supported)
	addedCount := 0
	skippedCount := 0

	for _, feature := range selectedFeatures {
		for _, shellName := range selectedShells {
			// Check if feature supports this shell
			if !feature.SupportsShell(shellName) {
				if len(selectedShells) == 1 || len(selectedFeatures) == 1 {
					fileops.ColorPrintfn(fileops.Yellow, "Skipping %s in %s (not supported)", feature.Name, shellName)
				}
				skippedCount++
				continue
			}

			// Check if feature is already installed in this shell
			if isFeatureInstalled(repoPath, shellName, feature.Name) {
				// Feature already installed, skip
				fileops.ColorPrintfn(fileops.Yellow, "Skipping %s in %s (already installed)", feature.Name, shellName)
				skippedCount++
				continue
			}

			fileops.ColorPrintfn(fileops.Cyan, "Adding %s to %s...", feature.Name, shellName)

			err = shell.AddFeatureToShell(repoPath, shellName, feature.Name, flagStrategy, flagOnCommand, flagDisabled)
			if err != nil {
				fileops.ColorPrintfn(fileops.Red, "  ✗ Failed: %s", err)
				skippedCount++
				continue
			}

			fileops.ColorPrintfn(fileops.Green, "  ✓ Feature added")
			addedCount++
		}
	}

	fmt.Println()
	if addedCount > 0 {
		fileops.ColorPrintfn(fileops.Green, "Successfully added %d feature(s)", addedCount)
	}
	if skippedCount > 0 {
		fileops.ColorPrintfn(fileops.Yellow, "Skipped %d feature(s)", skippedCount)
	}

	if addedCount > 0 {
		fmt.Println()
		fileops.ColorPrintln("Changes staged for commit.", fileops.Green)
		fileops.ColorPrintln("Run 'omdot apply' to activate the feature(s)", fileops.Cyan)
		fileops.ColorPrintln("Run 'omdot push' to commit and push changes", fileops.Cyan)
	}

	return nil
}

func selectShells(metadata catalog.FeatureMetadata) ([]string, error) {
	// Detect current shell
	currentShell, err := shell.DetectCurrentShell()
	if err != nil {
		currentShell = ""
	}

	// Filter supported shells
	supportedShells := metadata.SupportedShells
	if len(supportedShells) == 0 {
		supportedShells = []string{"bash", "zsh", "fish", "posix"}
	}

	// If feature only supports current shell, use it automatically
	if len(supportedShells) == 1 && supportedShells[0] == currentShell {
		return supportedShells, nil
	}

	// If current shell is the only supported option, use it
	currentShellSupported := false
	for _, s := range supportedShells {
		if s == currentShell {
			currentShellSupported = true
			break
		}
	}

	if currentShellSupported && len(supportedShells) > 1 {
		// Interactive mode: prompt user
		if !isInteractive() {
			return nil, fmt.Errorf("cannot prompt for shell selection in non-interactive mode\nPlease specify target shell(s):\n  --shell bash          Add to specific shell\n  --all                 Add to all supported shells\n\nExample: omdot feature add %s --shell bash", metadata.Name)
		}

		// Show interactive prompt
		return interactive.MultiSelect(
			fmt.Sprintf("Select shells to add feature '%s' to:", metadata.Name),
			supportedShells,
			func(s string) bool { return s == currentShell },
		)
	}

	// If current shell not supported, prompt with all options
	if !isInteractive() {
		return nil, fmt.Errorf("feature does not support current shell, cannot prompt in non-interactive mode\nPlease specify target shell(s) with --shell flag\n\nSupported shells: %v", supportedShells)
	}

	return interactive.MultiSelect(
		fmt.Sprintf("Select shells to add feature '%s' to:", metadata.Name),
		supportedShells,
		nil,
	)
}

func runFeatureRemove(cmd *cobra.Command, args []string) error {
	repoPath := viper.GetString("repo-path")

	// Interactive mode: browse features
	if flagInteractive {
		return runInteractiveFeatureRemove(repoPath)
	}

	// Non-interactive mode: require feature name
	if len(args) == 0 {
		return fmt.Errorf("feature name required (or use -i for interactive mode)\n\nExamples:\n  omdot feature remove git-prompt\n  omdot feature remove -i                    # Browse features interactively\n  omdot feature remove git-prompt --all      # Remove from all shells")
	}

	featureName := args[0]

	// Determine target shells
	var targetShells []string
	if flagAll || len(flagShell) > 0 {
		if flagAll {
			// Get all shells that have this feature
			allShells, err := shell.ListShellsWithFeatures(repoPath)
			if err != nil {
				return fmt.Errorf("failed to list shells: %w", err)
			}
			targetShells = allShells
		} else {
			targetShells = flagShell
		}
	} else {
		// Interactive selection
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

		// Filter shells that have this feature
		var shellsWithFeature []string
		for _, shellName := range allShells {
			manifestPath := shell.GetManifestPath(repoPath, shellName)
			m, err := manifest.ParseManifest(manifestPath)
			if err != nil {
				continue
			}
			if _, err := m.GetFeature(featureName); err == nil {
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

	// Remove feature from each shell
	for _, shellName := range targetShells {
		fileops.ColorPrintfn(fileops.Cyan, "Removing %s from %s...", featureName, shellName)

		err := shell.RemoveFeatureFromShell(repoPath, shellName, featureName)
		if err != nil {
			return fmt.Errorf("failed to remove feature from %s: %w", shellName, err)
		}

		fileops.ColorPrintfn(fileops.Green, "  ✓ Feature removed")

		// Check if cleanup is needed
		needsCleanup, err := shell.NeedsCleanup(repoPath, shellName)
		if err != nil {
			return fmt.Errorf("failed to check if cleanup needed: %w", err)
		}

		if needsCleanup {
			fileops.ColorPrintfn(fileops.Yellow, "\nNo features remaining in %s shell", shellName)

			shouldCleanup := flagForce
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

	fmt.Println()
	fileops.ColorPrintln("Changes staged for commit.", fileops.Green)
	fileops.ColorPrintln("Run 'omdot apply' to sync changes", fileops.Cyan)

	return nil
}

func runInteractiveFeatureRemove(repoPath string) error {
	if !isInteractive() {
		return fmt.Errorf("cannot run interactive mode in non-interactive environment")
	}

	// Get all shells with features
	allShells, err := shell.ListShellsWithFeatures(repoPath)
	if err != nil {
		return fmt.Errorf("failed to list shells: %w", err)
	}

	if len(allShells) == 0 {
		fileops.ColorPrintln("No shell features configured", fileops.Yellow)
		return nil
	}

	// Collect all features across all shells
	type featureInfo struct {
		name   string
		shells []string
	}
	featureMap := make(map[string][]string)

	for _, shellName := range allShells {
		manifestPath := shell.GetManifestPath(repoPath, shellName)
		localManifestPath := shell.GetLocalManifestPath(repoPath, shellName)
		merged, err := manifest.ParseManifestWithLocal(manifestPath, localManifestPath)
		if err != nil {
			continue
		}

		for _, f := range merged.Features {
			featureMap[f.Name] = append(featureMap[f.Name], shellName)
		}
	}

	if len(featureMap) == 0 {
		fileops.ColorPrintln("No features found", fileops.Yellow)
		return nil
	}

	// Build feature list with shell info
	var features []featureInfo
	var featureLabels []string
	for name, shells := range featureMap {
		features = append(features, featureInfo{name: name, shells: shells})
		label := fmt.Sprintf("%s (in %s)", name, strings.Join(shells, ", "))
		featureLabels = append(featureLabels, label)
	}

	// Sort alphabetically
	sort.Slice(features, func(i, j int) bool {
		return features[i].name < features[j].name
	})
	sort.Strings(featureLabels)

	// Prompt user to select features to remove
	selectedLabels, err := interactive.MultiSelect(
		"Select features to remove:",
		featureLabels,
		nil,
	)
	if err != nil {
		return fmt.Errorf("feature selection cancelled: %w", err)
	}

	if len(selectedLabels) == 0 {
		fileops.ColorPrintln("No features selected", fileops.Yellow)
		return nil
	}

	// Map selected labels back to feature names
	var selectedFeatures []string
	for _, label := range selectedLabels {
		// Extract feature name from label (before " (in ")
		name := strings.Split(label, " (in ")[0]
		selectedFeatures = append(selectedFeatures, name)
	}

	// For each selected feature, determine which shells to remove from
	removedCount := 0
	for _, featureName := range selectedFeatures {
		shellsWithFeature := featureMap[featureName]

		var targetShells []string
		if len(shellsWithFeature) == 1 {
			// Only one shell, remove from it
			targetShells = shellsWithFeature
		} else {
			// Multiple shells, prompt which to remove from
			targetShells, err = interactive.MultiSelect(
				fmt.Sprintf("Remove '%s' from which shells?", featureName),
				shellsWithFeature,
				nil,
			)
			if err != nil {
				fileops.ColorPrintfn(fileops.Yellow, "Skipping %s", featureName)
				continue
			}

			if len(targetShells) == 0 {
				fileops.ColorPrintfn(fileops.Yellow, "Skipping %s (no shells selected)", featureName)
				continue
			}
		}

		// Remove from selected shells
		for _, shellName := range targetShells {
			fileops.ColorPrintfn(fileops.Cyan, "Removing %s from %s...", featureName, shellName)

			err := shell.RemoveFeatureFromShell(repoPath, shellName, featureName)
			if err != nil {
				fileops.ColorPrintfn(fileops.Red, "  ✗ Failed: %s", err)
				continue
			}

			fileops.ColorPrintfn(fileops.Green, "  ✓ Feature removed")
			removedCount++

			// Check if cleanup is needed
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
		fileops.ColorPrintfn(fileops.Green, "Successfully removed %d feature(s)", removedCount)
		fmt.Println()
		fileops.ColorPrintln("Changes staged for commit.", fileops.Green)
		fileops.ColorPrintln("Run 'omdot apply' to sync changes", fileops.Cyan)
	} else {
		fileops.ColorPrintln("No features were removed", fileops.Yellow)
	}

	return nil
}

func runFeatureList(cmd *cobra.Command, args []string) error {
	repoPath := viper.GetString("repo-path")

	// Get shells to list
	var targetShells []string
	if len(flagShell) > 0 {
		targetShells = flagShell
	} else {
		var err error
		targetShells, err = shell.ListShellsWithFeatures(repoPath)
		if err != nil {
			return fmt.Errorf("failed to list shells: %w", err)
		}
	}

	if len(targetShells) == 0 {
		fileops.ColorPrintln("No shell features configured", fileops.Yellow)
		fileops.ColorPrintln("Use 'omdot feature add <feature>' to add features", fileops.Cyan)
		fileops.ColorPrintln("Use 'omdot feature add -i' to browse the catalog", fileops.Cyan)
		return nil
	}

	// List features for each shell
	for _, shellName := range targetShells {
		manifestPath := shell.GetManifestPath(repoPath, shellName)
		localManifestPath := shell.GetLocalManifestPath(repoPath, shellName)

		// Use merged manifest to show local overrides
		merged, err := manifest.ParseManifestWithLocal(manifestPath, localManifestPath)
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error reading %s manifest: %v", shellName, err)
			continue
		}

		fileops.ColorPrintfn(fileops.Cyan, "\n%s:", shellName)

		if len(merged.Features) == 0 {
			fileops.ColorPrintln("  (no features)", fileops.Yellow)
			continue
		}

		for _, featureWithOverride := range merged.Features {
			feature := featureWithOverride.FeatureConfig
			override := featureWithOverride.Override

			status := "✓"
			statusColor := fileops.Green
			if feature.Disabled {
				status = "✗"
				statusColor = fileops.Red
			}

			strategy := feature.Strategy
			if strategy == "" {
				if metadata, ok := catalog.GetFeature(feature.Name); ok {
					strategy = metadata.DefaultStrategy
				} else {
					strategy = "eager"
				}
			}

			// Build feature name with override indicator
			featureName := feature.Name
			if override.IsFromLocal {
				featureName += " (local)"
			} else if override.IsOverridden {
				featureName += " (overridden)"
			}

			fileops.ColorPrintf(statusColor, "  %s %s", status, featureName)
			fmt.Printf(" (%s", strategy)

			if len(feature.OnCommand) > 0 {
				fmt.Printf(": %v", feature.OnCommand)
			}

			fmt.Println(")")

			// Show file path
			featurePath, err := shell.GetFeatureFilePath(repoPath, shellName, feature.Name)
			if err == nil {
				// Expand home directory for display
				homeDir, _ := os.UserHomeDir()
				if homeDir != "" && strings.HasPrefix(featurePath, homeDir) {
					featurePath = "~" + strings.TrimPrefix(featurePath, homeDir)
				}
				fileops.ColorPrintf(fileops.Reset, "    %s\n", featurePath)
			}
		}
	}

	fmt.Println()
	return nil
}

func runFeatureEnable(cmd *cobra.Command, args []string) error {
	featureName := args[0]
	repoPath := viper.GetString("repo-path")

	targetShells := flagShell
	if len(targetShells) == 0 {
		// Use current shell
		currentShell, err := shell.DetectCurrentShell()
		if err != nil {
			return fmt.Errorf("could not detect current shell, please specify with --shell flag")
		}
		targetShells = []string{currentShell}
	}

	for _, shellName := range targetShells {
		// Use EnableFeatureWithOptions if strategy or onCommand flags are provided
		if flagStrategy != "" || len(flagOnCommand) > 0 {
			if err := shell.EnableFeatureWithOptions(repoPath, shellName, featureName, flagStrategy, flagOnCommand); err != nil {
				return fmt.Errorf("failed to enable feature in %s: %w", shellName, err)
			}
		} else {
			if err := shell.EnableFeature(repoPath, shellName, featureName); err != nil {
				return fmt.Errorf("failed to enable feature in %s: %w", shellName, err)
			}
		}
		fileops.ColorPrintfn(fileops.Green, "Enabled %s in %s", featureName, shellName)
	}

	fileops.ColorPrintln("\nRun 'omdot apply' to activate changes", fileops.Cyan)
	return nil
}

func runFeatureDisable(cmd *cobra.Command, args []string) error {
	featureName := args[0]
	repoPath := viper.GetString("repo-path")

	var targetShells []string
	if flagAll {
		var err error
		targetShells, err = shell.ListShellsWithFeatures(repoPath)
		if err != nil {
			return fmt.Errorf("failed to list shells: %w", err)
		}
	} else if len(flagShell) > 0 {
		targetShells = flagShell
	} else {
		// Use current shell
		currentShell, err := shell.DetectCurrentShell()
		if err != nil {
			return fmt.Errorf("could not detect current shell, please specify with --shell flag")
		}
		targetShells = []string{currentShell}
	}

	for _, shellName := range targetShells {
		if err := shell.DisableFeature(repoPath, shellName, featureName); err != nil {
			return fmt.Errorf("failed to disable feature in %s: %w", shellName, err)
		}
		fileops.ColorPrintfn(fileops.Yellow, "Disabled %s in %s", featureName, shellName)
	}

	fileops.ColorPrintln("\nRun 'omdot apply' to sync changes", fileops.Cyan)
	return nil
}

func runFeatureInfo(cmd *cobra.Command, args []string) error {
	featureName := args[0]
	repoPath := viper.GetString("repo-path")

	// Get metadata from catalog
	metadata, inCatalog := catalog.GetFeature(featureName)
	if !inCatalog {
		return fmt.Errorf("feature '%s' not found in catalog", featureName)
	}

	// Display metadata
	fileops.ColorPrintfn(fileops.Cyan, "\n%s", metadata.Name)
	fmt.Printf("  Category: %s\n", metadata.Category)
	fmt.Printf("  Description: %s\n", metadata.Description)
	fmt.Printf("  Default Strategy: %s\n", metadata.DefaultStrategy)

	if len(metadata.DefaultCommands) > 0 {
		fmt.Printf("  Default Commands: %v\n", metadata.DefaultCommands)
	}

	fmt.Printf("  Supported Shells: %v\n", metadata.SupportedShells)

	// Check current configuration
	fmt.Println("\nCurrent Configuration:")

	allShells, err := shell.ListShellsWithFeatures(repoPath)
	if err != nil {
		return fmt.Errorf("failed to list shells: %w", err)
	}

	configured := false
	for _, shellName := range allShells {
		manifestPath := shell.GetManifestPath(repoPath, shellName)
		m, err := manifest.ParseManifest(manifestPath)
		if err != nil {
			continue
		}

		feature, err := m.GetFeature(featureName)
		if err == nil {
			configured = true
			status := "enabled"
			if feature.Disabled {
				status = "disabled"
			}

			fileops.ColorPrintf(fileops.Cyan, "  %s: ", shellName)
			fmt.Print(status)

			if feature.Strategy != "" {
				fmt.Printf(" (%s", feature.Strategy)
				if len(feature.OnCommand) > 0 {
					fmt.Printf(": %v", feature.OnCommand)
				}
				fmt.Print(")")
			}

			fmt.Println()
		}
	}

	if !configured {
		fileops.ColorPrintln("  (not installed)", fileops.Yellow)
	}

	fmt.Println()
	return nil
}
