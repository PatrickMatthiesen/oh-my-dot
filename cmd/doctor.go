package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/hooks"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check shell framework health",
	Long: `Diagnose and validate shell framework configuration.

Checks performed:
  - Shell hooks are properly installed in profile files
  - Directory structure is correct
  - Manifest files are valid
  - Feature files exist
  - Local override security (permissions, ownership)
  - Init script syntax

Examples:
  oh-my-dot doctor              # Check all shells
  oh-my-dot doctor --shell bash # Check specific shell only`,
	GroupID: "dotfiles",
	RunE:    runDoctor,
}

var (
	flagDoctorShell []string
	flagFix         bool
)

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().StringSliceVar(&flagDoctorShell, "shell", nil, "Check specific shell(s) only")
	doctorCmd.Flags().BoolVar(&flagFix, "fix", false, "Attempt to fix issues automatically")
}

type checkResult struct {
	name    string
	status  string // "ok", "warning", "error"
	message string
	fixable bool
}

func runDoctor(cmd *cobra.Command, args []string) error {
	repoPath := viper.GetString("repo-path")
	alias := assumedAlias()

	fileops.ColorPrintln("\noh-my-dot Shell Framework Doctor", fileops.Cyan)
	fileops.ColorPrintln("=================================\n", fileops.Cyan)

	var allResults []checkResult
	overallStatus := "ok"

	// Get shells to check
	var shellsToCheck []string
	if len(flagDoctorShell) > 0 {
		shellsToCheck = flagDoctorShell
	} else {
		var err error
		shellsToCheck, err = shell.ListShellsWithFeatures(repoPath)
		if err != nil {
			return fmt.Errorf("failed to list shells: %w", err)
		}
	}

	if len(shellsToCheck) == 0 {
		fileops.ColorPrintln("No shell features configured", fileops.Yellow)
		fileops.ColorPrintfn(fileops.Cyan, "Run '%s feature add' to add features", alias)
		return nil
	}

	// Check each shell
	for _, shellName := range shellsToCheck {
		fileops.ColorPrintfn(fileops.Cyan, "\nChecking %s shell...", shellName)

		results := checkShell(repoPath, shellName)
		allResults = append(allResults, results...)

		// Update overall status
		for _, r := range results {
			if r.status == "error" {
				overallStatus = "error"
			} else if r.status == "warning" && overallStatus != "error" {
				overallStatus = "warning"
			}
		}
	}

	// Print summary
	fmt.Println()
	fileops.ColorPrintln("Summary", fileops.Cyan)
	fileops.ColorPrintln("-------", fileops.Cyan)

	okCount := 0
	warningCount := 0
	errorCount := 0
	fixableErrorCount := 0
	fixableWarningCount := 0

	for _, r := range allResults {
		switch r.status {
		case "ok":
			okCount++
		case "warning":
			warningCount++
			if r.fixable {
				fixableWarningCount++
			}
		case "error":
			errorCount++
			if r.fixable {
				fixableErrorCount++
			}
		}
	}

	fileops.ColorPrintfn(fileops.Green, "✓ %d checks passed", okCount)
	if warningCount > 0 {
		fileops.ColorPrintfn(fileops.Yellow, "⚠ %d warnings", warningCount)
	}
	if errorCount > 0 {
		fileops.ColorPrintfn(fileops.Red, "✗ %d errors", errorCount)
	}

	// Show fix tips
	totalFixable := fixableErrorCount + fixableWarningCount
	if totalFixable > 0 && !flagFix {
		fmt.Println()
		if fixableErrorCount > 0 && fixableWarningCount > 0 {
			fileops.ColorPrintfn(fileops.Cyan, "Tip: Run '%s doctor --fix' to automatically fix %d error(s) and %d warning(s)", alias, fixableErrorCount, fixableWarningCount)
		} else if fixableErrorCount > 0 {
			fileops.ColorPrintfn(fileops.Cyan, "Tip: Run '%s doctor --fix' to automatically fix %d error(s)", alias, fixableErrorCount)
		} else {
			fileops.ColorPrintfn(fileops.Cyan, "Tip: Run '%s doctor --fix' to automatically fix %d warning(s)", alias, fixableWarningCount)
		}
	} else if warningCount > 0 && fixableWarningCount == 0 {
		// Warnings that can't be fixed - explain they're optional
		fmt.Println()
		fileops.ColorPrintln("Note: Warnings are optional issues that don't affect functionality", fileops.Yellow)
	}

	fmt.Println()

	// Return error if there were errors
	if overallStatus == "error" {
		return fmt.Errorf("health check failed")
	}

	fileops.ColorPrintln("All checks passed! ✓", fileops.Green)
	return nil
}

