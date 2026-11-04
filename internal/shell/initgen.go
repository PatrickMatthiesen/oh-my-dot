package shell

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
)

// FeaturesByStrategy organizes features by their loading strategy
type FeaturesByStrategy struct {
	Eager     []string
	Defer     []string
	OnCommand map[string][]string // feature name -> trigger commands
}

// GenerateInitScript generates the shared entry from the base manifest only.
// It delegates to an ignored effective init when device-local overrides exist.
func GenerateInitScript(repoPath, shellName string) (string, error) {
	base, err := manifest.ParseManifest(GetManifestPath(repoPath, shellName))
	if err != nil {
		return "", fmt.Errorf("parse shared manifest: %w", err)
	}
	content, err := generateManifestInit(repoPath, shellName, manifest.MergeManifests(base, nil))
	if err != nil {
		return "", err
	}
	return withLocalDispatch(repoPath, shellName, content), nil
}

func generateManifestInit(repoPath, shellName string, merged *manifest.MergedManifest) (string, error) {
	features := categorizeFeaturesMerged(merged)
	switch shellName {
	case "bash":
		return generateBashInit(features), nil
	case "zsh":
		return generateZshInit(features), nil
	case "fish":
		return generateFishInit(features), nil
	case "powershell":
		return generatePowerShellInit(features), nil
	case "posix":
		return generatePosixInit(repoPath, features), nil
	default:
		return "", fmt.Errorf("unsupported shell: %s", shellName)
	}
}

