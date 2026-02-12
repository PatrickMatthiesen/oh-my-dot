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
		{"homebrew-path bash (fallback to posix)", "homebrew-path", "bash", true},
		{"homebrew-path zsh (fallback to posix)", "homebrew-path", "zsh", true},
		{"homebrew-path fish", "homebrew-path", "fish", true},
		{"homebrew-path posix", "homebrew-path", "posix", true},
		{"powershell-prompt powershell", "powershell-prompt", "powershell", true},
		{"powershell-aliases powershell", "powershell-aliases", "powershell", true},
		{"posh-git powershell", "posh-git", "powershell", true},
		{"oh-my-posh bash", "oh-my-posh", "bash", true},
		{"oh-my-posh zsh", "oh-my-posh", "zsh", true},
		{"oh-my-posh fish", "oh-my-posh", "fish", true},
		{"oh-my-posh powershell", "oh-my-posh", "powershell", true},
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
			name:        "homebrew-path bash (fallback to posix)",
			featureName: "homebrew-path",
			shellName:   "bash",
			wantError:   false,
			contains:    "linuxbrew",
		},
		{
			name:        "homebrew-path zsh (fallback to posix)",
			featureName: "homebrew-path",
			shellName:   "zsh",
			wantError:   false,
			contains:    "linuxbrew",
		},
		{
			name:        "homebrew-path fish",
			featureName: "homebrew-path",
			shellName:   "fish",
			wantError:   false,
			contains:    "linuxbrew",
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
		{
			name:        "powershell-prompt powershell",
			featureName: "powershell-prompt",
			shellName:   "powershell",
			wantError:   false,
			contains:    "Get-GitBranch",
		},
		{
			name:        "powershell-aliases powershell",
			featureName: "powershell-aliases",
			shellName:   "powershell",
			wantError:   false,
			contains:    "Set-Alias",
		},
		{
			name:        "posh-git powershell",
			featureName: "posh-git",
			shellName:   "powershell",
			wantError:   false,
			contains:    "Import-Module posh-git",
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

func TestRenderFeatureTemplate(t *testing.T) {
	t.Run("renders with oh-my-posh defaults", func(t *testing.T) {
		content := `url={{ .ThemeURL }} config={{ .ConfigFile }} fallback={{ .DefaultConfigPath }}`

		rendered, err := RenderFeatureTemplate(content, "oh-my-posh", "bash", nil)
		if err != nil {
			t.Fatalf("RenderFeatureTemplate() error = %v", err)
		}

		if !strings.Contains(rendered, "jandedobbeleer.omp.json") {
			t.Fatalf("expected default theme URL in rendered content, got %q", rendered)
		}
		if !strings.Contains(rendered, "$OMD_SHELL_ROOT/features/oh-my-posh.omp.json") {
			t.Fatalf("expected default config path in rendered content, got %q", rendered)
		}
	})

	t.Run("renders with option overrides", func(t *testing.T) {
		content := `url={{ .ThemeURL }} config={{ .ConfigFile }} auto={{ .AutoUpgrade }} custom={{ option "theme" }}`
		options := map[string]any{
			"theme":        "catppuccin",
			"config_file":  "/tmp/custom.omp.json",
			"auto_upgrade": true,
		}

		rendered, err := RenderFeatureTemplate(content, "oh-my-posh", "bash", options)
		if err != nil {
			t.Fatalf("RenderFeatureTemplate() error = %v", err)
		}

		if !strings.Contains(rendered, "catppuccin.omp.json") {
			t.Fatalf("expected overridden theme URL in rendered content, got %q", rendered)
		}
		if !strings.Contains(rendered, "/tmp/custom.omp.json") {
			t.Fatalf("expected overridden config file in rendered content, got %q", rendered)
		}
		if !strings.Contains(rendered, "auto=true") {
			t.Fatalf("expected auto upgrade boolean in rendered content, got %q", rendered)
		}
	})
}
