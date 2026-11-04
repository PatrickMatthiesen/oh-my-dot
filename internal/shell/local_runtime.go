package shell

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	gogit "github.com/go-git/go-git/v6"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/catalog"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
)

const localRuntimeIgnore = "# Device-only oh-my-dot runtime state\n*/enabled.local.json\n*/init.local.*\n*/.local/\n*/features/oh-my-posh*.omp.json\n"

// GetLocalInitScriptPath returns the private effective init beside the public entry.
func GetLocalInitScriptPath(repoPath, shellName string) (string, error) {
	config, ok := GetShellConfig(shellName)
	if !ok {
		return "", fmt.Errorf("unsupported shell: %s", shellName)
	}
	return filepath.Join(GetShellDirectory(repoPath, shellName), "init.local"+config.Extension), nil
}

// GetLocalFeatureFilePath returns a private rendered implementation, never a shared script.
func GetLocalFeatureFilePath(repoPath, shellName, featureName string) (string, error) {
	if err := manifest.ValidateFeatureName(featureName); err != nil {
		return "", err
	}
	config, ok := GetShellConfig(shellName)
	if !ok {
		return "", fmt.Errorf("unsupported shell: %s", shellName)
	}
	return filepath.Join(GetShellDirectory(repoPath, shellName), ".local", "features", featureName+config.Extension), nil
}

// IsDeviceLocalShellPath identifies files that must never enter a shared commit.
func IsDeviceLocalShellPath(path string) bool {
	normalized := filepath.ToSlash(filepath.Clean(path))
	parts := strings.Split(normalized, "/")
	if len(parts) < 3 || parts[0] != "omd-shells" {
		return false
	}
	name := parts[len(parts)-1]
	return name == LocalManifestFileName() || strings.HasPrefix(name, "init.local.") || parts[2] == ".local" || len(parts) >= 4 && parts[2] == "features" && strings.HasPrefix(name, "oh-my-posh") && strings.HasSuffix(name, ".omp.json")
}

// CheckLocalRuntimeTracking refuses to hide already-tracked private state behind
// ignore rules. Removing it from Git's index is an explicit recovery operation.
func CheckLocalRuntimeTracking(repoPath string) error {
	repo, err := gogit.PlainOpen(repoPath)
	if errors.Is(err, gogit.ErrRepositoryNotExists) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect runtime tracking: %w", err)
	}
	defer repo.Close()
	index, err := repo.Storer.Index()
	if err != nil {
		return fmt.Errorf("inspect runtime index: %w", err)
	}
	for _, entry := range index.Entries {
		localOnly, err := IsLocalOnlyFeaturePath(repoPath, entry.Name)
		if err != nil {
			return err
		}
		if IsDeviceLocalShellPath(entry.Name) || localOnly {
			return fmt.Errorf("device-local file %q is already tracked; preserve it and remove it from the Git index with git rm --cached before changing shell features", entry.Name)
		}
	}
	return nil
}

// EnsureLocalRuntimeIgnored records static private-runtime exclusions without
// modifying an existing user rule. No private name or option enters this file.
func EnsureLocalRuntimeIgnored(repoPath string) error {
	return ensureLocalRuntimeIgnored(repoPath, false)
}

func ensureLocalRuntimeIgnored(repoPath string, preview bool) error {
	if err := CheckLocalRuntimeTracking(repoPath); err != nil {
		return err
	}
	shellRoot := filepath.Join(repoPath, "omd-shells")
	if info, err := os.Lstat(shellRoot); err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
		return fmt.Errorf("shell runtime root must be a real directory: %s", shellRoot)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	path := filepath.Join(repoPath, "omd-shells", ".gitignore")
	data, err := readRuntimeFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if strings.Contains(string(data), localRuntimeIgnore) {
		return nil
	}
	if preview {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create shell ignore directory: %w", err)
	}
	if len(data) > 0 && data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	data = append(data, localRuntimeIgnore...)
	return writeRuntimeFile(path, data, 0644)
}

func readRuntimeFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("runtime file must be regular, refusing %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read runtime file %s: %w", path, err)
	}
	return data, nil
}

