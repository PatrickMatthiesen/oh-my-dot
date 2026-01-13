package catalog

import (
	"strings"
	"testing"
)

func TestHasFeatureTemplate(t *testing.T) {
	tests := []struct {
		name        string
		featureName string
		shellName   string
		want        bool
	}{
		{"ssh-agent bash (fallback to posix)", "ssh-agent", "bash", true},
		{"ssh-agent zsh (fallback to posix)", "ssh-agent", "zsh", true},
		{"ssh-agent fish", "ssh-agent", "fish", true},
		{"ssh-agent posix", "ssh-agent", "posix", true},
		{"ssh-agent sh (fallback to posix)", "ssh-agent", "sh", true},
		{"non-existent feature", "non-existent", "bash", false},
		{"ssh-agent powershell (no fallback)", "ssh-agent", "powershell", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasFeatureTemplate(tt.featureName, tt.shellName)
			if got != tt.want {
				t.Errorf("HasFeatureTemplate(%q, %q) = %v, want %v", tt.featureName, tt.shellName, got, tt.want)
			}
		})
	}
}

func TestGetFeatureTemplate(t *testing.T) {
	tests := []struct {
		name        string
		featureName string
		shellName   string
		wantError   bool
		contains    string
	}{
		{
			name:        "ssh-agent bash (fallback to posix)",
			featureName: "ssh-agent",
			shellName:   "bash",
			wantError:   false,
			contains:    "SSH_AUTH_SOCK",
		},
		{
			name:        "ssh-agent zsh (fallback to posix)",
			featureName: "ssh-agent",
			shellName:   "zsh",
			wantError:   false,
			contains:    "SSH_AUTH_SOCK",
		},
		{
			name:        "ssh-agent fish",
			featureName: "ssh-agent",
			shellName:   "fish",
			wantError:   false,
			contains:    "SSH_AUTH_SOCK",
		},
		{
			name:        "non-existent feature",
			featureName: "non-existent",
			shellName:   "bash",
			wantError:   true,
			contains:    "",
		},
		{
			name:        "ssh-agent powershell (no fallback)",
			featureName: "ssh-agent",
			shellName:   "powershell",
			wantError:   true,
			contains:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := GetFeatureTemplate(tt.featureName, tt.shellName)
			if (err != nil) != tt.wantError {
				t.Errorf("GetFeatureTemplate(%q, %q) error = %v, wantError %v", tt.featureName, tt.shellName, err, tt.wantError)
				return
			}
			if !tt.wantError && content == "" {
				t.Errorf("GetFeatureTemplate(%q, %q) returned empty content", tt.featureName, tt.shellName)
			}
			if tt.contains != "" && !strings.Contains(content, tt.contains) {
				t.Errorf("GetFeatureTemplate(%q, %q) content does not contain %q", tt.featureName, tt.shellName, tt.contains)
			}
		})
	}
}

func TestGetShellExtension(t *testing.T) {
	tests := []struct {
		shellName string
		want      string
	}{
		{"bash", ".sh"},
		{"zsh", ".sh"},
		{"posix", ".sh"},
		{"fish", ".fish"},
		{"powershell", ".ps1"},
		{"unknown", ".sh"},
	}

	for _, tt := range tests {
		t.Run(tt.shellName, func(t *testing.T) {
			got := GetShellExtension(tt.shellName)
			if got != tt.want {
				t.Errorf("GetShellExtension(%q) = %q, want %q", tt.shellName, got, tt.want)
			}
		})
	}
}
