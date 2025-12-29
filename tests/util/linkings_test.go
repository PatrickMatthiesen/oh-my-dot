package util_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/symlink"
)

func Fuzz_BuildLinkPath(f *testing.F) {
	home, err := os.UserHomeDir()
	if err != nil {
		f.Error(err)
	}

	f.Add(filepath.Join(home, "test.txt"), "~/test.txt")
	f.Add(filepath.Join(home, "test/test.txt"), "~/test/test.txt")
	f.Add(filepath.Join(home, "test/test/test.txt"), "~/test/test/test.txt")

	f.Fuzz(func(t *testing.T, path string, expected string) {
		link, err := symlink.BuildLinkPath(path)
		if err != nil {
			t.Error(err)
		}
		if link != expected {
			t.Errorf("Expected %s, got %s", expected, link)
		}
	})
}