func checkShell(repoPath, shellName string) []checkResult {
	var results []checkResult

	// 1. Check directory structure
	results = append(results, checkDirectoryStructure(repoPath, shellName)...)

	// 2. Check manifest validity
	results = append(results, checkManifest(repoPath, shellName)...)

	// 3. Check local override security
	results = append(results, checkLocalOverride(repoPath, shellName)...)

	// 4. Check feature files exist
	results = append(results, checkFeatureFiles(repoPath, shellName)...)

	// 5. Check hooks in profile
	results = append(results, checkProfileHooks(repoPath, shellName)...)

	// 6. Check init script syntax
	results = append(results, checkInitScriptSyntax(repoPath, shellName)...)

	return results
}

func checkDirectoryStructure(repoPath, shellName string) []checkResult {
	var results []checkResult

	shellDir := filepath.Join(repoPath, "omd-shells", shellName)

	// Check shell directory exists
	if _, err := os.Stat(shellDir); os.IsNotExist(err) {
		results = append(results, checkResult{
			name:    "Shell directory",
			status:  "error",
			message: fmt.Sprintf("Directory missing: %s", shellDir),
			fixable: false,
		})
		return results
	}

	printCheck("Shell directory", "ok", "")
	results = append(results, checkResult{
		name:   "Shell directory",
		status: "ok",
	})

	// Check features directory exists
	featuresDir := filepath.Join(shellDir, "features")
	if _, err := os.Stat(featuresDir); os.IsNotExist(err) {
		results = append(results, checkResult{
			name:    "Features directory",
			status:  "error",
			message: fmt.Sprintf("Directory missing: %s", featuresDir),
			fixable: true,
		})
		printCheck("Features directory", "error", "missing")

		if flagFix {
			if err := os.MkdirAll(featuresDir, 0755); err == nil {
				fileops.ColorPrintfn(fileops.Green, "  → Fixed: Created %s", featuresDir)
				results[len(results)-1].status = "ok"
			}
		}
	} else {
		printCheck("Features directory", "ok", "")
		results = append(results, checkResult{
			name:   "Features directory",
			status: "ok",
		})
	}

	// Check shared lib directory and helpers.sh exists
	libDir := filepath.Join(repoPath, "omd-shells", "lib")
	helpersFile := filepath.Join(libDir, "helpers.sh")

	// Check lib directory
	if _, err := os.Stat(libDir); os.IsNotExist(err) {
		results = append(results, checkResult{
			name:    "Shared lib directory",
			status:  "warning",
			message: fmt.Sprintf("Directory missing: %s (optional but recommended)", libDir),
			fixable: true,
		})
		printCheck("Shared lib directory", "warning", "missing (optional but recommended)")

		if flagFix {
			if err := os.MkdirAll(libDir, 0755); err == nil {
				fileops.ColorPrintfn(fileops.Green, "  → Fixed: Created %s", libDir)
				results[len(results)-1].status = "ok"
			}
		}
	} else {
		printCheck("Shared lib directory", "ok", "")
		results = append(results, checkResult{
			name:   "Shared lib directory",
			status: "ok",
		})
	}

	// Check helpers.sh file
	if _, err := os.Stat(helpersFile); os.IsNotExist(err) {
		results = append(results, checkResult{
			name:    "Helpers file",
			status:  "warning",
			message: fmt.Sprintf("File missing: %s (optional but recommended)", helpersFile),
			fixable: true,
		})
		printCheck("Helpers file", "warning", "missing (optional but recommended)")

		if flagFix {
			// Create the lib directory first if it doesn't exist
			if err := os.MkdirAll(libDir, 0755); err != nil {
				fileops.ColorPrintfn(fileops.Red, "  → Error creating lib directory: %v", err)
			} else {
				if err := os.WriteFile(helpersFile, []byte(shell.HelpersFileContent), 0644); err == nil {
					fileops.ColorPrintfn(fileops.Green, "  → Fixed: Created %s", helpersFile)
					results[len(results)-1].status = "ok"
				}
			}
		}
	} else {
		printCheck("Helpers file", "ok", "")
		results = append(results, checkResult{
			name:   "Helpers file",
			status: "ok",
		})
	}

	return results
}

