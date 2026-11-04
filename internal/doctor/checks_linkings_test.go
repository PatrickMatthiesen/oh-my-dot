package doctor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinkingDiagnosticsDetectInvalidMetadata(t *testing.T) {
	for _, data := range []string{"{broken", "null", `{"a":"relative"}`} {
		t.Run(data, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "linkings.json")
			if err := os.WriteFile(path, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			results := checkLinkings(context{repoPath: root, quiet: true, fix: true})
			if len(results) != 1 || results[0].status != statusError || results[0].fixable {
				t.Fatalf("invalid health %+v", results)
			}
			after, _ := os.ReadFile(path)
			if string(after) != data {
				t.Fatal("diagnosis rewrote metadata")
			}
		})
	}
}
