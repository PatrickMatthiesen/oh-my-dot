package shell

import (
	"fmt"
	"os"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
)

// FeaturesByStrategy organizes features by their loading strategy
type FeaturesByStrategy struct {
	Eager     []string
	Defer     []string
	OnCommand map[string][]string // feature name -> trigger commands
}

// GenerateInitScript generates a complete init script for a shell
// Supports eager, defer, and on-command loading strategies (Phase 5)
// Supports local overrides via enabled.local.json (Phase 6)
func GenerateInitScript(repoPath, shellName string) (string, error) {
	// Parse manifest with local overrides
	manifestPath := GetManifestPath(repoPath, shellName)
	localManifestPath := GetLocalManifestPath(repoPath, shellName)

	merged, err := manifest.ParseManifestWithLocal(manifestPath, localManifestPath)
	if err != nil {
		return "", fmt.Errorf("failed to parse manifests: %w", err)
	}

	// Organize features by strategy
	features := categorizeFeaturesMerged(merged)

	// Generate shell-specific init script
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
		return generatePosixInit(features), nil
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
  if [ -n "${_omd_load_eager_features:-}" ] && type -t _omd_load_eager_features >/dev/null 2>&1; then
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
		sb.WriteString("# Load deferred features (background)\n")
		sb.WriteString("_omd_load_deferred_features() {\n")
		sb.WriteString("  if [[ $- == *i* ]]; then\n")
		for _, feature := range features.Defer {
			sb.WriteString(fmt.Sprintf(`    ( [ -r "$OMD_SHELL_ROOT/features/%s.sh" ] && . "$OMD_SHELL_ROOT/features/%s.sh" ) &
`, feature, feature))
		}
		sb.WriteString("  fi\n")
		sb.WriteString("}\n\n")
	}

	// On-command loading
	if len(features.OnCommand) > 0 {
		sb.WriteString("# Register on-command features\n")
		sb.WriteString("_omd_register_oncommand_features() {\n")

		for featureName, commands := range features.OnCommand {
			// Group commands that share the same feature
			if len(commands) == 1 {
				// Single command - simple wrapper
				cmd := commands[0]
				sb.WriteString(fmt.Sprintf(`  %s() {
    unset -f %s
    local feature_file="$OMD_SHELL_ROOT/features/%s.sh"
    if [ -r "$feature_file" ]; then
      . "$feature_file"
    fi
    if command -v %s >/dev/null 2>&1; then
      command %s "$@"
    else
      echo "oh-my-dot: %s command not found after loading feature" >&2
      return 127
    fi
  }
`, cmd, cmd, featureName, cmd, cmd, cmd))
			} else {
				// Multiple commands - use helper function
				loaderFunc := fmt.Sprintf("__omd_load_%s", featureName)
				sb.WriteString(fmt.Sprintf(`  %s() {
    local feature_file="$OMD_SHELL_ROOT/features/%s.sh"
    [ -r "$feature_file" ] && . "$feature_file"
  }
`, loaderFunc, featureName))

				// Create wrappers for each command
				for _, cmd := range commands {
					allCommands := strings.Join(commands, " ")
					sb.WriteString(fmt.Sprintf(`  %s() { %s; unset -f %s %s; command %s "$@"; }
`, cmd, loaderFunc, allCommands, loaderFunc, cmd))
				}
			}
			sb.WriteString("\n")
		}

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
		sb.WriteString("# Load deferred features (background)\n")
		sb.WriteString("_omd_load_deferred_features() {\n")
		sb.WriteString("  if [[ -o interactive ]]; then\n")
		for _, feature := range features.Defer {
			sb.WriteString(fmt.Sprintf(`    ( [[ -r "$OMD_SHELL_ROOT/features/%s.zsh" ]] && . "$OMD_SHELL_ROOT/features/%s.zsh" ) &!
`, feature, feature))
		}
		sb.WriteString("  fi\n")
		sb.WriteString("}\n\n")
	}

	// On-command loading
	if len(features.OnCommand) > 0 {
		sb.WriteString("# Register on-command features\n")
		sb.WriteString("_omd_register_oncommand_features() {\n")

		for featureName, commands := range features.OnCommand {
			if len(commands) == 1 {
				// Single command - simple wrapper
				cmd := commands[0]
				sb.WriteString(fmt.Sprintf(`  %s() {
    unfunction %s
    local feature_file="$OMD_SHELL_ROOT/features/%s.zsh"
    [[ -r "$feature_file" ]] && . "$feature_file"
    if (( $+commands[%s] )); then
      %s "$@"
    else
      echo "oh-my-dot: %s command not found after loading feature" >&2
      return 127
    fi
  }
`, cmd, cmd, featureName, cmd, cmd, cmd))
			} else {
				// Multiple commands - use helper function
				loaderFunc := fmt.Sprintf("__omd_load_%s", featureName)
				sb.WriteString(fmt.Sprintf(`  %s() {
    local feature_file="$OMD_SHELL_ROOT/features/%s.zsh"
    [[ -r "$feature_file" ]] && . "$feature_file"
  }
`, loaderFunc, featureName))

				// Create wrappers for each command
				for _, cmd := range commands {
					allCommands := strings.Join(commands, " ")
					sb.WriteString(fmt.Sprintf(`  %s() { %s; unfunction %s %s; command %s "$@"; }
`, cmd, loaderFunc, allCommands, loaderFunc, cmd))
				}
			}
			sb.WriteString("\n")
		}

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
set -l OMD_SHELL_ROOT (dirname (status --current-filename))

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
		sb.WriteString("# Load deferred features (on first prompt)\n")
		sb.WriteString("function __omd_load_deferred --on-event fish_prompt\n")
		sb.WriteString("  # Remove this function after first run\n")
		sb.WriteString("  functions -e __omd_load_deferred\n")
		sb.WriteString("  \n")
		sb.WriteString("  # Load deferred features in background\n")
		for _, feature := range features.Defer {
			sb.WriteString(fmt.Sprintf(`  set -l feature_file "$OMD_SHELL_ROOT/features/%s.fish"
  test -r "$feature_file"; and source "$feature_file" &
`, feature))
		}
		sb.WriteString("end\n\n")
	}

	// On-command loading
	if len(features.OnCommand) > 0 {
		sb.WriteString("# Register on-command features\n")
		for featureName, commands := range features.OnCommand {
			if len(commands) == 1 {
				// Single command - simple wrapper
				cmd := commands[0]
				sb.WriteString(fmt.Sprintf(`function %s
  functions -e %s
  set -l feature_file "$OMD_SHELL_ROOT/features/%s.fish"
  test -r "$feature_file"; and source "$feature_file"
  if type -q %s
    command %s $argv
  else
    echo "oh-my-dot: %s command not found after loading feature" >&2
    return 127
  end
end

`, cmd, cmd, featureName, cmd, cmd, cmd))
			} else {
				// Multiple commands - use helper function
				loaderFunc := fmt.Sprintf("__omd_load_%s", featureName)
				sb.WriteString(fmt.Sprintf(`function %s
  set -l feature_file "$OMD_SHELL_ROOT/features/%s.fish"
  test -r "$feature_file"; and source "$feature_file"
end

`, loaderFunc, featureName))

				// Create wrappers for each command
				for _, cmd := range commands {
					allCommands := append(commands, loaderFunc)
					sb.WriteString(fmt.Sprintf(`function %s
  %s
  functions -e %s
  command %s $argv
end

`, cmd, loaderFunc, strings.Join(allCommands, " "), cmd))
				}
			}
		}
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
		sb.WriteString("# Load deferred features (background jobs)\n")
		sb.WriteString("if ($Host.UI.RawUI) {  # Interactive shell check\n")
		for _, feature := range features.Defer {
			sb.WriteString(fmt.Sprintf(`  $featureFile = Join-Path $OMD_SHELL_ROOT "features\%s.ps1"
  if (Test-Path $featureFile) {
    Start-Job -ScriptBlock { param($f) . $f } -ArgumentList $featureFile | Out-Null
  }
`, feature))
		}
		sb.WriteString("}\n\n")
	}

	// On-command loading
	if len(features.OnCommand) > 0 {
		sb.WriteString("# Register on-command features\n")
		for featureName, commands := range features.OnCommand {
			if len(commands) == 1 {
				// Single command - simple wrapper
				cmd := commands[0]
				sb.WriteString(fmt.Sprintf(`function %s {
  Remove-Item Function:%s -ErrorAction SilentlyContinue
  $featureFile = Join-Path $OMD_SHELL_ROOT "features\%s.ps1"
  if (Test-Path $featureFile) {
    . $featureFile
  }
  if (Get-Command %s -ErrorAction SilentlyContinue) {
    & %s @args
  } else {
    Write-Warning "oh-my-dot: %s command not found after loading feature"
    exit 127
  }
}

`, cmd, cmd, featureName, cmd, cmd, cmd))
			} else {
				// Multiple commands - use helper function
				loaderFunc := fmt.Sprintf("__omd_load_%s", featureName)
				sb.WriteString(fmt.Sprintf(`function %s {
  $featureFile = Join-Path $OMD_SHELL_ROOT "features\%s.ps1"
  if (Test-Path $featureFile) {
    . $featureFile
  }
}

`, loaderFunc, featureName))

				// Create wrappers for each command
				for _, cmd := range commands {
					allCommands := append(commands, loaderFunc)
					removeCmds := make([]string, len(allCommands))
					for i, c := range allCommands {
						removeCmds[i] = fmt.Sprintf("Remove-Item Function:%s -ErrorAction SilentlyContinue", c)
					}
					sb.WriteString(fmt.Sprintf(`function %s {
  %s
  %s
  & %s @args
}

`, cmd, loaderFunc, strings.Join(removeCmds, "; "), cmd))
				}
			}
		}
	}

	return sb.String()
}

