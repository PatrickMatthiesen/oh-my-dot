package shell

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
)

func TestGeneratePosixInitUsesGeneratedShellRoot(t *testing.T) {
	repoPath := filepath.Join(t.TempDir(), "repo with spaces")
	content := generatePosixInit(repoPath, FeaturesByStrategy{})

	if strings.Contains(content, `dirname "$0"`) {
		t.Fatal("POSIX init should not depend on $0 when sourced")
	}

	want := "OMD_SHELL_ROOT='" + filepath.ToSlash(filepath.Join(repoPath, "omd-shells", "posix")) + "'"
	if !strings.Contains(content, want) {
		t.Fatalf("POSIX init did not contain generated shell root %q\n%s", want, content)
	}
}

func TestDeferredFeaturesLoadInCurrentShell(t *testing.T) {
	features := FeaturesByStrategy{Defer: []string{"example"}}

	tests := []struct {
		name      string
		content   string
		forbidden []string
	}{
		{
			name:      "bash",
			content:   generateBashInit(features),
			forbidden: []string{`) &`},
		},
		{
			name:      "zsh",
			content:   generateZshInit(features),
			forbidden: []string{`) &!`},
		},
		{
			name:      "fish",
			content:   generateFishInit(features),
			forbidden: []string{`source "$feature_file" &`},
		},
		{
			name:      "powershell",
			content:   generatePowerShellInit(features),
			forbidden: []string{"Start-Job"},
		},
		{
			name:      "posix",
			content:   generatePosixInit(t.TempDir(), features),
			forbidden: []string{`) &`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, forbidden := range tt.forbidden {
				if strings.Contains(tt.content, forbidden) {
					t.Fatalf("deferred feature still uses background loading %q\n%s", forbidden, tt.content)
				}
			}
		})
	}
}

func TestGenerateFeatureTemplateOmitsShebang(t *testing.T) {
	content := generateFeatureTemplate("bash", "custom", catalog.FeatureMetadata{}, nil)

	if strings.HasPrefix(content, "#!") {
		t.Fatalf("generic feature template should not include a shebang:\n%s", content)
	}
	if !strings.Contains(content, "# oh-my-dot feature: custom") {
		t.Fatalf("generic feature template missing feature header:\n%s", content)
	}
}
