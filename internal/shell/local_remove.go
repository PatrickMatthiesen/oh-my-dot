package shell

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
)

// LocalFeatureRemovalPlan removes a device-only declaration and implementation.
// Removing an override restores the shared feature without modifying its files.
type LocalFeatureRemovalPlan struct {
	Changed          bool // Whether the selected local declaration exists.
	repoPath         string
	shellName        string
	manifestPath     string
	originalManifest []byte
	updatedManifest  *manifest.FeatureManifest
	scriptPath       string
	originalScript   []byte
	runtimeDirectory string
}

// PrepareLocalFeatureRemoval validates local ownership and snapshots all targets
// before removing a private feature or restoring an inherited shared feature.
func PrepareLocalFeatureRemoval(repoPath, shellName, featureName string) (*LocalFeatureRemovalPlan, error) {
	if err := CheckLocalRuntimeTracking(repoPath); err != nil {
		return nil, err
	}
	if !IsShellSupported(shellName) {
		return nil, fmt.Errorf("unsupported shell: %s", shellName)
	}
	if err := manifest.ValidateFeatureName(featureName); err != nil {
		return nil, err
	}
	plan := &LocalFeatureRemovalPlan{repoPath: repoPath, shellName: shellName, manifestPath: GetLocalManifestPath(repoPath, shellName)}
	file, err := manifest.OpenValidatedLocalManifest(plan.manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read local declaration: %w", err)
	}
	if file == nil {
		return plan, nil
	}
	defer file.Close()
	plan.originalManifest, err = io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("snapshot local declaration: %w", err)
	}
	local, err := manifest.ParseManifestReader(bytes.NewReader(plan.originalManifest))
	if err != nil {
		return nil, fmt.Errorf("parse local declaration: %w", err)
	}
	if _, err := local.GetFeature(featureName); err != nil {
		return plan, nil
	}
	base, err := manifest.ParseManifest(GetManifestPath(repoPath, shellName))
	if err != nil {
		return nil, fmt.Errorf("read shared declaration: %w", err)
	}
	path, err := GetLocalFeatureFilePath(repoPath, shellName, featureName)
	if err != nil {
		return nil, err
	}
	plan.runtimeDirectory = filepath.Dir(path)
	if err := ensurePrivateRuntimeDirectory(plan.runtimeDirectory, true); err != nil {
		return nil, err
	}
	if _, err := base.GetFeature(featureName); err != nil {
		path, err = GetEffectiveFeatureFilePath(repoPath, shellName, featureName)
		if err != nil {
			return nil, err
		}
	}
	data, err := readRuntimeFile(path)
	if err == nil {
		plan.scriptPath, plan.originalScript = path, data
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect private implementation: %w", err)
	}
	if err := local.RemoveFeature(featureName); err != nil {
		return nil, err
	}
	if err := ValidateRuntimeScripts(repoPath, shellName); err != nil {
		return nil, fmt.Errorf("check runtime before local removal: %w", err)
	}
	plan.updatedManifest, plan.Changed = local, true
	return plan, nil
}

// Apply removes the prepared private declaration and implementation, then
// regenerates runtime scripts. A repeated removal is a successful no-op.
func (plan *LocalFeatureRemovalPlan) Apply() error {
	if !plan.Changed {
		return nil
	}
	if err := ensurePrivateRuntimeDirectory(plan.runtimeDirectory, true); err != nil {
		return err
	}
	for path, expected := range map[string][]byte{plan.manifestPath: plan.originalManifest, plan.scriptPath: plan.originalScript} {
		if path == "" {
			continue
		}
		data, err := readRuntimeFile(path)
		if err != nil || !bytes.Equal(data, expected) {
			return fmt.Errorf("local feature changed while selecting; retry removal: %s", path)
		}
	}
	if err := EnsureLocalRuntimeIgnored(plan.repoPath); err != nil {
		return err
	}
	restore := func(cause error) error {
		manifestErr := manifest.WritePrivateFile(plan.manifestPath, plan.originalManifest)
		var scriptErr error
		if plan.scriptPath != "" {
			scriptErr = manifest.WritePrivateFile(plan.scriptPath, plan.originalScript)
		}
		return fmt.Errorf("remove local feature: %w", errors.Join(cause, manifestErr, scriptErr, RegenerateInitScript(plan.repoPath, plan.shellName)))
	}
	if err := manifest.WriteLocalManifest(plan.manifestPath, plan.updatedManifest); err != nil {
		return err
	}
	if plan.scriptPath != "" {
		if err := os.Remove(plan.scriptPath); err != nil {
			return restore(err)
		}
	}
	if err := RegenerateInitScript(plan.repoPath, plan.shellName); err != nil {
		return restore(err)
	}
	return nil
}
