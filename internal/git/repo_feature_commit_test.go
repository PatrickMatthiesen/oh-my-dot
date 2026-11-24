package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v6"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
)

func TestIsCommittableShellChangePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "commits shell manifest",
			path: "omd-shells/bash/enabled.json",
			want: true,
		},
		{
			name: "commits shell feature file",
			path: "omd-shells/bash/features/git-prompt.sh",
			want: true,
		},
		{
			name: "skips local override manifest",
			path: filepath.ToSlash(filepath.Join("omd-shells", "bash", shell.LocalManifestFileName())),
			want: false,
		},
		{
			name: "skips local override manifest with backslashes",
			path: filepath.Join("omd-shells", "powershell", shell.LocalManifestFileName()),
			want: false,
		},
		{
			name: "skips non shell file",
			path: "files/.gitconfig",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCommittableShellChangePath(tt.path)
			if got != tt.want {
				t.Errorf("isCommittableShellChangePath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestFormatPathSummaryDeterministicAndBounded(t *testing.T) {
	got := formatPathSummary([]string{
		"omd-shells/powershell/features/z.ps1",
		"omd-shells/powershell/enabled.json",
		"omd-shells/powershell/features/a.ps1",
	})
	want := "omd-shells/powershell/enabled.json, omd-shells/powershell/features/a.ps1, omd-shells/powershell/features/z.ps1"
	if got != want {
		t.Fatalf("formatPathSummary() = %q, want %q", got, want)
	}

	many := formatPathSummary([]string{"f", "e", "d", "c", "b", "a", "g"})
	if many != "a, b, c, d, e, and 2 more" {
		t.Fatalf("formatPathSummary(7) = %q, want %q", many, "a, b, c, d, e, and 2 more")
	}
}

func TestCheckShellFeatureCommitSafetyDeterministicOrder(t *testing.T) {
	repoPath := t.TempDir()
	repo, err := gogit.PlainInit(repoPath, false)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := repo.Config()
	if err != nil {
		t.Fatal(err)
	}
	cfg.User.Name = "Test"
	cfg.User.Email = "test@example.invalid"
	if err := repo.SetConfig(cfg); err != nil {
		t.Fatal(err)
	}

	for _, rel := range []string{
		"omd-shells/powershell/enabled.json",
		"omd-shells/powershell/features/oh-my-dot-completion.ps1",
	} {
		full := filepath.Join(repoPath, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("pending\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := CheckShellFeatureCommitSafety(""); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected empty repoPath to be rejected, got %v", err)
	}

	wantPaths := "omd-shells/powershell/enabled.json, omd-shells/powershell/features/oh-my-dot-completion.ps1"
	for range 10 {
		err := CheckShellFeatureCommitSafety(repoPath)
		if err == nil {
			t.Fatal("expected error for pending shell files")
		}
		if !strings.Contains(err.Error(), wantPaths) {
			t.Fatalf("expected deterministic paths %q in %q", wantPaths, err.Error())
		}
		if !strings.Contains(err.Error(), repoPath) || !strings.Contains(err.Error(), "--local") {
			t.Fatalf("expected repo path and --local hint in %q", err.Error())
		}
	}
}
