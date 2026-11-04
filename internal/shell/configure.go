package shell

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
)

// FeatureConfigurationPlan is a validated option change for an installed catalog
// feature. Preparing every target before applying any catches customized scripts
// and malformed manifests before a multi-shell command changes files.
type FeatureConfigurationPlan struct {
	Changed          bool // Whether saved options or the generated implementation change.
	manifestPath     string
	featurePath      string
	repoPath         string
	shellName        string
	originalManifest []byte
	originalScript   []byte
	updatedManifest  *manifest.FeatureManifest
	updatedScript    []byte
	scriptMode       os.FileMode
	local            bool
	sourcePath       string
	originalSource   []byte
	removeScript     bool
	refreshOnly      bool
}

// PrepareFeatureConfiguration validates an option change without modifying files.
// A customized feature implementation requires explicit force to replace it.
func PrepareFeatureConfiguration(repoPath, shellName, featureName string, values map[string]any, force bool) (*FeatureConfigurationPlan, error) {
	return prepareFeatureConfiguration(repoPath, shellName, featureName, values, force, false)
}

// PrepareFeatureRefresh validates a shared catalog refresh, preserving saved
// options and requiring force before replacing customized or older templates.
func PrepareFeatureRefresh(repoPath, shellName, featureName string, force bool) (*FeatureConfigurationPlan, error) {
	return prepareFeatureConfiguration(repoPath, shellName, featureName, nil, force, true)
}

func prepareFeatureConfiguration(repoPath, shellName, featureName string, values map[string]any, force, refresh bool) (*FeatureConfigurationPlan, error) {
	if err := CheckLocalRuntimeTracking(repoPath); err != nil {
		return nil, err
	}
	if !IsShellSupported(shellName) {
		return nil, fmt.Errorf("unsupported shell: %s", shellName)
	}
	if !catalog.HasFeatureTemplate(featureName, shellName) {
		return nil, fmt.Errorf("feature %q has no catalog template for %s", featureName, shellName)
	}
	path := GetManifestPath(repoPath, shellName)
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("inspect manifest: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("manifest must be a regular file: %s", path)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	base, err := manifest.ParseManifestReader(bytes.NewReader(original))
	if err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	feature, err := base.GetFeature(featureName)
	if err != nil {
		return nil, fmt.Errorf("shared feature %q is not installed in %s; use feature configure --local for local-only settings", featureName, shellName)
	}
	if refresh {
		values = feature.Options
		if err := ValidateRuntimeScripts(repoPath, shellName); err != nil {
			return nil, fmt.Errorf("check runtime before refreshing: %w", err)
		}
	}
	featurePath, err := GetFeatureFilePath(repoPath, shellName, featureName)
	if err != nil {
		return nil, err
	}
	scriptInfo, err := os.Lstat(featurePath)
	if err != nil {
		return nil, fmt.Errorf("inspect feature implementation (run doctor for recovery): %w", err)
	}
	if !scriptInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("feature implementation must be a regular file: %s", featurePath)
	}
	script, err := os.ReadFile(featurePath)
	if err != nil {
		return nil, fmt.Errorf("read feature implementation: %w", err)
	}
	plan := &FeatureConfigurationPlan{manifestPath: path, featurePath: featurePath, repoPath: repoPath, shellName: shellName, originalManifest: original, originalScript: script, updatedManifest: base, scriptMode: scriptInfo.Mode().Perm(), refreshOnly: refresh}
	if !refresh && reflect.DeepEqual(feature.Options, values) {
		return plan, nil
	}
	template, err := catalog.GetFeatureTemplate(featureName, shellName)
	if err != nil {
		return nil, fmt.Errorf("load feature template: %w", err)
	}
	expected, err := catalog.RenderFeatureTemplate(template, featureName, shellName, feature.Options)
	if err != nil {
		return nil, fmt.Errorf("render current options: %w", err)
	}
	if !matchesGeneratedFeature(script, expected, feature.Options) && !force {
		return nil, fmt.Errorf("%s has customized content or an older template; configuration was not changed. Back up the file and use --force to replace it with the configured catalog template", featurePath)
	}
	rendered, err := catalog.RenderFeatureTemplate(template, featureName, shellName, values)
	if err != nil {
		return nil, fmt.Errorf("render updated options: %w", err)
	}
	if err := base.UpdateFeature(featureName, func(config *manifest.FeatureConfig) { config.Options = values }); err != nil {
		return nil, fmt.Errorf("update feature configuration: %w", err)
	}
	plan.updatedScript = []byte(rendered)
	plan.Changed = !refresh || !bytes.Equal(script, plan.updatedScript)
	return plan, nil
}

