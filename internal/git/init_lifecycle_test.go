package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
)

func TestInitGitRepoMovesVerifiedCloneAndReturnsUsableRepository(t *testing.T) {
	for _, populated := range []bool{false, true} {
		name := "empty remote"
		if populated {
			name = "packed remote"
		}
		t.Run(name, func(t *testing.T) {
			remotePath := filepath.Join(t.TempDir(), "remote")
			remote, err := gogit.PlainInit(remotePath, false)
			if err != nil {
				t.Fatal(err)
			}
			if populated {
				if err := os.WriteFile(filepath.Join(remotePath, "README"), []byte("managed content"), 0600); err != nil {
					t.Fatal(err)
				}
				worktree, err := remote.Worktree()
				if err != nil {
					t.Fatal(err)
				}
				if _, err := worktree.Add("README"); err != nil {
					t.Fatal(err)
				}
				if _, err := worktree.Commit("initial", &gogit.CommitOptions{Author: &object.Signature{Name: "Test", Email: "test@example.invalid", When: time.Now()}}); err != nil {
					t.Fatal(err)
				}
			}
			if err := remote.Close(); err != nil {
				t.Fatal(err)
			}
			parent := t.TempDir()
			destination := filepath.Join(parent, "dotfiles")
			if err := os.Mkdir(destination, 0700); err != nil {
				t.Fatal(err)
			}
			// Verification reads a packed commit. On Windows an unclosed pack/index
			// descriptor prevents InitGitRepo from renaming its temporary clone.
			repo, err := InitGitRepo(destination, remotePath)
			if err != nil {
				t.Fatal(err)
			}
			if populated {
				head, err := repo.Head()
				if err != nil {
					t.Fatal(err)
				}
				if _, err := repo.CommitObject(head.Hash()); err != nil {
					t.Fatalf("returned repository retained the old clone path: %v", err)
				}
				data, err := os.ReadFile(filepath.Join(destination, "README"))
				if err != nil || string(data) != "managed content" {
					t.Fatalf("clone content %q, %v", data, err)
				}
			}
			if err := repo.Close(); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(destination, filepath.Join(parent, "moved")); err != nil {
				t.Fatalf("closed clone still holds descriptors: %v", err)
			}
			partials, err := filepath.Glob(filepath.Join(parent, ".omd-clone-*"))
			if err != nil {
				t.Fatal(err)
			}
			if len(partials) != 0 {
				t.Fatalf("temporary clones remain: %v", partials)
			}
		})
	}
}
