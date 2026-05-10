package git

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/spf13/viper"
)

func TestPlanDuplicateSubjectGroups(t *testing.T) {
	commits := []*object.Commit{
		{Message: "Refresh shell features\n"},
		{Message: "Refresh shell features\n"},
		{Message: "Add shell features (interactive)\n"},
		{Message: "Refresh shell features\n"},
		{Message: "Refresh shell features (interactive)\n"},
		{Message: "Refresh shell features (interactive)\n"},
	}

	got := planDuplicateSubjectGroups(commits)
	want := []PushCompactionGroup{
		{Subject: "Refresh shell features", Count: 2},
		{Subject: "Refresh shell features (interactive)", Count: 2},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("planDuplicateSubjectGroups() = %#v, want %#v", got, want)
	}
}

func TestPlanDuplicateSubjectGroups_NoGroups(t *testing.T) {
	tests := []struct {
		name    string
		commits []*object.Commit
	}{
		{name: "no unpublished commits"},
		{name: "one unpublished commit", commits: []*object.Commit{{Message: "Refresh shell features\n"}}},
		{
			name: "same subject separated by another commit",
			commits: []*object.Commit{
				{Message: "Refresh shell features\n"},
				{Message: "Add shell features\n"},
				{Message: "Refresh shell features\n"},
			},
		},
		{
			name: "suffix does not match non suffix",
			commits: []*object.Commit{
				{Message: "Refresh shell features\n"},
				{Message: "Refresh shell features (interactive)\n"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := planDuplicateSubjectGroups(tt.commits)
			if len(got) != 0 {
				t.Fatalf("planDuplicateSubjectGroups() = %#v, want no groups", got)
			}
		})
	}
}

func TestPlanPushCompaction_DetectsThreeIdenticalCommits(t *testing.T) {
	repoPath := setupCompactableRepo(t)

	for i := 0; i < 3; i++ {
		commitFile(t, repoPath, fmt.Sprintf("refresh-%d.txt", i), "Refresh shell features")
	}

	plan, err := PlanPushCompaction()
	if err != nil {
		t.Fatalf("PlanPushCompaction() error: %v", err)
	}

	if plan.OriginalCommitCount != 3 {
		t.Fatalf("OriginalCommitCount = %d, want 3", plan.OriginalCommitCount)
	}
	if plan.CompactedCommitCount != 1 {
		t.Fatalf("CompactedCommitCount = %d, want 1", plan.CompactedCommitCount)
	}
	if len(plan.Groups) != 1 || plan.Groups[0].Count != 3 || plan.Groups[0].Subject != "Refresh shell features" {
		t.Fatalf("Groups = %#v, want one 3-commit Refresh shell features group", plan.Groups)
	}
}

func TestCompactPushHistory_RewritesAndPushesCompactedHistory(t *testing.T) {
	repoPath := setupCompactableRepo(t)

	commitFile(t, repoPath, "refresh-1.txt", "Refresh shell features")
	commitFile(t, repoPath, "refresh-2.txt", "Refresh shell features")
	commitFile(t, repoPath, "add.txt", "Add shell features")

	plan, err := PlanPushCompaction()
	if err != nil {
		t.Fatalf("PlanPushCompaction() error: %v", err)
	}

	if _, err := CompactPushHistory(plan); err != nil {
		t.Fatalf("CompactPushHistory() error: %v", err)
	}

	pushed, err := PushRepo()
	if err != nil {
		t.Fatalf("PushRepo() error: %v", err)
	}
	if !pushed {
		t.Fatalf("PushRepo() pushed = false, want true")
	}

	subjects := remoteSubjects(t, viper.GetString("remote-url"))
	want := []string{"Add shell features", "Refresh shell features", "Initial commit"}
	if !reflect.DeepEqual(subjects[:3], want) {
		t.Fatalf("remote subjects = %#v, want prefix %#v", subjects[:3], want)
	}
}

func TestPlanPushCompaction_SkipsDirtyWorktree(t *testing.T) {
	repoPath := setupCompactableRepo(t)
	commitFile(t, repoPath, "refresh-1.txt", "Refresh shell features")
	commitFile(t, repoPath, "refresh-2.txt", "Refresh shell features")

	if err := os.WriteFile(filepath.Join(repoPath, "dirty.txt"), []byte("dirty"), 0644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	plan, err := PlanPushCompaction()
	if err != nil {
		t.Fatalf("PlanPushCompaction() error: %v", err)
	}
	if !strings.Contains(plan.SkippedReason, "worktree has uncommitted, staged, or untracked changes") {
		t.Fatalf("SkippedReason = %q, want dirty worktree skip", plan.SkippedReason)
	}
	if !strings.Contains(plan.SkippedReason, "dirty.txt") {
		t.Fatalf("SkippedReason = %q, want dirty path", plan.SkippedReason)
	}
}

func TestPlanPushCompaction_SkipsNotAheadBeforeCheckingDirtyWorktree(t *testing.T) {
	repoPath := setupCompactableRepo(t)

	if err := os.WriteFile(filepath.Join(repoPath, "dirty.txt"), []byte("dirty"), 0644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	plan, err := PlanPushCompaction()
	if err != nil {
		t.Fatalf("PlanPushCompaction() error: %v", err)
	}
	if plan.SkippedReason != "" {
		t.Fatalf("SkippedReason = %q, want no skip warning", plan.SkippedReason)
	}
	if plan.HasCompaction() {
		t.Fatalf("HasCompaction() = true, want false")
	}
}

func TestPlanPushCompaction_SkipsDivergedHistory(t *testing.T) {
	repoPath := setupCompactableRepo(t)
	remotePath := viper.GetString("remote-url")

	commitFile(t, repoPath, "refresh-1.txt", "Refresh shell features")
	commitFile(t, repoPath, "refresh-2.txt", "Refresh shell features")
	commitAndPushRemoteFile(t, remotePath, "remote.txt", "Remote commit")
	fetchOriginMain(t, repoPath)

	plan, err := PlanPushCompaction()
	if err != nil {
		t.Fatalf("PlanPushCompaction() error: %v", err)
	}
	if plan.SkippedReason != "local and remote have diverged" {
		t.Fatalf("SkippedReason = %q, want diverged skip", plan.SkippedReason)
	}
}

func TestPushRepo_AlreadyUpToDateIsSuccess(t *testing.T) {
	setupCompactableRepo(t)

	pushed, err := PushRepo()
	if err != nil {
		t.Fatalf("PushRepo() error: %v", err)
	}
	if pushed {
		t.Fatalf("PushRepo() pushed = true, want false")
	}
}

func setupCompactableRepo(t *testing.T) string {
	t.Helper()

	viper.Reset()

	repoPath := t.TempDir()
	remotePath := createRemoteRepo(t)
	viper.Set("repo-path", repoPath)
	viper.Set("remote-url", remotePath)

	if err := os.MkdirAll(repoPath, os.ModePerm); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	_, err := InitGitRepo(repoPath, remotePath, false)
	if err != nil {
		t.Fatalf("setup repo: %v", err)
	}

	return repoPath
}

func createRemoteRepo(t *testing.T) string {
	t.Helper()

	remotePath := t.TempDir()
	tempRepoPath := t.TempDir()

	if _, err := gogit.PlainInit(remotePath, true); err != nil {
		t.Fatalf("init remote: %v", err)
	}

	tempRepo, err := gogit.PlainInit(tempRepoPath, false)
	if err != nil {
		t.Fatalf("init temp repo: %v", err)
	}

	if err := tempRepo.Storer.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main"))); err != nil {
		t.Fatalf("set temp HEAD: %v", err)
	}

	if _, err := tempRepo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{remotePath}}); err != nil {
		t.Fatalf("create temp remote: %v", err)
	}

	commitFile(t, tempRepoPath, "initial.txt", "Initial commit")

	if err := tempRepo.Push(&gogit.PushOptions{RemoteName: "origin"}); err != nil {
		t.Fatalf("push initial commit: %v", err)
	}

	remoteRepo, err := gogit.PlainOpen(remotePath)
	if err != nil {
		t.Fatalf("open remote: %v", err)
	}

	if err := remoteRepo.Storer.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main"))); err != nil {
		t.Fatalf("set remote HEAD: %v", err)
	}

	return remotePath
}

