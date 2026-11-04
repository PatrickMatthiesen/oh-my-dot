package git

import (
	"fmt"

	gogit "github.com/go-git/go-git/v6"
)

// CheckShellFeatureCommitSafety rejects unrelated staged changes before a
// feature command modifies files. The commit path repeats the same check to
// catch staged changes made while an interactive command was open.
func CheckShellFeatureCommitSafety(repoPath string) error {
	repo, err := GetRepository(repoPath)
	if err != nil {
		return fmt.Errorf("inspect feature repository: %w", err)
	}
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
	if err := ensureNoStagedNonShellChanges(status); err != nil {
		return fmt.Errorf("commit or unstage unrelated files before changing features: %w", err)
	}
	return nil
}
