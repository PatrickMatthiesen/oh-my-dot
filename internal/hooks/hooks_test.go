package hooks

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestMalformedHookNeverChangesProfile(t *testing.T) {
	for _, content := range []string{
		"before\n" + HookStartMarker + "\nuser tail\n",
		"before\n" + HookEndMarker + "\nuser tail\n",
		HookStartMarker + "\n" + HookStartMarker + "\n" + HookEndMarker + "\nuser tail\n",
	} {
		t.Run(content, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "profile")
			if err := os.WriteFile(path, []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			if found, err := HasHook(path); err == nil || found {
				t.Fatal("malformed block reported healthy")
			}
			if _, err := InsertHook(path, GenerateHook("bash", "/repo/init")); err == nil {
				t.Fatal("insert accepted malformed block")
			}
			if err := RemoveHook(path); err == nil {
				t.Fatal("remove accepted malformed block")
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != content {
				t.Fatalf("profile changed: %q %v", data, err)
			}
		})
	}
}

func TestRemoveHookPreservesOtherBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profile")
	content := "before\r\n" + HookStartMarker + "\r\nmanaged\r\n" + HookEndMarker + "\r\nuser tail without newline"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	if err := RemoveHook(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "before\r\nuser tail without newline" {
		t.Fatalf("outside bytes changed: %q %v", data, err)
	}
}

func TestReplaceStaleHookPreservesProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profile")
	before := "# private prefix\r\n"
	after := "# private suffix\r\n"
	old := strings.ReplaceAll(GenerateHook("bash", "/old/init.sh"), "\n", "\r\n")
	if err := os.WriteFile(path, []byte(before+old+"\r\n"+after), 0600); err != nil {
		t.Fatal(err)
	}
	expected := GenerateHook("bash", "/new/init.sh")
	if matches, err := HookMatches(path, expected); err != nil || matches {
		t.Fatalf("stale match %v %v", matches, err)
	}
	if err := ReplaceHook(path, expected); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), before) || !strings.HasSuffix(string(data), after) {
		t.Fatal("outside profile content changed")
	}
	if matches, err := HookMatches(path, expected); err != nil || !matches {
		t.Fatalf("updated match %v %v", matches, err)
	}
	info, _ := os.Stat(path)
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatal("profile permissions changed")
	}
}
