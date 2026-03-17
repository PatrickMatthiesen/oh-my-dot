package doctor

import "testing"

func TestCountResults(t *testing.T) {
	results := []result{
		okResult("one"),
		warningResult("two", "warn", false),
		warningResult("three", "warn", true),
		errorResult("four", "err", false),
		errorResult("five", "err", true),
	}

	total := countResults(results)

	if total.okCount != 1 {
		t.Fatalf("okCount = %d, want 1", total.okCount)
	}
	if total.warningCount != 2 {
		t.Fatalf("warningCount = %d, want 2", total.warningCount)
	}
	if total.errorCount != 2 {
		t.Fatalf("errorCount = %d, want 2", total.errorCount)
	}
	if total.fixableWarningCount != 1 {
		t.Fatalf("fixableWarningCount = %d, want 1", total.fixableWarningCount)
	}
	if total.fixableErrorCount != 1 {
		t.Fatalf("fixableErrorCount = %d, want 1", total.fixableErrorCount)
	}
}

func TestSummaryHelpers(t *testing.T) {
	total := summary{
		errorCount:          2,
		fixableWarningCount: 1,
		fixableErrorCount:   3,
	}

	if !total.hasErrors() {
		t.Fatal("hasErrors() = false, want true")
	}

	if got := total.totalFixable(); got != 4 {
		t.Fatalf("totalFixable() = %d, want 4", got)
	}
}

func TestInitScriptSyntaxCommand(t *testing.T) {
	tests := []struct {
		name      string
		shellName string
		wantNil   bool
	}{
		{name: "bash", shellName: "bash"},
		{name: "zsh", shellName: "zsh"},
		{name: "fish", shellName: "fish"},
		{name: "posix", shellName: "posix"},
		{name: "powershell", shellName: "powershell", wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := initScriptSyntaxCommand(tt.shellName, "init-script")
			if tt.wantNil {
				if got != nil {
					t.Fatalf("initScriptSyntaxCommand(%q) = %v, want nil", tt.shellName, got)
				}
				return
			}

			if got == nil {
				t.Fatalf("initScriptSyntaxCommand(%q) = nil, want command", tt.shellName)
			}
		})
	}
}
