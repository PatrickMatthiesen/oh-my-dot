package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v6"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestApplyDistinguishesManagedTargets(t *testing.T) {
	for _, kind := range []string{"missing", "hard link", "managed symlink", "unrelated regular", "unrelated symlink", "broken symlink"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			source := filepath.Join(dir, "source")
			target := filepath.Join(dir, "target")
			if err := os.WriteFile(source, []byte("managed"), 0600); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "hard link":
				if err := os.Link(source, target); err != nil {
					t.Skip(err)
				}
			case "managed symlink":
				if err := os.Symlink(source, target); err != nil {
					t.Skip(err)
				}
			case "unrelated regular":
				if err := os.WriteFile(target, []byte("managed"), 0600); err != nil {
					t.Fatal(err)
				}
			case "unrelated symlink":
				other := filepath.Join(dir, "other")
				os.WriteFile(other, []byte("other"), 0600)
				if err := os.Symlink(other, target); err != nil {
					t.Skip(err)
				}
			case "broken symlink":
				if err := os.Symlink(filepath.Join(dir, "missing"), target); err != nil {
					t.Skip(err)
				}
			}
			action := inspectApplyLink(source, target, true)
			conflict := kind == "unrelated regular" || kind == "unrelated symlink" || kind == "broken symlink"
			if (action.err != nil) != conflict {
				t.Fatalf("classification %+v", action)
			}
			if (kind == "hard link" || kind == "managed symlink") && action.state != "already correct" {
				t.Fatalf("classification %+v", action)
			}
		})
	}
}

