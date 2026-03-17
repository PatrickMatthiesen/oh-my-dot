package hooks

import (
	"os"
	"path/filepath"
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
