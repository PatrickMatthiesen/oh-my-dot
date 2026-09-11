package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/PatrickMatthiesen/oh-my-dot/cmd"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/symlink"
)

func runAutoAdd(t *testing.T, source string, flags ...string) (string, error) {
	t.Helper()
	output, err := os.CreateTemp(t.TempDir(), "output")
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	stdout, stderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = output, output
	defer func() { os.Stdout, os.Stderr = stdout, stderr }()
	var reset func()
	defer func() {
		if reset != nil {
			reset()
		}
	}()
	runErr := cmd.Execute(func(root *cobra.Command) {
		add, _, err := root.Find([]string{"add"})
		if err != nil {
			t.Fatal(err)
		}
		reset = func() {
			for _, name := range []string{"file", "as", "copy-to", "move-to", "force", "no-commit"} {
				flag := add.Flags().Lookup(name)
				if flag == nil {
					t.Fatalf("missing flag %q", name)
				}
				if err := flag.Value.Set(flag.DefValue); err != nil {
					t.Fatal(err)
				}
				flag.Changed = false
			}
		}
		reset()
		args := append([]string{"add", "--no-interactive", "--no-commit"}, flags...)
		root.SetArgs(append(args, source))
	})
	data, err := os.ReadFile(output.Name())
	if err != nil {
		t.Fatal(err)
	}
	return stripANSI(string(data)), runErr
}

func assertAutoLink(t *testing.T, key, destination string) {
	t.Helper()
	links, err := symlink.GetLinkings()
	if err != nil {
		t.Fatal(err)
	}
	want, err := symlink.BuildLinkPath(destination)
	if err != nil {
		t.Fatal(err)
	}
	if got := links[key]; got != want {
		t.Fatalf("link[%q] = %q, want %q", key, got, want)
	}
}

func TestAddAutoNamesUseNearestAvailableParent(t *testing.T) {
	repoPath := setupNestedAdd(t)
	root := t.TempDir()
	sources := []string{
		filepath.Join(root, "first", "config"),
		filepath.Join(root, "second", "shared", "config"),
		filepath.Join(root, "third", "shared", "config"),
	}
	keys := []string{"config", "shared/config", "third/shared/config"}
	for i, source := range sources {
		writeNestedAddFile(t, source, keys[i])
		output, err := runAutoAdd(t, source)
		if err != nil {
			t.Fatalf("add %s: %v\n%s", source, err, output)
		}
		if !strings.Contains(output, " as "+keys[i]) {
			t.Errorf("output does not identify selected key %q: %s", keys[i], output)
		}
		assertAutoLink(t, keys[i], source)
	}
	// Naming later files never migrates or modifies earlier keys.
	for _, key := range keys {
		assertNestedAddContent(t, filepath.Join(repoPath, "files", filepath.FromSlash(key)), key)
	}
	links, err := symlink.GetLinkings()
	if err != nil || len(links) != 3 {
		t.Fatalf("links = %v, error = %v", links, err)
	}
}

func TestAddAutoNamesRejectSameDestination(t *testing.T) {
	setupNestedAdd(t)
	source := filepath.Join(t.TempDir(), "parent", "config")
	writeNestedAddFile(t, source, "original")
	if output, err := runAutoAdd(t, source); err != nil {
		t.Fatalf("initial add: %v\n%s", err, output)
	}
	if output, err := runAutoAdd(t, source); err == nil {
		t.Fatalf("re-add unexpectedly succeeded: %s", output)
	}
	links, err := symlink.GetLinkings()
	if err != nil || len(links) != 1 {
		t.Fatalf("re-add changed linkings: %v, %v", links, err)
	}
	assertAutoLink(t, "config", source)
	assertNestedAddContent(t, source, "original")
}

func TestAddAutoNamesUseFinalCopyMoveDestination(t *testing.T) {
	for _, operation := range []string{"--copy-to", "--move-to"} {
		t.Run(operation, func(t *testing.T) {
			repoPath := setupNestedAdd(t)
			seed := filepath.Join(t.TempDir(), "config")
			writeNestedAddFile(t, seed, "existing")
			if output, err := runAutoAdd(t, seed); err != nil {
				t.Fatalf("seed: %v\n%s", err, output)
			}
			source := filepath.Join(t.TempDir(), "source-parent", "incoming")
			destination := filepath.Join(t.TempDir(), "target-parent", "config")
			writeNestedAddFile(t, source, "new")
			if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
				t.Fatal(err)
			}
			output, err := runAutoAdd(t, source, operation, destination)
			if err != nil {
				t.Fatalf("add: %v\n%s", err, output)
			}
			assertAutoLink(t, "target-parent/config", destination)
			assertNestedAddContent(t, filepath.Join(repoPath, "files", "config"), "existing")
			assertNestedAddContent(t, filepath.Join(repoPath, "files", "target-parent", "config"), "new")
			assertNestedAddContent(t, destination, "new")
			if operation == "--copy-to" {
				assertNestedAddContent(t, source, "new")
			} else if _, err := os.Stat(source); !os.IsNotExist(err) {
				t.Fatalf("move did not remove source: %v", err)
			}
		})
	}
}