// Apply writes a prepared feature change. It refuses stale source state and
// restores the saved manifest/implementation if regeneration cannot complete.
func (plan *FeatureConfigurationPlan) Apply() error {
	if !plan.Changed {
		return nil
	}
	if plan.local {
		if err := ensurePrivateRuntimeDirectory(filepath.Dir(plan.featurePath), true); err != nil {
			return err
		}
	}
	snapshots := map[string][]byte{plan.manifestPath: plan.originalManifest, plan.featurePath: plan.originalScript}
	if plan.sourcePath != "" {
		snapshots[plan.sourcePath] = plan.originalSource
	}
	for path, expected := range snapshots {
		privateRefreshTarget := plan.local && plan.refreshOnly && path == plan.featurePath
		info, err := os.Lstat(path)
		if plan.local && expected == nil {
			if os.IsNotExist(err) {
				continue
			}
			if err == nil && !privateRefreshTarget {
				return fmt.Errorf("private configuration appeared while choosing options; retry configure: %s", path)
			}
		}
		if err != nil {
			return fmt.Errorf("recheck configuration: %w", err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("configuration target changed; retry configure: %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("recheck configuration: %w", err)
		}
		if !bytes.Equal(data, expected) {
			// Regenerating an earlier feature in this batch can already render
			// this private template. Accept only its exact prepared result;
			// settings and unrelated edits still require their original snapshots.
			if privateRefreshTarget && bytes.Equal(data, plan.updatedScript) {
				continue
			}
			return fmt.Errorf("configuration changed while selecting options; retry configure: %s", path)
		}
	}
	if plan.local {
		if err := EnsureLocalRuntimeIgnored(plan.repoPath); err != nil {
			return err
		}
		if err := ensurePrivateRuntimeDirectory(filepath.Dir(plan.featurePath), false); err != nil {
			return fmt.Errorf("create private feature directory: %w", err)
		}
	}
	restore := func(cause error) error {
		mode := os.FileMode(0644)
		if plan.local {
			mode = 0600
		}
		first := restoreFeatureSnapshot(plan.manifestPath, plan.originalManifest, mode)
		second := restoreFeatureSnapshot(plan.featurePath, plan.originalScript, plan.scriptMode)
		third := RegenerateInitScript(plan.repoPath, plan.shellName)
		return fmt.Errorf("configure feature: %w", errors.Join(cause, first, second, third))
	}
	writer := manifest.WriteManifest
	if plan.local {
		writer = manifest.WriteLocalManifest
	}
	if !plan.refreshOnly {
		if err := writer(plan.manifestPath, plan.updatedManifest); err != nil {
			return fmt.Errorf("save feature options: %w", err)
		}
	}
	writeScript := func() error { return os.WriteFile(plan.featurePath, plan.updatedScript, plan.scriptMode) }
	if plan.removeScript {
		writeScript = func() error {
			if plan.originalScript == nil {
				return nil
			}
			return os.Remove(plan.featurePath)
		}
	} else if plan.local {
		writeScript = func() error { return manifest.WritePrivateFile(plan.featurePath, plan.updatedScript) }
	}
	if err := writeScript(); err != nil {
		return restore(err)
	}
	if err := RegenerateInitScript(plan.repoPath, plan.shellName); err != nil {
		return restore(err)
	}
	return nil
}

// FeatureLoadPlan is a validated enabled/disabled or loading-strategy change.
// Local changes write only the local manifest; regeneration owns private output.
type FeatureLoadPlan struct {
	Changed   bool
	repoPath  string
	shellName string
	path      string
	local     bool
	original  []byte
	updated   *manifest.FeatureManifest
}

// PrepareFeatureLoadConfiguration resolves the effective feature and prepares a
// shared or device-local load setting without changing a feature implementation.
func PrepareFeatureLoadConfiguration(repoPath, shellName, featureName string, disabled bool, strategy string, triggers []string, local bool) (*FeatureLoadPlan, error) {
	if err := CheckLocalRuntimeTracking(repoPath); err != nil {
		return nil, err
	}
	if !IsShellSupported(shellName) {
		return nil, fmt.Errorf("unsupported shell: %s", shellName)
	}
	sharedSnapshot, err := readRuntimeFile(GetManifestPath(repoPath, shellName))
	if err != nil {
		return nil, fmt.Errorf("read shared manifest: %w", err)
	}
	base, err := manifest.ParseManifestReader(bytes.NewReader(sharedSnapshot))
	if err != nil {
		return nil, fmt.Errorf("read shared manifest: %w", err)
	}
	localManifest := &manifest.FeatureManifest{Features: []manifest.FeatureConfig{}}
	var localSnapshot []byte
	localFile, err := manifest.OpenValidatedLocalManifest(GetLocalManifestPath(repoPath, shellName))
	if err != nil {
		return nil, fmt.Errorf("read local manifest: %w", err)
	}
	if localFile != nil {
		defer localFile.Close()
		localSnapshot, err = io.ReadAll(localFile)
		if err != nil {
			return nil, fmt.Errorf("snapshot local manifest: %w", err)
		}
		localManifest, err = manifest.ParseManifestReader(bytes.NewReader(localSnapshot))
		if err != nil {
			return nil, fmt.Errorf("parse local manifest: %w", err)
		}
	}
	var effective *manifest.FeatureConfig
	for _, feature := range manifest.MergeManifests(base, localManifest).Features {
		if feature.Name == featureName {
			copy := feature.FeatureConfig
			effective = &copy
			break
		}
	}
	if effective == nil {
		return nil, fmt.Errorf("feature %q is not installed in %s", featureName, shellName)
	}
	chosen := base
	path := GetManifestPath(repoPath, shellName)
	if local {
		chosen = localManifest
		path = GetLocalManifestPath(repoPath, shellName)
	}
	current, findErr := chosen.GetFeature(featureName)
	if !local && findErr != nil {
		return nil, fmt.Errorf("feature %q is local-only in %s; use --local", featureName, shellName)
	}
	next := *effective
	if findErr == nil {
		next = *current
	} else if local {
		next = manifest.FeatureConfig{Name: featureName}
	}
	next.Disabled = disabled
	if local {
		next.DisabledSet = true
	}
	if strategy != "" {
		next.Strategy = strategy
	}
	if triggers != nil {
		next.OnCommand = append([]string(nil), triggers...)
	}
	if err := next.Validate(); err != nil {
		return nil, fmt.Errorf("invalid load configuration: %w", err)
	}
	plan := &FeatureLoadPlan{repoPath: repoPath, shellName: shellName, path: path, local: local, updated: chosen}
	if findErr == nil && reflect.DeepEqual(*current, next) {
		return plan, nil
	}
	plan.original = sharedSnapshot
	if local {
		plan.original = localSnapshot
	}
	if findErr == nil {
		err = chosen.UpdateFeature(featureName, func(feature *manifest.FeatureConfig) { *feature = next })
	} else {
		next.Options = nil
		err = chosen.AddFeature(next)
	}
	if err != nil {
		return nil, fmt.Errorf("prepare load settings: %w", err)
	}
	plan.Changed = true
	return plan, nil
}

// Apply saves the prepared load setting and regenerates the corresponding init.
func (plan *FeatureLoadPlan) Apply() error {
	if !plan.Changed {
		return nil
	}
	data, err := os.ReadFile(plan.path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("recheck manifest: %w", err)
	}
	if !bytes.Equal(data, plan.original) {
		return fmt.Errorf("manifest changed while choosing features; retry")
	}
	if info, err := os.Lstat(plan.path); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("manifest must remain a regular file: %s", plan.path)
	}
	if plan.local {
		if err := EnsureLocalRuntimeIgnored(plan.repoPath); err != nil {
			return err
		}
	}
	writer := manifest.WriteManifest
	if plan.local {
		writer = manifest.WriteLocalManifest
	}
	if err := writer(plan.path, plan.updated); err != nil {
		return fmt.Errorf("save load settings: %w", err)
	}
	if err := RegenerateInitScript(plan.repoPath, plan.shellName); err != nil {
		var restoreErr error
		if plan.original == nil {
			restoreErr = os.Remove(plan.path)
		} else {
			mode := os.FileMode(0644)
			if plan.local {
				mode = 0600
			}
			restoreErr = restoreFeatureSnapshot(plan.path, plan.original, mode)
		}
		return fmt.Errorf("regenerate load settings: %w", errors.Join(err, restoreErr))
	}
	return nil
}

func restoreFeatureSnapshot(path string, data []byte, mode os.FileMode) error {
	if data == nil {
		err := os.Remove(path)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if mode == 0600 {
		return manifest.WritePrivateFile(path, data)
	}
	return os.WriteFile(path, data, mode)
}

func matchesGeneratedFeature(actual []byte, expected string, values map[string]any) bool {
	normalize := func(value string) string { return strings.ReplaceAll(value, "\r\n", "\n") }
	if normalize(string(actual)) == normalize(expected) {
		return true
	}
	// Recognize the prior catalog cache name without mistaking an otherwise
	// unmodified implementation for user customization during a theme change.
	theme := "jandedobbeleer"
	if value, ok := values["theme"].(string); ok && value != "" {
		theme = value
	}
	current := "$OMD_SHELL_ROOT/.local/cache/oh-my-posh." + theme + ".omp.json"
	legacy := strings.ReplaceAll(expected, current, "$OMD_SHELL_ROOT/features/oh-my-posh.omp.json")
	legacy = strings.ReplaceAll(legacy, "  mkdir -p \"$OMD_SHELL_ROOT/.local/cache\" 2>/dev/null || :\n", "")
	legacy = strings.ReplaceAll(legacy, "    mkdir -p \"$OMD_SHELL_ROOT/.local/cache\" 2>/dev/null; or true\n", "")
	legacy = strings.ReplaceAll(legacy, "    New-Item -ItemType Directory -Path \"$OMD_SHELL_ROOT/.local/cache\" -Force -ErrorAction SilentlyContinue | Out-Null\n", "")
	return normalize(string(actual)) == normalize(legacy)
}

// PrepareLocalFeatureConfiguration changes a device-local option overlay and a
// private generated implementation. Shared files are never used as output.
func PrepareLocalFeatureConfiguration(repoPath, shellName, featureName string, values map[string]any, force bool) (*FeatureConfigurationPlan, error) {
	return prepareLocalFeatureConfiguration(repoPath, shellName, featureName, values, force, false)
}

// PrepareLocalFeatureRefresh refreshes an existing private catalog template with
// its effective options. Customized private scripts require explicit force.
func PrepareLocalFeatureRefresh(repoPath, shellName, featureName string, force bool) (*FeatureConfigurationPlan, error) {
	return prepareLocalFeatureConfiguration(repoPath, shellName, featureName, nil, force, true)
}

func prepareLocalFeatureConfiguration(repoPath, shellName, featureName string, values map[string]any, force, refresh bool) (*FeatureConfigurationPlan, error) {
	if err := CheckLocalRuntimeTracking(repoPath); err != nil {
		return nil, err
	}
	if !IsShellSupported(shellName) {
		return nil, fmt.Errorf("unsupported shell: %s", shellName)
	}
	if !catalog.HasFeatureTemplate(featureName, shellName) {
		return nil, fmt.Errorf("feature %q has no catalog template for %s", featureName, shellName)
	}
	base, err := manifest.ParseManifest(GetManifestPath(repoPath, shellName))
	if err != nil {
		return nil, fmt.Errorf("read shared manifest: %w", err)
	}
	local := &manifest.FeatureManifest{Features: []manifest.FeatureConfig{}}
	path := GetLocalManifestPath(repoPath, shellName)
	localFile, err := manifest.OpenValidatedLocalManifest(path)
	if err != nil {
		return nil, fmt.Errorf("read local manifest: %w", err)
	}
	var original []byte
	if localFile != nil {
		original, err = io.ReadAll(localFile)
		if err != nil {
			localFile.Close()
			return nil, fmt.Errorf("snapshot local manifest: %w", err)
		}
		local, err = manifest.ParseManifestReader(bytes.NewReader(original))
		localFile.Close()
		if err != nil {
			return nil, fmt.Errorf("parse local manifest: %w", err)
		}
	}
	var effective *manifest.FeatureConfig
	for _, feature := range manifest.MergeManifests(base, local).Features {
		if feature.Name == featureName {
			copy := feature.FeatureConfig
			effective = &copy
			break
		}
	}
	if effective == nil {
		return nil, fmt.Errorf("feature %q is not installed in %s", featureName, shellName)
	}
	if refresh {
		values = effective.Options
	}
	scriptPath, err := GetLocalFeatureFilePath(repoPath, shellName, featureName)
	if err != nil {
		return nil, err
	}
	if err := ensurePrivateRuntimeDirectory(filepath.Dir(scriptPath), true); err != nil {
		return nil, err
	}
	plan := &FeatureConfigurationPlan{repoPath: repoPath, shellName: shellName, manifestPath: path, featurePath: scriptPath, local: true, originalManifest: original, updatedManifest: local, scriptMode: 0600, refreshOnly: refresh}
	if !refresh && reflect.DeepEqual(effective.Options, values) {
		return plan, nil
	}
	shared, _ := base.GetFeature(featureName)
	if refresh && shared != nil && sameFeatureOptions(shared.Options, effective.Options) {
		return plan, nil
	}
	delta := make(map[string]any)
	for key, value := range values {
		if shared == nil || !reflect.DeepEqual(shared.Options[key], value) {
			delta[key] = value
		}
	}
	if shared != nil {
		for key := range shared.Options {
			if _, set := values[key]; !set {
				delta[key] = nil
			}
		}
	}
	current, findErr := local.GetFeature(featureName)
	next := manifest.FeatureConfig{Name: featureName}
	if findErr == nil {
		next = *current
	}
	if !refresh {
		next.Options = delta
	}
	if findErr == nil {
		err = local.UpdateFeature(featureName, func(feature *manifest.FeatureConfig) { *feature = next })
	} else {
		err = local.AddFeature(next)
	}
	if err != nil {
		return nil, fmt.Errorf("prepare local options: %w", err)
	}
	template, err := catalog.GetFeatureTemplate(featureName, shellName)
	if err != nil {
		return nil, err
	}
	oldOptions := effective.Options
	data, readErr := readRuntimeFile(scriptPath)
	if readErr == nil {
		info, err := os.Lstat(scriptPath)
		if err != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("private implementation must be a regular file: %s", scriptPath)
		}
		plan.originalScript = data
	} else if os.IsNotExist(readErr) {
		if shared != nil || len(effective.Options) == 0 {
			source, err := GetEffectiveFeatureFilePath(repoPath, shellName, featureName)
			if err != nil {
				return nil, err
			}
			info, err := os.Lstat(source)
			if err != nil || !info.Mode().IsRegular() {
				return nil, fmt.Errorf("shared implementation must be a regular file: %s", source)
			}
			data, err = os.ReadFile(source)
			if err != nil {
				return nil, fmt.Errorf("read shared implementation: %w", err)
			}
			plan.sourcePath = source
			plan.originalSource = data
			if shared != nil {
				oldOptions = shared.Options
			}
		}
	} else {
		return nil, fmt.Errorf("read private implementation: %w", readErr)
	}
	if data != nil {
		expected, err := catalog.RenderFeatureTemplate(template, featureName, shellName, oldOptions)
		if err != nil {
			return nil, err
		}
		knownGenerated := false
		if plan.sourcePath == "" {
			stateData, stateErr := readRuntimeFile(filepath.Join(GetShellDirectory(repoPath, shellName), ".local", "generated.json"))
			if stateErr == nil {
				var state localGenerationState
				if err := json.Unmarshal(stateData, &state); err != nil {
					return nil, fmt.Errorf("parse private template ownership: %w", err)
				}
				previous, ok := state[featureName]
				knownGenerated = ok && previous.RenderedHash == runtimeDigest(data)
			} else if !os.IsNotExist(stateErr) {
				return nil, stateErr
			}
		}
		if !matchesGeneratedFeature(data, expected, oldOptions) && !knownGenerated && !force {
			return nil, fmt.Errorf("feature implementation is customized; back it up and use --force to generate the private configured catalog version")
		}
	}
	rendered, err := catalog.RenderFeatureTemplate(template, featureName, shellName, values)
	if err != nil {
		return nil, err
	}
	plan.updatedScript = []byte(rendered)
	plan.removeScript = shared != nil && sameFeatureOptions(shared.Options, values)
	if refresh && bytes.Equal(plan.originalScript, plan.updatedScript) {
		return plan, nil
	}
	plan.Changed = true
	return plan, nil
}
