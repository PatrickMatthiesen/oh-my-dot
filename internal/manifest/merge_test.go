package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMergeManifests_NoLocal(t *testing.T) {
	base := &FeatureManifest{
		Features: []FeatureConfig{
			{Name: "git-prompt", Strategy: "eager"},
			{Name: "kubectl", Strategy: "on-command", OnCommand: []string{"kubectl"}},
		},
	}

	merged := MergeManifests(base, nil)

	if len(merged.Features) != 2 {
		t.Errorf("Expected 2 features, got %d", len(merged.Features))
	}

	for _, f := range merged.Features {
		if f.Override.HasLocal {
			t.Errorf("Feature %s should not have local override flag set", f.Name)
		}
		if f.Override.IsFromLocal {
			t.Errorf("Feature %s should not be marked as from local", f.Name)
		}
		if f.Override.IsOverridden {
			t.Errorf("Feature %s should not be marked as overridden", f.Name)
		}
	}
}

func TestMergeManifests_StrategyOverride(t *testing.T) {
	base := &FeatureManifest{
		Features: []FeatureConfig{
			{Name: "git-prompt", Strategy: "defer"},
			{Name: "kubectl", Strategy: "on-command", OnCommand: []string{"kubectl"}},
		},
	}

	local := &FeatureManifest{
		Features: []FeatureConfig{
			{Name: "git-prompt", Strategy: "eager"}, // Override to eager
		},
	}

	merged := MergeManifests(base, local)

	if len(merged.Features) != 2 {
		t.Errorf("Expected 2 features, got %d", len(merged.Features))
	}

	// Check git-prompt was overridden
	gitPrompt := merged.Features[0]
	if gitPrompt.Name != "git-prompt" {
		t.Errorf("Expected first feature to be git-prompt, got %s", gitPrompt.Name)
	}
	if gitPrompt.Strategy != "eager" {
		t.Errorf("Expected strategy to be overridden to 'eager', got '%s'", gitPrompt.Strategy)
	}
	if !gitPrompt.Override.IsOverridden {
		t.Error("Expected feature to be marked as overridden")
	}
	if gitPrompt.Override.IsFromLocal {
		t.Error("Expected feature to not be marked as from local")
	}

	// Check kubectl was not overridden
	kubectl := merged.Features[1]
	if kubectl.Name != "kubectl" {
		t.Errorf("Expected second feature to be kubectl, got %s", kubectl.Name)
	}
	if kubectl.Strategy != "on-command" {
		t.Errorf("Expected strategy to remain 'on-command', got '%s'", kubectl.Strategy)
	}
	if kubectl.Override.IsOverridden {
		t.Error("Expected feature to not be marked as overridden")
	}
	if !kubectl.Override.HasLocal {
		t.Error("Expected feature to have HasLocal flag set")
	}
}

func TestMergeManifests_OnCommandOverride(t *testing.T) {
	base := &FeatureManifest{
		Features: []FeatureConfig{
			{Name: "nvm", Strategy: "on-command", OnCommand: []string{"nvm"}},
		},
	}

	local := &FeatureManifest{
		Features: []FeatureConfig{
			{Name: "nvm", Strategy: "on-command", OnCommand: []string{"nvm", "node", "npm"}},
		},
	}

	merged := MergeManifests(base, local)

	if len(merged.Features) != 1 {
		t.Errorf("Expected 1 feature, got %d", len(merged.Features))
	}

	nvm := merged.Features[0]
	if len(nvm.OnCommand) != 3 {
		t.Errorf("Expected 3 commands, got %d", len(nvm.OnCommand))
	}

	expectedCommands := []string{"nvm", "node", "npm"}
	for i, cmd := range nvm.OnCommand {
		if cmd != expectedCommands[i] {
			t.Errorf("Expected command %d to be '%s', got '%s'", i, expectedCommands[i], cmd)
		}
	}

	if !nvm.Override.IsOverridden {
		t.Error("Expected feature to be marked as overridden")
	}
}

