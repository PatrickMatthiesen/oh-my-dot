package git_test

import (
	"errors"
	"os"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	internalgit "github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/PatrickMatthiesen/oh-my-dot/tests/testutil"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/viper"
)

func TestGetRemoteSyncState_UpToDate(t *testing.T) {
	_, err := testutil.SetupTestRepo(t)
	if err != nil {
		t.Fatalf("setup repo: %v", err)
	}

	state, err := internalgit.GetRemoteSyncState()
	if err != nil {
		t.Fatalf("GetRemoteSyncState error: %v", err)
	}

	if state != internalgit.RemoteSyncUpToDate {
		t.Fatalf("state = %q, want %q", state, internalgit.RemoteSyncUpToDate)
	}

	hasUpdates, err := internalgit.HasRemoteUpdates()
	if err != nil {
		t.Fatalf("HasRemoteUpdates error: %v", err)
	}
	if hasUpdates {
		t.Fatalf("HasRemoteUpdates = true, want false")
	}
}

func TestGetRemoteSyncState_RemoteAhead(t *testing.T) {
	_, err := testutil.SetupTestRepo(t)
	if err != nil {
		t.Fatalf("setup repo: %v", err)
	}

	remotePath := viper.GetString("remote-url")
	if err := commitAndPushToRemote(t, remotePath, "remote-ahead.txt", "remote ahead"); err != nil {
		t.Fatalf("commit remote: %v", err)
	}

	state, err := internalgit.GetRemoteSyncState()
	if err != nil {
		t.Fatalf("GetRemoteSyncState error: %v", err)
	}

	if state != internalgit.RemoteSyncRemoteAhead {
		t.Fatalf("state = %q, want %q", state, internalgit.RemoteSyncRemoteAhead)
	}

	hasUpdates, err := internalgit.HasRemoteUpdates()
	if err != nil {
		t.Fatalf("HasRemoteUpdates error: %v", err)
	}
	if !hasUpdates {
		t.Fatalf("HasRemoteUpdates = false, want true")
	}
}

func TestGetRemoteSyncState_RemoteSignificantlyAhead(t *testing.T) {
	_, err := testutil.SetupTestRepo(t)
	if err != nil {
		t.Fatalf("setup repo: %v", err)
	}

	repoPath := viper.GetString("repo-path")
	remotePath := viper.GetString("remote-url")

	if err := commitAndPushManyToRemote(t, remotePath, 101); err != nil {
		t.Fatalf("commit remote: %v", err)
	}
	if err := fetchLocalOriginMain(repoPath); err != nil {
		t.Fatalf("fetch local remote refs: %v", err)
	}

	state, err := internalgit.GetRemoteSyncState()
	if err != nil {
		t.Fatalf("GetRemoteSyncState error: %v", err)
	}

	if state != internalgit.RemoteSyncRemoteSignificantlyAhead {
		t.Fatalf("state = %q, want %q", state, internalgit.RemoteSyncRemoteSignificantlyAhead)
	}

	hasUpdates, err := internalgit.HasRemoteUpdates()
	if err != nil {
		t.Fatalf("HasRemoteUpdates error: %v", err)
	}
	if !hasUpdates {
		t.Fatalf("HasRemoteUpdates = false, want true")
	}
}

func TestGetRemoteSyncState_LocalAhead(t *testing.T) {
	_, err := testutil.SetupTestRepo(t)
	if err != nil {
		t.Fatalf("setup repo: %v", err)
	}

	repoPath := viper.GetString("repo-path")
	if err := commitToRepo(t, repoPath, "local-ahead.txt", "local ahead"); err != nil {
		t.Fatalf("commit local: %v", err)
	}

	state, err := internalgit.GetRemoteSyncState()
	if err != nil {
		t.Fatalf("GetRemoteSyncState error: %v", err)
	}

	if state != internalgit.RemoteSyncLocalAhead {
		t.Fatalf("state = %q, want %q", state, internalgit.RemoteSyncLocalAhead)
	}

	hasUpdates, err := internalgit.HasRemoteUpdates()
	if err != nil {
		t.Fatalf("HasRemoteUpdates error: %v", err)
	}
	if hasUpdates {
		t.Fatalf("HasRemoteUpdates = true, want false")
	}
}

func TestGetRemoteSyncState_Diverged(t *testing.T) {
	_, err := testutil.SetupTestRepo(t)
	if err != nil {
		t.Fatalf("setup repo: %v", err)
	}

	repoPath := viper.GetString("repo-path")
	remotePath := viper.GetString("remote-url")

	if err := commitToRepo(t, repoPath, "local-diverged.txt", "local diverged"); err != nil {
		t.Fatalf("commit local: %v", err)
	}
	if err := commitAndPushToRemote(t, remotePath, "remote-diverged.txt", "remote diverged"); err != nil {
		t.Fatalf("commit remote: %v", err)
	}
	if err := fetchLocalOriginMain(repoPath); err != nil {
		t.Fatalf("fetch local remote refs: %v", err)
	}

	state, err := internalgit.GetRemoteSyncState()
	if err != nil {
		t.Fatalf("GetRemoteSyncState error: %v", err)
	}

	if state != internalgit.RemoteSyncDiverged {
		t.Fatalf("state = %q, want %q", state, internalgit.RemoteSyncDiverged)
	}

	hasUpdates, err := internalgit.HasRemoteUpdates()
	if err != nil {
		t.Fatalf("HasRemoteUpdates error: %v", err)
	}
	if !hasUpdates {
		t.Fatalf("HasRemoteUpdates = false, want true")
	}
}

func commitToRepo(t *testing.T, repoPath, filename, content string) error {
	t.Helper()

	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return err
	}

	wt, err := r.Worktree()
	if err != nil {
		return err
	}

	fullPath := filepath.Join(repoPath, filename)
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return err
	}

	if _, err := wt.Add(filename); err != nil {
		return err
	}

	_, err = wt.Commit("test commit "+filename, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	return err
}

func commitAndPushToRemote(t *testing.T, remotePath, filename, content string) error {
	t.Helper()

	clonePath := t.TempDir()
	r, err := git.PlainClone(clonePath, false, &git.CloneOptions{URL: remotePath})
	if err != nil {
		return err
	}

	if err := commitToRepo(t, clonePath, filename, content); err != nil {
		return err
	}

	return r.Push(&git.PushOptions{})
}

func commitAndPushManyToRemote(t *testing.T, remotePath string, count int) error {
	t.Helper()

	clonePath := t.TempDir()
	r, err := git.PlainClone(clonePath, false, &git.CloneOptions{URL: remotePath})
	if err != nil {
		return err
	}

	for i := range count {
		filename := fmt.Sprint("remote-ahead-%03d.txt", i)
		content := fmt.Sprint("remote ahead %03d", i)
		if err := commitToRepo(t, clonePath, filename, content); err != nil {
			return err
		}
	}

	return r.Push(&git.PushOptions{})
}

func fetchLocalOriginMain(repoPath string) error {
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return err
	}

	err = r.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			"+refs/heads/main:refs/remotes/origin/main",
		},
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return err
	}

	return nil
}
