package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// preserveManagedSymlink turns a link into an independent local file before
// its repository source is removed. Other local files and links are untouched.
func preserveManagedSymlink(source, target string) error {
	info, err := os.Lstat(target)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect local file: %w", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return nil
	}
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		// An unrelated broken symlink has no local content to preserve.
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("resolve local symlink: %w", err)
	}
	canonicalSource, err := filepath.EvalSymlinks(source)
	if err != nil {
		return fmt.Errorf("resolve managed source: %w", err)
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return fmt.Errorf("resolve local file path: %w", err)
	}
	canonicalSource, err = filepath.Abs(canonicalSource)
	if err != nil {
		return fmt.Errorf("resolve source path: %w", err)
	}
	if filepath.Clean(resolved) != filepath.Clean(canonicalSource) {
		return nil
	}
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open managed content: %w", err)
	}
	defer input.Close()
	sourceInfo, err := input.Stat()
	if err != nil {
		return fmt.Errorf("inspect managed content: %w", err)
	}
	if !sourceInfo.Mode().IsRegular() {
		return fmt.Errorf("managed source must be a regular file")
	}
	output, err := os.CreateTemp(filepath.Dir(target), ".omd-keep-*")
	if err != nil {
		return fmt.Errorf("prepare local copy: %w", err)
	}
	temp := output.Name()
	defer os.Remove(temp)
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return fmt.Errorf("copy managed content: %w", err)
	}
	if err := output.Chmod(sourceInfo.Mode().Perm()); err != nil {
		output.Close()
		return fmt.Errorf("preserve file permissions: %w", err)
	}
	if err := output.Close(); err != nil {
		return fmt.Errorf("close local copy: %w", err)
	}
	current, err := os.Lstat(target)
	if err != nil || !os.SameFile(info, current) {
		return fmt.Errorf("local symlink changed while preparing its content; retry removal")
	}
	// Rename the symlink aside first. Unlike remove-and-recreate, this preserves
	// the original link for rollback on Windows as well as Unix.
	backup, err := os.CreateTemp(filepath.Dir(target), ".omd-link-*")
	if err != nil {
		return fmt.Errorf("prepare symlink backup: %w", err)
	}
	backupPath := backup.Name()
	if err := backup.Close(); err != nil {
		os.Remove(backupPath)
		return fmt.Errorf("close symlink backup: %w", err)
	}
	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("prepare backup path: %w", err)
	}
	if err := os.Rename(target, backupPath); err != nil {
		return fmt.Errorf("preserve original symlink: %w", err)
	}
	if err := os.Rename(temp, target); err != nil {
		restoreErr := os.Rename(backupPath, target)
		return fmt.Errorf("materialize local file: %w", errors.Join(err, restoreErr))
	}
	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("remove old symlink backup %s: %w", backupPath, err)
	}
	return nil
}
