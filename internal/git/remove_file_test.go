package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v5"
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
