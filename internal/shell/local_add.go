package shell

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
)

// IsUnmodifiedShellScaffold recognizes only canonical public files created by
// local-first setup. It lets a later shared command adopt this empty scaffolding
// without accepting user scripts or edits as part of its automatic commit.
func IsUnmodifiedShellScaffold(repoPath, relative string) bool {
	relative = filepath.ToSlash(relative)
	path := filepath.Join(repoPath, filepath.FromSlash(relative))
	data, err := readRuntimeFile(path)
	if err != nil {
		return false
	}
	switch relative {
	case "omd-shells/.gitignore":
		return string(data) == localRuntimeIgnore
	case "omd-shells/lib/helpers.sh":
		return string(data) == HelpersFileContent
	}
	parts := strings.Split(relative, "/")
	if len(parts) != 3 || parts[0] != "omd-shells" || !IsShellSupported(parts[1]) {
		return false
	}
	base, err := manifest.ParseManifest(GetManifestPath(repoPath, parts[1]))
	if err != nil || len(base.Features) != 0 {
		return false
	}
	if parts[2] == "enabled.json" {
		return string(data) == "{\n  \"features\": []\n}\n"
	}
	initPath, err := GetInitScriptPath(repoPath, parts[1])
	if err != nil || path != initPath {
		return false
	}
	expected, err := GenerateInitScript(repoPath, parts[1])
	return err == nil && string(data) == expected
}

// AddLocalFeatureToShell installs a private feature without changing a shared
// implementation or creating a commit. New shells get only ordinary public
// scaffolding; the declaration and implementation stay in ignored private files.
func AddLocalFeatureToShell(repoPath, shellName, featureName, strategy string, triggers []string, disabled bool, values map[string]any) error {
	if !IsShellSupported(shellName) {
		return fmt.Errorf("unsupported shell: %s", shellName)
	}
	metadata, inCatalog := catalog.GetFeature(featureName)
	if strategy == "" {
		strategy = "eager"
		if inCatalog {
			strategy = metadata.DefaultStrategy
		}
	}
	if len(triggers) == 0 && inCatalog && strategy == "on-command" {
		triggers = metadata.DefaultCommands
	}
	config := manifest.FeatureConfig{Name: featureName, Strategy: strategy, OnCommand: triggers, Disabled: disabled, DisabledSet: true, Options: values}
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid local feature: %w", err)
	}
	shellDir := GetShellDirectory(repoPath, shellName)
	if info, err := os.Stat(shellDir); err == nil && !info.IsDir() {
		return fmt.Errorf("shell path is not a directory: %s", shellDir)
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("inspect shell directory: %w", err)
	}
	privatePath, err := GetLocalFeatureFilePath(repoPath, shellName, featureName)
	if err != nil {
		return err
	}
	if err := ensurePrivateRuntimeDirectory(filepath.Dir(privatePath), true); err != nil {
		return err
	}
	if _, err := os.Lstat(privatePath); err == nil {
		return fmt.Errorf("private implementation already exists at %s; use feature configure --local for an installed feature", privatePath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect private implementation: %w", err)
	}
	if err := EnsureLocalRuntimeIgnored(repoPath); err != nil {
		return err
	}
	if !ShellDirectoryExists(repoPath, shellName) {
		if err := InitializeShellDirectory(repoPath, shellName); err != nil {
			return err
		}
	}
	base, err := manifest.ParseManifest(GetManifestPath(repoPath, shellName))
	if err != nil {
		return err
	}
	if _, err := base.GetFeature(featureName); err == nil {
		return fmt.Errorf("feature %q is already shared; use feature configure --local to add a private override", featureName)
	}
	localPath := GetLocalManifestPath(repoPath, shellName)
	local := &manifest.FeatureManifest{Features: []manifest.FeatureConfig{}}
	localFile, err := manifest.OpenValidatedLocalManifest(localPath)
	if err != nil {
		return err
	}
	var original []byte
	if localFile != nil {
		original, err = io.ReadAll(localFile)
		if err != nil {
			localFile.Close()
			return err
		}
		local, err = manifest.ParseManifestReader(bytes.NewReader(original))
		localFile.Close()
		if err != nil {
			return err
		}
	}
	if err := local.AddFeature(config); err != nil {
		return err
	}
	content := generateFeatureTemplate(shellName, featureName, metadata, values)
	if inCatalog && catalog.HasFeatureTemplate(featureName, shellName) {
		template, err := catalog.GetFeatureTemplate(featureName, shellName)
		if err != nil {
			return err
		}
		content, err = catalog.RenderFeatureTemplate(template, featureName, shellName, values)
		if err != nil {
			return err
		}
	}
	if err := ensurePrivateRuntimeDirectory(filepath.Dir(privatePath), false); err != nil {
		return err
	}
	file, err := os.OpenFile(privatePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("create private feature: %w", err)
	}
	if err := file.Close(); err != nil {
		os.Remove(privatePath)
		return err
	}
	// Reserve the new path exclusively, then establish its private permissions
	// before writing any options into the implementation, including on Windows.
	if err := manifest.WritePrivateFile(privatePath, []byte(content)); err != nil {
		os.Remove(privatePath)
		return err
	}
	restore := func(cause error) error {
		manifestErr := restoreFeatureSnapshot(localPath, original, 0600)
		fileErr := os.Remove(privatePath)
		return fmt.Errorf("add private feature: %w", errors.Join(cause, manifestErr, fileErr))
	}
	if err := manifest.WriteLocalManifest(localPath, local); err != nil {
		return restore(err)
	}
	if err := RegenerateInitScript(repoPath, shellName); err != nil {
		return restore(err)
	}
	return nil
}
