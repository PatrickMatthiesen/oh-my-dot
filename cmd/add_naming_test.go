package cmd

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestAddNameCandidates(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home", "person")
	for _, tc := range []struct {
		name, destination string
		want              []string
	}{
		{"home file", filepath.Join(home, ".bashrc"), []string{".bashrc"}},
		{"nested dotfile", filepath.Join(home, ".ssh", "config"), []string{"config", ".ssh/config"}},
		{"three parent limit", filepath.Join(home, "a", "b", "c", "d", "config"), []string{"config", "d/config", "c/d/config", "b/c/d/config"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := addNameCandidates(tc.destination, home)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
