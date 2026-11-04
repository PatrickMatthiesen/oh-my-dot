package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
)

func TestDoctorRestoresOnlyMissingCatalogImplementations(t *testing.T) {
	oldLookup := lookupFeatureExecutable
	lookupFeatureExecutable = func(string) (string, error) { return "/fake/oh-my-posh", nil }
	t.Cleanup(func() { lookupFeatureExecutable = oldLookup })
	for _, scenario := range []string{"catalog saved options", "custom missing", "existing custom content", "local catalog"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			if err := shell.InitializeShellDirectory(root, "bash"); err != nil {
				t.Fatal(err)
			}
			feature := "oh-my-posh"
			if scenario == "custom missing" {
				feature = "custom"
			}
			config := manifest.FeatureConfig{Name: feature, Strategy: "eager", Options: map[string]any{"theme": "pure", "auto_upgrade": true}}
			path := shell.GetManifestPath(root, "bash")
			if scenario == "local catalog" {
				path = shell.GetLocalManifestPath(root, "bash")
			}
			if err := manifest.WriteManifest(path, &manifest.FeatureManifest{Features: []manifest.FeatureConfig{config}}); err != nil {
				t.Fatal(err)
			}
			if scenario == "local catalog" {
				if err := os.Chmod(path, 0600); err != nil {
					t.Fatal(err)
				}
			}
			featurePath, err := shell.GetFeatureFilePath(root, "bash", feature)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "existing custom content" {
				if err := os.WriteFile(featurePath, []byte("custom content"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			results := checkFeatureFiles(context{repoPath: root, shellName: "bash", fix: true, quiet: true})
			if len(results) != 1 {
				t.Fatalf("unexpected results: %+v", results)
			}
			data, err := os.ReadFile(featurePath)
			switch scenario {
			case "custom missing":
				if !os.IsNotExist(err) || results[0].status != statusError || results[0].fixable {
					t.Fatalf("custom file falsely restored: %q %v %+v", data, err, results)
				}
			case "existing custom content":
				if err != nil || string(data) != "custom content" {
					t.Fatalf("existing file overwritten: %q %v", data, err)
				}
			default:
				if err != nil || !strings.Contains(string(data), "pure.omp.json") || !strings.Contains(string(data), "oh-my-posh upgrade --auto") || results[0].status != statusOK {
					t.Fatalf("template/options not restored: %q %v %+v", data, err, results)
				}
			}
		})
	}
}

func TestDoctorDoesNotFollowFeatureSymlink(t *testing.T) {
	root := t.TempDir()
	if err := shell.AddFeatureToShell(root, "bash", "git-prompt", "eager", nil, false, nil); err != nil {
		t.Fatal(err)
	}
	path, err := shell.GetFeatureFilePath(root, "bash", "git-prompt")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "private-file")
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	results := checkFeatureFiles(context{repoPath: root, shellName: "bash", fix: true, quiet: true})
	if len(results) != 1 || results[0].status != statusError || results[0].fixable {
		t.Fatalf("symlink accepted: %+v", results)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatal("doctor wrote through symlink")
	}
}

func TestDoctorReportsMissingOhMyPoshPrerequisite(t *testing.T) {
	oldLookup := lookupFeatureExecutable
	lookupFeatureExecutable = func(string) (string, error) { return "", os.ErrNotExist }
	t.Cleanup(func() { lookupFeatureExecutable = oldLookup })
	for _, disabled := range []bool{false, true} {
		name := "enabled"
		if disabled {
			name = "disabled"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if err := shell.AddFeatureToShell(root, "bash", "oh-my-posh", "eager", nil, disabled, map[string]any{"theme": "pure"}); err != nil {
				t.Fatal(err)
			}
			results := checkFeatureFiles(context{repoPath: root, shellName: "bash", quiet: true})
			warned := false
			for _, item := range results {
				if item.name == "Oh My Posh executable" {
					warned = true
					if item.status != statusWarning || item.fixable || !strings.Contains(item.message, "Install Oh My Posh") {
						t.Fatalf("unexpected prerequisite warning: %+v", item)
					}
				}
			}
			if warned == disabled {
				t.Fatalf("warning=%v disabled=%v", warned, disabled)
			}
		})
	}
}
