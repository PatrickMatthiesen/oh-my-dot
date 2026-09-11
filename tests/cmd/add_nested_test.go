package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v6"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/PatrickMatthiesen/oh-my-dot/cmd"
	internalgit "github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/symlink"
)

func setupNestedAdd(t *testing.T) string {
	t.Helper()
	_, repoPath := setupTestConfig(t)
	viper.Set("initialized", true)
	if _, err := git.PlainInit(repoPath, false); err != nil {
		t.Fatal(err)
	}
	return repoPath
}

func writeNestedAddFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func assertNestedAddContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil || string(got) != want {
		t.Fatalf("ReadFile(%q) = %q, %v; want %q", path, got, err, want)
	}
}

// Commands share Cobra flag state, so each invocation starts and finishes with
// add flags reset, including Changed bits used by mutually exclusive flags.
func runNestedAdd(t *testing.T, source string, extra ...string) error {
	t.Helper()
	output, err := os.CreateTemp(t.TempDir(), "output")
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	stdout, stderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = output, output
	defer func() { os.Stdout, os.Stderr = stdout, stderr }()
	return cmd.Execute(func(root *cobra.Command) {
		add, _, err := root.Find([]string{"add"})
		if err != nil {
			t.Fatal(err)
		}
		reset := func() {
			for _, name := range []string{"file", "as", "copy-to", "move-to", "force", "no-commit"} {
				flag := add.Flags().Lookup(name)
				if flag == nil {
					t.Fatalf("missing add flag %q", name)
				}
				if err := flag.Value.Set(flag.DefValue); err != nil {
					t.Fatal(err)
				}
				flag.Changed = false
			}
		}
		reset()
		t.Cleanup(reset)
		args := []string{"add", "--no-interactive", "--no-commit", "--force=false"}
		args = append(args, extra...)
		root.SetArgs(append(args, source))
	})
}

func TestAddNestedKeysKeepSameBasenamesDistinct(t *testing.T) {
	repoPath := setupNestedAdd(t)
	sources := []string{filepath.Join(t.TempDir(), "config"), filepath.Join(t.TempDir(), "config")}
	keys := []string{"editor/config", "terminal/config"}
	for i, source := range sources {
		writeNestedAddFile(t, source, keys[i])
		if err := runNestedAdd(t, source, "--as", keys[i]); err != nil {
			t.Fatalf("add %s: %v", keys[i], err)
		}
		assertNestedAddContent(t, filepath.Join(repoPath, "files", filepath.FromSlash(keys[i])), keys[i])
	}
	linkings, err := symlink.GetLinkings()
	if err != nil {
		t.Fatal(err)
	}
	for i, key := range keys {
		want, err := symlink.BuildLinkPath(sources[i])
		if err != nil {
			t.Fatal(err)
		}
		if linkings[key] != want {
			t.Errorf("linking[%q] = %q, want %q", key, linkings[key], want)
		}
	}
	files, err := internalgit.ListFiles()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(files, ",") != strings.Join(keys, ",") {
		t.Fatalf("listed files = %v, want %v", files, keys)
	}
	output, err := captureCommandOutput(t, []string{"list", "--no-interactive"})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range keys {
		if !strings.Contains(output, key) {
			t.Errorf("list output missing %q: %s", key, output)
		}
	}
}

func TestAddRejectsUnsafeKeysBeforeMoving(t *testing.T) {
	for _, key := range []string{".git/config", "nested/.GiT/config", "git~1/config", "nested/GIT~1/config", "../outside", "dir/../../outside", "/absolute", "C:/absolute", `dir\file`, "dir/../file", "dir//file", "CON", "dir/NUL.txt", "dir/trailing.", "dir/trailing "} {
		t.Run(key, func(t *testing.T) {
			repoPath := setupNestedAdd(t)
			source := filepath.Join(t.TempDir(), "source")
			target := filepath.Join(t.TempDir(), "destination")
			writeNestedAddFile(t, source, "original")
			if err := runNestedAdd(t, source, "--as", key, "--move-to", target); err == nil {
				t.Fatalf("accepted unsafe key %q", key)
			}
			assertNestedAddContent(t, source, "original")
			if _, err := os.Stat(target); !os.IsNotExist(err) {
				t.Fatalf("move target changed: %v", err)
			}
			files, err := os.ReadDir(filepath.Join(repoPath, "files"))
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			if len(files) != 0 {
				t.Fatalf("unsafe key created repository entries: %v", files)
			}
			repo, err := git.PlainOpen(repoPath)
			if err != nil {
				t.Fatal(err)
			}
			index, err := repo.Storer.Index()
			if err != nil {
				t.Fatal(err)
			}
			if len(index.Entries) != 0 {
				t.Fatalf("unsafe key changed Git index: %v", index.Entries)
			}
		})
	}
}

