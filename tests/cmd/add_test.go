package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	internalgit "github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/symlink"
	"github.com/go-git/go-git/v6"
	"github.com/spf13/viper"
)

func Test_Add_AllowsLocalOnlyRepoWithoutRemote(t *testing.T) {
	_, repoPath := setupTestConfig(t)
	viper.Set("initialized", true)

	if _, err := git.PlainInit(repoPath, false); err != nil {
		t.Fatalf("init local repo: %v", err)
	}

	sourcePath := filepath.Join(t.TempDir(), "sample.txt")
	if err := os.WriteFile(sourcePath, []byte("local only"), 0644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	output, err := captureCommandOutput(t, []string{"add", "--no-interactive", "--no-commit", sourcePath})
	if err != nil {
		t.Fatalf("add local-only repo error: %v\noutput:\n%s", err, output)
	}

	if strings.Contains(output, "Unable to verify remote push access") {
		t.Fatalf("expected local-only add to skip remote warning, got output:\n%s", output)
	}

	addedPath := filepath.Join(repoPath, "files", filepath.Base(sourcePath))
	if _, err := os.Stat(addedPath); err != nil {
		t.Fatalf("expected file to be added at %s: %v", addedPath, err)
	}

	linkings, err := symlink.GetLinkings()
	if err != nil {
		t.Fatalf("get linkings: %v", err)
	}
	if got := linkings[filepath.Base(sourcePath)]; got == "" {
		t.Fatalf("expected linking for %s", filepath.Base(sourcePath))
	}
}

func Test_Remove_AllowsLocalOnlyRepoWithoutRemote(t *testing.T) {
	_, repoPath := setupTestConfig(t)
	viper.Set("initialized", true)

	if _, err := git.PlainInit(repoPath, false); err != nil {
		t.Fatalf("init local repo: %v", err)
	}

	sourcePath := filepath.Join(t.TempDir(), "sample.txt")
	if err := os.WriteFile(sourcePath, []byte("local only"), 0644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	if err := internalgit.LinkAndAddFile(sourcePath); err != nil {
		t.Fatalf("seed linked file: %v", err)
	}

	normalizedPath, err := symlink.BuildLinkPath(sourcePath)
	if err != nil {
		t.Fatalf("normalize link path: %v", err)
	}
	if err := symlink.AddLinking(filepath.Base(sourcePath), normalizedPath); err != nil {
		t.Fatalf("seed linking: %v", err)
	}

	output, err := captureCommandOutput(t, []string{"remove", "--no-interactive", "--no-commit", filepath.Base(sourcePath)})
	if err != nil {
		t.Fatalf("remove local-only repo error: %v\noutput:\n%s", err, output)
	}

	if strings.Contains(output, "Unable to verify remote push access") {
		t.Fatalf("expected local-only remove to skip remote warning, got output:\n%s", output)
	}

	removedPath := filepath.Join(repoPath, "files", filepath.Base(sourcePath))
	if _, err := os.Stat(removedPath); !os.IsNotExist(err) {
		t.Fatalf("expected repository file to be removed, stat error = %v", err)
	}

	linkings, err := symlink.GetLinkings()
	if err != nil {
		t.Fatalf("get linkings: %v", err)
	}
	if got := linkings[filepath.Base(sourcePath)]; got != "" {
		t.Fatalf("expected linking to be removed, got %q", got)
	}
}
