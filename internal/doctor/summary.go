package doctor

import (
	"fmt"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
)

func countResults(results []result) summary {
	var total summary

	for _, item := range results {
		switch item.status {
		case statusOK:
			total.okCount++
		case statusWarning:
			total.warningCount++
			if item.fixable {
				total.fixableWarningCount++
			}
		case statusError:
			total.errorCount++
			if item.fixable {
				total.fixableErrorCount++
			}
		}
	}

	return total
}

func (total summary) hasErrors() bool {
	return total.errorCount > 0
}

func (total summary) totalFixable() int {
	return total.fixableErrorCount + total.fixableWarningCount
}

func printSummary(total summary, alias string, fix bool) {
	fmt.Println()
	fileops.ColorPrintln("Summary", fileops.Cyan)
	fileops.ColorPrintln("-------", fileops.Cyan)

	fileops.ColorPrintfn(fileops.Green, "✓ %d checks passed", total.okCount)
	if total.warningCount > 0 {
		fileops.ColorPrintfn(fileops.Yellow, "⚠ %d warnings", total.warningCount)
	}
	if total.errorCount > 0 {
		fileops.ColorPrintfn(fileops.Red, "✗ %d errors", total.errorCount)
	}

	if total.totalFixable() > 0 && !fix {
		fmt.Println()
		switch {
		case total.fixableErrorCount > 0 && total.fixableWarningCount > 0:
			fileops.ColorPrintfn(
				fileops.Cyan,
				"Tip: Run '%s doctor --fix' to automatically fix %d error(s) and %d warning(s)",
				alias,
				total.fixableErrorCount,
				total.fixableWarningCount,
			)
		case total.fixableErrorCount > 0:
			fileops.ColorPrintfn(
				fileops.Cyan,
				"Tip: Run '%s doctor --fix' to automatically fix %d error(s)",
				alias,
				total.fixableErrorCount,
			)
		default:
			fileops.ColorPrintfn(
				fileops.Cyan,
				"Tip: Run '%s doctor --fix' to automatically fix %d warning(s)",
				alias,
				total.fixableWarningCount,
			)
		}
	} else if total.warningCount > 0 && total.fixableWarningCount == 0 {
		fmt.Println()
		fileops.ColorPrintln("Note: Warnings are optional issues that don't affect functionality", fileops.Yellow)
	}

	fmt.Println()
}
