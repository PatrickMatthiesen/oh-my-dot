package featurecmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func runLifecycleCommand(t *testing.T, path string, args ...string) (string, error) {
	t.Helper()
	root := NewCommand(func() string { return "omd" }, func() string { return path })
	root.PersistentFlags().Bool("no-interactive", true, "")
	root.SilenceErrors = true
	root.SilenceUsage = true
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetErr(&output)
	root.SetArgs(args)
	err := root.Execute()
	return output.String(), err
}

func TestConfigureChangesOptionsAndIsRetrySafe(t *testing.T) {
	path, repo, _, _ := setupFeatureMutationFixture(t)
	if err := shell.AddFeatureToShell(path, "bash", "oh-my-posh", "", nil, false, map[string]any{"theme": "paradox"}); err != nil {
		t.Fatal(err)
	}
	if err := autoCommitShellFeatureChanges("fixture"); err != nil {
		t.Fatal(err)
	}
	output, err := runLifecycleCommand(t, path, "configure", "oh-my-posh", "--shell", "bash", "--option", "theme=pure")
	if err != nil {
		t.Fatalf("configure failed: %s %v", output, err)
	}
	shared, err := manifest.ParseManifest(shell.GetManifestPath(path, "bash"))
	if err != nil {
		t.Fatal(err)
	}
	feature, err := shared.GetFeature("oh-my-posh")
	if err != nil {
		t.Fatal(err)
	}
	if feature.Options["theme"] != "pure" {
		t.Fatalf("theme not changed: %v", feature.Options)
	}
	featurePath, _ := shell.GetFeatureFilePath(path, "bash", "oh-my-posh")
	script, err := os.ReadFile(featurePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(script), "pure.omp.json") || !strings.Contains(string(script), ".local/cache/oh-my-posh.pure.omp.json") {
		t.Fatalf("theme cache did not change: %s", script)
	}
	before, _ := repo.Head()
	if _, err := runLifecycleCommand(t, path, "configure", "oh-my-posh", "--shell", "bash", "--option", "theme=pure"); err != nil {
		t.Fatal(err)
	}
	after, _ := repo.Head()
	if before.Hash() != after.Hash() {
		t.Fatal("retry created another commit")
	}
}

func TestSharedRemovalWithPrivateOverrideRefusesWithoutMutation(t *testing.T) {
	for _, kind := range []string{"options", "load only"} {
		t.Run(kind, func(t *testing.T) {
			path, repo, _, _ := setupFeatureMutationFixture(t)
			if _, err := runLifecycleCommand(t, path, "add", "oh-my-posh", "--shell", "bash", "--option", "theme=pure"); err != nil {
				t.Fatal(err)
			}
			args := []string{"configure", "oh-my-posh", "--shell", "bash", "--local", "--option", "theme=paradox"}
			if kind == "load only" {
				args = []string{"disable", "oh-my-posh", "--shell", "bash", "--local"}
			}
			if _, err := runLifecycleCommand(t, path, args...); err != nil {
				t.Fatal(err)
			}
			snapshot := func() map[string]string {
				files := map[string]string{}
				if err := filepath.WalkDir(path, func(file string, entry os.DirEntry, err error) error {
					if err != nil || entry.IsDir() {
						return err
					}
					data, err := os.ReadFile(file)
					files[file] = string(data)
					return err
				}); err != nil {
					t.Fatal(err)
				}
				return files
			}
			before := snapshot()
			if _, err := runLifecycleCommand(t, path, "remove", "oh-my-posh", "--shell", "bash"); err == nil || !strings.Contains(err.Error(), "--local") {
				t.Fatalf("expected pre-write ownership refusal: %v", err)
			}
			if !reflect.DeepEqual(before, snapshot()) {
				t.Fatal("refused removal changed source, local settings, index or HEAD")
			}
			if _, err := runLifecycleCommand(t, path, "add", "core-aliases", "--shell", "bash"); err != nil {
				t.Fatalf("refused removal blocked later mutations: %v", err)
			}
			if _, err := runLifecycleCommand(t, path, "remove", "oh-my-posh", "--shell", "bash", "--local"); err != nil {
				t.Fatal(err)
			}
			if _, err := runLifecycleCommand(t, path, "remove", "oh-my-posh", "--shell", "bash"); err != nil {
				t.Fatal(err)
			}
			worktree, _ := repo.Worktree()
			if status, err := worktree.Status(); err != nil || !status.IsClean() {
				t.Fatalf("explicit removal left unexpected shared changes: %v %v", status, err)
			}
		})
	}
}

