package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateFeatureName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid simple name", "git-prompt", false},
		{"valid with underscore", "kubectl_completion", false},
		{"valid with numbers", "python3-venv", false},
		{"empty name", "", true},
		{"invalid spaces", "git prompt", true},
		{"invalid special chars", "git@prompt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFeatureName(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateFeatureName(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
			}
		})
	}
}

func TestValidateStrategy(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid eager", "eager", false},
		{"valid defer", "defer", false},
		{"valid on-command", "on-command", false},
		{"empty (defaults to catalog)", "", false},
		{"invalid strategy", "lazy", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStrategy(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateStrategy(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
			}
		})
	}
}

func TestFeatureConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    FeatureConfig
		wantError bool
	}{
		{
			"valid eager feature",
			FeatureConfig{Name: "git-prompt", Strategy: "eager"},
			false,
		},
		{
			"valid on-command with commands",
			FeatureConfig{Name: "kubectl", Strategy: "on-command", OnCommand: []string{"kubectl"}},
			false,
		},
		{
			"invalid on-command without commands",
			FeatureConfig{Name: "kubectl", Strategy: "on-command"},
			true,
		},
		{
			"invalid name",
			FeatureConfig{Name: "git prompt", Strategy: "eager"},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("FeatureConfig.Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestManifestAddRemoveFeature(t *testing.T) {
	manifest := &FeatureManifest{}

	// Add feature
	feature := FeatureConfig{Name: "git-prompt", Strategy: "defer"}
	if err := manifest.AddFeature(feature); err != nil {
		t.Fatalf("AddFeature() error = %v", err)
	}

	if len(manifest.Features) != 1 {
		t.Errorf("Expected 1 feature, got %d", len(manifest.Features))
	}

	// Add duplicate should fail
	if err := manifest.AddFeature(feature); err == nil {
		t.Error("Expected error when adding duplicate feature")
	}

	// Remove feature
	if err := manifest.RemoveFeature("git-prompt"); err != nil {
		t.Fatalf("RemoveFeature() error = %v", err)
	}

	if len(manifest.Features) != 0 {
		t.Errorf("Expected 0 features, got %d", len(manifest.Features))
	}

	// Remove non-existent should fail
	if err := manifest.RemoveFeature("git-prompt"); err == nil {
		t.Error("Expected error when removing non-existent feature")
	}
}

func TestParseAndWriteManifest(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "omdot-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manifestPath := filepath.Join(tmpDir, "enabled.json")

	// Create manifest
	manifest := &FeatureManifest{
		Features: []FeatureConfig{
			{Name: "git-prompt", Strategy: "defer"},
			{Name: "kubectl", Strategy: "on-command", OnCommand: []string{"kubectl"}},
		},
	}

	// Write manifest
	if err := WriteManifest(manifestPath, manifest); err != nil {
		t.Fatalf("WriteManifest() error = %v", err)
	}

	// Parse manifest
	parsed, err := ParseManifest(manifestPath)
	if err != nil {
		t.Fatalf("ParseManifest() error = %v", err)
	}

	if len(parsed.Features) != 2 {
		t.Errorf("Expected 2 features, got %d", len(parsed.Features))
	}

	// Verify content
	if parsed.Features[0].Name != "git-prompt" {
		t.Errorf("Expected first feature to be 'git-prompt', got '%s'", parsed.Features[0].Name)
	}
}

func TestGetEnabledFeatures(t *testing.T) {
	manifest := &FeatureManifest{
		Features: []FeatureConfig{
			{Name: "enabled-1", Strategy: "eager", Disabled: false},
			{Name: "disabled-1", Strategy: "eager", Disabled: true},
			{Name: "enabled-2", Strategy: "defer", Disabled: false},
		},
	}

	enabled := manifest.GetEnabledFeatures()
	if len(enabled) != 2 {
		t.Errorf("Expected 2 enabled features, got %d", len(enabled))
	}

	for _, f := range enabled {
		if f.Disabled {
			t.Errorf("GetEnabledFeatures() returned disabled feature: %s", f.Name)
		}
	}
}
