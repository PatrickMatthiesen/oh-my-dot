package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v6"
	"github.com/spf13/viper"
)

func TestRemoveFileRejectsPathsOutsideFilesDirectory(t *testing.T) {
	repoPath := t.TempDir()
	if _, err := gogit.PlainInit(repoPath, false); err != nil {
		t.Fatalf("PlainInit() error = %v", err)
	}

	originalRepoPath := viper.GetString("repo-path")
	viper.Set("repo-path", repoPath)
	t.Cleanup(func() {
		viper.Set("repo-path", originalRepoPath)
	})

	outsidePath := filepath.Join(repoPath, "outside.txt")
	if err := os.WriteFile(outsidePath, []byte("outside"), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	tests := []string{
		filepath.Join("..", "outside.txt"),
		outsidePath,
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			err := RemoveFile(input)
			if err == nil {
				t.Fatal("expected RemoveFile to reject path outside files directory")
			}
			if !strings.Contains(err.Error(), "outside the repository files directory") {
				t.Fatalf("expected outside-files error, got %v", err)
			}
		})
	}
}

func TestRemoveFileAcceptsRootPrefixedFileName(t *testing.T) {
	repoPath := t.TempDir()
	repo, err := gogit.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("PlainInit() error = %v", err)
	}

	originalRepoPath := viper.GetString("repo-path")
	viper.Set("repo-path", repoPath)
	t.Cleanup(func() {
		viper.Set("repo-path", originalRepoPath)
	})

	filesPath := filepath.Join(repoPath, "files")
	if err := os.MkdirAll(filesPath, 0755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	filePath := filepath.Join(filesPath, "test.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree() error = %v", err)
	}
	if _, err := worktree.Add(filepath.Join("files", "test.txt")); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	err = RemoveFile(string(filepath.Separator) + "test.txt")
	if err != nil {
		t.Fatalf("RemoveFile() error = %v", err)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected file to be removed, stat error = %v", err)
	}
}
