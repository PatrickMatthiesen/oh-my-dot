package doctor

import "github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"

const (
	statusOK      = "ok"
	statusWarning = "warning"
	statusError   = "error"
)

type context struct {
	repoPath  string
	shellName string
	fix       bool
}

type check struct {
	run func(ctx context) []result
}

type fixer func() (string, error)

type result struct {
	name    string
	status  string
	message string
	fixable bool
}

type summary struct {
	okCount             int
	warningCount        int
	errorCount          int
	fixableWarningCount int
	fixableErrorCount   int
}

func okResult(name string) result {
	return result{name: name, status: statusOK}
}

func warningResult(name, message string, fixable bool) result {
	return result{name: name, status: statusWarning, message: message, fixable: fixable}
}

func errorResult(name, message string, fixable bool) result {
	return result{name: name, status: statusError, message: message, fixable: fixable}
}

func addResult(results []result, ctx context, item result, applyFix fixer) []result {
	printCheck(item.name, item.status, item.message)

	if ctx.fix && item.fixable && applyFix != nil && item.status != statusOK {
		fixMessage, err := applyFix()
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "  → Error: %v", err)
		} else {
			fileops.ColorPrintfn(fileops.Green, "  → Fixed: %s", fixMessage)
			item.status = statusOK
			item.message = ""
		}
	}

	return append(results, item)
}

func printCheck(name, status, message string) {
	var icon string
	var color string

	switch status {
	case statusOK:
		icon = "✓"
		color = fileops.Green
	case statusWarning:
		icon = "⚠"
		color = fileops.Yellow
	case statusError:
		icon = "✗"
		color = fileops.Red
	}

	if message == "" {
		fileops.ColorPrintfn(color, "  %s %s", icon, name)
	} else {
		fileops.ColorPrintfn(color, "  %s %s: %s", icon, name, message)
	}
}
