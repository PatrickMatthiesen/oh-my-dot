package git

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/go-git/go-git/v6/plumbing/transport"
	"github.com/spf13/viper"
)

// PushCompactionGroup describes consecutive unpublished commits that can be squashed.
type PushCompactionGroup struct {
	Subject string
	Count   int
}

// PushCompactionPlan describes eligible unpublished commit compaction work.
type PushCompactionPlan struct {
	OriginalCommitCount  int
	CompactedCommitCount int
	Groups               []PushCompactionGroup
	SkippedReason        string

	repoPath   string
	headHash   plumbing.Hash
	commits    []*object.Commit
	remoteHash plumbing.Hash
	headName   plumbing.ReferenceName
}

// HasCompaction reports whether the plan would rewrite any commits.
func (p PushCompactionPlan) HasCompaction() bool {
	return len(p.Groups) > 0
}

// PlanPushCompaction inspects unpublished local commits for consecutive duplicate subjects.
func PlanPushCompaction() (PushCompactionPlan, error) {
	repoPath := strings.TrimSpace(viper.GetString("repo-path"))
	if repoPath == "" {
		return PushCompactionPlan{}, fmt.Errorf("repository path is not set")
	}

	r, err := gogit.PlainOpen(repoPath)
	if err != nil {
		return PushCompactionPlan{}, fmt.Errorf("failed to open repository: %w", err)
	}

	headRef, err := r.Head()
	if err != nil {
		return PushCompactionPlan{}, fmt.Errorf("failed to get current branch: %w", err)
	}
	if !headRef.Name().IsBranch() {
		return PushCompactionPlan{SkippedReason: "cannot compact from detached HEAD"}, nil
	}

	remoteHash, err := resolveRemoteBranchHash(r, headRef.Name().Short())
	if err != nil {
		if errors.Is(err, errRemoteBranchNotFound) {
			return PushCompactionPlan{SkippedReason: "branch has not been pushed yet"}, nil
		}
		return PushCompactionPlan{}, err
	}

	headCommit, err := r.CommitObject(headRef.Hash())
	if err != nil {
		return PushCompactionPlan{}, fmt.Errorf("failed to inspect HEAD commit: %w", err)
	}

	remoteCommit, err := r.CommitObject(remoteHash)
	if err != nil {
		return PushCompactionPlan{}, fmt.Errorf("failed to inspect remote commit: %w", err)
	}

	remoteHasLocal, remoteDepthExceeded, err := commitContainsAncestor(remoteCommit, headRef.Hash(), maxAncestorSearchDepth)
	if err != nil {
		return PushCompactionPlan{}, fmt.Errorf("failed to compare local and remote commits: %w", err)
	}
	if remoteHasLocal || remoteDepthExceeded {
		return PushCompactionPlan{}, nil
	}

	localHasRemote, localDepthExceeded, err := commitContainsAncestor(headCommit, remoteHash, maxAncestorSearchDepth)
	if err != nil {
		return PushCompactionPlan{}, fmt.Errorf("failed to compare local and remote commits: %w", err)
	}
	if !localHasRemote {
		reason := "local and remote have diverged"
		if localDepthExceeded {
			reason = "remote base is too far behind to compact safely"
		}
		return PushCompactionPlan{SkippedReason: reason}, nil
	}

	worktree, err := r.Worktree()
	if err != nil {
		return PushCompactionPlan{}, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return PushCompactionPlan{}, fmt.Errorf("failed to read worktree status: %w", err)
	}
	if !status.IsClean() {
		return PushCompactionPlan{SkippedReason: fmt.Sprintf("worktree has uncommitted, staged, or untracked changes: %s", formatDirtyStatus(status))}, nil
	}

	commitsNewestFirst, err := firstParentCommitsUntil(r, headCommit, remoteHash)
	if err != nil {
		return PushCompactionPlan{}, err
	}

	commits := reverseCommits(commitsNewestFirst)
	groups := planDuplicateSubjectGroups(commits)

	return PushCompactionPlan{
		OriginalCommitCount:  len(commits),
		CompactedCommitCount: len(commits) - duplicateCommitSavings(groups),
		Groups:               groups,
		repoPath:             filepath.Clean(repoPath),
		headHash:             headRef.Hash(),
		commits:              commits,
		remoteHash:           remoteHash,
		headName:             headRef.Name(),
	}, nil
}

