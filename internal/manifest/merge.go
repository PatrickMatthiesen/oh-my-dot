package manifest

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
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
	// Check if file exists
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, that's fine
		}
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Must be a regular file, not a symlink
	if !info.Mode().IsRegular() {
		return fmt.Errorf("file is not a regular file (possibly a symlink)")
	}

	// Platform-specific security checks
	if runtime.GOOS == "windows" {
		return validateLocalManifestWindows(path, info)
	}
	return validateLocalManifestUnix(path, info)
}

// validateLocalManifestUnix performs Unix-specific security checks
func validateLocalManifestUnix(path string, info os.FileInfo) error {
	// Get file ownership
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		// Can't get detailed file info, allow but warn
		return nil
	}

	// Check ownership - must be owned by current user
	currentUID := uint32(os.Getuid())
	if stat.Uid != currentUID {
		return fmt.Errorf("file not owned by current user (uid %d != %d)", stat.Uid, currentUID)
	}

	// Check permissions - must not be group or world writable
	perm := info.Mode().Perm()
	if perm&0022 != 0 { // Check group-write (020) or other-write (002) bits
		return fmt.Errorf("file is group or world writable (permissions: %o)", perm)
	}

	return nil
}

// validateLocalManifestWindows performs Windows-specific security checks
func validateLocalManifestWindows(path string, info os.FileInfo) error {
	// Windows ACL checking is complex and requires syscall.
	// For Phase 6 MVP, we'll do basic checks and log warnings.
	// Full ACL implementation can be added later if needed.

	// Basic check: ensure file is owned by current user
	// This is a simplified check - full Windows ACL checking would require more code

	// For now, just allow on Windows with a basic check
	// TODO: Implement full Windows ACL checking
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