// generatePosixInit generates a POSIX sh init script with all loading strategies
func generatePosixInit(features FeaturesByStrategy) string {
	var sb strings.Builder

	// Header
	sb.WriteString(`#!/usr/bin/env sh
# oh-my-dot shell framework - POSIX sh init script
# Auto-generated - do not edit manually

# Guard against double-loading
if [ "${OMD_POSIX_LOADED:-}" = "1" ]; then
  return 0 2>/dev/null || exit 0
fi
OMD_POSIX_LOADED=1

# Determine shell root (POSIX-compatible)
OMD_SHELL_ROOT="$(cd "$(dirname "$0")" && pwd)"

# Source helper library
if [ -r "$OMD_SHELL_ROOT/../lib/helpers.sh" ]; then
  . "$OMD_SHELL_ROOT/../lib/helpers.sh"
fi

`)

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
		sb.WriteString("# Load deferred features (background)\n")
		sb.WriteString("# Check for interactive shell\n")
		sb.WriteString("case $- in\n")
		sb.WriteString("  *i*)\n")
		for _, feature := range features.Defer {
			sb.WriteString(fmt.Sprintf(`    ( [ -r "$OMD_SHELL_ROOT/features/%s.sh" ] && . "$OMD_SHELL_ROOT/features/%s.sh" ) &
`, feature, feature))
		}
		sb.WriteString("    ;;\n")
		sb.WriteString("esac\n\n")
	}

	// On-command loading
	if len(features.OnCommand) > 0 {
		sb.WriteString("# Register on-command features\n")
		for featureName, commands := range features.OnCommand {
			if len(commands) == 1 {
				// Single command - simple wrapper
				cmd := commands[0]
				sb.WriteString(fmt.Sprintf(`%s() {
  unset -f %s
  feature_file="$OMD_SHELL_ROOT/features/%s.sh"
  if [ -r "$feature_file" ]; then
    . "$feature_file"
  fi
  if command -v %s >/dev/null 2>&1; then
    command %s "$@"
  else
    echo "oh-my-dot: %s command not found after loading feature" >&2
    return 127
  fi
}

`, cmd, cmd, featureName, cmd, cmd, cmd))
			} else {
				// Multiple commands - use helper function
				loaderFunc := fmt.Sprintf("__omd_load_%s", featureName)
				sb.WriteString(fmt.Sprintf(`%s() {
  feature_file="$OMD_SHELL_ROOT/features/%s.sh"
  [ -r "$feature_file" ] && . "$feature_file"
}

`, loaderFunc, featureName))

				// Create wrappers for each command
				for _, cmd := range commands {
					allCommands := strings.Join(commands, " ")
					sb.WriteString(fmt.Sprintf(`%s() { %s; unset -f %s %s; command %s "$@"; }
`, cmd, loaderFunc, allCommands, loaderFunc, cmd))
				}
				sb.WriteString("\n")
			}
		}
	}

	return sb.String()
}

// RegenerateInitScript regenerates the init script for a shell
// This should be called after any manifest changes
func RegenerateInitScript(repoPath, shellName string) error {
	// Generate new init script content
	content, err := GenerateInitScript(repoPath, shellName)
	if err != nil {
		return fmt.Errorf("failed to generate init script: %w", err)
	}

	// Get init script path
	initPath, err := GetInitScriptPath(repoPath, shellName)
	if err != nil {
		return err
	}

	// Write init script
	return writeFile(initPath, content)
}

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
	return os.WriteFile(path, []byte(content), 0644)
}
