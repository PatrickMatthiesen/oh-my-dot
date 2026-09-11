package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v6"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	appcmd "github.com/PatrickMatthiesen/oh-my-dot/cmd"
	internalgit "github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/symlink"
)

func seedNestedRemoval(t *testing.T, keys []string) (string, map[string]string) {
	t.Helper()
	_, repoPath := setupTestConfig(t)
	viper.Set("initialized", true)
	if _, err := git.PlainInit(repoPath, false); err != nil {
		t.Fatalf("initialize repository: %v", err)
	}
	links := make(map[string]string)
	for _, key := range keys {
		source := filepath.Join(t.TempDir(), filepath.Base(key))
		if err := os.WriteFile(source, []byte(key), 0644); err != nil {
			t.Fatalf("write linked file: %v", err)
		}
		target := filepath.Join(repoPath, "files", filepath.FromSlash(key))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatalf("create repository directory: %v", err)
		}
		if err := os.Link(source, target); err != nil {
			t.Fatalf("create repository file: %v", err)
		}
		if err := internalgit.StageChange("files/" + key); err != nil {
			t.Fatalf("stage repository file: %v", err)
		}
		if err := symlink.AddLinking(key, source); err != nil {
			t.Fatalf("save linking: %v", err)
		}
		links[key] = source
	}
	return repoPath, links
}

func runNestedRemoval(t *testing.T, selector string, deleteLinked bool) error {
	t.Helper()
	var restore []func()
	defer func() {
		for _, reset := range restore {
			reset()
		}
	}()
	return appcmd.Execute(func(root *cobra.Command) {
		command, _, err := root.Find([]string{"remove"})
		if err != nil {
			t.Fatalf("find remove command: %v", err)
		}
		for _, name := range []string{"file", "delete-linked", "keep-linked", "yes", "no-commit", "source"} {
			flag := command.Flags().Lookup(name)
			value, changed := flag.Value.String(), flag.Changed
			restore = append(restore, func() {
				_ = flag.Value.Set(value)
				flag.Changed = changed
			})
			if err := flag.Value.Set(flag.DefValue); err != nil {
				t.Fatalf("reset %s flag: %v", name, err)
			}
			flag.Changed = false
		}
		args := []string{"remove", "--no-interactive", "--no-commit"}
		if deleteLinked {
			args = append(args, "--delete-linked")
		}
		root.SetArgs(append(args, selector))
	})
}

func Test_Remove_NestedRepositoryKeys(t *testing.T) {
	for _, selection := range []string{"qualified", "absolute", "unique basename", "qualified flat"} {
		t.Run(selection, func(t *testing.T) {
			keys := []string{"first/config", "second/config"}
			if selection == "unique basename" {
				keys[1] = "second/other"
			} else if selection == "qualified flat" {
				keys[0] = "config"
			}
			repoPath, links := seedNestedRemoval(t, keys)
			selector := keys[0]
			if selection == "absolute" {
				selector = filepath.Join(repoPath, "files", filepath.FromSlash(keys[0]))
			} else if selection == "unique basename" {
				selector = "config"
			} else if selection == "qualified flat" {
				selector = "./config"
			}
			if err := runNestedRemoval(t, selector, false); err != nil {
				t.Fatalf("remove nested file: %v", err)
			}
			if _, err := os.Stat(filepath.Join(repoPath, "files", filepath.FromSlash(keys[0]))); !os.IsNotExist(err) {
				t.Errorf("selected file should be removed, stat error = %v", err)
			}
			data, err := os.ReadFile(filepath.Join(repoPath, "files", filepath.FromSlash(keys[1])))
			if err != nil || string(data) != keys[1] {
				t.Errorf("other repository file = %q, error = %v", data, err)
			}
			remaining, err := symlink.GetLinkings()
			if err != nil {
				t.Fatalf("read linkings: %v", err)
			}
			if len(remaining) != 1 || remaining[keys[1]] != links[keys[1]] {
				t.Errorf("remaining linkings = %v", remaining)
			}
			for key, source := range links {
				if data, err := os.ReadFile(source); err != nil || string(data) != key {
					t.Errorf("source %s = %q, error = %v", key, data, err)
				}
			}
		})
	}
}