func writeRuntimeFile(path string, data []byte, mode os.FileMode) error {
	if previous, err := readRuntimeFile(path); err == nil {
		if string(previous) == string(data) {
			if mode == 0600 {
				return manifest.WritePrivateFile(path, data)
			}
			return nil
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if mode == 0600 {
		return manifest.WritePrivateFile(path, data)
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		return fmt.Errorf("write runtime file %s: %w", path, err)
	}
	return nil
}

func ensurePrivateRuntimeDirectory(path string, preview bool) error {
	// Each private component is checked so a repository-provided symlink cannot
	// redirect local option values into another directory.
	parent := filepath.Dir(path)
	for _, directory := range []string{filepath.Dir(parent), parent, path} {
		info, err := os.Lstat(directory)
		if err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
			return fmt.Errorf("private runtime directory must not be a symlink or file: %s", directory)
		}
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("inspect private runtime directory: %w", err)
		}
	}
	if preview {
		return nil
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		return fmt.Errorf("create private runtime directory: %w", err)
	}
	return nil
}

func withLocalDispatch(repoPath, shellName, content string) string {
	localPath, _ := GetLocalInitScriptPath(repoPath, shellName)
	var dispatch string
	switch shellName {
	case "bash":
		dispatch = `_omd_local_init="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/init.local.sh"
if [ -r "$_omd_local_init" ]; then
  . "$_omd_local_init"
  unset _omd_local_init
  return 0
fi
unset _omd_local_init

`
	case "zsh":
		dispatch = `typeset _omd_local_init="${${(%):-%x}:A:h}/init.local.zsh"
if [[ -r "$_omd_local_init" ]]; then
  source "$_omd_local_init"
  unset _omd_local_init
  return 0
fi
unset _omd_local_init

`
	case "fish":
		dispatch = `set -l _omd_local_init (dirname (status --current-filename))/init.local.fish
if test -r "$_omd_local_init"
  source "$_omd_local_init"
  return $status
end

`
	case "powershell":
		dispatch = `$omdLocalInit = Join-Path (Split-Path -Parent $PSCommandPath) 'init.local.ps1'
if (Test-Path -LiteralPath $omdLocalInit -PathType Leaf) {
  . $omdLocalInit
  Remove-Variable omdLocalInit -ErrorAction SilentlyContinue
  return
}
Remove-Variable omdLocalInit -ErrorAction SilentlyContinue

`
	case "posix":
		dispatch = fmt.Sprintf("if [ -r %s ]; then\n  . %s\n  return 0 2>/dev/null || exit 0\nfi\n\n", shellSingleQuote(filepath.ToSlash(localPath)), shellSingleQuote(filepath.ToSlash(localPath)))
	}
	if strings.HasPrefix(content, "#!") {
		first, rest, _ := strings.Cut(content, "\n")
		return first + "\n" + dispatch + rest
	}
	return dispatch + content
}

type generatedLocalFeature struct {
	RenderedHash string `json:"renderedHash"`
	OptionsHash  string `json:"optionsHash"`
}

type localGenerationState map[string]generatedLocalFeature

