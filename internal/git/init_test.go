package git

import (
	"os"
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v6"
)

func TestInitGitRepoFailurePreservesDestination(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing destination", true: "existing files"}[existing], func(t *testing.T) {
			root := t.TempDir()
			destination := filepath.Join(root, "dotfiles")
			if existing {
				if err := os.Mkdir(destination, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(destination, "keep"), []byte("keep"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := InitGitRepo(destination, filepath.Join(root, "missing-remote")); err == nil {
				t.Fatal("expected clone failure")
			}
			if IsGitRepo(destination) {
				t.Fatal("failed clone left a repository")
			}
			if existing {
				data, err := os.ReadFile(filepath.Join(destination, "keep"))
				if err != nil || string(data) != "keep" {
					t.Fatalf("existing file changed: %q, %v", data, err)
				}
			} else if _, err := os.Stat(destination); !os.IsNotExist(err) {
				t.Fatalf("partial destination remains: %v", err)
			}
			matches, _ := filepath.Glob(filepath.Join(root, ".omd-clone-*"))
			if len(matches) != 0 {
				t.Fatalf("partial clones remain: %v", matches)
			}
		})
	}
}

func TestInitFromExistingLocalRepo(t *testing.T) {
	path := t.TempDir()
	if _, err := gogit.PlainInit(path, false); err != nil {
		t.Fatal(err)
	}
	if err := InitFromExistingRepo(path); err != nil {
		t.Fatal(err)
	}
}
