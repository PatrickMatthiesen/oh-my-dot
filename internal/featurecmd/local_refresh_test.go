package featurecmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
)

func TestPrivateBulkRefreshAfterCatalogUpgrade(t *testing.T) {
	for _, scenario := range []string{"outdated second", "missing second", "customized second"} {
		t.Run(scenario, func(t *testing.T) {
			path, repo, _, _ := setupFeatureMutationFixture(t)
			if _, err := runLifecycleCommand(t, path, "add", "git-prompt", "--shell", "bash"); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"core-aliases", "oh-my-posh"} {
				args := []string{"add", name, "--shell", "bash", "--local"}
				if name == "oh-my-posh" {
					args = append(args, "--option", "theme=pure")
				}
				if _, err := runLifecycleCommand(t, path, args...); err != nil {
					t.Fatal(err)
				}
			}
			statePath := filepath.Join(shell.GetShellDirectory(path, "bash"), ".local", "generated.json")
			data, err := os.ReadFile(statePath)
			if err != nil {
				t.Fatal(err)
			}
			var state map[string]map[string]string
			if err := json.Unmarshal(data, &state); err != nil {
				t.Fatal(err)
			}
			current := map[string][]byte{}
			for _, name := range []string{"core-aliases", "oh-my-posh"} {
				file, _ := shell.GetLocalFeatureFilePath(path, "bash", name)
				content, err := os.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				current[file] = content
				previous := append(append([]byte(nil), content...), []byte("\n# previous bundled catalog version\n")...)
				if err := manifest.WritePrivateFile(file, previous); err != nil {
					t.Fatal(err)
				}
				state[name]["renderedHash"] = fmt.Sprintf("%x", sha256.Sum256(previous))
			}
			data, err = json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			if err := manifest.WritePrivateFile(statePath, data); err != nil {
				t.Fatal(err)
			}
			second, _ := shell.GetLocalFeatureFilePath(path, "bash", "oh-my-posh")
			if scenario == "missing second" {
				if err := os.Remove(second); err != nil {
					t.Fatal(err)
				}
			} else if scenario == "customized second" {
				if err := manifest.WritePrivateFile(second, []byte("# user customization\n")); err != nil {
					t.Fatal(err)
				}
			}
			localPath := shell.GetLocalManifestPath(path, "bash")
			data, err = os.ReadFile(localPath)
			if err != nil {
				t.Fatal(err)
			}
			var compact bytes.Buffer
			if err := json.Compact(&compact, data); err != nil {
				t.Fatal(err)
			}
			beforeManifest := compact.Bytes()
			if err := manifest.WritePrivateFile(localPath, beforeManifest); err != nil {
				t.Fatal(err)
			}
			beforeScripts := map[string][]byte{}
			for file := range current {
				beforeScripts[file], _ = os.ReadFile(file)
			}
			beforeHead, _ := repo.Head()
			output, err := runLifecycleCommand(t, path, "update", "--local", "--all")
			if scenario == "customized second" {
				if err == nil || !strings.Contains(err.Error(), "customized") {
					t.Fatalf("customized batch did not refuse: %s %v", output, err)
				}
				current = beforeScripts
			} else if err != nil || !strings.Contains(output, "Refreshed 2 private template(s)") {
				t.Fatalf("bulk private refresh failed or undercounted: %s %v", output, err)
			}
			if scenario != "customized second" {
				output, err := runLifecycleCommand(t, path, "update", "--local", "--all")
				if err != nil || !strings.Contains(output, "Refreshed 0 private template(s)") {
					t.Fatalf("private refresh retry was not a no-op: %s %v", output, err)
				}
			}
			for file, expected := range current {
				got, err := os.ReadFile(file)
				if err != nil || !bytes.Equal(got, expected) {
					t.Fatalf("wrong private implementation %s: %v", file, err)
				}
			}
			if got, _ := os.ReadFile(localPath); !bytes.Equal(got, beforeManifest) {
				t.Fatal("template refresh rewrote private settings")
			}
			afterHead, _ := repo.Head()
			worktree, _ := repo.Worktree()
			status, err := worktree.Status()
			if err != nil || !status.IsClean() || beforeHead.Hash() != afterHead.Hash() {
				t.Fatalf("private refresh changed shared Git state: %s %v", status, err)
			}
		})
	}
}

func TestPrivateRefreshStillRejectsUnexpectedChanges(t *testing.T) {
	for _, change := range []string{"edited script", "new script", "settings"} {
		t.Run(change, func(t *testing.T) {
			path, _, _, _ := setupFeatureMutationFixture(t)
			if _, err := runLifecycleCommand(t, path, "add", "oh-my-posh", "--shell", "bash", "--local", "--option", "theme=pure"); err != nil {
				t.Fatal(err)
			}
			file, _ := shell.GetLocalFeatureFilePath(path, "bash", "oh-my-posh")
			if err := manifest.WritePrivateFile(file, []byte("# previous implementation\n")); err != nil {
				t.Fatal(err)
			}
			if change == "new script" {
				if err := os.Remove(file); err != nil {
					t.Fatal(err)
				}
			}
			plan, err := shell.PrepareLocalFeatureRefresh(path, "bash", "oh-my-posh", true)
			if err != nil || !plan.Changed {
				t.Fatalf("prepare explicit replacement: %v", err)
			}
			if change == "settings" {
				localPath := shell.GetLocalManifestPath(path, "bash")
				local, err := manifest.ParseManifest(localPath)
				if err != nil {
					t.Fatal(err)
				}
				if err := local.UpdateFeature("oh-my-posh", func(f *manifest.FeatureConfig) { f.Disabled = true }); err != nil {
					t.Fatal(err)
				}
				if err := manifest.WriteLocalManifest(localPath, local); err != nil {
					t.Fatal(err)
				}
			} else if err := manifest.WritePrivateFile(file, []byte("# unexpected user edit\n")); err != nil {
				t.Fatal(err)
			}
			beforeScript, _ := os.ReadFile(file)
			beforeManifest, _ := os.ReadFile(shell.GetLocalManifestPath(path, "bash"))
			if err := plan.Apply(); err == nil || !strings.Contains(err.Error(), "changed") {
				t.Fatalf("stale private replacement was accepted: %v", err)
			}
			if after, _ := os.ReadFile(file); !bytes.Equal(beforeScript, after) {
				t.Fatal("stale refresh overwrote a user edit")
			}
			if after, _ := os.ReadFile(shell.GetLocalManifestPath(path, "bash")); !bytes.Equal(beforeManifest, after) {
				t.Fatal("stale refresh rewrote settings")
			}
		})
	}
}
