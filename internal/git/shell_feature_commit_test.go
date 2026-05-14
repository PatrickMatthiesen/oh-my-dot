package git

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/spf13/viper"
)

func TestStageAndCommitShellFeatureChanges_AmendsUnpublishedManagedCommit(t *testing.T) {
	repoPath := setupCompactableRepo(t)

	writeShellFeatureFile(t, repoPath, "bash", "first", "alias first='echo first'\n")
	if committed, err := StageAndCommitShellFeatureChanges("Add shell feature: first"); err != nil || !committed {
		t.Fatalf("StageAndCommitShellFeatureChanges() committed = %t, error = %v", committed, err)
	}

	writeShellFeatureFile(t, repoPath, "bash", "second", "alias second='echo second'\n")
	if committed, err := StageAndCommitShellFeatureChanges("Refresh shell features"); err != nil || !committed {
		t.Fatalf("StageAndCommitShellFeatureChanges() committed = %t, error = %v", committed, err)
	}

	got := localSubjects(t, repoPath)
	want := []string{"Refresh shell features", "Initial commit"}
	if !reflect.DeepEqual(got[:2], want) {
		t.Fatalf("local subjects = %#v, want prefix %#v", got[:2], want)
	}
}

func TestStageAndCommitShellFeatureChanges_DoesNotAmendPushedManagedCommit(t *testing.T) {
	repoPath := setupCompactableRepo(t)

	writeShellFeatureFile(t, repoPath, "bash", "first", "alias first='echo first'\n")
	if committed, err := StageAndCommitShellFeatureChanges("Add shell feature: first"); err != nil || !committed {
		t.Fatalf("StageAndCommitShellFeatureChanges() committed = %t, error = %v", committed, err)
	}
	if pushed, err := PushRepo(); err != nil || !pushed {
		t.Fatalf("PushRepo() pushed = %t, error = %v", pushed, err)
	}

	writeShellFeatureFile(t, repoPath, "bash", "second", "alias second='echo second'\n")
	if committed, err := StageAndCommitShellFeatureChanges("Refresh shell features"); err != nil || !committed {
		t.Fatalf("StageAndCommitShellFeatureChanges() committed = %t, error = %v", committed, err)
	}

	got := localSubjects(t, repoPath)
	want := []string{"Refresh shell features", "Add shell feature: first", "Initial commit"}
	if !reflect.DeepEqual(got[:3], want) {
		t.Fatalf("local subjects = %#v, want prefix %#v", got[:3], want)
	}
}

func TestStageAndCommitShellFeatureChanges_DoesNotAmendUserAuthoredCommit(t *testing.T) {
	repoPath := setupCompactableRepo(t)
	commitFile(t, repoPath, "manual.txt", "Manual commit")

	writeShellFeatureFile(t, repoPath, "bash", "first", "alias first='echo first'\n")
	if committed, err := StageAndCommitShellFeatureChanges("Add shell feature: first"); err != nil || !committed {
		t.Fatalf("StageAndCommitShellFeatureChanges() committed = %t, error = %v", committed, err)
	}

	got := localSubjects(t, repoPath)
	want := []string{"Add shell feature: first", "Manual commit", "Initial commit"}
	if !reflect.DeepEqual(got[:3], want) {
		t.Fatalf("local subjects = %#v, want prefix %#v", got[:3], want)
	}
}

func TestStageAndCommitShellFeatureChanges_DoesNotAmendManagedSubjectWithNonShellChanges(t *testing.T) {
	repoPath := setupCompactableRepo(t)
	commitFile(t, repoPath, "manual.txt", "Refresh shell features")

	writeShellFeatureFile(t, repoPath, "bash", "first", "alias first='echo first'\n")
	if committed, err := StageAndCommitShellFeatureChanges("Add shell feature: first"); err != nil || !committed {
		t.Fatalf("StageAndCommitShellFeatureChanges() committed = %t, error = %v", committed, err)
	}

	got := localSubjects(t, repoPath)
	want := []string{"Add shell feature: first", "Refresh shell features", "Initial commit"}
	if !reflect.DeepEqual(got[:3], want) {
		t.Fatalf("local subjects = %#v, want prefix %#v", got[:3], want)
	}
}

