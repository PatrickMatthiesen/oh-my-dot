package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/spf13/viper"
)

func TestIsSSHRemote(t *testing.T) {
	for _, tt := range []struct {
		url  string
		want bool
	}{
		{"git@github.com:user/repo.git", true}, {"example.com:repo.git", true}, {"ssh://git@example.com:443/repo", true},
		{"https://github.com/user/repo", false}, {`C:\Users\dotfiles`, false}, {"C:/dotfiles", false}, {"../dotfiles", false}, {"/tmp/a:b", false},
	} {
		t.Run(tt.url, func(t *testing.T) {
			if got := IsSSHRemote(tt.url); got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}

func TestSSHOperationsUseNativeGitConfiguration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture requires POSIX sh")
	}
	root := t.TempDir()
	remote := filepath.Join(root, "empty.git")
	if _, err := gogit.PlainInit(remote, true); err != nil {
		t.Fatal(err)
	}
	wrapper := filepath.Join(root, "ssh wrapper")
	// Git invokes the configured SSH command; execute the upload/receive pack
	// locally so this tests real protocol exchanges without network credentials.
	if err := os.WriteFile(wrapper, []byte("#!/bin/sh\nif [ \"$1\" = '-G' ]; then exit 0; fi\nfor arg do command=$arg; done\nexec sh -c \"$command\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_SSH_COMMAND", "'"+wrapper+"'")
	t.Setenv("GIT_SSH_VARIANT", "ssh")
	remoteURL := "git@example.invalid:" + filepath.ToSlash(remote)
	path := filepath.Join(root, "dotfiles")
	repo, err := InitGitRepo(path, remoteURL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Head(); !errors.Is(err, plumbing.ErrReferenceNotFound) {
		t.Fatalf("expected unborn HEAD: %v", err)
	}
	refs, err := ListRemote(path, remoteURL, 5)
	if err != nil || len(refs) != 0 {
		t.Fatalf("empty remote refs %v, %v", refs, err)
	}
	if err := os.WriteFile(filepath.Join(path, "README"), []byte("first"), 0600); err != nil {
		t.Fatal(err)
	}
	tree, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tree.Add("README"); err != nil {
		t.Fatal(err)
	}
	hash, err := tree.Commit("initial", &gogit.CommitOptions{Author: &object.Signature{Name: "Test", Email: "test@example.invalid", When: time.Now()}})
	if err != nil {
		t.Fatal(err)
	}
	oldPath := viper.GetString("repo-path")
	viper.Set("repo-path", path)
	t.Cleanup(func() { viper.Set("repo-path", oldPath) })
	if err := CheckRemotePushPermission(); err != nil {
		t.Fatalf("empty remote access: %v", err)
	}
	plan, err := PlanPushCompaction()
	if err != nil || plan.HasCompaction() {
		t.Fatalf("first push plan: %+v, %v", plan, err)
	}
	pushed, err := PushRepo()
	if err != nil || !pushed {
		t.Fatalf("push: %v %v", pushed, err)
	}
	pushed, err = PushRepo()
	if err != nil || pushed {
		t.Fatalf("repeat push: %v %v", pushed, err)
	}
	refs, err = ListRemote(path, remoteURL, 5)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, ref := range refs {
		if ref.Hash() == hash {
			found = true
		}
	}
	if !found {
		t.Fatal("first commit absent from remote")
	}
	second := filepath.Join(root, "second")
	if _, err := CloneSSH(second, remoteURL); err != nil {
		t.Fatal(err)
	}
	if err := FetchSSH(second, remoteURL, "+refs/heads/*:refs/remotes/origin/*"); err != nil {
		t.Fatal(err)
	}
	if err := ReinitializeRepository(second, remoteURL); err != nil {
		t.Fatal(err)
	}
	secondRepo, err := GetRepository(second)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(second, "README"), []byte("second"), 0600); err != nil {
		t.Fatal(err)
	}
	secondTree, err := secondRepo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := secondTree.Add("README"); err != nil {
		t.Fatal(err)
	}
	if _, err := secondTree.Commit("second", &gogit.CommitOptions{Author: &object.Signature{Name: "Test", Email: "test@example.invalid", When: time.Now()}}); err != nil {
		t.Fatal(err)
	}
	if _, err := runRemoteGit(second, time.Minute, "push", "origin", "HEAD"); err != nil {
		t.Fatal(err)
	}
	pulled, err := PullRepo()
	if err != nil || !pulled {
		t.Fatalf("pull %v %v", pulled, err)
	}
	pulled, err = PullRepo()
	if err != nil || pulled {
		t.Fatalf("repeat pull %v %v", pulled, err)
	}
	bareRepo, err := GetRepository(remote)
	if err != nil {
		t.Fatal(err)
	}
	if err := bareRepo.Storer.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("missing"))); err != nil {
		t.Fatal(err)
	}
	broken := filepath.Join(root, "broken")
	if _, err := InitGitRepo(broken, remoteURL); err == nil {
		t.Fatal("populated remote with broken default branch accepted")
	}
	if _, err := os.Stat(broken); !os.IsNotExist(err) {
		t.Fatalf("failed clone left destination: %v", err)
	}

}

