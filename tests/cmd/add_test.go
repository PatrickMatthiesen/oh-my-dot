package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	internalgit "github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/symlink"
	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/spf13/viper"
)

func Test_LocalFileCommands_AllowInaccessibleRemote(t *testing.T) {
	for _, command := range []string{"add", "remove"} {
		t.Run(command, func(t *testing.T) {
			_, repoPath := setupTestConfig(t)
			viper.Set("initialized", true)
			repo, err := git.PlainInit(repoPath, false)
			if err != nil {
				t.Fatalf("init repository: %v", err)
			}
			cfg, err := repo.Config()
			if err != nil {
				t.Fatalf("read repository config: %v", err)
			}
			cfg.User.Name = "Local Test User"
			cfg.User.Email = "local-test@example.com"
			if err := repo.SetConfig(cfg); err != nil {
				t.Fatalf("set repository identity: %v", err)
			}
			if _, err := repo.CreateRemote(&config.RemoteConfig{
				Name: "origin",
				URLs: []string{filepath.Join(t.TempDir(), "missing-remote.git")},
			}); err != nil {
				t.Fatalf("configure inaccessible remote: %v", err)
			}

			sourcePath := filepath.Join(t.TempDir(), "sample.txt")
			const contents = "local operations work offline"
			if err := os.WriteFile(sourcePath, []byte(contents), 0644); err != nil {
				t.Fatalf("write source: %v", err)
			}
			normalizedPath, err := symlink.BuildLinkPath(sourcePath)
			if err != nil {
				t.Fatalf("normalize source path: %v", err)
			}
			argument := sourcePath
			if command == "remove" {
				if err := internalgit.LinkAndAddFile(sourcePath); err != nil {
					t.Fatalf("seed repository file: %v", err)
				}
				if err := symlink.AddLinking(filepath.Base(sourcePath), normalizedPath); err != nil {
					t.Fatalf("seed linking: %v", err)
				}
				if err := internalgit.Commit("Seed file"); err != nil {
					t.Fatalf("seed commit: %v", err)
				}
				argument = filepath.Base(sourcePath)
			}

			output, err := captureCommandOutput(t, []string{command, "--no-interactive", "--no-commit=false", argument})
			if err != nil {
				t.Fatalf("%s error: %v\noutput:\n%s", command, err, output)
			}
			if strings.Contains(output, "Warning:") || strings.Contains(output, "cannot access remote repository") {
				t.Errorf("expected no remote diagnostic for %s, got:\n%s", command, output)
			}
			linkings, err := symlink.GetLinkings()
			if err != nil {
				t.Fatalf("read linkings: %v", err)
			}
			repositoryFile := filepath.Join(repoPath, "files", filepath.Base(sourcePath))
			if command == "add" {
				data, err := os.ReadFile(repositoryFile)
				if err != nil || string(data) != contents {
					t.Errorf("repository content = %q, error = %v", data, err)
				}
				if got := linkings[filepath.Base(sourcePath)]; got != normalizedPath {
					t.Errorf("linking = %q, want %q", got, normalizedPath)
				}
			} else {
				if _, err := os.Stat(repositoryFile); !os.IsNotExist(err) {
					t.Errorf("expected removed repository file, stat error = %v", err)
				}
				if _, exists := linkings[filepath.Base(sourcePath)]; exists {
					t.Error("linking still exists after removal")
				}
			}
			if data, err := os.ReadFile(sourcePath); err != nil || string(data) != contents {
				t.Errorf("source content = %q, error = %v", data, err)
			}
			head, err := repo.Head()
			if err != nil {
				t.Fatalf("read committed HEAD: %v", err)
			}
			commit, err := repo.CommitObject(head.Hash())
			if err != nil {
				t.Fatalf("read commit: %v", err)
			}
			wantMessage := "Added " + sourcePath
			if command == "remove" {
				wantMessage = "Removed " + filepath.Base(sourcePath)
			}
			if commit.Message != wantMessage {
				t.Errorf("commit message = %q, want %q", commit.Message, wantMessage)
			}
			if commit.Author.Name != cfg.User.Name || commit.Author.Email != cfg.User.Email {
				t.Errorf("commit author = %v, want repository-local identity", commit.Author)
			}
			worktree, err := repo.Worktree()
			if err != nil {
				t.Fatalf("open worktree: %v", err)
			}
			status, err := worktree.Status()
			if err != nil {
				t.Fatalf("read worktree status: %v", err)
			}
			if !status.IsClean() {
				t.Errorf("expected committed changes, got worktree status:\n%s", status)
			}
		})
	}
}

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
