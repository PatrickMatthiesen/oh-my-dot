package featurecmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v6"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
)

func setupFeatureMutationFixture(t *testing.T) (string, *gogit.Repository, *commandState, *cobra.Command) {
	t.Helper()
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
	state := &commandState{repoPathProvider: func() string { return path }, aliasProvider: func() string { return "oh-my-dot" }, flagShell: []string{"bash"}}
	command := &cobra.Command{}
	command.Flags().Bool("no-interactive", true, "")
	return path, repo, state, command
}

func TestInstalledAddValidatesConfigurationBeforeNoOp(t *testing.T) {
	for _, tt := range []struct {
		name     string
		options  []string
		strategy string
		commands []string
		disabled bool
		want     string
	}{
		{name: "unknown option", options: []string{"unknown=x"}, want: "unknown option"},
		{name: "invalid enum", options: []string{"theme=missing-theme"}, want: "invalid value"},
		{name: "invalid strategy", strategy: "later", want: "invalid strategy"},
		{name: "invalid trigger", strategy: "on-command", commands: []string{"node;exit"}, want: "onCommand"},
		{name: "explicit changed options", options: []string{"theme=paradox"}, want: "already installed"},
		{name: "identical options", options: []string{"theme=pure"}},
		{name: "explicit disabled", disabled: true, want: "already installed"},
		{name: "idempotent rerun"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path, repo, state, command := setupFeatureMutationFixture(t)
			if err := shell.AddFeatureToShell(path, "bash", "oh-my-posh", "", nil, false, map[string]any{"theme": "pure"}); err != nil {
				t.Fatal(err)
			}
			if err := autoCommitShellFeatureChanges("fixture"); err != nil {
				t.Fatal(err)
			}
			before, err := repo.Head()
			if err != nil {
				t.Fatal(err)
			}
			state.flagOption = tt.options
			state.flagStrategy = tt.strategy
			state.flagOnCommand = tt.commands
			state.flagDisabled = tt.disabled
			err = state.runFeatureAdd(command, []string{"oh-my-posh"})
			if tt.want == "" && err != nil || tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)) {
				t.Fatalf("unexpected error: %v", err)
			}
			after, err := repo.Head()
			if err != nil {
				t.Fatal(err)
			}
			wt, _ := repo.Worktree()
			status, err := wt.Status()
			if err != nil {
				t.Fatal(err)
			}
			if before.Hash() != after.Hash() || !status.IsClean() {
				t.Fatal("installed-add validation mutated repository")
			}
		})
	}
}

func TestFeatureMutationPreflightPreservesStagedWork(t *testing.T) {
	for _, mutation := range []string{"add", "update", "remove", "enable", "disable"} {
		t.Run(mutation, func(t *testing.T) {
			path, repo, state, command := setupFeatureMutationFixture(t)
			if err := shell.AddFeatureToShell(path, "bash", "git-prompt", "", nil, mutation == "enable", nil); err != nil {
				t.Fatal(err)
			}
			if err := autoCommitShellFeatureChanges("fixture"); err != nil {
				t.Fatal(err)
			}
			before, err := repo.Head()
			if err != nil {
				t.Fatal(err)
			}
			manifestPath := shell.GetManifestPath(path, "bash")
			original, err := os.ReadFile(manifestPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(path, "unrelated"), []byte("staged user work"), 0600); err != nil {
				t.Fatal(err)
			}
			wt, _ := repo.Worktree()
			if _, err := wt.Add("unrelated"); err != nil {
				t.Fatal(err)
			}
			statusBefore, err := wt.Status()
			if err != nil {
				t.Fatal(err)
			}
			var runErr error
			switch mutation {
			case "add":
				runErr = state.runFeatureAdd(command, []string{"core-aliases"})
			case "update":
				runErr = state.runFeatureUpdate(command, []string{"git-prompt"})
			case "remove":
				runErr = state.runFeatureRemove(command, []string{"git-prompt"})
			case "enable":
				runErr = state.runFeatureEnable(command, []string{"git-prompt"})
			case "disable":
				runErr = state.runFeatureDisable(command, []string{"git-prompt"})
			}
			if runErr == nil || !strings.Contains(runErr.Error(), "unstage") {
				t.Fatalf("unexpected preflight result: %v", runErr)
			}
			after, err := repo.Head()
			if err != nil {
				t.Fatal(err)
			}
			final, err := os.ReadFile(manifestPath)
			if err != nil {
				t.Fatal(err)
			}
			statusAfter, err := wt.Status()
			if err != nil {
				t.Fatal(err)
			}
			if before.Hash() != after.Hash() || !bytes.Equal(original, final) || statusBefore.String() != statusAfter.String() {
				t.Fatal("preflight changed manifest, index, or HEAD")
			}
		})
	}
}

