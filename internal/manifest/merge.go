package manifest

import (
	"fmt"
	"os"
)

// LocalOverride represents metadata about local overrides
type LocalOverride struct {
	HasLocal     bool
	IsFromLocal  bool
	IsOverridden bool
}

// FeatureWithOverride wraps a FeatureConfig with local override info
type FeatureWithOverride struct {
	FeatureConfig
	Override LocalOverride
}

// MergedManifest represents a manifest with local overrides applied
type MergedManifest struct {
	Features []FeatureWithOverride
}

// MergeManifests merges a base manifest with a local override manifest
// Local manifest can:
// - Override strategy for existing features
// - Set disabled flag for existing features
// - Add new local-only features
// The merge preserves the order of the base manifest and appends local-only features
func MergeManifests(base, local *FeatureManifest) *MergedManifest {
	merged := &MergedManifest{
		Features: make([]FeatureWithOverride, 0),
	}

	if local == nil {
		// No local overrides, just wrap base features
		for _, f := range base.Features {
			merged.Features = append(merged.Features, FeatureWithOverride{
				FeatureConfig: f,
				Override: LocalOverride{
					HasLocal:     false,
					IsFromLocal:  false,
					IsOverridden: false,
				},
			})
		}
		return merged
	}

	// Create a map of local features for quick lookup
	localFeatures := make(map[string]FeatureConfig)
	for _, f := range local.Features {
		localFeatures[f.Name] = f
	}

	// Process base features, applying local overrides
	for _, baseFeature := range base.Features {
		if localFeature, exists := localFeatures[baseFeature.Name]; exists {
			// Feature exists in both - apply overrides
			mergedFeature := baseFeature // Start with base

			// Override strategy if set in local
			if localFeature.Strategy != "" {
				mergedFeature.Strategy = localFeature.Strategy
			}

			// Override onCommand if set in local
			if len(localFeature.OnCommand) > 0 {
				mergedFeature.OnCommand = localFeature.OnCommand
			}

			// Always use local disabled flag (explicit override)
			mergedFeature.Disabled = localFeature.Disabled

			merged.Features = append(merged.Features, FeatureWithOverride{
				FeatureConfig: mergedFeature,
				Override: LocalOverride{
					HasLocal:     true,
					IsFromLocal:  false,
					IsOverridden: true,
				},
			})

			// Mark as processed
			delete(localFeatures, baseFeature.Name)
		} else {
			// Feature only in base
			merged.Features = append(merged.Features, FeatureWithOverride{
				FeatureConfig: baseFeature,
				Override: LocalOverride{
					HasLocal:     true, // local file exists, but feature not in it
					IsFromLocal:  false,
					IsOverridden: false,
				},
			})
		}
	}

	// Add remaining local-only features (not in base)
	for _, localFeature := range localFeatures {
		merged.Features = append(merged.Features, FeatureWithOverride{
			FeatureConfig: localFeature,
			Override: LocalOverride{
				HasLocal:     true,
				IsFromLocal:  true,
				IsOverridden: false,
			},
		})
	}

	return merged
}

// GetEnabledFeatures returns all enabled features from merged manifest
func (m *MergedManifest) GetEnabledFeatures() []FeatureConfig {
	var enabled []FeatureConfig
	for _, f := range m.Features {
		if !f.Disabled {
			enabled = append(enabled, f.FeatureConfig)
		}
	}
	return enabled
}

// ValidateLocalManifest checks if a local manifest file is safe to load
// Returns nil if safe, error with reason if unsafe
func ValidateLocalManifest(path string) error {
	f, err := openAndValidateConfig(path)
	if err != nil {
		return err
	}
	f.Close()
	return nil
}



// ParseManifestWithLocal reads both base and local manifests, validates security,
// and returns a merged manifest
func ParseManifestWithLocal(basePath, localPath string) (*MergedManifest, error) {
	// Parse base manifest (required)
	base, err := ParseManifest(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base manifest: %w", err)
	}

	// Check if local manifest exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		// No local manifest, return base only
		return MergeManifests(base, nil), nil
	}

	// Validate local manifest security
	if err := ValidateLocalManifest(localPath); err != nil {
		// Security validation failed - log warning and ignore local manifest
		fmt.Fprintf(os.Stderr, "oh-my-dot: warning: %s is unsafe (%v), ignoring\n", localPath, err)
		return MergeManifests(base, nil), nil
	}

	// Parse local manifest
	local, err := ParseManifest(localPath)
	if err != nil {
		// Local manifest is invalid - log warning and ignore
		fmt.Fprintf(os.Stderr, "oh-my-dot: warning: failed to parse %s (%v), ignoring\n", localPath, err)
		return MergeManifests(base, nil), nil
	}

	// Merge manifests
	return MergeManifests(base, local), nil
}
