package featurecmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
	"github.com/spf13/cobra"
)

func (state *commandState) runFeatureList(cmd *cobra.Command, args []string) error {
	repoPath := state.repoPathProvider()
	alias := state.aliasProvider()

	var targetShells []string
	if len(state.flagShell) > 0 {
		targetShells = state.flagShell
	} else {
		var err error
		targetShells, err = shell.ListShellsWithFeatures(repoPath)
		if err != nil {
			return fmt.Errorf("failed to list shells: %w", err)
		}
	}

	if len(targetShells) == 0 {
		fileops.ColorPrintln("No shell features configured", fileops.Yellow)
		fileops.ColorPrintfn(fileops.Cyan, "Use '%s feature add <feature>' to add features", alias)
		fileops.ColorPrintfn(fileops.Cyan, "Use '%s feature add -i' to browse the catalog", alias)
		return nil
	}

	for _, shellName := range targetShells {
		manifestPath := shell.GetManifestPath(repoPath, shellName)
		localManifestPath := shell.GetLocalManifestPath(repoPath, shellName)

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

			featurePath, err := shell.GetFeatureFilePath(repoPath, shellName, feature.Name)
			if err == nil {
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

func (state *commandState) runFeatureEnable(cmd *cobra.Command, args []string) error {
	featureName := args[0]
	repoPath := state.repoPathProvider()

	targetShells := state.flagShell
	if len(targetShells) == 0 {
		currentShell, err := shell.DetectCurrentShell()
		if err != nil {
			return fmt.Errorf("could not detect current shell, please specify with --shell flag")
		}
		targetShells = []string{currentShell}
	}

	for _, shellName := range targetShells {
		if state.flagStrategy != "" || len(state.flagOnCommand) > 0 {
			if err := shell.EnableFeatureWithOptions(repoPath, shellName, featureName, state.flagStrategy, state.flagOnCommand); err != nil {
				return fmt.Errorf("failed to enable feature in %s: %w", shellName, err)
			}
		} else {
			if err := shell.EnableFeature(repoPath, shellName, featureName); err != nil {
				return fmt.Errorf("failed to enable feature in %s: %w", shellName, err)
			}
		}
		fileops.ColorPrintfn(fileops.Green, "Enabled %s in %s", featureName, shellName)
	}

	if err := autoCommitShellFeatureChanges("Enable shell feature: " + featureName); err != nil {
		return fmt.Errorf("failed to commit shell feature changes: %w", err)
	}

	fileops.ColorPrintfn(fileops.Cyan, "\nRun '%s apply' to activate changes", state.aliasProvider())
	fileops.ColorPrintfn(fileops.Cyan, "Run '%s push' to push committed changes", state.aliasProvider())
	return nil
}

func (state *commandState) runFeatureDisable(cmd *cobra.Command, args []string) error {
	featureName := args[0]
	repoPath := state.repoPathProvider()

	var targetShells []string
	if state.flagAll {
		var err error
		targetShells, err = shell.ListShellsWithFeatures(repoPath)
		if err != nil {
			return fmt.Errorf("failed to list shells: %w", err)
		}
	} else if len(state.flagShell) > 0 {
		targetShells = state.flagShell
	} else {
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

	if err := autoCommitShellFeatureChanges("Disable shell feature: " + featureName); err != nil {
		return fmt.Errorf("failed to commit shell feature changes: %w", err)
	}

	fileops.ColorPrintfn(fileops.Cyan, "\nRun '%s apply' to sync changes", state.aliasProvider())
	fileops.ColorPrintfn(fileops.Cyan, "Run '%s push' to push committed changes", state.aliasProvider())
	return nil
}

func (state *commandState) runFeatureInfo(cmd *cobra.Command, args []string) error {
	featureName := args[0]
	repoPath := state.repoPathProvider()

	metadata, inCatalog := catalog.GetFeature(featureName)
	if !inCatalog {
		return fmt.Errorf("feature '%s' not found in catalog", featureName)
	}

	fileops.ColorPrintfn(fileops.Cyan, "\n%s", metadata.Name)
	fmt.Printf("  Category: %s\n", metadata.Category)
	fmt.Printf("  Description: %s\n", metadata.Description)
	fmt.Printf("  Default Strategy: %s\n", metadata.DefaultStrategy)

	if len(metadata.DefaultCommands) > 0 {
		fmt.Printf("  Default Commands: %v\n", metadata.DefaultCommands)
	}

	fmt.Printf("  Supported Shells: %v\n", metadata.SupportedShells)
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