func TestAddAutoNamesExplicitKeyDoesNotFallback(t *testing.T) {
	repoPath := setupNestedAdd(t)
	original := filepath.Join(t.TempDir(), "config")
	incoming := filepath.Join(t.TempDir(), "second", "config")
	writeNestedAddFile(t, original, "first")
	writeNestedAddFile(t, incoming, "second")
	if _, err := runAutoAdd(t, original); err != nil {
		t.Fatal(err)
	}
	if _, err := runAutoAdd(t, incoming, "--as", "config"); err == nil {
		t.Fatal("explicit key silently fell back")
	}
	assertNestedAddContent(t, filepath.Join(repoPath, "files", "config"), "first")
	if _, err := os.Stat(filepath.Join(repoPath, "files", "second", "config")); !os.IsNotExist(err) {
		t.Fatalf("fallback was created: %v", err)
	}
}

func TestAddAutoNamesExhaustionPrecedesCopyMove(t *testing.T) {
	for _, operation := range []string{"--copy-to", "--move-to"} {
		t.Run(operation, func(t *testing.T) {
			repoPath := setupNestedAdd(t)
			keys := []string{"config", "c/config", "b/c/config", "a/b/c/config"}
			for _, key := range keys {
				source := filepath.Join(t.TempDir(), "seed")
				writeNestedAddFile(t, source, key)
				if _, err := runAutoAdd(t, source, "--as", key); err != nil {
					t.Fatal(err)
				}
			}
			source := filepath.Join(t.TempDir(), "incoming")
			destination := filepath.Join(t.TempDir(), "a", "b", "c", "config")
			writeNestedAddFile(t, source, "incoming")
			writeNestedAddFile(t, destination, "destination")
			output, err := runAutoAdd(t, source, operation, destination, "--force")
			if err == nil {
				t.Fatalf("expected exhausted candidate error: %s", output)
			}
			if !strings.Contains(err.Error(), "--as") {
				t.Errorf("error lacks naming guidance: %v", err)
			}
			assertNestedAddContent(t, source, "incoming")
			assertNestedAddContent(t, destination, "destination")
			for _, key := range keys {
				assertNestedAddContent(t, filepath.Join(repoPath, "files", filepath.FromSlash(key)), key)
			}
		})
	}
}

func TestAddAutoNamesReserveKeysWithMissingStoredFiles(t *testing.T) {
	repoPath := setupNestedAdd(t)
	original := filepath.Join(t.TempDir(), "config")
	incoming := filepath.Join(t.TempDir(), "second", "config")
	writeNestedAddFile(t, original, "first")
	writeNestedAddFile(t, incoming, "second")
	if _, err := runAutoAdd(t, original); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(repoPath, "files", "config")); err != nil {
		t.Fatal(err)
	}
	if output, err := runAutoAdd(t, incoming); err != nil {
		t.Fatalf("add: %v\n%s", err, output)
	}
	assertAutoLink(t, "config", original)
	assertAutoLink(t, "second/config", incoming)
	assertNestedAddContent(t, filepath.Join(repoPath, "files", "second", "config"), "second")
}

func TestAddAutoNamesStopBeforeHomeDirectory(t *testing.T) {
	repoPath := setupNestedAdd(t)
	home := filepath.Join(t.TempDir(), "private-user-name")
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	for _, key := range []string{"config", "child/config"} {
		seed := filepath.Join(t.TempDir(), "seed")
		writeNestedAddFile(t, seed, key)
		if _, err := runAutoAdd(t, seed, "--as", key); err != nil {
			t.Fatal(err)
		}
	}
	source := filepath.Join(home, "child", "config")
	writeNestedAddFile(t, source, "private")
	if output, err := runAutoAdd(t, source); err == nil {
		t.Fatalf("used home directory as namespace: %s", output)
	} else if !strings.Contains(err.Error(), "--as") {
		t.Errorf("missing --as guidance: %v", err)
	}
	assertNestedAddContent(t, source, "private")
	if _, err := os.Stat(filepath.Join(repoPath, "files", "private-user-name")); !os.IsNotExist(err) {
		t.Fatalf("home name leaked into repository keys: %v", err)
	}
}
