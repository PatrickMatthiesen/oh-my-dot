package cmd

import (
	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
)

func TestFilterFeaturesByShells(t *testing.T) {
	tests := []struct {
		name           string
		features       []catalog.FeatureMetadata
		shells         []string
		expectedCount  int
		expectedNames  []string
	}{
		{
			name: "filter bash features",
			features: []catalog.FeatureMetadata{
				{Name: "bash-only", SupportedShells: []string{"bash"}},
				{Name: "zsh-only", SupportedShells: []string{"zsh"}},
				{Name: "multi-shell", SupportedShells: []string{"bash", "zsh"}},
			},
			shells:        []string{"bash"},
			expectedCount: 2,
			expectedNames: []string{"bash-only", "multi-shell"},
		},
		{
			name: "filter multiple shells",
			features: []catalog.FeatureMetadata{
				{Name: "bash-only", SupportedShells: []string{"bash"}},
				{Name: "zsh-only", SupportedShells: []string{"zsh"}},
				{Name: "powershell-only", SupportedShells: []string{"powershell"}},
				{Name: "multi-shell", SupportedShells: []string{"bash", "zsh"}},
			},
			shells:        []string{"bash", "zsh"},
			expectedCount: 3,
			expectedNames: []string{"bash-only", "zsh-only", "multi-shell"},
		},
		{
			name: "no matching shells",
			features: []catalog.FeatureMetadata{
				{Name: "bash-only", SupportedShells: []string{"bash"}},
				{Name: "zsh-only", SupportedShells: []string{"zsh"}},
			},
			shells:        []string{"powershell"},
			expectedCount: 0,
			expectedNames: []string{},
		},
		{
			name: "empty shells list",
			features: []catalog.FeatureMetadata{
				{Name: "feature1", SupportedShells: []string{"bash"}},
				{Name: "feature2", SupportedShells: []string{"zsh"}},
			},
			shells:        []string{},
			expectedCount: 2,
			expectedNames: []string{"feature1", "feature2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterFeaturesByShells(tt.features, tt.shells)

			if len(result) != tt.expectedCount {
				t.Errorf("expected %d features, got %d", tt.expectedCount, len(result))
			}

			// Check if expected names are present
			resultNames := make(map[string]bool)
			for _, f := range result {
				resultNames[f.Name] = true
			}

			for _, expectedName := range tt.expectedNames {
				if !resultNames[expectedName] {
					t.Errorf("expected feature '%s' not found in results", expectedName)
				}
			}
		})
	}
}