func TestRemoteErrorGuidance(t *testing.T) {
	for _, tt := range []struct{ name, message, url, platform, state, want, absent string }{
		{"timeout", "dial tcp: i/o timeout", "git@example.com:repo", "windows", "Running", "network", "Start-Service"},
		{"context timeout", context.DeadlineExceeded.Error(), "git@example.com:repo", "linux", "", "timed out", "ssh-agent -s"},
		{"trust", "Host key verification failed", "git@github.com:user/repo", "windows", "", "ssh -T git@github.com", "Start-Service"},
		{"changed key", "REMOTE HOST IDENTIFICATION HAS CHANGED", "git@github.com:user/repo", "windows", "", "Verify the new fingerprint", "Start-Service"},
		{"running agent", "SSH agent unavailable", "git@github.com:user/repo", "windows", "Running", "ssh-add -l", "Start-Service"},
		{"stopped agent", "SSH agent unavailable", "git@github.com:user/repo", "windows", "Stopped", "Start-Service", "Add-WindowsCapability"},
		{"missing agent", "SSH agent unavailable", "git@github.com:user/repo", "windows", "missing", "Add-WindowsCapability", "ssh-agent -s"},
		{"https", "401 Repository not found", "https://github.com/user/repo", "windows", "", "gh auth setup-git", "Start-Service"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			help := remoteErrorHelp(errors.New(tt.message), tt.url, tt.platform, func() string { return tt.state })
			if !strings.Contains(help, tt.want) || strings.Contains(help, tt.absent) {
				t.Fatalf("unexpected guidance %q", help)
			}
		})
	}
}

func TestSSHHelpPreservesEndpoint(t *testing.T) {
	if got := sshHelpCommand("ssh://alice@example.com:2222/repo"); got != "ssh -T -p 2222 alice@example.com" {
		t.Fatal(got)
	}
}

func TestBatchSSHCommandPreservesConfiguration(t *testing.T) {
	t.Setenv("GIT_SSH_COMMAND", "custom-ssh -i '/path/key name'")
	t.Setenv("GIT_SSH_VARIANT", "ssh")
	if got := batchSSHCommand(t.TempDir()); got != "custom-ssh -i '/path/key name' -oBatchMode=yes" {
		t.Fatal(got)
	}
	t.Setenv("GIT_SSH_COMMAND", "plink -i key.ppk")
	t.Setenv("GIT_SSH_VARIANT", "plink")
	if got := batchSSHCommand(t.TempDir()); got != "plink -i key.ppk -batch" {
		t.Fatal(got)
	}
}
