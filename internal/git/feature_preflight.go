package git

import (
	"fmt"
	"sort"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
	gogit "github.com/go-git/go-git/v6"
)

const maxDisplayedPreflightPaths = 5

func formatPathSummary(paths []string) string {
	sorted := append([]string(nil), paths...)
	sort.Strings(sorted)
	if len(sorted) <= maxDisplayedPreflightPaths {
		return strings.Join(sorted, ", ")
	}
	return fmt.Sprintf("%s, and %d more", strings.Join(sorted[:maxDisplayedPreflightPaths], ", "), len(sorted)-maxDisplayedPreflightPaths)
}

// CheckShellFeatureCommitSafety rejects unrelated staged changes before a
// feature command modifies files. The commit path repeats the same check to
// catch staged changes made while an interactive command was open.
func CheckShellFeatureCommitSafety(repoPath string) error {
	if strings.TrimSpace(repoPath) == "" {
		return fmt.Errorf("repository path is not configured")
	}
	if err := shell.CheckLocalRuntimeTracking(repoPath); err != nil {
		return err
	}
	repo, err := GetRepository(repoPath)
	if err != nil {
		return fmt.Errorf("inspect feature repository: %w", err)
	}
	defer repo.Close()
	if err := (&gogit.CommitOptions{}).Validate(repo); err != nil {
		return fmt.Errorf("cannot commit feature changes: %w; configure Git user.name and user.email, or run oh-my-dot doctor", err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("inspect feature repository: %w", err)
	}
	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("inspect staged changes before modifying features: %w", err)
	}
	if err := ensureNoStagedNonShellChangesForRepo(repoPath, status); err != nil {
		return fmt.Errorf("commit or unstage unrelated files in %s before changing features: %w", repoPath, err)
	}
	var pending []string
	for path, entry := range status {
		if entry.Staging == gogit.Unmodified && entry.Worktree == gogit.Unmodified {
			continue
		}
		if isDeviceLocalShellChangePathForRepo(repoPath, path) {
			continue
		}
		staged := entry.Staging != gogit.Unmodified && entry.Staging != gogit.Untracked
		if !staged && entry.Staging == gogit.Untracked && shell.IsUnmodifiedShellScaffold(repoPath, path) {
			continue
		}
		if staged || isCommittableShellChangePathForRepo(repoPath, path) {
			pending = append(pending, path)
		}
	}
	if len(pending) > 0 {
		return fmt.Errorf(
			"commit or stash existing shell edits in %s before changing features (%s); automatic feature commits must not include your pending work (or use --local for a device-only change without committing)",
			repoPath,
			formatPathSummary(pending),
		)
	}
	return nil
}
