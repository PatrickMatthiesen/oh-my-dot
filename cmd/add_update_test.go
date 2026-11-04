package cmd

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/symlink"
)

type captureFixture struct {
	repo               *gogit.Repository
	root, live, stored string
}

func setupCapture(t *testing.T) captureFixture {
	t.Helper()
	t.Setenv("CI", "true")
	viper.Reset()
	t.Cleanup(viper.Reset)
	root := filepath.Join(t.TempDir(), "repo")
	repo, err := gogit.PlainInit(root, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repo.Close() })
	cfg, err := repo.Config()
	if err != nil {
		t.Fatal(err)
	}
	cfg.User.Name = "Capture Test"
	cfg.User.Email = "capture@example.invalid"
	if err := repo.SetConfig(cfg); err != nil {
		t.Fatal(err)
	}
	viper.Set("repo-path", root)
	live := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(live, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := processAddFile(captureCommand(false, false), live, false); err != nil {
		t.Fatal(err)
	}
	return captureFixture{repo: repo, root: root, live: live, stored: filepath.Join(root, "files", "profile", "config")}
}

func captureCommand(update, noCommit bool) *cobra.Command {
	command := &cobra.Command{}
	command.Flags().String("as", "profile/config", "")
	command.Flags().String("copy-to", "", "")
	command.Flags().String("move-to", "", "")
	command.Flags().Bool("update", update, "")
	command.Flags().Bool("no-commit", noCommit, "")
	command.Flags().Bool("no-interactive", true, "")
	return command
}

func captureHead(t *testing.T, repo *gogit.Repository) plumbing.Hash {
	t.Helper()
	head, err := repo.Head()
	if err != nil {
		t.Fatal(err)
	}
	return head.Hash()
}

func replaceLiveFile(t *testing.T, path, content string) {
	t.Helper()
	replacement := path + ".replacement"
	if err := os.WriteFile(replacement, []byte(content), 0640); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(replacement, 0640); err != nil {
		t.Fatal(err)
	}
	// Dropping the old directory entry emulates editors that replace the inode,
	// without relying on OS-specific rename-over-open-file behavior in a fixture.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
}

func assertCaptureContent(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil || string(data) != want {
		t.Fatalf("content %q: %q %v", path, data, err)
	}
}

func TestAddCaptureAtomicEditorReplacement(t *testing.T) {
	fixture := setupCapture(t)
	before := captureHead(t, fixture.repo)
	replaceLiveFile(t, fixture.live, "edited atomically")
	if err := processAddFile(captureCommand(false, false), fixture.live, false); err == nil || !strings.Contains(err.Error(), "--update") {
		t.Fatalf("missing capture guidance: %v", err)
	}
	assertCaptureContent(t, fixture.stored, "original")
	if err := processAddFile(captureCommand(true, false), fixture.live, false); err != nil {
		t.Fatal(err)
	}
	assertCaptureContent(t, fixture.stored, "edited atomically")
	assertCaptureContent(t, fixture.live, "edited atomically")
	storedInfo, _ := os.Stat(fixture.stored)
	liveInfo, _ := os.Stat(fixture.live)
	if !os.SameFile(storedInfo, liveInfo) {
		t.Fatal("capture did not restore managed hard link")
	}
	if runtime.GOOS != "windows" && storedInfo.Mode().Perm() != 0640 {
		t.Fatal("capture changed permissions")
	}
	after := captureHead(t, fixture.repo)
	if after == before {
		t.Fatal("capture did not commit content")
	}
	committed, err := fixture.repo.CommitObject(after)
	if err != nil {
		t.Fatal(err)
	}
	file, err := committed.File("files/profile/config")
	if err != nil {
		t.Fatal(err)
	}
	content, err := file.Contents()
	if err != nil || content != "edited atomically" {
		t.Fatalf("wrong committed capture %q %v", content, err)
	}
	if err := processAddFile(captureCommand(true, false), fixture.live, false); err != nil {
		t.Fatal(err)
	}
	if captureHead(t, fixture.repo) != after {
		t.Fatal("repeat capture created another commit")
	}
	links, err := symlink.GetLinkings()
	if err != nil || len(links) != 1 || links["profile/config"] == "" {
		t.Fatalf("registration changed: %v %v", links, err)
	}
}

