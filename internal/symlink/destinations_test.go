package symlink

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckDestinationAvailable(t *testing.T) {
	root := t.TempDir()
	for _, tc := range []struct {
		name, key, destination string
		wantError              bool
	}{
		{"same destination", "work/config", filepath.Join(root, "config"), true},
		{"normalized destination", "work/config", filepath.Join(root, "sub", "..", "config"), true},
		{"case-only key", "SHARED/config", filepath.Join(root, "other"), true},
		{"different destination", "work/config", filepath.Join(root, "other"), false},
		{"foreign destination", "work/config", "relative/config", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckDestinationAvailable(Linkings{"shared/config": filepath.Join(root, "config")}, tc.key, tc.destination)
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v, wantError %v", err, tc.wantError)
			}
		})
	}
}

func TestCheckDestinationDirectoryAlias(t *testing.T) {
	root := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	err := CheckDestinationAvailable(Linkings{"one/config": filepath.Join(root, "config")}, "two/config", filepath.Join(alias, "config"))
	if err == nil {
		t.Fatal("expected duplicate destination through directory symlink")
	}
}

func TestBuildLinkPathPreservesHomeSibling(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	file := home + "-other" + string(filepath.Separator) + "config"
	got, err := BuildLinkPath(file)
	if err != nil || got != filepath.ToSlash(file) {
		t.Fatalf("got %q, error %v", got, err)
	}
	got, err = BuildLinkPath(filepath.Join(home, ".config", "app", "config"))
	if err != nil || got != "~/.config/app/config" {
		t.Fatalf("got %q, error %v", got, err)
	}
}
