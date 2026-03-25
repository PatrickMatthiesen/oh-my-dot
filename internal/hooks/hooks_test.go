package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInsertHookCreatesParentDirectory(t *testing.T) {
	t.Parallel()

	profilePath := filepath.Join(t.TempDir(), "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")

	added, err := InsertHook(profilePath, GenerateHook("powershell", "C:\\repo\\omd-shells\\powershell\\init.ps1"))
	if err != nil {
		t.Fatalf("InsertHook returned error: %v", err)
	}
	if !added {
		t.Fatalf("InsertHook reported hook was not added")
	}

	data, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatalf("failed to read created profile: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected created profile to contain hook content")
	}
}

func TestGenerateHookNormalizesShellPaths(t *testing.T) {
	tests := []struct {
		name      string
		shell     string
		initPath  string
		wantSlash bool
	}{
		{name: "bash normalizes path", shell: "bash", initPath: `C:\repo\omd-shells\bash\init.sh`, wantSlash: true},
		{name: "zsh normalizes path", shell: "zsh", initPath: `C:\repo\omd-shells\zsh\init.zsh`, wantSlash: true},
		{name: "powershell preserves path", shell: "powershell", initPath: `C:\repo\omd-shells\powershell\init.ps1`, wantSlash: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := GenerateHook(tt.shell, tt.initPath)
			if tt.wantSlash {
				if strings.Contains(hook, "\\") {
					t.Fatalf("GenerateHook(%q, %q) kept backslashes in hook: %q", tt.shell, tt.initPath, hook)
				}
				if !strings.Contains(hook, strings.ReplaceAll(tt.initPath, "\\", "/")) {
					t.Fatalf("GenerateHook(%q, %q) did not include a slash-normalized path: %q", tt.shell, tt.initPath, hook)
				}
				return
			}

			if !strings.Contains(hook, tt.initPath) {
				t.Fatalf("GenerateHook(%q, %q) = %q, want original path preserved", tt.shell, tt.initPath, hook)
			}
		})
	}
}