func checkManifest(repoPath, shellName string) []checkResult {
	var results []checkResult

	manifestPath := shell.GetManifestPath(repoPath, shellName)

	// Check manifest file exists
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		results = append(results, checkResult{
			name:    "Manifest file",
			status:  "error",
			message: fmt.Sprintf("File missing: %s", manifestPath),
			fixable: false,
		})
		printCheck("Manifest file", "error", "missing")
		return results
	}

	// Parse manifest
	m, err := manifest.ParseManifest(manifestPath)
	if err != nil {
		results = append(results, checkResult{
			name:    "Manifest validity",
			status:  "error",
			message: fmt.Sprintf("Invalid manifest: %v", err),
			fixable: false,
		})
		printCheck("Manifest validity", "error", err.Error())
		return results
	}

	printCheck("Manifest file", "ok", "")
	results = append(results, checkResult{
		name:   "Manifest file",
		status: "ok",
	})

	// Validate each feature config
	for _, feature := range m.Features {
		if err := feature.Validate(); err != nil {
			results = append(results, checkResult{
				name:    fmt.Sprintf("Feature '%s' config", feature.Name),
				status:  "error",
				message: err.Error(),
				fixable: false,
			})
			printCheck(fmt.Sprintf("Feature '%s' config", feature.Name), "error", err.Error())
		}
	}

	if len(m.Features) > 0 {
		printCheck(fmt.Sprintf("Feature configs (%d)", len(m.Features)), "ok", "")
		results = append(results, checkResult{
			name:   "Feature configs",
			status: "ok",
		})
	}

	return results
}

func checkLocalOverride(repoPath, shellName string) []checkResult {
	var results []checkResult

	localManifestPath := shell.GetLocalManifestPath(repoPath, shellName)

	// Check if local manifest exists
	if _, err := os.Stat(localManifestPath); os.IsNotExist(err) {
		// No local manifest, that's fine
		return results
	}

	// Validate security
	if err := manifest.ValidateLocalManifest(localManifestPath); err != nil {
		results = append(results, checkResult{
			name:    "Local override security",
			status:  "error",
			message: fmt.Sprintf("Unsafe: %v", err),
			fixable: false,
		})
		printCheck("Local override security", "error", err.Error())
		return results
	}

	// Parse local manifest
	if _, err := manifest.ParseManifest(localManifestPath); err != nil {
		results = append(results, checkResult{
			name:    "Local override validity",
			status:  "error",
			message: fmt.Sprintf("Invalid: %v", err),
			fixable: false,
		})
		printCheck("Local override validity", "error", err.Error())
		return results
	}

	printCheck("Local override", "ok", "")
	results = append(results, checkResult{
		name:   "Local override",
		status: "ok",
	})

	return results
}

func checkFeatureFiles(repoPath, shellName string) []checkResult {
	var results []checkResult

	manifestPath := shell.GetManifestPath(repoPath, shellName)
	m, err := manifest.ParseManifest(manifestPath)
	if err != nil {
		// Already reported in checkManifest
		return results
	}

	missingCount := 0
	for _, feature := range m.Features {
		featurePath, err := shell.GetFeatureFilePath(repoPath, shellName, feature.Name)
		if err != nil {
			continue
		}

		if _, err := os.Stat(featurePath); os.IsNotExist(err) {
			results = append(results, checkResult{
				name:    fmt.Sprintf("Feature file '%s'", feature.Name),
				status:  "error",
				message: fmt.Sprintf("File missing: %s", featurePath),
				fixable: true,
			})
			printCheck(fmt.Sprintf("Feature file '%s'", feature.Name), "error", "missing")
			missingCount++

			if flagFix {
				// Create empty feature file
				if err := os.WriteFile(featurePath, []byte("# Feature: "+feature.Name+"\n"), 0644); err == nil {
					fileops.ColorPrintfn(fileops.Green, "  → Fixed: Created %s", featurePath)
					results[len(results)-1].status = "ok"
				}
			}
		}
	}

	if missingCount == 0 && len(m.Features) > 0 {
		printCheck(fmt.Sprintf("Feature files (%d)", len(m.Features)), "ok", "")
		results = append(results, checkResult{
			name:   "Feature files",
			status: "ok",
		})
	}

	return results
}