func TestStageAndCommitShellFeatureChanges_CommitsWhenRemoteCannotBeResolved(t *testing.T) {
	viper.Reset()

	repoPath := t.TempDir()
	viper.Set("repo-path", repoPath)
	if _, err := InitGitRepo(repoPath, ""); err != nil {
		t.Fatalf("init repo: %v", err)
	}
	commitFile(t, repoPath, "initial.txt", "Initial commit")

	writeShellFeatureFile(t, repoPath, "bash", "first", "alias first='echo first'\n")
	if committed, err := StageAndCommitShellFeatureChanges("Add shell feature: first"); err != nil || !committed {
		t.Fatalf("StageAndCommitShellFeatureChanges() committed = %t, error = %v", committed, err)
	}

	writeShellFeatureFile(t, repoPath, "bash", "second", "alias second='echo second'\n")
	if committed, err := StageAndCommitShellFeatureChanges("Refresh shell features"); err != nil || !committed {
		t.Fatalf("StageAndCommitShellFeatureChanges() committed = %t, error = %v", committed, err)
	}

	got := localSubjects(t, repoPath)
	want := []string{"Refresh shell features", "Add shell feature: first", "Initial commit"}
	if !reflect.DeepEqual(got[:3], want) {
		t.Fatalf("local subjects = %#v, want prefix %#v", got[:3], want)
	}
}

func TestStageAndCommitShellFeatureChanges_DropsUnpublishedManagedCommitWhenNetEmpty(t *testing.T) {
	repoPath := setupCompactableRepo(t)
	featurePath := writeShellFeatureFile(t, repoPath, "bash", "first", "alias first='echo first'\n")
	if committed, err := StageAndCommitShellFeatureChanges("Add shell feature: first"); err != nil || !committed {
		t.Fatalf("StageAndCommitShellFeatureChanges() committed = %t, error = %v", committed, err)
	}

	if err := os.Remove(featurePath); err != nil {
		t.Fatalf("remove feature file: %v", err)
	}
	if committed, err := StageAndCommitShellFeatureChanges("Remove shell feature: first"); err != nil || !committed {
		t.Fatalf("StageAndCommitShellFeatureChanges() committed = %t, error = %v", committed, err)
	}

	got := localSubjects(t, repoPath)
	want := []string{"Initial commit"}
	if !reflect.DeepEqual(got[:1], want) {
		t.Fatalf("local subjects = %#v, want prefix %#v", got[:1], want)
	}
}

func TestStageAndCommitShellFeatureChanges_RejectsUnrelatedStagedChanges(t *testing.T) {
	repoPath := setupCompactableRepo(t)

	r, err := gogit.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	worktree, err := r.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repoPath, "manual.txt"), []byte("manual"), 0644); err != nil {
		t.Fatalf("write manual file: %v", err)
	}
	if _, err := worktree.Add("manual.txt"); err != nil {
		t.Fatalf("stage manual file: %v", err)
	}

	writeShellFeatureFile(t, repoPath, "bash", "first", "alias first='echo first'\n")
	committed, err := StageAndCommitShellFeatureChanges("Add shell feature: first")
	if err == nil {
		t.Fatalf("StageAndCommitShellFeatureChanges() error = nil, want staged unrelated change error")
	}
	if committed {
		t.Fatalf("StageAndCommitShellFeatureChanges() committed = true, want false")
	}
	if !strings.Contains(err.Error(), "non-shell changes are staged: manual.txt") {
		t.Fatalf("StageAndCommitShellFeatureChanges() error = %q, want staged manual.txt error", err)
	}

	status, err := worktree.Status()
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.File("omd-shells/bash/features/first.sh").Staging != gogit.Untracked {
		t.Fatalf("shell feature staging = %q, want untracked", status.File("omd-shells/bash/features/first.sh").Staging)
	}
}

func writeShellFeatureFile(t *testing.T, repoPath, shellName, featureName, content string) string {
	t.Helper()

	path := filepath.Join(repoPath, "omd-shells", shellName, "features", featureName+".sh")
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		t.Fatalf("mkdir feature dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write feature file: %v", err)
	}

	return path
}

func localSubjects(t *testing.T, repoPath string) []string {
	t.Helper()

	r, err := gogit.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	head, err := r.Head()
	if err != nil {
		t.Fatalf("head: %v", err)
	}

	iter, err := r.Log(&gogit.LogOptions{From: head.Hash()})
	if err != nil {
		t.Fatalf("log: %v", err)
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
