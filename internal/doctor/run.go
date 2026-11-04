package doctor

import (
	"fmt"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
)

// ResolveShellsToCheck returns the configured shells to inspect for doctor runs.
func ResolveShellsToCheck(repoPath string, selectedShells []string) ([]string, error) {
	if len(selectedShells) > 0 {
		return selectedShells, nil
	}

	shellsToCheck, err := shell.ListShellsWithFeatures(repoPath)
	if err != nil {
		return nil, fmt.Errorf("list shells with features: %w", err)
	}

	return shellsToCheck, nil
}

// Run checks the repository and provided shells and prints the result summary.
func Run(repoPath string, shellsToCheck []string, alias string, fix bool) error {
	return RunWithOptions(repoPath, shellsToCheck, alias, fix, false)
}

// RunWithOptions runs human-readable checks with optional offline diagnostics.
func RunWithOptions(repoPath string, shellsToCheck []string, alias string, fix, offline bool) error {
	fileops.ColorPrintln("\noh-my-dot Doctor", fileops.Cyan)
	fileops.ColorPrintln("=================================\n", fileops.Cyan)

	if len(shellsToCheck) == 0 {
		fileops.ColorPrintln("No shell features configured", fileops.Yellow)
		fileops.ColorPrintfn(fileops.Cyan, "Run '%s feature add' to add features", alias)
	}

	allResults := runChecksWithOptions(repoPath, shellsToCheck, fix, false, offline)
	summary := countResults(allResults)

	printSummary(summary, alias, fix)

	if summary.hasErrors() {
		return fmt.Errorf("health check failed")
	}

	if summary.warningCount == 0 {
		fileops.ColorPrintln("All checks passed! ✓", fileops.Green)
	} else {
		fileops.ColorPrintln("Health check completed with warnings", fileops.Yellow)
	}
	return nil
}

// CheckResult describes one repository or shell diagnostic.
type CheckResult struct {
	Shell   string `json:"shell,omitempty"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Fixable bool   `json:"fixable"`
}

// Report is the versioned, machine-readable result of a doctor invocation.
type Report struct {
	SchemaVersion int           `json:"schemaVersion"`
	Checks        []CheckResult `json:"checks"`
	Errors        int           `json:"errors"`
	Warnings      int           `json:"warnings"`
}

// Inspect runs the same diagnostics without presentation output.
func Inspect(repoPath string, shells []string, fix, offline bool) (Report, error) {
	results := runChecksWithOptions(repoPath, shells, fix, true, offline)
	report := Report{SchemaVersion: 1, Checks: make([]CheckResult, 0, len(results))}
	for _, item := range results {
		report.Checks = append(report.Checks, CheckResult{Shell: item.shell, Name: item.name, Status: item.status, Message: item.message, Fixable: item.fixable})
		if item.status == statusError {
			report.Errors++
		}
		if item.status == statusWarning {
			report.Warnings++
		}
	}
	if report.Errors > 0 {
		return report, fmt.Errorf("health check failed (%d errors)", report.Errors)
	}
	return report, nil
}