func TestMergeManifests_DisabledFlag(t *testing.T) {
	base := &FeatureManifest{
		Features: []FeatureConfig{
			{Name: "git-prompt", Strategy: "eager", Disabled: false},
			{Name: "kubectl", Strategy: "on-command", OnCommand: []string{"kubectl"}, Disabled: false},
		},
	}

	local := &FeatureManifest{
		Features: []FeatureConfig{
			{Name: "git-prompt", Disabled: true}, // Disable via local
		},
	}

	merged := MergeManifests(base, local)

	gitPrompt := merged.Features[0]
	if !gitPrompt.Disabled {
		t.Error("Expected git-prompt to be disabled")
	}
	if !gitPrompt.Override.IsOverridden {
		t.Error("Expected feature to be marked as overridden")
	}

	kubectl := merged.Features[1]
	if kubectl.Disabled {
		t.Error("Expected kubectl to remain enabled")
	}
}

func TestMergeManifests_LocalOnlyFeatures(t *testing.T) {
	base := &FeatureManifest{
		Features: []FeatureConfig{
			{Name: "git-prompt", Strategy: "eager"},
		},
	}

	local := &FeatureManifest{
		Features: []FeatureConfig{
			{Name: "local-only-feature", Strategy: "defer"},
		},
	}

	merged := MergeManifests(base, local)

	if len(merged.Features) != 2 {
		t.Errorf("Expected 2 features, got %d", len(merged.Features))
	}

	// Base feature should come first
	if merged.Features[0].Name != "git-prompt" {
		t.Errorf("Expected first feature to be git-prompt, got %s", merged.Features[0].Name)
	}

	// Local-only feature should be appended
	localFeature := merged.Features[1]
	if localFeature.Name != "local-only-feature" {
		t.Errorf("Expected second feature to be local-only-feature, got %s", localFeature.Name)
	}
	if !localFeature.Override.IsFromLocal {
		t.Error("Expected feature to be marked as from local")
	}
	if localFeature.Override.IsOverridden {
		t.Error("Expected feature to not be marked as overridden")
	}
	if !localFeature.Override.HasLocal {
		t.Error("Expected feature to have HasLocal flag set")
	}
}

func TestMergeManifests_OrderPreservation(t *testing.T) {
	base := &FeatureManifest{
		Features: []FeatureConfig{
			{Name: "feature-a", Strategy: "eager"},
			{Name: "feature-b", Strategy: "defer"},
			{Name: "feature-c", Strategy: "on-command", OnCommand: []string{"cmd"}},
		},
	}

	local := &FeatureManifest{
		Features: []FeatureConfig{
			{Name: "feature-b", Strategy: "eager"}, // Override middle feature
			{Name: "local-feature", Strategy: "defer"},
		},
	}

	merged := MergeManifests(base, local)

	if len(merged.Features) != 4 {
		t.Errorf("Expected 4 features, got %d", len(merged.Features))
	}

	// Check order: base features first (in original order), then local-only
	expectedOrder := []string{"feature-a", "feature-b", "feature-c", "local-feature"}
	for i, expected := range expectedOrder {
		if merged.Features[i].Name != expected {
			t.Errorf("Expected feature at position %d to be %s, got %s", i, expected, merged.Features[i].Name)
		}
	}

	// Verify override marker on feature-b
	if !merged.Features[1].Override.IsOverridden {
		t.Error("Expected feature-b to be marked as overridden")
	}

	// Verify local marker on local-feature
	if !merged.Features[3].Override.IsFromLocal {
		t.Error("Expected local-feature to be marked as from local")
	}
}

