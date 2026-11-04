package manifest

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCreatePrivateFilePreservesExistingContents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private-runtime")
	if err := CreatePrivateFile(path, []byte("first private contents")); err != nil {
		t.Fatal(err)
	}
	if err := CreatePrivateFile(path, []byte("replacement")); !errors.Is(err, os.ErrExist) {
		t.Fatalf("exclusive create did not report existing file: %v", err)
	}
	contents, err := os.ReadFile(path)
	if err != nil || string(contents) != "first private contents" {
		t.Fatalf("exclusive create replaced existing contents: %q, %v", contents, err)
	}
}

func TestWritePrivateFileRestrictsExistingFileBeforeReplacingContents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private-runtime")
	if err := os.WriteFile(path, []byte("old contents longer than replacement"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := WritePrivateFile(path, []byte("private")); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil || string(contents) != "private" {
		t.Fatalf("private replacement failed: %q, %v", contents, err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatalf("private permissions not restricted: %v, %v", info, err)
		}
	}
}

func TestWritePrivateFileRejectsSymlinkWithoutChangingTarget(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "user-file")
	if err := os.WriteFile(target, []byte("preserve"), 0644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "private-link")
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := WritePrivateFile(path, []byte("private")); err == nil {
		t.Fatal("private writer followed a symlink")
	}
	contents, err := os.ReadFile(target)
	if err != nil || string(contents) != "preserve" {
		t.Fatal("private writer changed symlink target")
	}
}