func TestFeatureMutationPreservesExistingShellScriptEdits(t *testing.T) {
	for _, staged := range []bool{false, true} {
		t.Run(fmt.Sprint("staged=", staged), func(t *testing.T) {
			path, repo, state, command := setupFeatureMutationFixture(t)
			if err := shell.AddFeatureToShell(path, "bash", "git-prompt", "", nil, false, nil); err != nil {
				t.Fatal(err)
			}
			if err := autoCommitShellFeatureChanges("fixture"); err != nil {
				t.Fatal(err)
			}
			script, _ := shell.GetFeatureFilePath(path, "bash", "git-prompt")
			if err := os.WriteFile(script, []byte("# independent user customization\n"), 0644); err != nil {
				t.Fatal(err)
			}
			wt, _ := repo.Worktree()
			if staged {
				if _, err := wt.Add("omd-shells/bash/features/git-prompt.sh"); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := repo.Head()
			statusBefore, _ := wt.Status()
			if err := state.runFeatureAdd(command, []string{"core-aliases"}); err == nil || !strings.Contains(err.Error(), "pending work") {
				t.Fatalf("existing shell edit not protected: %v", err)
			}
			after, _ := repo.Head()
			statusAfter, _ := wt.Status()
			if before.Hash() != after.Hash() || statusBefore.String() != statusAfter.String() {
				t.Fatal("refused mutation changed HEAD or index")
			}
		})
	}
}

func TestLocalFirstScaffoldAllowsLaterSharedAdd(t *testing.T) {
	path, _, _, _ := setupFeatureMutationFixture(t)
	if _, err := runLifecycleCommand(t, path, "add", "oh-my-posh", "--shell", "bash", "--local", "--option", "theme=paradox"); err != nil {
		t.Fatal(err)
	}
	if _, err := runLifecycleCommand(t, path, "add", "git-prompt", "--shell", "bash"); err != nil {
		t.Fatalf("canonical local-first scaffold prevented shared setup: %v", err)
	}
}

func TestMalformedSelectedManifestStopsAddBeforeAnyWrite(t *testing.T) {
	path, repo, state, command := setupFeatureMutationFixture(t)
	if err := shell.InitializeShellDirectory(path, "zsh"); err != nil {
		t.Fatal(err)
	}
	if err := autoCommitShellFeatureChanges("fixture"); err != nil {
		t.Fatal(err)
	}
	before, err := repo.Head()
	if err != nil {
		t.Fatal(err)
	}
	broken := []byte(`{"features":`)
	if err := os.WriteFile(shell.GetManifestPath(path, "zsh"), broken, 0600); err != nil {
		t.Fatal(err)
	}
	wt, _ := repo.Worktree()
	statusBefore, err := wt.Status()
	if err != nil {
		t.Fatal(err)
	}
	state.flagShell = []string{"bash", "zsh"}
	if err := state.runFeatureAdd(command, []string{"git-prompt"}); err == nil {
		t.Fatal("malformed manifest accepted")
	}
	after, err := repo.Head()
	if err != nil {
		t.Fatal(err)
	}
	statusAfter, err := wt.Status()
	if err != nil {
		t.Fatal(err)
	}
	if before.Hash() != after.Hash() || statusBefore.String() != statusAfter.String() {
		t.Fatal("preflight staged or committed malformed manifest")
	}
	if _, err := os.Stat(shell.GetShellDirectory(path, "bash")); !os.IsNotExist(err) {
		t.Fatal("preflight wrote first shell")
	}
}

func TestNamedUpdateAndRemoveValidateAllShellsBeforeWriting(t *testing.T) {
	for _, mutation := range []string{"update", "remove"} {
		t.Run(mutation, func(t *testing.T) {
			path, repo, state, command := setupFeatureMutationFixture(t)
			if err := shell.AddFeatureToShell(path, "bash", "oh-my-posh", "", nil, false, nil); err != nil {
				t.Fatal(err)
			}
			if mutation == "update" {
				// Simulate an installation from an older catalog that supported this shell.
				if err := shell.AddFeatureToShell(path, "posix", "oh-my-posh", "", nil, false, nil); err != nil {
					t.Fatal(err)
				}
				featurePath, err := shell.GetFeatureFilePath(path, "bash", "oh-my-posh")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(featurePath, []byte("# stale catalog template\n"), 0600); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := shell.AddFeatureToShell(path, "posix", "core-aliases", "", nil, false, nil); err != nil {
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
			state.flagShell = []string{"bash", "posix"}
			var runErr error
			if mutation == "update" {
				runErr = state.runFeatureUpdate(command, []string{"oh-my-posh"})
			} else {
				runErr = state.runFeatureRemove(command, []string{"oh-my-posh"})
			}
			if runErr == nil {
				t.Fatal("later-shell failure was not reported")
			}
			after, err := repo.Head()
			if err != nil {
				t.Fatal(err)
			}
			if before.Hash() != after.Hash() {
				t.Fatal("known later-shell failure changed HEAD before validation completed")
			}
			wt, _ := repo.Worktree()
			status, err := wt.Status()
			if err != nil {
				t.Fatal(err)
			}
			if !status.IsClean() {
				t.Fatalf("partial operation left changes uncommitted: %s", status)
			}
			m, err := manifest.ParseManifest(shell.GetManifestPath(path, "bash"))
			if err != nil {
				t.Fatal(err)
			}
			_, featureErr := m.GetFeature("oh-my-posh")
			if mutation == "remove" && featureErr != nil {
				t.Fatal("failed batch changed first shell")
			}
		})
	}
}

func TestMissingCommitIdentityStopsAddBeforeWrites(t *testing.T) {
	path, repo, state, command := setupFeatureMutationFixture(t)
	if err := os.WriteFile(filepath.Join(path, ".git", "config"), []byte("[core]\nrepositoryformatversion = 0\nbare = false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	// Isolate any machine identity and author environment variables.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GIT_CONFIG_GLOBAL", "")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_SYSTEM", filepath.Join(home, "absent-system"))
	for _, name := range []string{"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL", "EMAIL"} {
		t.Setenv(name, "")
	}
	runErr := state.runFeatureAdd(command, []string{"git-prompt"})
	if runErr == nil || !strings.Contains(runErr.Error(), "user.name") {
		t.Fatalf("missing identity guidance: %v", runErr)
	}
	if _, err := os.Stat(filepath.Join(path, "omd-shells")); !os.IsNotExist(err) {
		t.Fatalf("identity error left shell writes: %v", err)
	}
	wt, _ := repo.Worktree()
	status, err := wt.Status()
	if err != nil {
		t.Fatal(err)
	}
	if !status.IsClean() {
		t.Fatal("identity error staged changes")
	}
}

func TestFeatureInfoDocumentsScriptOptions(t *testing.T) {
	path, _, state, command := setupFeatureMutationFixture(t)
	_ = path
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stdout
	os.Stdout = write
	runErr := state.runFeatureInfo(command, []string{"oh-my-posh"})
	os.Stdout = original
	write.Close()
	output, err := io.ReadAll(read)
	read.Close()
	if err != nil {
		t.Fatal(err)
	}
	if runErr != nil {
		t.Fatal(runErr)
	}
	for _, want := range []string{"--option name=value", "theme (enum, required)", "config_file (file, optional)", "auto_upgrade (bool, optional); default: false", "values:", "pure"} {
		if !strings.Contains(string(output), want) {
			t.Fatalf("missing %q in %s", want, output)
		}
	}
}