func TestSharedRefreshProtectsCommittedCustomizationsBeforeBatchWrites(t *testing.T) {
	path, repo, _, _ := setupFeatureMutationFixture(t)
	for _, name := range []string{"oh-my-posh", "core-aliases"} {
		args := []string{"add", name, "--shell", "bash"}
		if name == "oh-my-posh" {
			args = append(args, "--option", "theme=pure")
		}
		if _, err := runLifecycleCommand(t, path, args...); err != nil {
			t.Fatal(err)
		}
	}
	feature, _ := shell.GetFeatureFilePath(path, "bash", "oh-my-posh")
	custom := []byte("# committed independent customization\n")
	if err := os.WriteFile(feature, custom, 0644); err != nil {
		t.Fatal(err)
	}
	if err := autoCommitShellFeatureChanges("User customization"); err != nil {
		t.Fatal(err)
	}
	head, _ := repo.Head()
	for _, args := range [][]string{{"update", "oh-my-posh", "--shell", "bash"}, {"update", "--all"}} {
		if _, err := runLifecycleCommand(t, path, args...); err == nil || !strings.Contains(err.Error(), "--force") {
			t.Fatalf("committed customization not protected: %v", err)
		}
		if current, _ := os.ReadFile(feature); !bytes.Equal(current, custom) {
			t.Fatal("refused refresh changed custom code")
		}
		after, _ := repo.Head()
		if head.Hash() != after.Hash() {
			t.Fatal("refused batch committed another target")
		}
	}
	if _, err := runLifecycleCommand(t, path, "update", "--all", "--force"); err != nil {
		t.Fatal(err)
	}
	if current, _ := os.ReadFile(feature); bytes.Equal(current, custom) || !bytes.Contains(current, []byte("pure.omp.json")) {
		t.Fatal("explicit refresh did not restore the configured catalog version")
	}
}

func TestConfigurePreservesCustomScriptsUnlessForced(t *testing.T) {
	path, _, _, _ := setupFeatureMutationFixture(t)
	if err := shell.AddFeatureToShell(path, "bash", "oh-my-posh", "", nil, false, map[string]any{"theme": "paradox"}); err != nil {
		t.Fatal(err)
	}
	featurePath, _ := shell.GetFeatureFilePath(path, "bash", "oh-my-posh")
	custom := []byte("# customized by user\n")
	if err := os.WriteFile(featurePath, custom, 0644); err != nil {
		t.Fatal(err)
	}
	if err := autoCommitShellFeatureChanges("fixture"); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(shell.GetManifestPath(path, "bash"))
	if _, err := runLifecycleCommand(t, path, "configure", "oh-my-posh", "--shell", "bash", "--option", "theme=pure"); err == nil || !strings.Contains(err.Error(), "customized") {
		t.Fatalf("customized file accepted: %v", err)
	}
	after, _ := os.ReadFile(shell.GetManifestPath(path, "bash"))
	data, _ := os.ReadFile(featurePath)
	if !bytes.Equal(before, after) || !bytes.Equal(data, custom) {
		t.Fatal("refused configuration mutated user files")
	}
	if _, err := runLifecycleCommand(t, path, "configure", "oh-my-posh", "--shell", "bash", "--option", "theme=pure", "--force"); err != nil {
		t.Fatal(err)
	}
}