func commitFile(t *testing.T, repoPath, filename, message string) plumbing.Hash {
	t.Helper()

	r, err := gogit.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	wt, err := r.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repoPath, filename), []byte(filename), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if _, err := wt.Add(filename); err != nil {
		t.Fatalf("add file: %v", err)
	}

	hash, err := wt.Commit(message, &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("commit file: %v", err)
	}

	return hash
}

func remoteSubjects(t *testing.T, remotePath string) []string {
	t.Helper()

	r, err := gogit.PlainOpen(remotePath)
	if err != nil {
		t.Fatalf("open remote: %v", err)
	}

	ref, err := r.Reference(plumbing.NewBranchReferenceName("main"), true)
	if err != nil {
		t.Fatalf("main ref: %v", err)
	}

	iter, err := r.Log(&gogit.LogOptions{From: ref.Hash()})
	if err != nil {
		t.Fatalf("remote log: %v", err)
	}
	defer iter.Close()

	var subjects []string
	err = iter.ForEach(func(commit *object.Commit) error {
		subjects = append(subjects, commitSubject(commit))
		return nil
	})
	if err != nil {
		t.Fatalf("iterate log: %v", err)
	}

	return subjects
}

func commitAndPushRemoteFile(t *testing.T, remotePath, filename, message string) {
	t.Helper()

	clonePath := t.TempDir()
	r, err := gogit.PlainClone(clonePath, &gogit.CloneOptions{URL: remotePath})
	if err != nil {
		t.Fatalf("clone remote: %v", err)
	}

	commitFile(t, clonePath, filename, message)

	if err := r.Push(&gogit.PushOptions{}); err != nil {
		t.Fatalf("push remote commit: %v", err)
	}
}

func fetchOriginMain(t *testing.T, repoPath string) {
	t.Helper()

	r, err := gogit.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	err = r.Fetch(&gogit.FetchOptions{
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			"+refs/heads/main:refs/remotes/origin/main",
		},
	})
	if err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		t.Fatalf("fetch origin main: %v", err)
	}
}
