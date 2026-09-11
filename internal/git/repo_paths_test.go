package git

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	gogit "github.com/go-git/go-git/v6"
	"github.com/spf13/viper"
)

func TestLinkAndAddFileAsNestedKeys(t *testing.T) {
	repoPath := t.TempDir()
	repo, err := gogit.PlainInit(repoPath, false)
	if err != nil {
		t.Fatal(err)
	}
	old := viper.GetString("repo-path")
	viper.Set("repo-path", repoPath)
	t.Cleanup(func() { viper.Set("repo-path", old) })
	for _, folder := range []string{"work", "personal"} {
		source := filepath.Join(t.TempDir(), "README.md")
		if err := os.WriteFile(source, []byte(folder), 0644); err != nil {
			t.Fatal(err)
		}
		if err := LinkAndAddFileAs(source, folder+"/README.md"); err != nil {
			t.Fatal(err)
		}
		sourceInfo, _ := os.Stat(source)
		targetInfo, err := os.Stat(filepath.Join(repoPath, "files", folder, "README.md"))
		if err != nil || !os.SameFile(sourceInfo, targetInfo) {
			t.Fatalf("expected hard link: %v", err)
		}
	}
	keys, err := ListFiles()
	if err != nil || !reflect.DeepEqual(keys, []string{"personal/README.md", "work/README.md"}) {
		t.Fatalf("keys=%v error=%v", keys, err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	status, err := wt.Status()
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range keys {
		if status.File("files/"+key).Staging != gogit.Added {
			t.Fatalf("not staged: %s", key)
		}
	}
	duplicate := filepath.Join(t.TempDir(), "README.md")
	if err := os.WriteFile(duplicate, []byte("replacement"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"work/README.md", "WORK/new.md", "personal/README.md/child", "../escape"} {
		if err := LinkAndAddFileAs(duplicate, key); err == nil {
			t.Fatalf("accepted conflicting key %s", key)
		}
	}
	content, err := os.ReadFile(filepath.Join(repoPath, "files", "work", "README.md"))
	if err != nil || string(content) != "work" {
		t.Fatalf("overwrote existing file: %q %v", content, err)
	}
	if err := RemoveFile("work/README.md"); err != nil {
		t.Fatal(err)
	}
	keys, err = ListFiles()
	if err != nil || !reflect.DeepEqual(keys, []string{"personal/README.md"}) {
		t.Fatalf("after removal keys=%v error=%v", keys, err)
	}
}

func TestRemoveFileRejectsSymlinkParent(t *testing.T) {
	repoPath := t.TempDir()
	if _, err := gogit.PlainInit(repoPath, false); err != nil {
		t.Fatal(err)
	}
	old := viper.GetString("repo-path")
	viper.Set("repo-path", repoPath)
	t.Cleanup(func() { viper.Set("repo-path", old) })
	outside := t.TempDir()
	target := filepath.Join(outside, "config")
	if err := os.WriteFile(target, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(repoPath, "files"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repoPath, "files", "linked")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := RemoveFile("linked/config"); err == nil {
		t.Fatal("removed through symlink parent")
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("outside file modified: %v", err)
	}
}
