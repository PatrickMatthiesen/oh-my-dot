package main

import (
	"bytes"
	"encoding/json"
	gogit "github.com/go-git/go-git/v6"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadOnlyCommandsDoNotCreateConfig(t *testing.T) {
	for _, args := range [][]string{nil, {"add"}, {"apply", "--dry-run"}, {"list", "--json"}, {"doctor", "--offline"}, {"status", "--json"}, {"feature", "list"}, {"init", "--help"}, {"version"}} {
		if shouldCreateConfig(args) {
			t.Fatalf("unexpected config creation for %v", args)
		}
	}
	for _, args := range [][]string{{"init"}, {"i", "--folder", "repo", "--no-interactive"}, {"--no-interactive", "init"}, {"config", "set", "allow-gh-auth", "false"}} {
		if !shouldCreateConfig(args) {
			t.Fatalf("missing config creation for %v", args)
		}
	}
}

// Exercise the real process boundary: JSON diagnostics can fail while stdout
// remains one complete JSON value, with all error presentation on stderr.
func TestMachineInspectionProcess(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "omd.exe")
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build test CLI: %v\n%s", err, output)
	}
	repoPath := filepath.Join(root, "repo")
	if _, err := gogit.PlainInit(repoPath, false); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "config.json")
	data, _ := json.Marshal(map[string]any{"initialized": true, "repo-path": repoPath})
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OH_MY_DOT_CONFIG", configPath)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(root, "no-global-config"))
	t.Setenv("GIT_AUTHOR_NAME", "")
	t.Setenv("GIT_AUTHOR_EMAIL", "")
	t.Setenv("CI", "true")
	run := func(args ...string) ([]byte, string, error) {
		command := exec.Command(binary, args...)
		var stderr bytes.Buffer
		command.Stderr = &stderr
		stdout, err := command.Output()
		return stdout, stderr.String(), err
	}
	stdout, stderr, err := run("doctor", "--json", "--offline")
	if err == nil {
		t.Fatal("expected missing identity health failure")
	}
	decoder := json.NewDecoder(bytes.NewReader(stdout))
	var report map[string]any
	if err := decoder.Decode(&report); err != nil {
		t.Fatalf("JSON stdout invalid: %q, stderr %s", stdout, stderr)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		t.Fatalf("extra stdout after report: %q", stdout)
	}
	if report["schemaVersion"] != float64(1) || !strings.Contains(stderr, "health check failed") {
		t.Fatalf("wrong health output %q %s", stdout, stderr)
	}
	if err := os.Rename(repoPath, repoPath+"-moved"); err != nil {
		t.Fatal(err)
	}
	stdout, _, err = run("list", "--json")
	if err == nil || len(stdout) != 0 {
		t.Fatalf("missing repo falsely listed: %s %v", stdout, err)
	}
	fresh := filepath.Join(root, "fresh", "config.json")
	t.Setenv("OH_MY_DOT_CONFIG", fresh)
	if _, _, err := run("list", "--json"); err == nil {
		t.Fatal("uninitialized inventory succeeded")
	}
	if _, err := os.Stat(filepath.Dir(fresh)); !os.IsNotExist(err) {
		t.Fatal("inspection created config storage")
	}
	if stdout, stderr, err := run("i", "--folder", filepath.Join(root, "alias-repo"), "--no-interactive"); err != nil {
		t.Fatalf("init alias: %s %s %v", stdout, stderr, err)
	}
}