func TestFeatureInventoryJSONIsEffectiveAndStrict(t *testing.T) {
	path, _, _, _ := setupFeatureMutationFixture(t)
	if err := shell.AddFeatureToShell(path, "bash", "oh-my-posh", "", nil, false, map[string]any{"theme": "pure"}); err != nil {
		t.Fatal(err)
	}
	localPath := shell.GetLocalManifestPath(path, "bash")
	local := &manifest.FeatureManifest{Features: []manifest.FeatureConfig{{Name: "oh-my-posh", Disabled: true}, {Name: "private-aliases", Strategy: "eager"}}}
	if err := manifest.WriteLocalManifest(localPath, local); err != nil {
		t.Fatal(err)
	}
	output, err := runLifecycleCommand(t, path, "list", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var report featureInventory
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("inventory not JSON: %s %v", output, err)
	}
	if report.SchemaVersion != 1 || len(report.Features) != 2 {
		t.Fatalf("wrong inventory %+v", report)
	}
	if report.Features[0].Name != "oh-my-posh" || !report.Features[0].Disabled || !report.Features[0].Overridden || report.Features[0].Options["theme"] != "pure" || report.Features[1].Origin != "local" {
		t.Fatalf("not effective configuration %+v", report.Features)
	}
	output, err = runLifecycleCommand(t, path, "info", "oh-my-posh", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var info featureInfoReport
	if err := json.Unmarshal([]byte(output), &info); err != nil {
		t.Fatal(err)
	}
	if len(info.Installed) != 1 || !info.Installed[0].Disabled {
		t.Fatalf("info ignored override %+v", info)
	}
	if err := shell.AddFeatureToShell(path, "zsh", "oh-my-posh", "", nil, false, map[string]any{"theme": "paradox"}); err != nil {
		t.Fatal(err)
	}
	output, err = runLifecycleCommand(t, path, "info", "oh-my-posh", "--shell", "zsh", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(output), &info); err != nil || len(info.Installed) != 1 || info.Installed[0].Shell != "zsh" || info.Installed[0].Options["theme"] != "paradox" {
		t.Fatalf("shell filter returned wrong configuration: %s %v", output, err)
	}
	if err := os.WriteFile(localPath, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"list", "--json"}, {"info", "oh-my-posh", "--json"}} {
		output, err := runLifecycleCommand(t, path, args...)
		if err == nil || strings.Contains(output, `"features":[]`) {
			t.Fatalf("invalid local manifest hidden: %s %v", output, err)
		}
	}
}

func TestEnableDisableNoOpDoesNotCommit(t *testing.T) {
	path, repo, _, _ := setupFeatureMutationFixture(t)
	if err := shell.AddFeatureToShell(path, "bash", "git-prompt", "", nil, false, nil); err != nil {
		t.Fatal(err)
	}
	if err := autoCommitShellFeatureChanges("fixture"); err != nil {
		t.Fatal(err)
	}
	if _, err := runLifecycleCommand(t, path, "disable", "git-prompt", "--shell", "bash"); err != nil {
		t.Fatal(err)
	}
	before, _ := repo.Head()
	if _, err := runLifecycleCommand(t, path, "disable", "git-prompt", "--shell", "bash"); err != nil {
		t.Fatal(err)
	}
	after, _ := repo.Head()
	if before.Hash() != after.Hash() {
		t.Fatal("no-op disable created commit")
	}
	if _, err := runLifecycleCommand(t, path, "enable", "git-prompt", "--shell", "bash"); err != nil {
		t.Fatal(err)
	}
	shared, _ := manifest.ParseManifest(shell.GetManifestPath(path, "bash"))
	feature, _ := shared.GetFeature("git-prompt")
	if feature.Disabled {
		t.Fatal("feature not enabled")
	}
}

func TestMissingFeatureSelectionFiltersCurrentState(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("OMDOT_NON_INTERACTIVE", "")
	path, _, state, _ := setupFeatureMutationFixture(t)
	if err := shell.AddFeatureToShell(path, "bash", "git-prompt", "", nil, true, nil); err != nil {
		t.Fatal(err)
	}
	if err := shell.AddFeatureToShell(path, "bash", "core-aliases", "", nil, false, nil); err != nil {
		t.Fatal(err)
	}
	command := &cobra.Command{}
	command.Flags().Bool("interactive", true, "")
	old := selectLifecycleFeature
	t.Cleanup(func() { selectLifecycleFeature = old })
	selectLifecycleFeature = func(question string, choices []string) (int, error) {
		if len(choices) != 1 || choices[0] != "git-prompt (bash)" {
			t.Fatalf("wrong enable choices %v", choices)
		}
		return 0, nil
	}
	name, targets, err := state.loadSelection(command, nil, false)
	if err != nil || name != "git-prompt" || len(targets) != 1 || targets[0] != "bash" {
		t.Fatalf("wrong picker result %s %v %v", name, targets, err)
	}
}

