package agentcmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func copyDirectory(srcDir, dstDir string, force bool) error {
	srcAbs, err := filepath.Abs(srcDir)
	if err != nil {
		return fmt.Errorf("failed to resolve source directory: %w", err)
	}
	dstAbs, err := filepath.Abs(dstDir)
	if err != nil {
		return fmt.Errorf("failed to resolve destination directory: %w", err)
	}
	if srcAbs == dstAbs {
		return fmt.Errorf("source and destination are the same directory")
	}

	if _, err := os.Stat(dstAbs); err == nil {
		if !force {
			return fmt.Errorf("%s already exists (use --force to overwrite)", dstAbs)
		}
		if err := os.RemoveAll(dstAbs); err != nil {
			return fmt.Errorf("failed to remove existing destination: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("cannot inspect destination: %w", err)
	}

	if err := os.MkdirAll(dstAbs, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	return filepath.WalkDir(srcAbs, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		name := entry.Name()
		if entry.IsDir() && (name == ".git" || name == ".svn" || name == ".hg") {
			return filepath.SkipDir
		}

		relativePath, err := filepath.Rel(srcAbs, path)
		if err != nil {
			return fmt.Errorf("failed to resolve relative path: %w", err)
		}
		if relativePath == "." {
			return nil
		}

		targetPath := filepath.Join(dstAbs, relativePath)
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("failed to inspect %s: %w", path, err)
		}

		if entry.IsDir() {
			if err := os.MkdirAll(targetPath, info.Mode().Perm()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
			return nil
		}

		if entry.Type()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("failed to read symlink %s: %w", path, err)
			}
			if err := os.Symlink(linkTarget, targetPath); err != nil {
				return fmt.Errorf("failed to create symlink %s: %w", targetPath, err)
			}
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}
		if err := os.WriteFile(targetPath, content, info.Mode().Perm()); err != nil {
			return fmt.Errorf("failed to write %s: %w", targetPath, err)
		}

		return nil
	})
}

func removeDirectory(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to remove %s: %w", path, err)
	}
	return nil
}

func ensurePathInside(parent, child string) error {
	parentAbs, err := filepath.Abs(parent)
	if err != nil {
		return fmt.Errorf("failed to resolve parent path: %w", err)
	}
	childAbs, err := filepath.Abs(child)
	if err != nil {
		return fmt.Errorf("failed to resolve child path: %w", err)
	}

	relative, err := filepath.Rel(parentAbs, childAbs)
	if err != nil {
		return fmt.Errorf("failed to compare paths: %w", err)
	}
	if relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || relative == ".." {
		return fmt.Errorf("%s is outside %s", childAbs, parentAbs)
	}
	return nil
}
