package git

import (
	"path/filepath"
	"testing"
)

func TestIsCommittableAgentChangePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "commits agent skill file",
			path: "omd-agents/skills/code-review/SKILL.md",
			want: true,
		},
		{
			name: "commits agent skill file with backslashes",
			path: filepath.Join("omd-agents", "skills", "code-review", "SKILL.md"),
			want: true,
		},
		{
			name: "skips shell feature file",
			path: "omd-shells/bash/features/git-prompt.sh",
			want: false,
		},
		{
			name: "skips ordinary dotfile",
			path: "files/.gitconfig",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCommittableAgentChangePath(tt.path)
			if got != tt.want {
				t.Fatalf("isCommittableAgentChangePath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
