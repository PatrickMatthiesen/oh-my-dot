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

func TestInitEmptyRemotePreservesBranchThroughPushAndReclone(t *testing.T) {
	for _, branch := range []string{"main", "machines/windows"} {
		t.Run(branch, func(t *testing.T) {
			remotePath := t.TempDir()
			branchRef := plumbing.NewBranchReferenceName(branch)
			remote, err := gogit.PlainInit(remotePath, true, gogit.WithDefaultBranch(branchRef))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { remote.Close() })
			destination := filepath.Join(t.TempDir(), "dotfiles")
			repo, err := InitGitRepo(destination, remotePath)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { repo.Close() })
			head, err := repo.Reference(plumbing.HEAD, false)
			if err != nil || head.Target() != branchRef {
				t.Fatalf("initial HEAD %v %v; want %s", head, err, branchRef)
			}
			if err := os.WriteFile(filepath.Join(destination, "first"), []byte("first commit"), 0600); err != nil {
				t.Fatal(err)
			}
			worktree, err := repo.Worktree()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := worktree.Add("first"); err != nil {
				t.Fatal(err)
			}
			hash, err := worktree.Commit("initial", &gogit.CommitOptions{Author: &object.Signature{Name: "Test", Email: "test@example.invalid", When: time.Now()}})
			if err != nil {
				t.Fatal(err)
			}
			origin, err := repo.Remote("origin")
			if err != nil {
				t.Fatal(err)
			}
			if err := origin.Push(&gogit.PushOptions{}); err != nil {
				t.Fatal(err)
			}
			remoteHead, err := remote.Reference(plumbing.HEAD, false)
			if err != nil || remoteHead.Target() != branchRef {
				t.Fatalf("remote HEAD changed: %v %v", remoteHead, err)
			}
			secondPath := filepath.Join(t.TempDir(), "second")
			second, err := InitGitRepo(secondPath, remotePath)
			if err != nil {
				t.Fatalf("reclone after first push: %v", err)
			}
			t.Cleanup(func() { second.Close() })
			secondHead, err := second.Head()
			if err != nil || secondHead.Name() != branchRef || secondHead.Hash() != hash {
				t.Fatalf("recloned HEAD %v %v", secondHead, err)
			}
			if content, err := os.ReadFile(filepath.Join(secondPath, "first")); err != nil || string(content) != "first commit" {
				t.Fatalf("recloned file %q %v", content, err)
			}
		})
	}
}