func checkProfileHooks(repoPath, shellName string) []checkResult {
	var results []checkResult

	shellConfig, ok := shell.GetShellConfig(shellName)
	if !ok {
		results = append(results, checkResult{
			name:    "Profile hook",
			status:  "warning",
			message: "Shell not supported for hook checking",
			fixable: false,
		})
		printCheck("Profile hook", "warning", "shell not supported")
		return results
	}

	// Resolve profile path
	profilePath, err := shell.ResolveProfilePath(shellConfig)
	if err != nil {
		results = append(results, checkResult{
			name:    "Profile hook",
			status:  "warning",
			message: fmt.Sprintf("Cannot resolve profile: %v", err),
			fixable: false,
		})
		printCheck("Profile hook", "warning", err.Error())
		return results
	}

	// Check if hook exists
	hasHook, err := hooks.HasHook(profilePath)
	if err != nil {
		results = append(results, checkResult{
			name:    "Profile hook",
			status:  "error",
			message: fmt.Sprintf("Cannot check hook: %v", err),
			fixable: false,
		})
		printCheck("Profile hook", "error", err.Error())
		return results
	}

	if !hasHook {
		results = append(results, checkResult{
			name:    "Profile hook",
			status:  "error",
			message: fmt.Sprintf("Hook missing in %s", profilePath),
			fixable: true,
		})
		printCheck("Profile hook", "error", "not installed")

		if flagFix {
			initScriptPath, _ := shell.GetInitScriptPath(repoPath, shellName)
			hookContent := hooks.GenerateHook(shellName, initScriptPath)
			if added, err := hooks.InsertHook(profilePath, hookContent); err == nil && added {
				fileops.ColorPrintfn(fileops.Green, "  → Fixed: Added hook to %s", profilePath)
				results[len(results)-1].status = "ok"
			}
		}
	} else {
		printCheck("Profile hook", "ok", "")
		results = append(results, checkResult{
			name:   "Profile hook",
			status: "ok",
		})
	}

	return results
}

func checkInitScriptSyntax(repoPath, shellName string) []checkResult {
	var results []checkResult

	initScriptPath, err := shell.GetInitScriptPath(repoPath, shellName)
	if err != nil {
		return results
	}

	// Check if init script exists
	if _, err := os.Stat(initScriptPath); os.IsNotExist(err) {
		results = append(results, checkResult{
			name:    "Init script",
			status:  "error",
			message: fmt.Sprintf("File missing: %s", initScriptPath),
			fixable: true,
		})
		printCheck("Init script", "error", "missing")

		if flagFix {
			// Regenerate init script
			if err := shell.RegenerateInitScript(repoPath, shellName); err == nil {
				fileops.ColorPrintfn(fileops.Green, "  → Fixed: Generated %s", initScriptPath)
				results[len(results)-1].status = "ok"
			}
		}
		return results
	}

	printCheck("Init script", "ok", "")
	results = append(results, checkResult{
		name:   "Init script",
		status: "ok",
	})

	// Validate syntax based on shell type
	var cmd *exec.Cmd
	switch shellName {
	case "bash":
		cmd = exec.Command("bash", "-n", initScriptPath)
	case "zsh":
		cmd = exec.Command("zsh", "-n", initScriptPath)
	case "fish":
		cmd = exec.Command("fish", "-n", initScriptPath)
	case "posix":
		cmd = exec.Command("sh", "-n", initScriptPath)
	default:
		// PowerShell syntax checking is complex, skip for now
		return results
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		results = append(results, checkResult{
			name:    "Init script syntax",
			status:  "error",
			message: fmt.Sprintf("Syntax error: %s", strings.TrimSpace(string(output))),
			fixable: true,
		})
		printCheck("Init script syntax", "error", strings.TrimSpace(string(output)))

		if flagFix {
			// Regenerate init script
			if err := shell.RegenerateInitScript(repoPath, shellName); err == nil {
				fileops.ColorPrintfn(fileops.Green, "  → Fixed: Regenerated %s", initScriptPath)
				results[len(results)-1].status = "ok"
			}
		}
	} else {
		printCheck("Init script syntax", "ok", "")
		results = append(results, checkResult{
			name:   "Init script syntax",
			status: "ok",
		})
	}

	return results
}

func printCheck(name, status, message string) {
	var icon string
	var color string

	switch status {
	case "ok":
		icon = "✓"
		color = fileops.Green
	case "warning":
		icon = "⚠"
		color = fileops.Yellow
	case "error":
		icon = "✗"
		color = fileops.Red
	}

	if message == "" {
		fileops.ColorPrintfn(color, "  %s %s", icon, name)
	} else {
		fileops.ColorPrintfn(color, "  %s %s: %s", icon, name, message)
	}
}
