package featurecmd

import (
	"os"
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v6"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
)

func TestNamedAddSkipsExistingAndPreflightsBlockedShells(t *testing.T) {
	for _, scenario := range []string{"already installed", "second shell blocked"} {
		t.Run(scenario, func(t *testing.T) {
			path := t.TempDir()
			repo, err := gogit.PlainInit(path, false)
			if err != nil {
				t.Fatal(err)
			}
			cfg, err := repo.Config()
			if err != nil {
				t.Fatal(err)
			}
			cfg.User.Name = "Test"
			cfg.User.Email = "test@example.invalid"
			if err := repo.SetConfig(cfg); err != nil {
				t.Fatal(err)
			}
			oldPath := viper.Get("repo-path")
			viper.Set("repo-path", path)
			t.Cleanup(func() { viper.Set("repo-path", oldPath) })
			if scenario == "already installed" {
				if err := shell.AddFeatureToShell(path, "zsh", "git-prompt", "", nil, false, nil); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.MkdirAll(filepath.Join(path, "omd-shells"), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(path, "omd-shells", "zsh"), []byte("blocking file"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := autoCommitShellFeatureChanges("fixture"); err != nil {
				t.Fatal(err)
			}
			before, err := repo.Head()
			if err != nil {
				t.Fatal(err)
			}
			command := &cobra.Command{}
			command.Flags().Bool("no-interactive", true, "")
			state := &commandState{repoPathProvider: func() string { return path }, aliasProvider: func() string { return "oh-my-dot" }, flagShell: []string{"bash", "zsh"}}
			runErr := state.runFeatureAdd(command, []string{"git-prompt"})
			if (runErr != nil) != (scenario == "second shell blocked") {
				t.Fatalf("run error=%v", runErr)
			}
			if scenario == "second shell blocked" {
				if _, err := os.Stat(shell.GetManifestPath(path, "bash")); !os.IsNotExist(err) {
					t.Fatalf("preflight changed bash: %v", err)
				}
				after, err := repo.Head()
				if err != nil {
					t.Fatal(err)
				}
				if after.Hash() != before.Hash() {
					t.Fatal("preflight failure changed HEAD")
				}
				return
			}
			m, err := manifest.ParseManifest(shell.GetManifestPath(path, "bash"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := m.GetFeature("git-prompt"); err != nil {
				t.Fatal(err)
			}
			after, err := repo.Head()
			if err != nil {
				t.Fatal(err)
			}
			if before.Hash() == after.Hash() {
				t.Fatal("successful shell installation was not committed")
			}
			tree, err := repo.Worktree()
			if err != nil {
				t.Fatal(err)
			}
			status, err := tree.Status()
			if err != nil {
				t.Fatal(err)
			}
			if !status.IsClean() {
				t.Fatalf("partial batch left uncommitted changes: %s", status)
			}
		})
	}
}

func TestPreflightFeatureManifestsRejectsBlockedAncestor(t *testing.T) {
	for _, blocked := range []string{"omd-shells", filepath.Join("omd-shells", "zsh")} {
		t.Run(blocked, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, blocked)
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("private file"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := preflightFeatureManifests(root, []string{"bash", "zsh"}); err == nil {
				t.Fatal("blocked shell ancestor treated as missing configuration")
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != "private file" {
				t.Fatalf("blocking file changed: %q %v", data, err)
			}
		})
	}
}
