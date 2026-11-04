package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestLoadConfigReadOnlyDoesNotCreateStorage(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	path := filepath.Join(t.TempDir(), "absent", "config.json")
	if err := LoadConfig(path, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatalf("read-only load created storage: %v", err)
	}
}

func TestLoadConfigPreservesMalformedFile(t *testing.T) {
	for _, create := range []bool{false, true} {
		t.Run(map[bool]string{false: "read only", true: "create allowed"}[create], func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			path := filepath.Join(t.TempDir(), "config.json")
			original := []byte(`{"repo-path":`)
			if err := os.WriteFile(path, original, 0600); err != nil {
				t.Fatal(err)
			}
			err := LoadConfig(path, create)
			if err == nil || !strings.Contains(err.Error(), path) {
				t.Fatalf("missing contextual error: %v", err)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != string(original) {
				t.Fatal("malformed configuration overwritten")
			}
		})
	}
}