func TestAddCollisionPreflightPreservesFiles(t *testing.T) {
	for _, operation := range []string{"--copy-to", "--move-to"} {
		for _, secondKey := range []string{"config", "CONFIG"} {
			t.Run(operation+secondKey, func(t *testing.T) {
				repoPath := setupNestedAdd(t)
				original := filepath.Join(t.TempDir(), "config")
				source := filepath.Join(t.TempDir(), "incoming")
				target := filepath.Join(t.TempDir(), "destination")
				writeNestedAddFile(t, original, "tracked")
				writeNestedAddFile(t, source, "incoming")
				writeNestedAddFile(t, target, "destination")
				if err := runNestedAdd(t, original); err != nil {
					t.Fatal(err)
				}
				if err := runNestedAdd(t, source, "--as", secondKey, operation, target, "--force"); err == nil {
					t.Fatal("expected repository key conflict despite --force")
				}
				assertNestedAddContent(t, source, "incoming")
				assertNestedAddContent(t, target, "destination")
				assertNestedAddContent(t, original, "tracked")
				assertNestedAddContent(t, filepath.Join(repoPath, "files", "config"), "tracked")
			})
		}
	}
}

func TestAddCopyToDirectoryUsesResolvedFile(t *testing.T) {
	repoPath := setupNestedAdd(t)
	source := filepath.Join(t.TempDir(), "config")
	directory := t.TempDir()
	writeNestedAddFile(t, source, "copied")
	if err := runNestedAdd(t, source, "--copy-to", directory); err != nil {
		t.Fatal(err)
	}
	assertNestedAddContent(t, source, "copied")
	assertNestedAddContent(t, filepath.Join(directory, "config"), "copied")
	assertNestedAddContent(t, filepath.Join(repoPath, "files", "config"), "copied")
	linkings, err := symlink.GetLinkings()
	if err != nil {
		t.Fatal(err)
	}
	want, err := symlink.BuildLinkPath(filepath.Join(directory, "config"))
	if err != nil {
		t.Fatal(err)
	}
	if linkings["config"] != want {
		t.Fatalf("linking = %q, want copied destination %q", linkings["config"], want)
	}
}

func TestAddRejectsDuplicateDestinationUnderDifferentKey(t *testing.T) {
	repoPath := setupNestedAdd(t)
	source := filepath.Join(t.TempDir(), "config")
	writeNestedAddFile(t, source, "original")
	if err := runNestedAdd(t, source, "--as", "first/config"); err != nil {
		t.Fatal(err)
	}
	if err := runNestedAdd(t, source, "--as", "second/config", "--force"); err == nil {
		t.Fatal("accepted duplicate destination")
	}
	assertNestedAddContent(t, source, "original")
	if _, err := os.Stat(filepath.Join(repoPath, "files", "second", "config")); !os.IsNotExist(err) {
		t.Fatalf("duplicate key was created: %v", err)
	}
	linkings, err := symlink.GetLinkings()
	if err != nil {
		t.Fatal(err)
	}
	if len(linkings) != 1 || linkings["first/config"] == "" {
		t.Fatalf("linkings changed after conflict: %v", linkings)
	}
}

func TestAddRejectsDestinationInsideRepositoryViaDirectoryAlias(t *testing.T) {
	for _, operation := range []string{"--copy-to", "--move-to"} {
		t.Run(operation, func(t *testing.T) {
			repoPath := setupNestedAdd(t)
			files := filepath.Join(repoPath, "files")
			if err := os.MkdirAll(files, 0755); err != nil {
				t.Fatal(err)
			}
			alias := filepath.Join(t.TempDir(), "alias")
			if err := os.Symlink(files, alias); err != nil {
				t.Skipf("directory symlinks unavailable: %v", err)
			}
			source := filepath.Join(t.TempDir(), "source")
			writeNestedAddFile(t, source, "original")
			if err := runNestedAdd(t, source, "--as", "nested/config", operation, filepath.Join(alias, "destination")); err == nil {
				t.Fatal("accepted destination inside repository through directory alias")
			}
			assertNestedAddContent(t, source, "original")
			entries, err := os.ReadDir(files)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Fatalf("preflight failure modified repository storage: %v", entries)
			}
		})
	}
}

func TestAddRejectsMovingTrackedRepositorySource(t *testing.T) {
	for _, useAlias := range []bool{false, true} {
		name := "direct"
		if useAlias {
			name = "directory alias"
		}
		t.Run(name, func(t *testing.T) {
			repoPath := setupNestedAdd(t)
			original := filepath.Join(t.TempDir(), "original")
			writeNestedAddFile(t, original, "tracked")
			if err := runNestedAdd(t, original, "--as", "first/config"); err != nil {
				t.Fatal(err)
			}
			source := filepath.Join(repoPath, "files", "first", "config")
			if useAlias {
				alias := filepath.Join(t.TempDir(), "alias")
				if err := os.Symlink(filepath.Dir(source), alias); err != nil {
					t.Skipf("directory symlinks unavailable: %v", err)
				}
				source = filepath.Join(alias, "config")
			}
			destination := filepath.Join(t.TempDir(), "destination")
			if err := runNestedAdd(t, source, "--as", "second/config", "--move-to", destination); err == nil {
				t.Fatal("accepted move of existing repository source")
			}
			assertNestedAddContent(t, source, "tracked")
			assertNestedAddContent(t, original, "tracked")
			if _, err := os.Stat(destination); !os.IsNotExist(err) {
				t.Fatalf("move destination changed: %v", err)
			}
			linkings, err := symlink.GetLinkings()
			if err != nil {
				t.Fatal(err)
			}
			if len(linkings) != 1 || linkings["first/config"] == "" {
				t.Fatalf("linkings changed: %v", linkings)
			}
		})
	}
}