func TestApplyDryRunCreatesNothingAndMatchesExecution(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	target := filepath.Join(root, "local")
	if _, err := gogit.PlainInit(repoPath, false); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repoPath, "files"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "files", "config"), []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(map[string]string{"config": target})
	if err := os.WriteFile(filepath.Join(repoPath, "linkings.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	viper.Set("repo-path", repoPath)
	cmd := &cobra.Command{}
	cmd.Flags().Bool("dry-run", true, "")
	cmd.Flags().Bool("no-shell", true, "")
	cmd.Flags().Bool("allow-outside-home", true, "")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := runApply(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("preview wrote target: %v", err)
	}
	if !bytes.Contains(output.Bytes(), []byte("create")) {
		t.Fatalf("missing planned action %s", output.String())
	}
	if err := cmd.Flags().Set("dry-run", "false"); err != nil {
		t.Fatal(err)
	}
	if err := runApply(cmd, nil); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(target)
	if err != nil || string(content) != "content" {
		t.Fatalf("actual output %q %v", content, err)
	}
	if err := runApply(cmd, nil); err != nil {
		t.Fatalf("repeat apply %v", err)
	}
}

func TestApplyPreflightConflictDoesNotPartiallyLink(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	os.MkdirAll(filepath.Join(repo, "files"), 0700)
	links := map[string]string{"first": filepath.Join(root, "first"), "second": filepath.Join(root, "second")}
	for key := range links {
		os.WriteFile(filepath.Join(repo, "files", key), []byte(key), 0600)
	}
	os.WriteFile(links["second"], []byte("unrelated"), 0600)
	data, _ := json.Marshal(links)
	os.WriteFile(filepath.Join(repo, "linkings.json"), data, 0600)
	viper.Set("repo-path", repo)
	cmd := &cobra.Command{}
	cmd.Flags().Bool("no-shell", true, "")
	cmd.Flags().Bool("allow-outside-home", true, "")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := runApply(cmd, nil); err == nil {
		t.Fatal("conflicting target accepted")
	}
	if _, err := os.Lstat(links["first"]); !os.IsNotExist(err) {
		t.Fatalf("preflight mutated valid target %v", err)
	}
	content, _ := os.ReadFile(links["second"])
	if string(content) != "unrelated" {
		t.Fatal("conflict overwritten")
	}
}

func TestKeepManagedSymlinkMaterializesContent(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(source, []byte("keep this"), 0640); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(source, 0640); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(source, target); err != nil {
		t.Skip(err)
	}
	if err := preserveManagedSymlink(source, target); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(source); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(target)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("local not materialized %v %v", info, err)
	}
	content, err := os.ReadFile(target)
	if err != nil || string(content) != "keep this" {
		t.Fatalf("local content %q %v", content, err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0640 {
		t.Fatalf("mode %v", info.Mode().Perm())
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("temporary backup leaked: %v", entries)
	}
}

func TestKeepLeavesUnrelatedSymlinkUntouched(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	other := filepath.Join(dir, "other")
	target := filepath.Join(dir, "target")
	os.WriteFile(source, []byte("managed"), 0600)
	os.WriteFile(other, []byte("private"), 0600)
	if err := os.Symlink(other, target); err != nil {
		t.Skip(err)
	}
	if err := preserveManagedSymlink(source, target); err != nil {
		t.Fatal(err)
	}
	actual, err := os.Readlink(target)
	if err != nil || actual != other {
		t.Fatalf("unrelated symlink changed %q %v", actual, err)
	}
	content, _ := os.ReadFile(target)
	if string(content) != "private" {
		t.Fatal("unrelated content changed")
	}
}

func TestApplyDryRunPreviewsHookWithoutProfileWrites(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash home-profile convention on Unix")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip(err)
	}
	viper.Reset()
	t.Cleanup(viper.Reset)
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := filepath.Join(t.TempDir(), "repo")
	shellDir := filepath.Join(repo, "omd-shells", "bash")
	if err := os.MkdirAll(shellDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shellDir, "enabled.json"), []byte(`{"features":[{"name":"custom","strategy":"eager"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shellDir, "init.sh"), []byte("# generated\n"), 0600); err != nil {
		t.Fatal(err)
	}
	viper.Set("repo-path", repo)
	command := &cobra.Command{}
	command.Flags().Bool("dry-run", true, "")
	var output bytes.Buffer
	command.SetOut(&output)
	if err := runApply(command, nil); err != nil {
		t.Fatal(err)
	}
	profile := filepath.Join(home, ".bashrc")
	if _, err := os.Lstat(profile); !os.IsNotExist(err) {
		t.Fatalf("preview wrote profile %v", err)
	}
	if !strings.Contains(output.String(), "hook: "+profile+" (create)") {
		t.Fatalf("hook omitted from preview %q", output.String())
	}
	command.Flags().Set("dry-run", "false")
	if err := runApply(command, nil); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(profile)
	if err != nil || !strings.Contains(string(content), "oh-my-dot shell") {
		t.Fatalf("actual hook %q %v", content, err)
	}
}

func TestApplyRejectsMalformedHookBeforeLinking(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash home-profile convention on Unix")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip(err)
	}
	viper.Reset()
	t.Cleanup(viper.Reset)
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := filepath.Join(t.TempDir(), "repo")
	shellDir := filepath.Join(repo, "omd-shells", "bash")
	os.MkdirAll(shellDir, 0700)
	os.WriteFile(filepath.Join(shellDir, "enabled.json"), []byte(`{"features":[{"name":"custom","strategy":"eager"}]}`), 0600)
	os.WriteFile(filepath.Join(shellDir, "init.sh"), []byte("# generated\n"), 0600)
	profile := filepath.Join(home, ".bashrc")
	original := "# >>> oh-my-dot shell >>>\nkeep this private content\n"
	os.WriteFile(profile, []byte(original), 0600)
	viper.Set("repo-path", repo)
	command := &cobra.Command{}
	var output bytes.Buffer
	command.SetOut(&output)
	if err := runApply(command, nil); err == nil {
		t.Fatal("malformed hook accepted")
	}
	content, _ := os.ReadFile(profile)
	if string(content) != original {
		t.Fatal("malformed profile changed")
	}
}
