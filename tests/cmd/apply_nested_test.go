package cmd_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v6"
	"github.com/spf13/viper"
)

func Test_ApplyNestedFiles(t *testing.T) {
	_, repoPath := setupTestConfig(t)
	viper.Set("initialized", true)
	if _, err := git.PlainInit(repoPath, false); err != nil {
		t.Fatal(err)
	}
	targetDir := t.TempDir()
	links := map[string]string{}
	for _, group := range []string{"work", "personal"} {
		key := group + "/config"
		stored := filepath.Join(repoPath, "files", filepath.FromSlash(key))
		if err := os.MkdirAll(filepath.Dir(stored), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(stored, []byte(group), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(targetDir, group), 0700); err != nil {
			t.Fatal(err)
		}
		links[key] = filepath.Join(targetDir, group, "config")
	}
	data, _ := json.Marshal(links)
	if err := os.WriteFile(filepath.Join(repoPath, "linkings.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	output, err := captureCommandOutput(t, []string{"apply", "--no-shell", "--no-interactive", "--allow-outside-home"})
	if err != nil {
		t.Fatalf("apply: %v\n%s", err, output)
	}
	for key, target := range links {
		content, err := os.ReadFile(target)
		if err != nil || string(content) != strings.Split(key, "/")[0] {
			t.Fatalf("%s content %q: %v", key, content, err)
		}
	}
}

func Test_ApplyRejectsConflictsBeforeLinking(t *testing.T) {
	for _, mode := range []string{"duplicate destination", "unsafe key"} {
		t.Run(mode, func(t *testing.T) {
			_, repoPath := setupTestConfig(t)
			viper.Set("initialized", true)
			if _, err := git.PlainInit(repoPath, false); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(repoPath, "files"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(repoPath, "files", "safe"), []byte("safe"), 0600); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(t.TempDir(), "target")
			links := map[string]string{"safe": target, "nested/config": target}
			if mode == "unsafe key" {
				links = map[string]string{"safe": target, "../escape": target + "-other"}
			}
			data, _ := json.Marshal(links)
			if err := os.WriteFile(filepath.Join(repoPath, "linkings.json"), data, 0600); err != nil {
				t.Fatal(err)
			}
			output, err := captureCommandOutput(t, []string{"apply", "--no-shell", "--no-interactive", "--allow-outside-home"})
			if err == nil {
				t.Fatalf("expected preflight error: %s", output)
			}
			if _, err := os.Lstat(target); !os.IsNotExist(err) {
				t.Fatalf("target mutated before validation: %v", err)
			}
		})
	}
}