// CompactPushHistory rewrites eligible unpublished commits according to the supplied plan.
func CompactPushHistory(plan PushCompactionPlan) (PushCompactionPlan, error) {
	if !plan.HasCompaction() {
		return plan, nil
	}

	r, err := gogit.PlainOpen(viper.GetString("repo-path"))
	if err != nil {
		return plan, fmt.Errorf("failed to open repository: %w", err)
	}

	if filepath.Clean(viper.GetString("repo-path")) != plan.repoPath {
		return plan, fmt.Errorf("repository changed since compaction was planned; retry push")
	}
	head, err := r.Head()
	if err != nil {
		return plan, fmt.Errorf("inspect current branch: %w", err)
	}
	if head.Name() != plan.headName || head.Hash() != plan.headHash {
		return plan, fmt.Errorf("branch changed since compaction was planned; retry push")
	}
	worktree, err := r.Worktree()
	if err != nil {
		return plan, fmt.Errorf("inspect compaction worktree: %w", err)
	}
	status, err := worktree.Status()
	if err != nil {
		return plan, fmt.Errorf("inspect compaction worktree: %w", err)
	}
	if !status.IsClean() {
		return plan, fmt.Errorf("worktree changed since compaction was planned; commit or stash changes and retry push")
	}
	newParent := plan.remoteHash
	for i := 0; i < len(plan.commits); {
		runEnd := i + 1
		subject := commitSubject(plan.commits[i])
		for runEnd < len(plan.commits) && commitSubject(plan.commits[runEnd]) == subject {
			runEnd++
		}

		source := plan.commits[runEnd-1]
		newParent, err = writeReparentedCommit(r, source, newParent)
		if err != nil {
			return plan, err
		}
		i = runEnd
	}

	if err := r.Storer.CheckAndSetReference(plumbing.NewHashReference(plan.headName, newParent), plumbing.NewHashReference(plan.headName, plan.headHash)); err != nil {
		return plan, fmt.Errorf("failed to update %s: %w", plan.headName.Short(), err)
	}

	return plan, nil
}

func resolveRemoteBranchHash(r *gogit.Repository, branchName string) (plumbing.Hash, error) {
	remoteBranchRefName := plumbing.NewRemoteReferenceName("origin", branchName)

	origin, err := r.Remote("origin")
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("inspect origin: %w", err)
	}
	var fetchErr error
	if IsSSHRemote(origin.Config().URLs[0]) {
		worktree, err := r.Worktree()
		if err != nil {
			return plumbing.ZeroHash, fmt.Errorf("open worktree: %w", err)
		}
		rooted, ok := worktree.Filesystem().(interface{ Root() string })
		if !ok {
			return plumbing.ZeroHash, fmt.Errorf("native SSH fetch requires a filesystem-backed worktree")
		}
		fetchErr = FetchSSH(rooted.Root(), origin.Config().URLs[0], fmt.Sprintf("+refs/heads/%s:%s", branchName, remoteBranchRefName))
	} else {
		fetchErr = r.Fetch(&gogit.FetchOptions{
			ClientOptions: repositoryClientOptions(r),
			RemoteName:    "origin",
			RefSpecs: []config.RefSpec{
				config.RefSpec(fmt.Sprintf("+refs/heads/%s:%s", branchName, remoteBranchRefName)),
			},
		})
	}
	if fetchErr != nil && !errors.Is(fetchErr, gogit.NoErrAlreadyUpToDate) {
		// A new remote has no branch to compact against. Native Git reports
		// this as a failed refspec; list once to distinguish it from lost access.
		refs, listErr := ListRemote(viper.GetString("repo-path"), origin.Config().URLs[0], 15)
		if errors.Is(listErr, transport.ErrEmptyRemoteRepository) {
			return plumbing.ZeroHash, fmt.Errorf("%w: %s", errRemoteBranchNotFound, branchName)
		}
		if listErr == nil {
			found := false
			for _, ref := range refs {
				if ref.Name() == plumbing.NewBranchReferenceName(branchName) {
					found = true
					break
				}
			}
			if !found {
				return plumbing.ZeroHash, fmt.Errorf("%w: %s", errRemoteBranchNotFound, branchName)
			}
		}
		return plumbing.ZeroHash, fmt.Errorf("failed to fetch remote branch: %w", fetchErr)
	}

	remoteRef, err := r.Reference(remoteBranchRefName, true)
	if err == nil {
		return remoteRef.Hash(), nil
	}

	remote, remoteErr := r.Remote("origin")
	if remoteErr != nil {
		return plumbing.ZeroHash, fmt.Errorf("no remote 'origin' configured: %w", remoteErr)
	}

	remoteRefs, remoteErr := ListRemote(viper.GetString("repo-path"), remote.Config().URLs[0], 15)
	if remoteErr != nil {
		return plumbing.ZeroHash, fmt.Errorf("unable to access remote repository: %w", remoteErr)
	}

	branchRefName := plumbing.NewBranchReferenceName(branchName)
	for _, ref := range remoteRefs {
		if ref.Name() == branchRefName {
			return ref.Hash(), nil
		}
	}

	return plumbing.ZeroHash, fmt.Errorf("%w: %s", errRemoteBranchNotFound, branchRefName.String())
}