// categorizeFeaturesMerged organizes enabled features from a merged manifest
func categorizeFeaturesMerged(m *manifest.MergedManifest) FeaturesByStrategy {
	features := FeaturesByStrategy{
		OnCommand: make(map[string][]string),
	}

	for _, f := range m.GetEnabledFeatures() {
		strategy := f.Strategy
		if strategy == "" {
			strategy = "eager" // Default to eager
		}

		switch strategy {
		case "eager":
			features.Eager = append(features.Eager, f.Name)
		case "defer":
			features.Defer = append(features.Defer, f.Name)
		case "on-command":
			if len(f.OnCommand) > 0 {
				features.OnCommand[f.Name] = f.OnCommand
			}
		}
	}

	return features
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

// generateBashInit generates a bash init script with all loading strategies
func generateBashInit(features FeaturesByStrategy) string {
	var sb strings.Builder

	// Header
	sb.WriteString(`#!/usr/bin/env bash
# oh-my-dot shell framework - bash init script
# Auto-generated - do not edit manually

# Guard against double-loading functions (but allow eager re-loading)
if [ "${OMD_BASH_LOADED:-}" = "1" ]; then
  # If already loaded, just re-apply eager features that modify environment
  # This handles the case where .bashrc is re-sourced and resets PS1
  if type -t _omd_load_eager_features >/dev/null 2>&1; then
    _omd_load_eager_features
  fi
  return 0
fi
OMD_BASH_LOADED=1

# Determine shell root
OMD_SHELL_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source helper library
if [ -r "$OMD_SHELL_ROOT/../lib/helpers.sh" ]; then
  . "$OMD_SHELL_ROOT/../lib/helpers.sh"
fi

`)

	// Eager loading
	if len(features.Eager) > 0 {
		sb.WriteString("# Load eager features\n")
		sb.WriteString("_omd_load_eager_features() {\n")
		for _, feature := range features.Eager {
			sb.WriteString(fmt.Sprintf(`  local feature_file="$OMD_SHELL_ROOT/features/%s.sh"
  if [ -r "$feature_file" ]; then
    . "$feature_file"
  else
    echo "oh-my-dot: warning: feature '%s' not found" >&2
  fi
`, feature, feature))
		}
		sb.WriteString("}\n\n")
	}

	// Defer loading
	if len(features.Defer) > 0 {
		sb.WriteString("# Load deferred features in the current interactive shell\n")
		sb.WriteString("_omd_load_deferred_features() {\n")
		sb.WriteString("  if [[ $- == *i* ]]; then\n")
		for _, feature := range features.Defer {
			sb.WriteString(fmt.Sprintf(`    local feature_file="$OMD_SHELL_ROOT/features/%s.sh"
    [ -r "$feature_file" ] && . "$feature_file"
`, feature))
		}
		sb.WriteString("  fi\n")
		sb.WriteString("}\n\n")
	}

	// Register wrappers that remove themselves before loading so feature-defined
	// functions remain available after the first invocation.
	if len(features.OnCommand) > 0 {
		sb.WriteString("_omd_register_oncommand_features() {\n")
		writeOnCommandWrappers(&sb, "bash", features.OnCommand)
		sb.WriteString("}\n\n")
	}

	// Execute loading
	sb.WriteString("# Execute loading\n")
	if len(features.Eager) > 0 {
		sb.WriteString("_omd_load_eager_features\n")
	}
	if len(features.OnCommand) > 0 {
		sb.WriteString("_omd_register_oncommand_features\n")
	}
	if len(features.Defer) > 0 {
		sb.WriteString("_omd_load_deferred_features\n")
	}

	// Cleanup (keep _omd_load_eager_features for re-sourcing)
	sb.WriteString("\n# Cleanup (keep _omd_load_eager_features for re-sourcing)\n")
	funcsToClean := []string{}
	// Don't clean up eager features function - needed for re-sourcing .bashrc
	if len(features.Defer) > 0 {
		funcsToClean = append(funcsToClean, "_omd_load_deferred_features")
	}
	if len(features.OnCommand) > 0 {
		funcsToClean = append(funcsToClean, "_omd_register_oncommand_features")
	}
	if len(funcsToClean) > 0 {
		sb.WriteString(fmt.Sprintf("unset -f %s\n", strings.Join(funcsToClean, " ")))
	}

	return sb.String()
}

// generateZshInit generates a zsh init script with all loading strategies
func generateZshInit(features FeaturesByStrategy) string {
	var sb strings.Builder

	// Header
	sb.WriteString(`#!/usr/bin/env zsh
# oh-my-dot shell framework - zsh init script
# Auto-generated - do not edit manually

# Guard against double-loading
if [[ -n "$OMD_ZSH_LOADED" ]]; then
  return 0
fi
OMD_ZSH_LOADED=1

# Determine shell root
OMD_SHELL_ROOT="${${(%):-%x}:A:h}"

# Source helper library
if [[ -r "$OMD_SHELL_ROOT/../lib/helpers.sh" ]]; then
  . "$OMD_SHELL_ROOT/../lib/helpers.sh"
fi

`)

	// Eager loading
	if len(features.Eager) > 0 {
		sb.WriteString("# Load eager features\n")
		sb.WriteString("_omd_load_eager_features() {\n")
		for _, feature := range features.Eager {
			sb.WriteString(fmt.Sprintf(`  local feature_file="$OMD_SHELL_ROOT/features/%s.zsh"
  if [[ -r "$feature_file" ]]; then
    . "$feature_file"
  else
    echo "oh-my-dot: warning: feature '%s' not found" >&2
  fi
`, feature, feature))
		}
		sb.WriteString("}\n\n")
	}

	// Defer loading
	if len(features.Defer) > 0 {
		sb.WriteString("# Load deferred features in the current interactive shell\n")
		sb.WriteString("_omd_load_deferred_features() {\n")
		sb.WriteString("  if [[ -o interactive ]]; then\n")
		for _, feature := range features.Defer {
			sb.WriteString(fmt.Sprintf(`    local feature_file="$OMD_SHELL_ROOT/features/%s.zsh"
    [[ -r "$feature_file" ]] && . "$feature_file"
`, feature))
		}
		sb.WriteString("  fi\n")
		sb.WriteString("}\n\n")
	}

	// Register wrappers that remove themselves before loading so feature-defined
	// functions remain available after the first invocation.
	if len(features.OnCommand) > 0 {
		sb.WriteString("_omd_register_oncommand_features() {\n")
		writeOnCommandWrappers(&sb, "zsh", features.OnCommand)
		sb.WriteString("}\n\n")
	}

	// Execute loading
	sb.WriteString("# Execute loading\n")
	if len(features.Eager) > 0 {
		sb.WriteString("_omd_load_eager_features\n")
	}
	if len(features.OnCommand) > 0 {
		sb.WriteString("_omd_register_oncommand_features\n")
	}
	if len(features.Defer) > 0 {
		sb.WriteString("_omd_load_deferred_features\n")
	}

	// Cleanup
	sb.WriteString("\n# Cleanup\n")
	funcsToClean := []string{}
	if len(features.Eager) > 0 {
		funcsToClean = append(funcsToClean, "_omd_load_eager_features")
	}
	if len(features.Defer) > 0 {
		funcsToClean = append(funcsToClean, "_omd_load_deferred_features")
	}
	if len(features.OnCommand) > 0 {
		funcsToClean = append(funcsToClean, "_omd_register_oncommand_features")
	}
	if len(funcsToClean) > 0 {
		sb.WriteString(fmt.Sprintf("unset -f %s\n", strings.Join(funcsToClean, " ")))
	}

	return sb.String()
}

// generateFishInit generates a fish init script with all loading strategies
func generateFishInit(features FeaturesByStrategy) string {
	var sb strings.Builder

	// Header
	sb.WriteString(`#!/usr/bin/env fish
# oh-my-dot shell framework - fish init script
# Auto-generated - do not edit manually

# Guard against double-loading
if set -q OMD_FISH_LOADED
  exit 0
end
set -g OMD_FISH_LOADED 1

# Determine shell root
set -g OMD_SHELL_ROOT (dirname (status --current-filename))

`)

	// Eager loading
	if len(features.Eager) > 0 {
		sb.WriteString("# Load eager features\n")
		for _, feature := range features.Eager {
			sb.WriteString(fmt.Sprintf(`set -l feature_file "$OMD_SHELL_ROOT/features/%s.fish"
if test -r "$feature_file"
  source "$feature_file"
else
  echo "oh-my-dot: warning: feature '%s' not found" >&2
end

`, feature, feature))
		}
	}

	// Defer loading (using fish_prompt event)
	if len(features.Defer) > 0 {
		sb.WriteString("# Load deferred features in the current shell on first prompt\n")
		sb.WriteString("function __omd_load_deferred --on-event fish_prompt\n")
		sb.WriteString("  # Remove this function after first run\n")
		sb.WriteString("  functions -e __omd_load_deferred\n")
		sb.WriteString("  \n")
		for _, feature := range features.Defer {
			sb.WriteString(fmt.Sprintf(`  set -l feature_file "$OMD_SHELL_ROOT/features/%s.fish"
  test -r "$feature_file"; and source "$feature_file"
`, feature))
		}
		sb.WriteString("end\n\n")
	}

	// Register wrappers that remove themselves before loading so feature-defined
	// functions remain available after the first invocation.
	if len(features.OnCommand) > 0 {
		writeOnCommandWrappers(&sb, "fish", features.OnCommand)
	}

	return sb.String()
}

// generatePowerShellInit generates a PowerShell init script with all loading strategies
func generatePowerShellInit(features FeaturesByStrategy) string {
	var sb strings.Builder

	// Header
	sb.WriteString(`# oh-my-dot shell framework - PowerShell init script
# Auto-generated - do not edit manually

# Guard against double-loading
if ($global:OMD_POWERSHELL_LOADED) {
  return
}
$global:OMD_POWERSHELL_LOADED = $true

# Determine shell root
$OMD_SHELL_ROOT = Split-Path -Parent $PSCommandPath

`)

	// Eager loading
	if len(features.Eager) > 0 {
		sb.WriteString("# Load eager features\n")
		for _, feature := range features.Eager {
			sb.WriteString(fmt.Sprintf(`$featureFile = Join-Path $OMD_SHELL_ROOT "features\%s.ps1"
if (Test-Path $featureFile) {
  . $featureFile
} else {
  Write-Warning "oh-my-dot: feature '%s' not found"
}

`, feature, feature))
		}
	}

	// Defer loading (using background jobs)
	if len(features.Defer) > 0 {
		sb.WriteString("# Load deferred features in the current interactive shell\n")
		sb.WriteString("if ($Host.UI.RawUI) {  # Interactive shell check\n")
		for _, feature := range features.Defer {
			sb.WriteString(fmt.Sprintf(`  $featureFile = Join-Path $OMD_SHELL_ROOT "features\%s.ps1"
  if (Test-Path $featureFile) {
    . $featureFile
  }
`, feature))
		}
		sb.WriteString("}\n\n")
	}

	// Register wrappers that remove themselves before loading so feature-defined
	// functions remain available after the first invocation.
	if len(features.OnCommand) > 0 {
		writeOnCommandWrappers(&sb, "powershell", features.OnCommand)
	}

	return sb.String()
}

// generatePosixInit generates a POSIX sh init script with all loading strategies
func generatePosixInit(repoPath string, features FeaturesByStrategy) string {
	var sb strings.Builder
	shellRoot := filepath.ToSlash(GetShellDirectory(repoPath, "posix"))

	// Header
	sb.WriteString(fmt.Sprintf(`#!/usr/bin/env sh
# oh-my-dot shell framework - POSIX sh init script
# Auto-generated - do not edit manually

# Guard against double-loading
if [ "${OMD_POSIX_LOADED:-}" = "1" ]; then
  return 0 2>/dev/null || exit 0
fi
OMD_POSIX_LOADED=1

# Shell root is generated as an absolute path because POSIX sh cannot
# reliably discover the path of a sourced script.
OMD_SHELL_ROOT=%s

# Source helper library
if [ -r "$OMD_SHELL_ROOT/../lib/helpers.sh" ]; then
  . "$OMD_SHELL_ROOT/../lib/helpers.sh"
fi

`, shellSingleQuote(shellRoot)))

	// Eager loading
	if len(features.Eager) > 0 {
		sb.WriteString("# Load eager features\n")
		for _, feature := range features.Eager {
			sb.WriteString(fmt.Sprintf(`feature_file="$OMD_SHELL_ROOT/features/%s.sh"
if [ -r "$feature_file" ]; then
  . "$feature_file"
else
  echo "oh-my-dot: warning: feature '%s' not found" >&2
fi

`, feature, feature))
		}
	}

	// Defer loading (basic background sourcing)
	if len(features.Defer) > 0 {
		sb.WriteString("# Load deferred features in the current interactive shell\n")
		sb.WriteString("# Check for interactive shell\n")
		sb.WriteString("case $- in\n")
		sb.WriteString("  *i*)\n")
		for _, feature := range features.Defer {
			sb.WriteString(fmt.Sprintf(`    feature_file="$OMD_SHELL_ROOT/features/%s.sh"
    [ -r "$feature_file" ] && . "$feature_file"
`, feature))
		}
		sb.WriteString("    ;;\n")
		sb.WriteString("esac\n\n")
	}

	// Register wrappers that remove themselves before loading so feature-defined
	// functions remain available after the first invocation.
	if len(features.OnCommand) > 0 {
		writeOnCommandWrappers(&sb, "posix", features.OnCommand)
	}

	return sb.String()
}

// writeOnCommandWrappers loads a feature in the invoking shell, then calls the
// resulting function or executable without bypassing shell functions.
func writeOnCommandWrappers(sb *strings.Builder, shellName string, features map[string][]string) {
	for feature, commands := range features {
		allCommands := strings.Join(commands, " ")
		for _, commandName := range commands {
			switch shellName {
			case "bash", "zsh", "posix":
				extension, declaration, remove := "sh", "local ", "unset -f"
				if shellName == "zsh" {
					extension, remove = "zsh", "unfunction"
				}
				if shellName == "posix" {
					declaration = ""
				}
				fmt.Fprintf(sb, `%s() {
  %sfeature_file="$OMD_SHELL_ROOT/features/%s.%s"
  if [ ! -r "$feature_file" ]; then
    echo "oh-my-dot: feature '%s' not found" >&2
    return 127
  fi
  %s %s
  . "$feature_file" || return $?
  if command -v %s >/dev/null 2>&1; then
    %s "$@"
  else
    echo "oh-my-dot: %s command not found after loading feature" >&2
    return 127
  fi
}

`, commandName, declaration, feature, extension, feature, remove, allCommands, commandName, commandName, commandName)
			case "fish":
				fmt.Fprintf(sb, `function %s
  set -l feature_file "$OMD_SHELL_ROOT/features/%s.fish"
  if not test -r "$feature_file"
    echo "oh-my-dot: feature '%s' not found" >&2
    return 127
  end
  functions -e %s
  source "$feature_file"; or return $status
  if type -q %s
    %s $argv
  else
    echo "oh-my-dot: %s command not found after loading feature" >&2
    return 127
  end
end

`, commandName, feature, feature, allCommands, commandName, commandName, commandName)
			case "powershell":
				var remove, promote strings.Builder
				for _, name := range commands {
					fmt.Fprintf(&remove, "  Remove-Item Function:global:%s -ErrorAction SilentlyContinue\n", name)
					fmt.Fprintf(&promote, "  $loadedCommand = Get-Command %s -CommandType Function -ErrorAction SilentlyContinue\n  if ($loadedCommand) { Set-Item Function:global:%s -Value $loadedCommand.ScriptBlock }\n", name, name)
				}
				fmt.Fprintf(sb, `function global:%s {
  $featureFile = Join-Path $OMD_SHELL_ROOT "features\%s.ps1"
  if (-not (Test-Path $featureFile -PathType Leaf)) {
    Write-Warning "oh-my-dot: feature '%s' not found"
    $global:LASTEXITCODE = 127
    return
  }
%s  try { . $featureFile } catch { Write-Warning "oh-my-dot: feature '%s' failed: $_"; $global:LASTEXITCODE = 1; return }
%s  if (Get-Command %s -ErrorAction SilentlyContinue) {
    & %s @args
  } else {
    Write-Warning "oh-my-dot: %s command not found after loading feature"
    $global:LASTEXITCODE = 127
  }
}

`, commandName, feature, feature, remove.String(), feature, promote.String(), commandName, commandName, commandName)
			}
		}
	}
}

// RegenerateInitScript regenerates the init script for a shell
// This should be called after any manifest changes
func RegenerateInitScript(repoPath, shellName string) error {
	return regenerateRuntimeScripts(repoPath, shellName)
}

// TODO: Hook up somehow so the user can use it
// RegenerateAllInitScripts regenerates init scripts for all initialized shells
func RegenerateAllInitScripts(repoPath string) error {
	shells, err := ListShellsWithFeatures(repoPath)
	if err != nil {
		return fmt.Errorf("failed to list shells: %w", err)
	}

	for _, shell := range shells {
		if err := RegenerateInitScript(repoPath, shell); err != nil {
			return fmt.Errorf("failed to regenerate init script for %s: %w", shell, err)
		}
	}

	return nil
}

// writeFile is a helper to write content to a file
func writeFile(path, content string) error {
	return fileops.WriteTextFileLF(path, content, 0644)
}
