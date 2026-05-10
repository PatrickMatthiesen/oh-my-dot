package git

import (
	"errors"
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
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
			return PushCompactionPlan{SkippedReason: "remote branch could not be resolved"}, nil
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

	if err := r.Storer.SetReference(plumbing.NewHashReference(plan.headName, newParent)); err != nil {
		return plan, fmt.Errorf("failed to update %s: %w", plan.headName.Short(), err)
	}

	return plan, nil
}

func resolveRemoteBranchHash(r *gogit.Repository, branchName string) (plumbing.Hash, error) {
	remoteBranchRefName := plumbing.NewRemoteReferenceName("origin", branchName)

	err := r.Fetch(&gogit.FetchOptions{
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("+refs/heads/%s:%s", branchName, remoteBranchRefName)),
		},
	})
	if err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		return plumbing.ZeroHash, fmt.Errorf("failed to fetch remote branch: %w", err)
	}

	remoteRef, err := r.Reference(remoteBranchRefName, true)
	if err == nil {
		return remoteRef.Hash(), nil
	}

	remote, remoteErr := r.Remote("origin")
	if remoteErr != nil {
		return plumbing.ZeroHash, fmt.Errorf("no remote 'origin' configured: %w", remoteErr)
	}

	remoteRefs, remoteErr := remote.List(&gogit.ListOptions{})
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