var errRemoteBranchNotFound = errors.New("remote branch not found")

func firstParentCommitsUntil(r *gogit.Repository, start *object.Commit, stop plumbing.Hash) ([]*object.Commit, error) {
	var commits []*object.Commit
	current := start

	for current.Hash != stop {
		if current.NumParents() > 1 {
			return nil, fmt.Errorf("cannot compact history containing merge commit %s", current.Hash.String())
		}

		commits = append(commits, current)

		parent, err := current.Parent(0)
		if err != nil {
			return nil, fmt.Errorf("remote branch is not an ancestor of local HEAD")
		}

		if _, err := r.CommitObject(parent.Hash); err != nil {
			return nil, fmt.Errorf("failed to inspect parent commit: %w", err)
		}

		current = parent
	}

	return commits, nil
}

func reverseCommits(commits []*object.Commit) []*object.Commit {
	reversed := make([]*object.Commit, len(commits))
	for i := range commits {
		reversed[i] = commits[len(commits)-1-i]
	}
	return reversed
}

func planDuplicateSubjectGroups(commits []*object.Commit) []PushCompactionGroup {
	var groups []PushCompactionGroup
	for i := 0; i < len(commits); {
		subject := commitSubject(commits[i])
		runEnd := i + 1
		for runEnd < len(commits) && commitSubject(commits[runEnd]) == subject {
			runEnd++
		}
		if runEnd-i > 1 {
			groups = append(groups, PushCompactionGroup{Subject: subject, Count: runEnd - i})
		}
		i = runEnd
	}
	return groups
}

func duplicateCommitSavings(groups []PushCompactionGroup) int {
	savings := 0
	for _, group := range groups {
		savings += group.Count - 1
	}
	return savings
}

func commitSubject(commit *object.Commit) string {
	message := strings.TrimRight(commit.Message, "\n")
	if i := strings.IndexByte(message, '\n'); i >= 0 {
		return message[:i]
	}
	return message
}

func formatDirtyStatus(status gogit.Status) string {
	dirty := make([]string, 0, len(status))
	for path, fileStatus := range status {
		if fileStatus.Staging == gogit.Unmodified && fileStatus.Worktree == gogit.Unmodified {
			continue
		}

		dirty = append(dirty, fmt.Sprintf("%s(%c%c)", path, fileStatus.Staging, fileStatus.Worktree))
	}

	return strings.Join(dirty, ", ")
}

func writeReparentedCommit(r *gogit.Repository, source *object.Commit, parent plumbing.Hash) (plumbing.Hash, error) {
	commit := &object.Commit{
		Author:       source.Author,
		Committer:    source.Committer,
		Message:      source.Message,
		TreeHash:     source.TreeHash,
		ParentHashes: []plumbing.Hash{parent},
	}

	obj := &plumbing.MemoryObject{}
	if err := commit.Encode(obj); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to encode compacted commit: %w", err)
	}

	hash, err := r.Storer.SetEncodedObject(obj)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to write compacted commit: %w", err)
	}

	return hash, nil
}
