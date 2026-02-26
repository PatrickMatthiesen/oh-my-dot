package git

import (
	"path/filepath"
	"testing"

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
