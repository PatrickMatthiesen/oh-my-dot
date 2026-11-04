package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOhMyPoshYAMLThemeConfiguration(t *testing.T) {
	for _, shellName := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shellName, func(t *testing.T) {
			root := t.TempDir()
			legacyPath := ""
			const legacyContent = "# Preserve the user's existing Zsh implementation.\n"
			if shellName == "zsh" {
				if err := InitializeShellDirectory(root, shellName); err != nil {
					t.Fatal(err)
				}
				legacyPath = filepath.Join(root, "omd-shells", shellName, "features", "oh-my-posh.sh")
				if err := os.WriteFile(legacyPath, []byte(legacyContent), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := AddFeatureToShell(root, shellName, "oh-my-posh", "eager", nil, false, map[string]any{"theme": "devious-diamonds"}); err != nil {
				t.Fatal(err)
			}
			path, err := GetFeatureFilePath(root, shellName, "oh-my-posh")
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"glowsticks", "agnoster.minimal", "M365Princess", "devious-diamonds"} {
				plan, err := PrepareFeatureConfiguration(root, shellName, "oh-my-posh", map[string]any{"theme": name}, false)
				if err != nil {
					t.Fatalf("configure %s: %v", name, err)
				}
				if err := plan.Apply(); err != nil {
					t.Fatal(err)
				}
				plan, err = PrepareFeatureRefresh(root, shellName, "oh-my-posh", false)
				if err != nil {
					t.Fatalf("refresh %s: %v", name, err)
				}
				if plan.Changed {
					t.Fatalf("refresh unexpectedly changed a current %s template", name)
				}
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), "devious-diamonds.omp.yaml") {
				t.Fatal("YAML configuration lost its actual extension")
			}
			// No YAML template used the old shared JSON cache. Removing the
			// cache directory creation is therefore a user edit, not migration.
			lines := strings.Split(string(data), "\n")
			var edited []string
			for _, line := range lines {
				if strings.Contains(line, "mkdir -p") || strings.Contains(line, "New-Item -ItemType Directory") {
					continue
				}
				edited = append(edited, line)
			}
			if err := os.WriteFile(path, []byte(strings.Join(edited, "\n")), 0644); err != nil {
				t.Fatal(err)
			}
			if _, err := PrepareFeatureConfiguration(root, shellName, "oh-my-posh", map[string]any{"theme": "pure"}, false); err == nil || !strings.Contains(err.Error(), "customized") {
				t.Fatalf("expected edited YAML template protection, got %v", err)
			}
			if _, err := PrepareFeatureRefresh(root, shellName, "oh-my-posh", false); err == nil || !strings.Contains(err.Error(), "customized") {
				t.Fatalf("expected edited YAML template refresh protection, got %v", err)
			}
			if legacyPath != "" {
				data, err := os.ReadFile(legacyPath)
				if err != nil || string(data) != legacyContent {
					t.Fatalf("existing Zsh .sh file changed: content=%q error=%v", data, err)
				}
			}
		})
	}
}