func TestMergedManifest_GetEnabledFeatures(t *testing.T) {
	base := &FeatureManifest{
		Features: []FeatureConfig{
			{Name: "enabled-1", Strategy: "eager", Disabled: false},
			{Name: "disabled-1", Strategy: "defer", Disabled: true},
			{Name: "enabled-2", Strategy: "on-command", OnCommand: []string{"cmd"}, Disabled: false},
		},
	}

	merged := MergeManifests(base, nil)
	enabled := merged.GetEnabledFeatures()

	if len(enabled) != 2 {
		t.Errorf("Expected 2 enabled features, got %d", len(enabled))
	}

	expectedNames := []string{"enabled-1", "enabled-2"}
	for i, expected := range expectedNames {
		if enabled[i].Name != expected {
			t.Errorf("Expected enabled feature %d to be %s, got %s", i, expected, enabled[i].Name)
		}
	}
}

func TestValidateLocalManifest_NotExist(t *testing.T) {
	err := ValidateLocalManifest("/nonexistent/path/enabled.local.json")
	if err != nil {
		t.Errorf("Expected nil error for non-existent file, got: %v", err)
	}
}

func TestValidateLocalManifest_RegularFile(t *testing.T) {
	// Create a temp file with proper permissions
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "enabled.local.json")

	err := os.WriteFile(testFile, []byte(`{"features":[]}`), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = ValidateLocalManifest(testFile)
	if err != nil {
		t.Errorf("Expected nil error for valid file, got: %v", err)
	}
}

func TestValidateLocalManifest_Symlink(t *testing.T) {
	// Create a temp file and symlink
	tmpDir := t.TempDir()
	realFile := filepath.Join(tmpDir, "real.json")
	symlinkFile := filepath.Join(tmpDir, "enabled.local.json")

	err := os.WriteFile(realFile, []byte(`{"features":[]}`), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = os.Symlink(realFile, symlinkFile)
	if err != nil {
		t.Skipf("Skipping symlink test (symlink creation failed: %v)", err)
	}

	err = ValidateLocalManifest(symlinkFile)
	if err == nil {
		t.Error("Expected error for symlink, got nil")
	}
	if err != nil && err.Error() != "file is not a regular file (possibly a symlink)" {
		t.Errorf("Expected symlink error, got: %v", err)
	}
}

func TestValidateLocalManifest_GroupWritable(t *testing.T) {
	// Skip on Windows
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping Unix permission test on Windows")
	}

	// Create a temp file with group-writable permissions
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "enabled.local.json")

	err := os.WriteFile(testFile, []byte(`{"features":[]}`), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Explicitly set group-writable permissions (bypasses umask)
	err = os.Chmod(testFile, 0620)
	if err != nil {
		t.Fatalf("Failed to chmod file: %v", err)
	}

	err = ValidateLocalManifest(testFile)
	if err == nil {
		t.Error("Expected error for group-writable file, got nil")
	}
}

func TestValidateLocalManifest_WorldWritable(t *testing.T) {
	// Skip on Windows
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping Unix permission test on Windows")
	}

	// Create a temp file with world-writable permissions
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "enabled.local.json")

	err := os.WriteFile(testFile, []byte(`{"features":[]}`), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Explicitly set world-writable permissions (bypasses umask)
	err = os.Chmod(testFile, 0602)
	if err != nil {
		t.Fatalf("Failed to chmod file: %v", err)
	}

	err = ValidateLocalManifest(testFile)
	if err == nil {
		t.Error("Expected error for world-writable file, got nil")
	}
}

