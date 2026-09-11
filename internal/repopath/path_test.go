package repopath

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestValidate(t *testing.T) {
	for _, key := range []string{"README.md", "work/README.md", ".config/git/config", "My Project/café.conf", "com10.txt"} {
		t.Run("valid_"+key, func(t *testing.T) {
			if err := Validate(key); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, key := range []string{"", "/a", ".git", "nested/.GiT/config", "git~1/config", "nested/GIT~1", "../a", "a/../b", "a/./b", "a//b", "a/", `a\b`, "C:/a", "a:b", "a?b", "a\x00b", "a\nb", "a.", "a ", "CON", "aux.txt", "NUL/x", "COM1.txt", "LPT9", "COM¹.log", "CONOUT$"} {
		t.Run("invalid_"+key, func(t *testing.T) {
			if err := Validate(key); err == nil {
				t.Fatalf("accepted %q", key)
			}
		})
	}
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("unchanged"), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckAvailable(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "files", "Work", "README.md"))
	writeFile(t, filepath.Join(repo, "files", "flat"))
	for _, tc := range []struct {
		key       string
		wantError bool
	}{
		{"Work/other.md", false}, {"Personal/README.md", false}, {"flat", true},
		{"Work/README.md", true}, {"Work/readme.md", true}, {"work/new.md", true},
		{"flat/nested", true}, {"Work", true}, {"WORK", true}, {"../outside", true},
	} {
		t.Run(tc.key, func(t *testing.T) {
			if err := CheckAvailable(repo, tc.key); (err != nil) != tc.wantError {
				t.Fatalf("error=%v, wantError=%v", err, tc.wantError)
			}
		})
	}
	content, err := os.ReadFile(filepath.Join(repo, "files", "Work", "README.md"))
	if err != nil || string(content) != "unchanged" {
		t.Fatalf("existing file changed: %q %v", content, err)
	}
}

func TestResolveAndListNested(t *testing.T) {
	repo := t.TempDir()
	keys, err := List(repo)
	if err != nil || len(keys) != 0 {
		t.Fatalf("missing directory: %v %v", keys, err)
	}
	resolved, err := Resolve(repo, "new/README.md")
	if err != nil || resolved != filepath.Join(repo, "files", "new", "README.md") {
		t.Fatalf("resolve: %q %v", resolved, err)
	}
	for _, key := range []string{"z/README.md", "README.md", "a/README.md"} {
		writeFile(t, filepath.Join(repo, "files", filepath.FromSlash(key)))
	}
	if err := os.MkdirAll(filepath.Join(repo, "files", "empty"), 0755); err != nil {
		t.Fatal(err)
	}
	keys, err = List(repo)
	if err != nil || !reflect.DeepEqual(keys, []string{"README.md", "a/README.md", "z/README.md"}) {
		t.Fatalf("list: %v %v", keys, err)
	}
}

func TestRejectSymlinkComponents(t *testing.T) {
	for _, component := range []string{"files", "files/linked", "files/linked/item"} {
		t.Run(component, func(t *testing.T) {
			repo := t.TempDir()
			outside := t.TempDir()
			link := filepath.Join(repo, filepath.FromSlash(component))
			if err := os.MkdirAll(filepath.Dir(link), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, link); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			if _, err := Resolve(repo, "linked/item"); err == nil {
				t.Fatal("resolved a symlink")
			}
			if err := CheckAvailable(repo, "linked/item"); err == nil {
				t.Fatal("accepted a symlink")
			}
			if _, err := List(repo); err == nil {
				t.Fatal("listed a symlink")
			}
		})
	}
}