func runtimeDigest(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func strictLocalManifest(path string) (*manifest.FeatureManifest, error) {
	file, err := manifest.OpenValidatedLocalManifest(path)
	if err != nil {
		return nil, fmt.Errorf("load private manifest: %w", err)
	}
	if file == nil {
		return nil, nil
	}
	defer file.Close()
	value, err := manifest.ParseManifestReader(file)
	if err != nil {
		return nil, fmt.Errorf("parse private manifest %s: %w", path, err)
	}
	return value, nil
}

// ValidateRuntimeScripts checks runtime generation and ownership without writing
// files, creating directories, changing ignore rules, or removing stale output.
func ValidateRuntimeScripts(repoPath, shellName string) error {
	return prepareRuntimeScripts(repoPath, shellName, true)
}

func regenerateRuntimeScripts(repoPath, shellName string) error {
	return prepareRuntimeScripts(repoPath, shellName, false)
}

func prepareRuntimeScripts(repoPath, shellName string, preview bool) error {
	public, err := GenerateInitScript(repoPath, shellName)
	if err != nil {
		return err
	}
	base, err := manifest.ParseManifest(GetManifestPath(repoPath, shellName))
	if err != nil {
		return err
	}
	local, err := strictLocalManifest(GetLocalManifestPath(repoPath, shellName))
	if err != nil {
		return err
	}
	localInit, err := GetLocalInitScriptPath(repoPath, shellName)
	if err != nil {
		return err
	}
	if err := ensureLocalRuntimeIgnored(repoPath, preview); err != nil {
		return err
	}
	initPath, err := GetInitScriptPath(repoPath, shellName)
	if err != nil {
		return err
	}
	for _, path := range []string{initPath, localInit} {
		if _, err := readRuntimeFile(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if local == nil || len(local.Features) == 0 {
		statePath := filepath.Join(GetShellDirectory(repoPath, shellName), ".local", "generated.json")
		if _, stateErr := os.Lstat(statePath); stateErr == nil {
			if _, err := renderLocalFeatureVariants(repoPath, shellName, base, manifest.MergeManifests(base, nil), "", preview); err != nil {
				return err
			}
		} else if !os.IsNotExist(stateErr) {
			return stateErr
		}
		if _, err := readRuntimeFile(localInit); err == nil {
			if !preview {
				if err := os.Remove(localInit); err != nil {
					return fmt.Errorf("remove obsolete private init: %w", err)
				}
			}
		} else if !os.IsNotExist(err) {
			return err
		}
	} else {
		merged := manifest.MergeManifests(base, local)
		effective, err := generateManifestInit(repoPath, shellName, merged)
		if err != nil {
			return err
		}
		effective, err = renderLocalFeatureVariants(repoPath, shellName, base, merged, effective, preview)
		if err != nil {
			return err
		}
		if !preview {
			if err := writeRuntimeFile(localInit, []byte(effective), 0600); err != nil {
				return err
			}
		}
	}
	if preview {
		return nil
	}
	return writeRuntimeFile(initPath, []byte(public), 0644)
}

func renderLocalFeatureVariants(repoPath, shellName string, base *manifest.FeatureManifest, merged *manifest.MergedManifest, content string, preview bool) (string, error) {
	directory := filepath.Join(GetShellDirectory(repoPath, shellName), ".local", "features")
	if err := ensurePrivateRuntimeDirectory(directory, preview); err != nil {
		return "", err
	}
	statePath := filepath.Join(filepath.Dir(directory), "generated.json")
	state := localGenerationState{}
	if data, err := readRuntimeFile(statePath); err == nil {
		if err := json.Unmarshal(data, &state); err != nil {
			return "", fmt.Errorf("parse local generation state: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	nextState := localGenerationState{}
	pendingWrites := map[string][]byte{}
	var pendingRemovals []string
	config, _ := GetShellConfig(shellName)
	for _, effective := range merged.Features {
		shared, findErr := base.GetFeature(effective.Name)
		private := effective.Override.IsFromLocal || findErr == nil && !sameFeatureOptions(shared.Options, effective.Options)
		if !private {
			continue
		}
		path, err := GetLocalFeatureFilePath(repoPath, shellName, effective.Name)
		if err != nil {
			return "", err
		}
		// Existing local-only scripts are user-owned. Keep their old location
		// when there is no rendered option variant rather than silently moving
		// or replacing them. The staging guard recognizes this local ownership.
		if effective.Override.IsFromLocal && len(effective.Options) == 0 {
			if _, err := os.Lstat(path); os.IsNotExist(err) {
				legacy, _ := GetFeatureFilePath(repoPath, shellName, effective.Name)
				if _, legacyErr := readRuntimeFile(legacy); legacyErr == nil {
					continue
				} else if !os.IsNotExist(legacyErr) {
					return "", legacyErr
				}
			}
		}
		if catalog.HasFeatureTemplate(effective.Name, shellName) {
			template, err := catalog.GetFeatureTemplate(effective.Name, shellName)
			if err != nil {
				return "", err
			}
			desired, err := catalog.RenderFeatureTemplate(template, effective.Name, shellName, effective.Options)
			if err != nil {
				return "", err
			}
			optionsJSON, err := json.Marshal(effective.Options)
			if err != nil {
				return "", fmt.Errorf("encode local options: %w", err)
			}
			current, readErr := readRuntimeFile(path)
			expected := generatedLocalFeature{RenderedHash: runtimeDigest([]byte(desired)), OptionsHash: runtimeDigest(optionsJSON)}
			previous, recorded := state[effective.Name]
			if readErr == nil && string(current) != desired {
				if !recorded || runtimeDigest(current) != previous.RenderedHash {
					if recorded && previous.OptionsHash == expected.OptionsHash {
						nextState[effective.Name] = previous
					} else {
						return "", fmt.Errorf("private feature %s has customized content; configure with --force after backing it up", path)
					}
				} else {
					readErr = os.ErrNotExist
				}
			}
			if readErr != nil && !os.IsNotExist(readErr) {
				return "", readErr
			}
			if _, preserved := nextState[effective.Name]; !preserved {
				pendingWrites[path] = []byte(desired)
				nextState[effective.Name] = expected
			}
		} else {
			if _, err := readRuntimeFile(path); err != nil {
				return "", fmt.Errorf("local custom feature requires its private implementation at %s: %w", path, err)
			}
		}
		old := "features/" + effective.Name + config.Extension
		replacement := ".local/features/" + effective.Name + config.Extension
		if shellName == "powershell" {
			old = strings.ReplaceAll(old, "/", `\`)
			replacement = strings.ReplaceAll(replacement, "/", `\`)
		}
		content = strings.ReplaceAll(content, old, replacement)
	}
	for name, previous := range state {
		if _, used := nextState[name]; used {
			continue
		}
		path, err := GetLocalFeatureFilePath(repoPath, shellName, name)
		if err != nil {
			return "", err
		}
		current, readErr := readRuntimeFile(path)
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			return "", readErr
		}
		if runtimeDigest(current) == previous.RenderedHash {
			pendingRemovals = append(pendingRemovals, path)
		}
	}
	if preview {
		return content, nil
	}
	paths := make([]string, 0, len(pendingWrites))
	for path := range pendingWrites {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if err := writeRuntimeFile(path, pendingWrites[path], 0600); err != nil {
			return "", err
		}
	}
	for _, path := range pendingRemovals {
		if err := os.Remove(path); err != nil {
			return "", fmt.Errorf("remove obsolete generated private feature: %w", err)
		}
	}
	data, err := json.MarshalIndent(nextState, "", "  ")
	if err != nil {
		return "", err
	}
	if err := writeRuntimeFile(statePath, append(data, '\n'), 0600); err != nil {
		return "", err
	}
	return content, nil
}

// IsLocalOnlyFeaturePath identifies legacy user scripts owned only by a local
// manifest. Their location is retained for compatibility, but automatic shared
// commits must exclude them just like implementations under .local/features.
func IsLocalOnlyFeaturePath(repoPath, path string) (bool, error) {
	parts := strings.Split(filepath.ToSlash(filepath.Clean(path)), "/")
	if len(parts) != 4 || parts[0] != "omd-shells" || parts[2] != "features" {
		return false, nil
	}
	shellName := parts[1]
	config, ok := GetShellConfig(shellName)
	if !ok {
		return false, nil
	}
	if !strings.HasSuffix(parts[3], config.Extension) {
		return false, nil
	}
	name := strings.TrimSuffix(parts[3], config.Extension)
	local, err := strictLocalManifest(GetLocalManifestPath(repoPath, shellName))
	if err != nil {
		return false, err
	}
	if local == nil {
		return false, nil
	}
	if _, err := local.GetFeature(name); err != nil {
		return false, nil
	}
	base, err := manifest.ParseManifest(GetManifestPath(repoPath, shellName))
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("read shared ownership: %w", err)
	}
	_, err = base.GetFeature(name)
	return err != nil, nil
}

// GetEffectiveFeatureFilePath returns the implementation used by the private
// effective init when local options or local-only features change its ownership.
// It only reads metadata and never generates or repairs files.
func GetEffectiveFeatureFilePath(repoPath, shellName, featureName string) (string, error) {
	sharedPath, err := GetFeatureFilePath(repoPath, shellName, featureName)
	if err != nil {
		return "", err
	}
	base, err := manifest.ParseManifest(GetManifestPath(repoPath, shellName))
	if err != nil {
		return "", err
	}
	local, err := strictLocalManifest(GetLocalManifestPath(repoPath, shellName))
	if err != nil {
		return "", err
	}
	for _, effective := range manifest.MergeManifests(base, local).Features {
		if effective.Name != featureName {
			continue
		}
		shared, findErr := base.GetFeature(featureName)
		if !effective.Override.IsFromLocal && findErr == nil && sameFeatureOptions(shared.Options, effective.Options) {
			return sharedPath, nil
		}
		privatePath, err := GetLocalFeatureFilePath(repoPath, shellName, featureName)
		if err != nil {
			return "", err
		}
		if effective.Override.IsFromLocal && len(effective.Options) == 0 {
			if _, err := os.Lstat(privatePath); os.IsNotExist(err) {
				if _, legacyErr := readRuntimeFile(sharedPath); legacyErr == nil {
					return sharedPath, nil
				} else if !os.IsNotExist(legacyErr) {
					return "", legacyErr
				}
			}
		}
		return privatePath, nil
	}
	return "", fmt.Errorf("feature %q is not configured in %s", featureName, shellName)
}

func sameFeatureOptions(first, second map[string]any) bool {
	return len(first) == 0 && len(second) == 0 || reflect.DeepEqual(first, second)
}
