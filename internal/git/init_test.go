package git

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
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

func TestInitGitRepoAcceptsEmptyRemote(t *testing.T) {
	remote := t.TempDir()
	if _, err := gogit.PlainInit(remote, true); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "dotfiles")
	repo, err := InitGitRepo(destination, remote)
	if err != nil {
		t.Fatalf("initialize empty remote: %v", err)
	}
	origin, err := repo.Remote("origin")
	if err != nil || origin.Config().URLs[0] != remote {
		t.Fatalf("origin = %v, %v", origin, err)
	}
	if _, err := repo.Head(); !errors.Is(err, plumbing.ErrReferenceNotFound) {
		t.Fatalf("empty remote HEAD error = %v", err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "first.txt"), []byte("first"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := worktree.Add("first.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := worktree.Commit("First commit", &gogit.CommitOptions{Author: &object.Signature{Name: "Test", Email: "test@example.invalid", When: time.Now()}}); err != nil {
		t.Fatal(err)
	}
	if err := origin.Push(&gogit.PushOptions{}); err != nil {
		t.Fatalf("first push to empty remote: %v", err)
	}
}