func TestParseManifestWithLocal_NoLocal(t *testing.T) {
	// Create base manifest
	tmpDir := t.TempDir()
	baseFile := filepath.Join(tmpDir, "enabled.json")
	localFile := filepath.Join(tmpDir, "enabled.local.json")

	baseContent := `{
		"features": [
			{"name": "git-prompt", "strategy": "eager"}
		]
	}`
	err := os.WriteFile(baseFile, []byte(baseContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create base file: %v", err)
	}

	// Local file doesn't exist
	merged, err := ParseManifestWithLocal(baseFile, localFile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(merged.Features) != 1 {
		t.Errorf("Expected 1 feature, got %d", len(merged.Features))
	}

	if merged.Features[0].Override.HasLocal {
		t.Error("Expected HasLocal to be false when no local file exists")
	}
}

func TestParseManifestWithLocal_WithLocal(t *testing.T) {
	// Create base and local manifests
	tmpDir := t.TempDir()
	baseFile := filepath.Join(tmpDir, "enabled.json")
	localFile := filepath.Join(tmpDir, "enabled.local.json")

	baseContent := `{
		"features": [
			{"name": "git-prompt", "strategy": "defer"}
		]
	}`
	err := os.WriteFile(baseFile, []byte(baseContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create base file: %v", err)
	}

	localContent := `{
		"features": [
			{"name": "git-prompt", "strategy": "eager"}
		]
	}`
	err = os.WriteFile(localFile, []byte(localContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create local file: %v", err)
	}

	merged, err := ParseManifestWithLocal(baseFile, localFile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(merged.Features) != 1 {
		t.Errorf("Expected 1 feature, got %d", len(merged.Features))
	}

	// Check override was applied
	if merged.Features[0].Strategy != "eager" {
		t.Errorf("Expected strategy to be overridden to 'eager', got '%s'", merged.Features[0].Strategy)
	}

	if !merged.Features[0].Override.IsOverridden {
		t.Error("Expected feature to be marked as overridden")
	}
}

func TestParseManifestWithLocal_InvalidLocal(t *testing.T) {
	// Create base and invalid local manifests
	tmpDir := t.TempDir()
	baseFile := filepath.Join(tmpDir, "enabled.json")
	localFile := filepath.Join(tmpDir, "enabled.local.json")

	baseContent := `{
		"features": [
			{"name": "git-prompt", "strategy": "eager"}
		]
	}`
	err := os.WriteFile(baseFile, []byte(baseContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create base file: %v", err)
	}

	// Invalid JSON
	localContent := `{invalid json`
	err = os.WriteFile(localFile, []byte(localContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create local file: %v", err)
	}

	// Should fall back to base manifest with warning
	merged, err := ParseManifestWithLocal(baseFile, localFile)
	if err != nil {
		t.Fatalf("Expected no error (should fall back to base), got: %v", err)
	}

	if len(merged.Features) != 1 {
		t.Errorf("Expected 1 feature from base, got %d", len(merged.Features))
	}

	if merged.Features[0].Override.HasLocal {
		t.Error("Expected HasLocal to be false when local is invalid")
	}
}

func TestParseManifestWithLocal_UnsafeLocal(t *testing.T) {
	// Skip on Windows
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping Unix permission test on Windows")
	}

	// Create base and unsafe local manifests
	tmpDir := t.TempDir()
	baseFile := filepath.Join(tmpDir, "enabled.json")
	localFile := filepath.Join(tmpDir, "enabled.local.json")

	baseContent := `{
		"features": [
			{"name": "git-prompt", "strategy": "eager"}
		]
	}`
	err := os.WriteFile(baseFile, []byte(baseContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create base file: %v", err)
	}

	localContent := `{
		"features": [
			{"name": "git-prompt", "strategy": "defer"}
		]
	}`
	err = os.WriteFile(localFile, []byte(localContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create local file: %v", err)
	}

	// Explicitly set world-writable permissions (unsafe)
	err = os.Chmod(localFile, 0666)
	if err != nil {
		t.Fatalf("Failed to chmod local file: %v", err)
	}

	// Should fall back to base manifest with warning
	merged, err := ParseManifestWithLocal(baseFile, localFile)
	if err != nil {
		t.Fatalf("Expected no error (should fall back to base), got: %v", err)
	}

	if len(merged.Features) != 1 {
		t.Errorf("Expected 1 feature from base, got %d", len(merged.Features))
	}

	// Should use base strategy (eager), not local strategy (defer)
	if merged.Features[0].Strategy != "eager" {
		t.Errorf("Expected base strategy 'eager', got '%s'", merged.Features[0].Strategy)
	}

	if merged.Features[0].Override.HasLocal {
		t.Error("Expected HasLocal to be false when local is unsafe")
	}
}
