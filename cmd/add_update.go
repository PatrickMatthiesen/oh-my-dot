package cmd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/filemode"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/go-git/go-git/v6/utils/convert"
	"github.com/spf13/cobra"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/repopath"
)

func processManagedAdd(cmd *cobra.Command, repoPath, live, key string, liveInfo os.FileInfo, update bool) error {
	explicit, _ := cmd.Flags().GetString("as")
	if explicit != "" || cmd.Flags().Changed("as") {
		if err := repopath.Validate(explicit); err != nil {
			return fmt.Errorf("invalid --as path: %w", err)
		}
		if explicit != key {
			return fmt.Errorf("local file is already managed as %q; --as cannot rename it to %q", key, explicit)
		}
	}
	stored, err := repopath.Resolve(repoPath, key)
	if err != nil {
		return fmt.Errorf("resolve managed source: %w", err)
	}
	storedInfo, err := os.Lstat(stored)
	if err != nil {
		return fmt.Errorf("inspect managed source: %w", err)
	}
	if !storedInfo.Mode().IsRegular() {
		return fmt.Errorf("managed source %s must be a regular file", stored)
	}
	resolvedInfo, err := os.Stat(live)
	if err != nil {
		return fmt.Errorf("inspect managed local file: %w", err)
	}
	same := os.SameFile(storedInfo, resolvedInfo)
	if liveInfo.Mode()&os.ModeSymlink != 0 && !same {
		return fmt.Errorf("managed destination is a symlink to another file; restore its intended target before capturing changes")
	}
	if !update {
		if !same {
			return fmt.Errorf("%s is managed as %q but its local file was replaced; use 'add %s --update' to capture it after reviewing both copies", live, key, live)
		}
		fileops.ColorPrintfn(fileops.Green, "Already managed as %s (use --update to stage and commit current edits)", key)
		return nil
	}
	repo, err := git.GetRepository(repoPath)
	if err != nil {
		return err
	}
	defer repo.Close()
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("inspect capture worktree: %w", err)
	}
	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("inspect capture state: %w", err)
	}
	relative := filepath.ToSlash(filepath.Join("files", filepath.FromSlash(key)))
	current := status[relative]
	noCommit, _ := cmd.Flags().GetBool("no-commit")

	// A capture never changes registration metadata or another index entry.
	if !noCommit {
		if err := git.CheckCommitSafety(repoPath); err != nil {
			return err
		}
	}
	if !same {
		storedData, err := os.ReadFile(stored)
		if err != nil {
			return fmt.Errorf("read stored content: %w", err)
		}
		liveData, err := os.ReadFile(live)
		if err != nil {
			return fmt.Errorf("read local content: %w", err)
		}
		equal := bytes.Equal(storedData, liveData) && storedInfo.Mode().Perm() == resolvedInfo.Mode().Perm()
		if !equal {
			changed, err := capturedSourceChanged(repo, relative, storedData, storedInfo.Mode())
			if err != nil {
				return err
			}
			staged := current != nil && current.Staging != gogit.Unmodified && current.Staging != gogit.Untracked
			if changed || staged {
				return fmt.Errorf("stored file %q also has uncommitted changes; compare and resolve the stored and local copies before using --update", key)
			}
		}
		if err := relinkCapturedFile(live, stored, liveInfo, storedInfo, storedData); err != nil {
			return err
		}
	}
	// Explicit capture must hash the file. A same-size replacement within the
	// index timestamp resolution can otherwise look unchanged to stat caches.
	if err := worktree.AddWithOptions(&gogit.AddOptions{Path: relative, SkipStatus: true}); err != nil {
		return fmt.Errorf("stage captured file: %w", err)
	}
	status, err = worktree.Status()
	if err != nil {
		return fmt.Errorf("inspect captured change: %w", err)
	}
	captured := status[relative]
	if captured == nil || captured.Staging == gogit.Unmodified || captured.Staging == gogit.Untracked {
		fileops.ColorPrintfn(fileops.Green, "Captured %s; stored content is unchanged", key)
		return nil
	}
	if !noCommit {
		if err := git.Commit("Updated " + key); err != nil {
			return fmt.Errorf("commit captured file: %w", err)
		}
	}
	fileops.ColorPrintfn(fileops.Green, "Captured %s", key)
	return nil
}