func TestAddCaptureRefusesIndependentStoredChanges(t *testing.T) {
	fixture := setupCapture(t)
	before := captureHead(t, fixture.repo)
	replaceLiveFile(t, fixture.live, "local edit")
	if err := os.WriteFile(fixture.stored, []byte("independent repo edit"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := processAddFile(captureCommand(true, false), fixture.live, true); err == nil || !strings.Contains(err.Error(), "also has uncommitted changes") {
		t.Fatalf("divergence not refused: %v", err)
	}
	assertCaptureContent(t, fixture.stored, "independent repo edit")
	assertCaptureContent(t, fixture.live, "local edit")
	if captureHead(t, fixture.repo) != before {
		t.Fatal("refused capture changed HEAD")
	}
	wt, _ := fixture.repo.Worktree()
	status, err := wt.Status()
	if err != nil {
		t.Fatal(err)
	}
	if status["files/profile/config"].Staging != gogit.Unmodified {
		t.Fatal("refused capture staged source")
	}
}

func TestAddCaptureNoCommitAndStagedWork(t *testing.T) {
	fixture := setupCapture(t)
	before := captureHead(t, fixture.repo)
	replaceLiveFile(t, fixture.live, "capture me")
	unrelated := filepath.Join(fixture.root, "unrelated")
	if err := os.WriteFile(unrelated, []byte("keep staged"), 0600); err != nil {
		t.Fatal(err)
	}
	wt, _ := fixture.repo.Worktree()
	if _, err := wt.Add("unrelated"); err != nil {
		t.Fatal(err)
	}
	if err := processAddFile(captureCommand(true, false), fixture.live, false); err == nil || !strings.Contains(err.Error(), "unrelated staged") {
		t.Fatalf("staged work not protected: %v", err)
	}
	assertCaptureContent(t, fixture.stored, "original")
	if err := processAddFile(captureCommand(true, true), fixture.live, false); err != nil {
		t.Fatal(err)
	}
	if captureHead(t, fixture.repo) != before {
		t.Fatal("--no-commit created a commit")
	}
	// The command opens its own repository instance. Observe its persisted
	// index through a fresh instance rather than this fixture's old index cache.
	observed, err := gogit.PlainOpen(fixture.root)
	if err != nil {
		t.Fatal(err)
	}
	defer observed.Close()
	observedTree, err := observed.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	status, err := observedTree.Status()
	if err != nil {
		t.Fatal(err)
	}
	if status["unrelated"].Staging != gogit.Added || status["files/profile/config"].Staging != gogit.Modified {
		t.Fatalf("wrong staged state: %v", status)
	}
	index, err := observed.Storer.Index()
	if err != nil {
		t.Fatal(err)
	}
	entry, err := index.Entry("files/profile/config")
	if err != nil {
		t.Fatal(err)
	}
	blob, err := observed.BlobObject(entry.Hash)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := blob.Reader()
	if err != nil {
		t.Fatal(err)
	}
	captured, err := io.ReadAll(reader)
	reader.Close()
	if err != nil || string(captured) != "capture me" {
		t.Fatalf("wrong persisted captured content: %q %v", captured, err)
	}

}

func TestAddManagedRetriesAndSymlinkIdentity(t *testing.T) {
	for _, scenario := range []string{"hard link", "managed symlink", "wrong symlink", "different key"} {
		t.Run(scenario, func(t *testing.T) {
			fixture := setupCapture(t)
			before := captureHead(t, fixture.repo)
			command := captureCommand(false, false)
			if strings.Contains(scenario, "symlink") {
				target := fixture.stored
				if scenario == "wrong symlink" {
					target = filepath.Join(t.TempDir(), "unrelated")
					if err := os.WriteFile(target, []byte("private"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.Remove(fixture.live); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, fixture.live); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
			}
			if scenario == "different key" {
				command.Flags().Set("as", "different/config")
			}
			// A harmless no-op must also preserve an unrelated staged edit.
			wt, _ := fixture.repo.Worktree()
			os.WriteFile(filepath.Join(fixture.root, "other"), []byte("staged"), 0600)
			wt.Add("other")
			err := processAddFile(command, fixture.live, false)
			if scenario == "wrong symlink" || scenario == "different key" {
				if err == nil {
					t.Fatal("conflicting registration accepted")
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if captureHead(t, fixture.repo) != before {
				t.Fatal("retry changed HEAD")
			}
			assertCaptureContent(t, fixture.stored, "original")
		})
	}
}

func TestCaptureRequiresExistingDestinationAndDirectPath(t *testing.T) {
	fixture := setupCapture(t)
	unknown := filepath.Join(t.TempDir(), "unknown")
	os.WriteFile(unknown, []byte("unknown"), 0600)
	command := captureCommand(true, false)
	if err := processAddFile(command, unknown, false); err == nil || !strings.Contains(err.Error(), "not a managed destination") {
		t.Fatalf("unknown capture: %v", err)
	}
	command.Flags().Set("copy-to", filepath.Join(t.TempDir(), "copy"))
	if err := processAddFile(command, fixture.live, false); err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("copy capture: %v", err)
	}
	assertCaptureContent(t, fixture.live, "original")
}

func TestCaptureHashesSameSizeEditsWithPreservedTimestamp(t *testing.T) {
	for _, replacement := range []bool{false, true} {
		t.Run(map[bool]string{false: "in-place", true: "editor replacement"}[replacement], func(t *testing.T) {
			fixture := setupCapture(t)
			before, err := os.Stat(fixture.live)
			if err != nil {
				t.Fatal(err)
			}
			if replacement {
				replaceLiveFile(t, fixture.live, "newvalue")
			} else if err := os.WriteFile(fixture.live, []byte("newvalue"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chtimes(fixture.live, before.ModTime(), before.ModTime()); err != nil {
				t.Fatal(err)
			}
			if err := processAddFile(captureCommand(true, true), fixture.live, false); err != nil {
				t.Fatal(err)
			}
			opened, err := gogit.PlainOpen(fixture.root)
			if err != nil {
				t.Fatal(err)
			}
			defer opened.Close()
			index, err := opened.Storer.Index()
			if err != nil {
				t.Fatal(err)
			}
			entry, err := index.Entry("files/profile/config")
			if err != nil {
				t.Fatal(err)
			}
			blob, err := opened.BlobObject(entry.Hash)
			if err != nil {
				t.Fatal(err)
			}
			reader, err := blob.Reader()
			if err != nil {
				t.Fatal(err)
			}
			staged, err := io.ReadAll(reader)
			reader.Close()
			if err != nil || string(staged) != "newvalue" {
				t.Fatalf("explicit capture staged cached content: %q %v", staged, err)
			}
		})
	}
}

func TestCaptureDetectsSameSizeStoredDivergenceWithoutTimestampChange(t *testing.T) {
	fixture := setupCapture(t)
	before, err := os.Stat(fixture.stored)
	if err != nil {
		t.Fatal(err)
	}
	replaceLiveFile(t, fixture.live, "liveedit")
	if err := os.WriteFile(fixture.stored, []byte("repoedit"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(fixture.stored, before.ModTime(), before.ModTime()); err != nil {
		t.Fatal(err)
	}
	if err := processAddFile(captureCommand(true, true), fixture.live, false); err == nil || !strings.Contains(err.Error(), "also has uncommitted changes") {
		t.Fatalf("hidden divergence not refused: %v", err)
	}
	assertCaptureContent(t, fixture.stored, "repoedit")
	assertCaptureContent(t, fixture.live, "liveedit")
}

func TestCaptureRespectsGitLineEndingConversion(t *testing.T) {
	fixture := setupCapture(t)
	config, err := fixture.repo.Config()
	if err != nil {
		t.Fatal(err)
	}
	config.Core.AutoCRLF = "true"
	if err := fixture.repo.SetConfig(config); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture.live, []byte("first\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := processAddFile(captureCommand(true, false), fixture.live, false); err != nil {
		t.Fatal(err)
	}
	replaceLiveFile(t, fixture.live, "second\r\n")
	if err := processAddFile(captureCommand(true, false), fixture.live, false); err != nil {
		t.Fatal(err)
	}
	assertCaptureContent(t, fixture.stored, "second\r\n")
	commit, err := fixture.repo.CommitObject(captureHead(t, fixture.repo))
	if err != nil {
		t.Fatal(err)
	}
	file, err := commit.File("files/profile/config")
	if err != nil {
		t.Fatal(err)
	}
	content, err := file.Contents()
	if err != nil || content != "second\n" {
		t.Fatalf("wrong normalized capture %q %v", content, err)
	}
}
