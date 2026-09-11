package doctor

import (
	"errors"
	"fmt"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/transport"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
)

// checkRepository runs once, independently of configured shells.
func checkRepository(ctx context) []result {
	fileops.ColorPrintfn(fileops.Cyan, "\nChecking repository...")
	var results []result
	repo, err := git.PlainOpen(ctx.repoPath)
	if err != nil {
		return addResult(results, ctx, errorResult("Git repository", fmt.Sprintf("Cannot open %s: %v", ctx.repoPath, err), false), nil)
	}
	results = addResult(results, ctx, okResult("Git repository"), nil)

	// Use the same configuration resolution and validation as Commit, including
	// author overrides and repository, global, and system identity settings.
	options := &git.CommitOptions{}
	if err := options.Validate(repo); err != nil {
		message := fmt.Sprintf("Cannot prepare a commit: %v", err)
		if errors.Is(err, git.ErrMissingAuthor) {
			message = fmt.Sprintf("Missing Git author identity. Run git -C %q config user.name \"Your Name\" and git -C %q config user.email \"you@example.com\" (or set them with git config --global).", ctx.repoPath, ctx.repoPath)
		}
		results = addResult(results, ctx, errorResult("Git commit identity", message, false), nil)
	} else {
		results = addResult(results, ctx, okResult("Git commit identity"), nil)
	}

	remote, err := repo.Remote("origin")
	if errors.Is(err, git.ErrRemoteNotFound) {
		return addResult(results, ctx, warningResult("Git remote", "No origin remote configured; local operations work, but push/pull require a remote.", false), nil)
	}
	if err != nil {
		return addResult(results, ctx, warningResult("Git remote", fmt.Sprintf("Cannot inspect origin: %v", err), false), nil)
	}

	// Listing refs checks connectivity and read access, not push permission.
	// Bound the transport check so an unreachable remote cannot wait forever.
	// An empty remote was reached successfully and is ready for its first push.
	if _, err := remote.List(&git.ListOptions{Timeout: 10}); err != nil && !errors.Is(err, transport.ErrEmptyRemoteRepository) {
		return addResult(results, ctx, warningResult("Git remote", fmt.Sprintf("Cannot access origin (check its URL, credentials, and network): %v. Local operations can continue.", err), false), nil)
	}
	return addResult(results, ctx, okResult("Git remote read access"), nil)
}
