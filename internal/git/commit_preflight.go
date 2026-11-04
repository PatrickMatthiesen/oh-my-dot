package git

import (
	"fmt"

	gogit "github.com/go-git/go-git/v6"
)

// CheckCommitSafety rejects an automatic dotfile commit before mutation if it
// would include unrelated staged content or cannot resolve the author identity.
// Explicit --no-commit operations bypass this check and leave staging to the user.
func CheckCommitSafety(path string) error {
	repo, err := GetRepository(path)
	if err != nil {
		return err
	}
	if err := (&gogit.CommitOptions{}).Validate(repo); err != nil {
		return fmt.Errorf("cannot prepare a commit: %w; configure git user.name and user.email, or use --no-commit to stage only", err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("inspect index before commit: %w", err)
	}
	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("inspect index before commit: %w", err)
	}
	for path, item := range status {
		if item.Staging != gogit.Unmodified && item.Staging != gogit.Untracked {
			return fmt.Errorf("unrelated staged changes at %s; commit or unstage them first, or use --no-commit", path)
		}
	}
	return nil
}