func Test_Remove_AmbiguousBasenameDoesNotChangeFiles(t *testing.T) {
	for _, flat := range []bool{false, true} {
		name := "nested only"
		keys := []string{"z/config", "a/config"}
		if flat {
			name = "flat and nested"
			keys = append(keys, "config")
		}
		t.Run(name, func(t *testing.T) {
			repoPath, links := seedNestedRemoval(t, keys)
			err := runNestedRemoval(t, "config", true)
			wantMatches := "a/config, z/config"
			if flat {
				wantMatches = "a/config, config, z/config"
			}
			if err == nil || !strings.Contains(err.Error(), "ambiguous") || !strings.Contains(err.Error(), wantMatches) {
				t.Fatalf("expected ambiguity with sorted keys %q, got %v", wantMatches, err)
			}
			remaining, err := symlink.GetLinkings()
			if err != nil {
				t.Fatalf("read linkings: %v", err)
			}
			if len(remaining) != len(links) {
				t.Errorf("linkings changed: %v", remaining)
			}
			for key, source := range links {
				if remaining[key] != source {
					t.Errorf("linking for %s changed", key)
				}
				for _, file := range []string{source, filepath.Join(repoPath, "files", filepath.FromSlash(key))} {
					if data, err := os.ReadFile(file); err != nil || string(data) != key {
						t.Errorf("file %s changed: content = %q, error = %v", file, data, err)
					}
				}
			}
		})
	}
}

func Test_Remove_DeleteLinkedExpandsHome(t *testing.T) {
	repoPath, links := seedNestedRemoval(t, []string{"first/config", "second/config"})
	t.Setenv("HOME", filepath.Dir(links["first/config"]))
	t.Setenv("USERPROFILE", filepath.Dir(links["first/config"]))
	if err := symlink.AddLinking("first/config", "~/config"); err != nil {
		t.Fatalf("set home-relative linking: %v", err)
	}
	if err := runNestedRemoval(t, "first/config", true); err != nil {
		t.Fatalf("remove linked file: %v", err)
	}
	for _, file := range []string{links["first/config"], filepath.Join(repoPath, "files", "first", "config")} {
		if _, err := os.Stat(file); !os.IsNotExist(err) {
			t.Errorf("expected %s removed, stat error = %v", file, err)
		}
	}
	if data, err := os.ReadFile(links["second/config"]); err != nil || string(data) != "second/config" {
		t.Errorf("other linked file = %q, error = %v", data, err)
	}
}

func Test_Remove_InvalidSelectorPreservesLinkedFile(t *testing.T) {
	repoPath, links := seedNestedRemoval(t, []string{"first/config"})
	for _, selector := range []string{"../outside", filepath.Join(t.TempDir(), "outside"), "first/missing"} {
		if err := runNestedRemoval(t, selector, true); err == nil {
			t.Errorf("expected error for %q", selector)
		}
	}
	for _, file := range []string{links["first/config"], filepath.Join(repoPath, "files", "first", "config")} {
		if data, err := os.ReadFile(file); err != nil || string(data) != "first/config" {
			t.Errorf("file %s changed: content = %q, error = %v", file, data, err)
		}
	}
}

func Test_Remove_UnsafeRepositoryPathPreservesLinkedFile(t *testing.T) {
	repoPath, links := seedNestedRemoval(t, []string{"first/config"})
	repositoryFile := filepath.Join(repoPath, "files", "first", "config")
	if err := os.Remove(repositoryFile); err != nil {
		t.Fatalf("remove seeded repository file: %v", err)
	}
	if err := os.Symlink(links["first/config"], repositoryFile); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := runNestedRemoval(t, "first/config", true); err == nil {
		t.Fatal("expected symlinked repository file to be rejected")
	}
	if data, err := os.ReadFile(links["first/config"]); err != nil || string(data) != "first/config" {
		t.Errorf("linked file changed: content = %q, error = %v", data, err)
	}
	remaining, err := symlink.GetLinkings()
	if err != nil {
		t.Fatalf("read linkings: %v", err)
	}
	if remaining["first/config"] != links["first/config"] {
		t.Errorf("linking changed: %v", remaining)
	}
}
