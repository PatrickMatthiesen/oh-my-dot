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

// Run executes the doctor checks for the provided shells and prints the result summary.
func Run(repoPath string, shellsToCheck []string, alias string, fix bool) error {
	fileops.ColorPrintln("\noh-my-dot Shell Framework Doctor", fileops.Cyan)
	fileops.ColorPrintln("=================================\n", fileops.Cyan)

	if len(shellsToCheck) == 0 {
		fileops.ColorPrintln("No shell features configured", fileops.Yellow)
		fileops.ColorPrintfn(fileops.Cyan, "Run '%s feature add' to add features", alias)
		return nil
	}

	allResults := runChecks(repoPath, shellsToCheck, fix)
	summary := countResults(allResults)

	printSummary(summary, alias, fix)

	if summary.hasErrors() {
		return fmt.Errorf("health check failed")
	}

	fileops.ColorPrintln("All checks passed! ✓", fileops.Green)
	return nil
}
