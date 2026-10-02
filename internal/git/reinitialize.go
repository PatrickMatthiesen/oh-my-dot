package git

import (
	"errors"
	"fmt"
	"os"
	"runtime"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/transport"
)

// ReinitializeRepository validates a selected origin and recovers an empty repository.
// Repositories with commits retain their history and working files.
func ReinitializeRepository(path, remoteURL string) error {
	repo, err := GetRepository(path)
	if err != nil {
		return fmt.Errorf("open existing repository: %w", err)
	}
	if remoteURL == "" {
		return fmt.Errorf("a remote URL is required for forced initialization")
	}
	options := RemoteClientOptions(path, remoteURL)
	remoteConfig := &config.RemoteConfig{Name: "origin", URLs: []string{remoteURL}, Fetch: []config.RefSpec{"+refs/heads/*:refs/remotes/origin/*"}}
	remote := gogit.NewRemote(repo.Storer, remoteConfig)
	refs, err := remote.List(&gogit.ListOptions{ClientOptions: options, Timeout: 10})
	if errors.Is(err, transport.ErrEmptyRemoteRepository) || (err == nil && len(refs) == 0) {
		return fmt.Errorf("selected remote repository has no commits; remote initialization requires a populated repository")
	}
	if err != nil {
		if IsSSHAgentError(err) {
			return fmt.Errorf("access selected remote: %w\n%s", err, sshAgentHelp(runtime.GOOS))
		}
		return fmt.Errorf("access selected remote: %w", err)
	}
	_, headErr := repo.Head()
	if headErr != nil && !errors.Is(headErr, plumbing.ErrReferenceNotFound) {
		return fmt.Errorf("inspect repository HEAD: %w", headErr)
	}
	if headErr != nil && len(refs) > 0 {
		// An interrupted legacy init can leave only .git. Never replace user files.
		entries, err := os.ReadDir(path)
		if err != nil {
			return fmt.Errorf("inspect empty repository: %w", err)
		}
		for _, entry := range entries {
			if entry.Name() != ".git" {
				return fmt.Errorf("cannot populate an empty repository containing %s; move its files or use another folder", entry.Name())
			}
		}
		var branch plumbing.ReferenceName
		var hash plumbing.Hash
		for _, ref := range refs {
			if ref.Name() == plumbing.HEAD {
				if ref.Type() == plumbing.SymbolicReference {
					branch = ref.Target()
				} else {
					hash = ref.Hash()
				}
			}
		}
		for _, ref := range refs {
			if ref.Name().IsBranch() && (ref.Name() == branch || (branch == "" && ref.Hash() == hash)) {
				branch, hash = ref.Name(), ref.Hash()
				break
			}
		}
		if !branch.IsBranch() || hash.IsZero() {
			return fmt.Errorf("selected remote has no usable default branch")
		}
		if err := remote.Fetch(&gogit.FetchOptions{ClientOptions: options}); err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) {
			return fmt.Errorf("fetch selected remote: %w", err)
		}
		worktree, err := repo.Worktree()
		if err != nil {
			return fmt.Errorf("open existing worktree: %w", err)
		}
		if err := worktree.Checkout(&gogit.CheckoutOptions{Branch: branch, Create: true, Hash: hash}); err != nil {
			return fmt.Errorf("check out remote default branch: %w", err)
		}
		cfg, err := repo.Config()
		if err != nil {
			return fmt.Errorf("read repository configuration: %w", err)
		}
		cfg.Branches[branch.Short()] = &config.Branch{Name: branch.Short(), Remote: "origin", Merge: branch}
		cfg.Remotes["origin"] = remoteConfig
		if err := repo.SetConfig(cfg); err != nil {
			return fmt.Errorf("save selected origin: %w", err)
		}
		return nil
	}
	if err := remote.Fetch(&gogit.FetchOptions{ClientOptions: options}); err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		return fmt.Errorf("fetch selected remote: %w", err)
	}
	cfg, err := repo.Config()
	if err != nil {
		return fmt.Errorf("read repository configuration: %w", err)
	}
	cfg.Remotes["origin"] = remoteConfig
	if err := repo.SetConfig(cfg); err != nil {
		return fmt.Errorf("save selected origin: %w", err)
	}
	return nil
}