// relinkCapturedFile restores the existing hard-link representation without
// replacing the user's local file. A failed rename preserves the old source.
func relinkCapturedFile(live, stored string, expected, expectedStored os.FileInfo, storedData []byte) error {
	placeholder, err := os.CreateTemp(filepath.Dir(stored), ".omd-capture-*")
	if err != nil {
		return fmt.Errorf("prepare capture: %w", err)
	}
	temp := placeholder.Name()
	if err := placeholder.Close(); err != nil {
		os.Remove(temp)
		return fmt.Errorf("close capture placeholder: %w", err)
	}
	if err := os.Remove(temp); err != nil {
		return fmt.Errorf("prepare capture link: %w", err)
	}
	defer os.Remove(temp)
	if err := os.Link(live, temp); err != nil {
		return fmt.Errorf("restore managed hard link (local file and repository must share a filesystem): %w", err)
	}
	linked, err := os.Lstat(temp)
	if err != nil || !os.SameFile(expected, linked) {
		return fmt.Errorf("local file changed while capturing; retry --update")
	}
	currentLive, err := os.Lstat(live)
	if err != nil || !os.SameFile(expected, currentLive) {
		return fmt.Errorf("local file changed while capturing; retry --update")
	}
	currentStored, err := os.Lstat(stored)
	if err != nil || !os.SameFile(expectedStored, currentStored) {
		return fmt.Errorf("stored source changed while capturing; compare both copies before retrying")
	}
	currentData, err := os.ReadFile(stored)
	if err != nil {
		return fmt.Errorf("recheck stored source: %w", err)
	}
	if !bytes.Equal(currentData, storedData) || currentStored.Mode().Perm() != expectedStored.Mode().Perm() {
		return fmt.Errorf("stored source changed while capturing; compare both copies before retrying")
	}
	// Go's rename replaces an existing regular file on Windows and Unix. No
	// delete-first fallback is used, so failure leaves the stored source intact.
	if err := os.Rename(temp, stored); err != nil {
		return fmt.Errorf("replace managed source: %w", err)
	}
	return nil
}

// capturedSourceChanged compares the stored bytes with HEAD, rather than using
// timestamp-based worktree status to decide whether overwriting them is safe.
func capturedSourceChanged(repo *gogit.Repository, path string, data []byte, mode os.FileMode) (bool, error) {
	head, err := repo.Head()
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect stored history: %w", err)
	}
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return false, fmt.Errorf("inspect stored commit: %w", err)
	}
	file, err := commit.File(path)
	if errors.Is(err, object.ErrFileNotFound) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect stored file history: %w", err)
	}
	config, err := repo.Config()
	if err != nil {
		return false, fmt.Errorf("inspect capture configuration: %w", err)
	}
	// Match go-git's normal clean conversion before comparing the object hash.
	if config.Core.AutoCRLF == "true" || config.Core.AutoCRLF == "input" {
		stat, err := convert.GetStat(bufio.NewReader(bytes.NewReader(data)))
		if err != nil {
			return false, fmt.Errorf("inspect stored line endings: %w", err)
		}
		if !stat.IsBinary() {
			var normalized bytes.Buffer
			if _, err := io.Copy(convert.NewLFWriter(&normalized), bytes.NewReader(data)); err != nil {
				return false, fmt.Errorf("normalize stored line endings: %w", err)
			}
			data = normalized.Bytes()
		}
	}
	expected, err := file.Contents()
	if err != nil {
		return false, fmt.Errorf("read stored history: %w", err)
	}
	if !bytes.Equal([]byte(expected), data) {
		return true, nil
	}
	if config.Core.FileMode {
		actual, err := filemode.NewFromOSFileMode(mode)
		if err != nil {
			return false, fmt.Errorf("inspect stored file mode: %w", err)
		}
		return actual != file.Mode, nil
	}
	return false, nil
}