func TestFeatureInfoBeforeInitialization(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-created")
	output, err := runLifecycleCommand(t, path, "info", "oh-my-posh", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var info featureInfoReport
	if err := json.Unmarshal([]byte(output), &info); err != nil {
		t.Fatal(err)
	}
	if info.Catalog.Name != "oh-my-posh" || len(info.Installed) != 0 {
		t.Fatalf("unexpected catalog info %+v", info)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("inspection created configuration")
	}
}

func TestFeatureInspectionRejectsMissingConfiguredRepository(t *testing.T) {
	old := viper.Get("initialized")
	viper.Set("initialized", true)
	t.Cleanup(func() { viper.Set("initialized", old) })
	path := filepath.Join(t.TempDir(), "moved-repository")
	for _, args := range [][]string{{"list", "--json"}, {"info", "oh-my-posh", "--json"}} {
		output, err := runLifecycleCommand(t, path, args...)
		if err == nil || output != "" || !strings.Contains(err.Error(), path) {
			t.Fatalf("missing configured repository reported healthy inventory: %q %v", output, err)
		}
	}
}

func TestFeatureInfoIncludesInstalledCustomFeature(t *testing.T) {
	path, _, _, _ := setupFeatureMutationFixture(t)
	if err := shell.AddFeatureToShell(path, "bash", "my-aliases", "eager", nil, false, nil); err != nil {
		t.Fatal(err)
	}
	output, err := runLifecycleCommand(t, path, "info", "my-aliases", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var info featureInfoReport
	if err := json.Unmarshal([]byte(output), &info); err != nil {
		t.Fatal(err)
	}
	if info.Catalog != nil || len(info.Installed) != 1 || info.Installed[0].Name != "my-aliases" || info.Installed[0].Origin != "shared" || info.Installed[0].Path == "" {
		t.Fatalf("custom feature missing effective inspection: %+v", info)
	}
}

func TestLocalConfigureKeepsPrivateOptionsOutOfSharedCommits(t *testing.T) {
	path, repo, _, _ := setupFeatureMutationFixture(t)
	if err := shell.AddFeatureToShell(path, "bash", "oh-my-posh", "", nil, false, map[string]any{"theme": "paradox"}); err != nil {
		t.Fatal(err)
	}
	if err := autoCommitShellFeatureChanges("fixture"); err != nil {
		t.Fatal(err)
	}
	sharedPath := shell.GetManifestPath(path, "bash")
	scriptPath, _ := shell.GetFeatureFilePath(path, "bash", "oh-my-posh")
	publicInit, _ := shell.GetInitScriptPath(path, "bash")
	originalManifest, _ := os.ReadFile(sharedPath)
	originalScript, _ := os.ReadFile(scriptPath)
	originalInit, _ := os.ReadFile(publicInit)
	before, _ := repo.Head()
	if output, err := runLifecycleCommand(t, path, "configure", "oh-my-posh", "--shell", "bash", "--local", "--option", "theme=pure"); err != nil {
		t.Fatalf("local configure %s %v", output, err)
	}
	after, _ := repo.Head()
	if before.Hash() != after.Hash() {
		t.Fatal("local configuration created a commit")
	}
	finalManifest, _ := os.ReadFile(sharedPath)
	finalScript, _ := os.ReadFile(scriptPath)
	finalInit, _ := os.ReadFile(publicInit)
	if !bytes.Equal(originalManifest, finalManifest) || !bytes.Equal(originalScript, finalScript) || !bytes.Equal(originalInit, finalInit) {
		t.Fatal("local options changed shared files")
	}
	localScript, _ := shell.GetLocalFeatureFilePath(path, "bash", "oh-my-posh")
	data, err := os.ReadFile(localScript)
	if err != nil || !strings.Contains(string(data), "oh-my-posh.pure.omp.json") {
		t.Fatalf("private theme not rendered %s %v", data, err)
	}
	if _, err := runLifecycleCommand(t, path, "configure", "oh-my-posh", "--shell", "bash", "--local", "--option", "theme=pure"); err != nil {
		t.Fatal(err)
	}
	if _, err := runLifecycleCommand(t, path, "configure", "oh-my-posh", "--shell", "bash", "--option", "auto_upgrade=true"); err != nil {
		t.Fatal(err)
	}
	shared, _ := manifest.ParseManifest(sharedPath)
	feature, _ := shared.GetFeature("oh-my-posh")
	if feature.Options["theme"] != "paradox" {
		t.Fatalf("shared configuration copied private theme: %v", feature.Options)
	}
	if _, err := runLifecycleCommand(t, path, "update", "--all"); err != nil {
		t.Fatal(err)
	}
	refreshed, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(refreshed), "pure.omp.json") || !strings.Contains(string(refreshed), "paradox.omp.json") {
		t.Fatalf("catalog refresh copied private options into shared implementation: %s", refreshed)
	}
	private, err := os.ReadFile(localScript)
	if err != nil || !strings.Contains(string(private), "pure.omp.json") {
		t.Fatalf("catalog refresh lost effective private implementation: %s %v", private, err)
	}
	history, err := repo.Log(&gogit.LogOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer history.Close()
	commit, err := history.Next()
	if err != nil {
		t.Fatal(err)
	}
	tree, err := commit.Tree()
	if err != nil {
		t.Fatal(err)
	}
	files := tree.Files()
	defer files.Close()
	if err := files.ForEach(func(file *object.File) error {
		if strings.Contains(file.Name, "enabled.local") || strings.Contains(file.Name, "init.local") || strings.Contains(file.Name, "/.local/") {
			t.Fatalf("private file committed: %s", file.Name)
		}
		contents, err := file.Contents()
		if err != nil {
			return err
		}
		if strings.Contains(contents, "oh-my-posh.pure.omp.json") || strings.Contains(contents, `"theme": "pure"`) {
			t.Fatalf("private theme leaked into %s", file.Name)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	output, err := runLifecycleCommand(t, path, "info", "oh-my-posh", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var report featureInfoReport
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Installed) != 1 || report.Installed[0].Options["theme"] != "pure" || !report.Installed[0].Overridden || report.Installed[0].Path != localScript {
		t.Fatalf("private effective state hidden: %+v", report)
	}
}

func TestLocalEnableOverridesSharedDisabledWithoutCommit(t *testing.T) {
	path, repo, _, _ := setupFeatureMutationFixture(t)
	if err := shell.AddFeatureToShell(path, "bash", "git-prompt", "", nil, true, nil); err != nil {
		t.Fatal(err)
	}
	if err := autoCommitShellFeatureChanges("fixture"); err != nil {
		t.Fatal(err)
	}
	before, _ := repo.Head()
	if _, err := runLifecycleCommand(t, path, "enable", "git-prompt", "--shell", "bash", "--local"); err != nil {
		t.Fatal(err)
	}
	after, _ := repo.Head()
	if before.Hash() != after.Hash() {
		t.Fatal("private enable created commit")
	}
	records, err := featureRecords(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Disabled || !records[0].Overridden {
		t.Fatalf("explicit local enable lost: %+v", records)
	}
	base, _ := manifest.ParseManifest(shell.GetManifestPath(path, "bash"))
	feature, _ := base.GetFeature("git-prompt")
	if !feature.Disabled {
		t.Fatal("local enable changed shared disabled state")
	}
}

func TestLocalEnablePinsIntentBeforeSharedDisabledChanges(t *testing.T) {
	path, _, _, _ := setupFeatureMutationFixture(t)
	if err := shell.AddFeatureToShell(path, "bash", "git-prompt", "", nil, false, nil); err != nil {
		t.Fatal(err)
	}
	if err := autoCommitShellFeatureChanges("fixture"); err != nil {
		t.Fatal(err)
	}
	if _, err := runLifecycleCommand(t, path, "enable", "git-prompt", "--shell", "bash", "--local"); err != nil {
		t.Fatal(err)
	}
	local, err := manifest.ParseManifest(shell.GetLocalManifestPath(path, "bash"))
	if err != nil {
		t.Fatal(err)
	}
	pinned, err := local.GetFeature("git-prompt")
	if err != nil || !pinned.DisabledSet || pinned.Disabled || pinned.Strategy != "" {
		t.Fatalf("local enabled intent was not persisted: %+v %v", pinned, err)
	}
	if _, err := runLifecycleCommand(t, path, "disable", "git-prompt", "--shell", "bash"); err != nil {
		t.Fatal(err)
	}
	records, err := featureRecords(path, nil)
	if err != nil || len(records) != 1 || records[0].Disabled {
		t.Fatalf("later shared state overrode local intent: %+v %v", records, err)
	}
	if _, err := runLifecycleCommand(t, path, "enable", "git-prompt", "--shell", "bash"); err != nil {
		t.Fatal(err)
	}
	shared, _ := manifest.ParseManifest(shell.GetManifestPath(path, "bash"))
	feature, _ := shared.GetFeature("git-prompt")
	if feature.Disabled {
		t.Fatal("matching effective local state prevented the requested shared change")
	}
}

func TestLocalUpdateAndRemoveCompletePrivateLifecycle(t *testing.T) {
	path, repo, _, _ := setupFeatureMutationFixture(t)
	if _, err := runLifecycleCommand(t, path, "add", "oh-my-posh", "--shell", "bash", "--local", "--option", "theme=pure"); err != nil {
		t.Fatal(err)
	}
	sharedPath := shell.GetManifestPath(path, "bash")
	publicPath, _ := shell.GetInitScriptPath(path, "bash")
	beforeManifest, _ := os.ReadFile(sharedPath)
	beforeInit, _ := os.ReadFile(publicPath)
	privatePath, _ := shell.GetLocalFeatureFilePath(path, "bash", "oh-my-posh")
	custom := []byte("# custom private prompt\n")
	if err := manifest.WritePrivateFile(privatePath, custom); err != nil {
		t.Fatal(err)
	}
	if _, err := runLifecycleCommand(t, path, "update", "oh-my-posh", "--shell", "bash", "--local"); err == nil || !strings.Contains(err.Error(), "customized") {
		t.Fatalf("private refresh replaced custom content without force: %v", err)
	}
	unchanged, _ := os.ReadFile(privatePath)
	if !bytes.Equal(unchanged, custom) {
		t.Fatal("refused refresh modified custom private script")
	}
	if _, err := runLifecycleCommand(t, path, "update", "--all", "--local", "--force"); err != nil {
		t.Fatal(err)
	}
	refreshed, _ := os.ReadFile(privatePath)
	if !strings.Contains(string(refreshed), "pure.omp.json") {
		t.Fatal("forced private refresh did not render its effective options")
	}
	if _, err := runLifecycleCommand(t, path, "remove", "oh-my-posh", "--shell", "bash", "--local"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(privatePath); !os.IsNotExist(err) {
		t.Fatalf("removed private implementation remains: %v", err)
	}
	if _, err := runLifecycleCommand(t, path, "remove", "oh-my-posh", "--shell", "bash", "--local"); err != nil {
		t.Fatalf("private removal retry failed: %v", err)
	}
	afterManifest, _ := os.ReadFile(sharedPath)
	afterInit, _ := os.ReadFile(publicPath)
	if !bytes.Equal(beforeManifest, afterManifest) || !bytes.Equal(beforeInit, afterInit) {
		t.Fatal("private maintenance changed shared files")
	}
	if _, err := repo.Head(); err == nil {
		t.Fatal("private maintenance created a commit")
	}
	if _, err := runLifecycleCommand(t, path, "add", "oh-my-posh", "--shell", "bash", "--local", "--option", "theme=pure"); err != nil {
		t.Fatalf("removed private feature cannot be added again: %v", err)
	}
}

func TestRemovingLocalOverrideRestoresSharedFeature(t *testing.T) {
	path, repo, _, _ := setupFeatureMutationFixture(t)
	if err := shell.AddFeatureToShell(path, "bash", "oh-my-posh", "", nil, false, map[string]any{"theme": "paradox"}); err != nil {
		t.Fatal(err)
	}
	if err := autoCommitShellFeatureChanges("fixture"); err != nil {
		t.Fatal(err)
	}
	if _, err := runLifecycleCommand(t, path, "configure", "oh-my-posh", "--shell", "bash", "--local", "--option", "theme=pure"); err != nil {
		t.Fatal(err)
	}
	before, _ := repo.Head()
	if _, err := runLifecycleCommand(t, path, "remove", "oh-my-posh", "--shell", "bash", "--local"); err != nil {
		t.Fatal(err)
	}
	after, _ := repo.Head()
	if before.Hash() != after.Hash() {
		t.Fatal("removing override created a shared commit")
	}
	records, err := featureRecords(path, nil)
	if err != nil || len(records) != 1 || records[0].Overridden || records[0].Options["theme"] != "paradox" {
		t.Fatalf("shared feature was not restored: %+v %v", records, err)
	}
	privatePath, _ := shell.GetLocalFeatureFilePath(path, "bash", "oh-my-posh")
	if _, err := os.Stat(privatePath); !os.IsNotExist(err) {
		t.Fatalf("orphaned private override script: %v", err)
	}
}

func TestLocalRequiredOptionResetInheritsSharedValue(t *testing.T) {
	path, _, _, _ := setupFeatureMutationFixture(t)
	if err := shell.AddFeatureToShell(path, "bash", "oh-my-posh", "", nil, false, map[string]any{"theme": "paradox"}); err != nil {
		t.Fatal(err)
	}
	if err := autoCommitShellFeatureChanges("fixture"); err != nil {
		t.Fatal(err)
	}
	if _, err := runLifecycleCommand(t, path, "configure", "oh-my-posh", "--shell", "bash", "--local", "--option", "theme=pure"); err != nil {
		t.Fatal(err)
	}
	if _, err := runLifecycleCommand(t, path, "configure", "oh-my-posh", "--shell", "bash", "--local", "--reset-option", "theme"); err != nil {
		t.Fatal(err)
	}
	records, err := featureRecords(path, nil)
	if err != nil || len(records) != 1 || records[0].Options["theme"] != "paradox" {
		t.Fatalf("local reset did not inherit required shared theme: %+v %v", records, err)
	}
	private, _ := shell.GetLocalFeatureFilePath(path, "bash", "oh-my-posh")
	if _, err := os.Stat(private); !os.IsNotExist(err) {
		t.Fatalf("reset left an unused private implementation: %v", err)
	}
	if _, err := runLifecycleCommand(t, path, "configure", "oh-my-posh", "--shell", "bash", "--option", "theme=pure"); err != nil {
		t.Fatal(err)
	}
	records, err = featureRecords(path, nil)
	if err != nil || records[0].Options["theme"] != "pure" {
		t.Fatalf("reset theme stopped inheriting shared changes: %+v %v", records, err)
	}
}

func TestLocalAddCreatesPrivateOnlyFeatureWithoutIdentity(t *testing.T) {
	path, repo, _, _ := setupFeatureMutationFixture(t)
	cfg, err := repo.Config()
	if err != nil {
		t.Fatal(err)
	}
	cfg.User.Name = ""
	cfg.User.Email = ""
	if err := repo.SetConfig(cfg); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "missing"))
	if output, err := runLifecycleCommand(t, path, "add", "oh-my-posh", "--shell", "bash", "--local", "--option", "theme=pure"); err != nil {
		t.Fatalf("local add required identity: %s %v", output, err)
	}
	if _, err := repo.Head(); err == nil {
		t.Fatal("private add created a commit")
	}
	base, err := manifest.ParseManifest(shell.GetManifestPath(path, "bash"))
	if err != nil {
		t.Fatal(err)
	}
	if len(base.Features) != 0 {
		t.Fatal("private feature declared in shared manifest")
	}
	localPath := shell.GetLocalManifestPath(path, "bash")
	data, err := os.ReadFile(localPath)
	if err != nil || !strings.Contains(string(data), "oh-my-posh") {
		t.Fatalf("local feature missing: %s %v", data, err)
	}
	private, _ := shell.GetLocalFeatureFilePath(path, "bash", "oh-my-posh")
	original, err := os.ReadFile(private)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runLifecycleCommand(t, path, "add", "oh-my-posh", "--shell", "bash", "--local", "--option", "theme=pure"); err != nil {
		t.Fatalf("identical private retry failed: %v", err)
	}
	after, _ := os.ReadFile(private)
	if !bytes.Equal(original, after) {
		t.Fatal("private retry rewrote user script")
	}
	if _, err := runLifecycleCommand(t, path, "configure", "oh-my-posh", "--shell", "bash", "--local", "--option", "theme=paradox"); err != nil {
		t.Fatal(err)
	}
	final, _ := os.ReadFile(private)
	if !strings.Contains(string(final), "oh-my-posh.paradox.omp.json") {
		t.Fatal("local-only configuration did not change implementation")
	}
}
