package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

// FeatureConfig represents a single feature configuration in enabled.json
type FeatureConfig struct {
	Name      string         `json:"name"`
	Strategy  string         `json:"strategy,omitempty"`  // "eager", "defer", or "on-command"
	OnCommand []string       `json:"onCommand,omitempty"` // Commands that trigger on-command loading
	Disabled  bool           `json:"disabled,omitempty"`  // If true, feature is disabled
	Options   map[string]any `json:"options,omitempty"`   // User-provided option values
}

// FeatureManifest represents the enabled.json file structure
type FeatureManifest struct {
	Features []FeatureConfig `json:"features"`
}

var (
	// featureNameRegex validates feature names (alphanumeric, hyphens, underscores)
	featureNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

	// validStrategies are the allowed load strategies
	validStrategies = map[string]bool{
		"eager":      true,
		"defer":      true,
		"on-command": true,
	}
)

// ValidateFeatureName checks if a feature name is valid
func ValidateFeatureName(name string) error {
	if name == "" {
		return fmt.Errorf("feature name cannot be empty")
	}
	if !featureNameRegex.MatchString(name) {
		return fmt.Errorf("feature name must contain only alphanumeric characters, hyphens, and underscores")
	}
	return nil
}

// ValidateStrategy checks if a strategy is valid
func ValidateStrategy(strategy string) error {
	if strategy == "" {
		return nil // Empty strategy is allowed (defaults to catalog)
	}
	if !validStrategies[strategy] {
		return fmt.Errorf("invalid strategy '%s': must be 'eager', 'defer', or 'on-command'", strategy)
	}
	return nil
}

// Validate validates a feature configuration
func (f *FeatureConfig) Validate() error {
	if err := ValidateFeatureName(f.Name); err != nil {
		return err
	}
	if err := ValidateStrategy(f.Strategy); err != nil {
		return err
	}
	// Require onCommand if strategy is on-command
	if f.Strategy == "on-command" && len(f.OnCommand) == 0 {
		return fmt.Errorf("feature '%s' uses on-command strategy but has no trigger commands", f.Name)
	}
	return nil
}

// ParseManifest reads and parses a manifest file
func ParseManifest(path string) (*FeatureManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest FeatureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest JSON: %w", err)
	}

	// Validate all features
	for i, feature := range manifest.Features {
		if err := feature.Validate(); err != nil {
			return nil, fmt.Errorf("feature at index %d: %w", i, err)
		}
	}

	return &manifest, nil
}

// WriteManifest writes a manifest to a file with pretty formatting
func WriteManifest(path string, manifest *FeatureManifest) error {
	// Pretty-print with 2-space indent
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Add newline at end of file
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// AddFeature adds a feature to the manifest
func (m *FeatureManifest) AddFeature(feature FeatureConfig) error {
	if err := feature.Validate(); err != nil {
		return err
	}

	// Check if feature already exists
	for _, f := range m.Features {
		if f.Name == feature.Name {
			return fmt.Errorf("feature '%s' already exists", feature.Name)
		}
	}

	m.Features = append(m.Features, feature)
	return nil
}

// RemoveFeature removes a feature from the manifest
func (m *FeatureManifest) RemoveFeature(name string) error {
	for i, f := range m.Features {
		if f.Name == name {
			m.Features = append(m.Features[:i], m.Features[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("feature '%s' not found", name)
}

// GetFeature retrieves a feature by name
func (m *FeatureManifest) GetFeature(name string) (*FeatureConfig, error) {
	for i := range m.Features {
		if m.Features[i].Name == name {
			return &m.Features[i], nil
		}
	}
	return nil, fmt.Errorf("feature '%s' not found", name)
}

// UpdateFeature updates an existing feature
func (m *FeatureManifest) UpdateFeature(name string, updater func(*FeatureConfig)) error {
	for i := range m.Features {
		if m.Features[i].Name == name {
			updater(&m.Features[i])
			return m.Features[i].Validate()
		}
	}
	return fmt.Errorf("feature '%s' not found", name)
}

// HasFeatures returns true if the manifest has any features
func (m *FeatureManifest) HasFeatures() bool {
	return len(m.Features) > 0
}

// GetEnabledFeatures returns all enabled features
func (m *FeatureManifest) GetEnabledFeatures() []FeatureConfig {
	var enabled []FeatureConfig
	for _, f := range m.Features {
		if !f.Disabled {
			enabled = append(enabled, f)
		}
	}
	return enabled
}
