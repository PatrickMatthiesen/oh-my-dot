package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestShellValidationPathForGOOS(t *testing.T) {
	tests := []struct {
		name string
		goos string
		path string
		want string
	}{
		{name: "windows path", goos: "windows", path: `C:\Users\patr7\dotfiles\omd-shells\bash\init.sh`, want: `C:/Users/patr7/dotfiles/omd-shells/bash/init.sh`},
		{name: "unix path", goos: "linux", path: `/home/user/dotfiles/omd-shells/bash/init.sh`, want: `/home/user/dotfiles/omd-shells/bash/init.sh`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shellValidationPathForGOOS(tt.goos, tt.path); got != tt.want {
				t.Fatalf("shellValidationPathForGOOS(%q, %q) = %q, want %q", tt.goos, tt.path, got, tt.want)
			}
		})
	}
}

func TestCheckLineEndingsFixesCRLF(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(tmpDir, "omd-shells", "lib"), 0755); err != nil {
		t.Fatalf("failed to create lib dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "omd-shells", "bash", "features"), 0755); err != nil {
		t.Fatalf("failed to create features dir: %v", err)
	}

	initPath := filepath.Join(tmpDir, "omd-shells", "bash", "init.sh")
	helperPath := filepath.Join(tmpDir, "omd-shells", "lib", "helpers.sh")

	if err := os.WriteFile(initPath, []byte("line1\r\nline2\r\n"), 0644); err != nil {
		t.Fatalf("failed to create init script: %v", err)
	}
	if err := os.WriteFile(helperPath, []byte("helper\r\ncontent\r\n"), 0644); err != nil {
		t.Fatalf("failed to create helper file: %v", err)
	}

	results := checkLineEndings(context{repoPath: tmpDir, shellName: "bash", fix: true})
	if len(results) == 0 {
		t.Fatal("expected checkLineEndings to report a result")
	}

	for _, path := range []string{initPath, helperPath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}
		if strings.Contains(string(data), "\r") {
			t.Fatalf("expected LF-only content in %s, got %q", path, string(data))
		}
	}
}
