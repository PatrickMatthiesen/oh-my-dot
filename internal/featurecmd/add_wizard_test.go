package featurecmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v6"
	"github.com/spf13/viper"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
)

func TestFeatureInstallLabelShowsSelectedShellCoverage(t *testing.T) {
	feature := catalog.FeatureMetadata{Name: "test", SupportedShells: []string{"bash", "powershell"}}
	installed := map[string]map[string]bool{"bash": {"test": true}, "powershell": {}}
	label := featureInstallLabel(feature, []string{"bash", "powershell"}, installed)
	if !strings.Contains(label, "installed: bash") || strings.Contains(label, "installed: powershell") {
		t.Fatalf("label=%q", label)
	}
	if label := featureInstallLabel(feature, []string{"powershell"}, installed); strings.Contains(label, "installed:") {
		t.Fatalf("label=%q", label)
	}
}

func TestFeatureWizardStopsOnCancelButContinuesOnSkip(t *testing.T) {
	for _, test := range []struct {
		name        string
		promptError error
		wantLast    bool
		wantError   bool
		summary     string
	}{
		{"cancel", fmt.Errorf("option: %w", interactive.ErrCancelled), false, false, "Cancelled."},
		{"skip", fmt.Errorf("option: %w", interactive.ErrSkip), true, false, "Skipped oh-my-posh"},
		{"failure", fmt.Errorf("unexpected prompt failure"), true, true, "Failed to add 1 feature(s)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			repoPath := t.TempDir()
			repo, err := gogit.PlainInit(repoPath, false)
			if err != nil {
				t.Fatal(err)
			}
			cfg, err := repo.Config()
			if err != nil {
				t.Fatal(err)
			}
			cfg.User.Name = "Wizard Test"
			cfg.User.Email = "wizard@example.invalid"
			if err := repo.SetConfig(cfg); err != nil {
				t.Fatal(err)
			}
			oldRepoPath := viper.Get("repo-path")
			viper.Set("repo-path", repoPath)
			t.Cleanup(func() { viper.Set("repo-path", oldRepoPath) })
			var features []catalog.FeatureMetadata
			for _, name := range []string{"powershell-aliases", "oh-my-posh", "terminal-icons"} {
				feature, ok := catalog.GetFeature(name)
				if !ok {
					t.Fatalf("missing feature %s", name)
				}
				features = append(features, feature)
			}
			read, write, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			stdout := os.Stdout
			os.Stdout = write
			state := &commandState{aliasProvider: func() string { return "oh-my-dot" }}
			runErr := state.addSelectedFeatures(repoPath, []string{"powershell"}, features, map[string]map[string]bool{"powershell": {}}, func(catalog.FeatureMetadata) (map[string]any, error) { return nil, test.promptError })
			os.Stdout = stdout
			write.Close()
			output, err := io.ReadAll(read)
			read.Close()
			if err != nil {
				t.Fatal(err)
			}
			if (runErr != nil) != test.wantError {
				t.Fatalf("run error=%v", runErr)
			}
			if !strings.Contains(string(output), test.summary) {
				t.Fatalf("missing summary %q in %s", test.summary, output)
			}
			m, err := manifest.ParseManifest(shell.GetManifestPath(repoPath, "powershell"))
			if err != nil {
				t.Fatal(err)
			}
			names := map[string]bool{}
			for _, feature := range m.Features {
				names[feature.Name] = true
			}
			if !names["powershell-aliases"] || names["oh-my-posh"] || names["terminal-icons"] != test.wantLast {
				t.Fatalf("installed=%v", names)
			}
			head, err := repo.Head()
			if err != nil {
				t.Fatalf("completed features were not committed: %v", err)
			}
			commit, err := repo.CommitObject(head.Hash())
			if err != nil {
				t.Fatal(err)
			}
			if commit.Message != "Add shell features (interactive)" {
				t.Fatalf("commit=%q", commit.Message)
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
				t.Fatalf("uncommitted changes: %s", status)
			}
		})
	}
}
