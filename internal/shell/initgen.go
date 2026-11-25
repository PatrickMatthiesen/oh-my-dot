package shell

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
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
	features := categorizeFeaturesMerged(merged, shellName)
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
func categorizeFeaturesMerged(m *manifest.MergedManifest, shellName string) FeaturesByStrategy {
	features := FeaturesByStrategy{
		OnCommand: make(map[string][]string),
	}
	registeredTriggers := make(map[string]bool)

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
			for _, trigger := range f.OnCommand {
				key := trigger
				if shellName == "powershell" {
					key = strings.ToLower(key)
				}
				if !registeredTriggers[key] {
					registeredTriggers[key] = true
					features.OnCommand[f.Name] = append(features.OnCommand[f.Name], trigger)
				}
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

	// Schedule deferred loading after eager features have configured the prompt.
	if len(features.Defer) > 0 {
		sb.WriteString(`
# Load deferred features once at the first interactive prompt.
if [[ $- == *i* ]]; then
  _omd_load_deferred_features() {
    if (( BASH_VERSINFO[0] * 100 + BASH_VERSINFO[1] >= 501 )) &&
      [[ $(declare -p PROMPT_COMMAND 2>/dev/null) == 'declare -a'* ]]; then
      local hook_index
      for hook_index in "${!PROMPT_COMMAND[@]}"; do
        if [[ ${PROMPT_COMMAND[hook_index]} == _omd_load_deferred_features ]]; then
          unset 'PROMPT_COMMAND[hook_index]'
        fi
      done
    else
      # Match the generated command line even if the profile appended hooks.
      local prompt_hooks="${PROMPT_COMMAND-}"$'\n'
      prompt_hooks=${prompt_hooks/$'\n'_omd_load_deferred_features[$'\n;']/$'\n'}
      PROMPT_COMMAND=${prompt_hooks%$'\n'}
    fi
    unset -f _omd_load_deferred_features
`)
		for _, feature := range features.Defer {
			sb.WriteString(fmt.Sprintf(`    local feature_file="$OMD_SHELL_ROOT/features/%s.sh"
    if [[ -r "$feature_file" ]]; then
      . "$feature_file"
    fi
`, feature))
		}
		sb.WriteString(`    return 0
  }

  # Bash versions before 5.1 execute only the first array element.
  if (( BASH_VERSINFO[0] * 100 + BASH_VERSINFO[1] >= 501 )) &&
    [[ $(declare -p PROMPT_COMMAND 2>/dev/null) == 'declare -a'* ]]; then
    PROMPT_COMMAND+=(_omd_load_deferred_features)
  else
    PROMPT_COMMAND="${PROMPT_COMMAND-}"$'\n'_omd_load_deferred_features
  fi
fi
`)
	}

	// Keep the eager loader for re-sourcing.
	if len(features.OnCommand) > 0 {
		sb.WriteString("\nunset -f _omd_register_oncommand_features\n")
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

# Re-apply eager features when the init script is sourced again.
if [[ -n "${OMD_ZSH_LOADED:-}" ]]; then
  if (( $+functions[_omd_load_eager_features] )); then
    _omd_load_eager_features
  fi
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

	// Schedule deferred loading after eager features have configured the prompt.
	if len(features.Defer) > 0 {
		sb.WriteString(`
# Load deferred features once at the first interactive prompt.
if [[ -o interactive ]]; then
  autoload -Uz add-zsh-hook
  _omd_load_deferred_features() {
    add-zsh-hook -d precmd _omd_load_deferred_features
    unfunction _omd_load_deferred_features
`)
		for _, feature := range features.Defer {
			sb.WriteString(fmt.Sprintf(`    local feature_file="$OMD_SHELL_ROOT/features/%s.zsh"
    if [[ -r "$feature_file" ]]; then
      . "$feature_file"
    fi
`, feature))
		}
		sb.WriteString(`    return 0
  }
  add-zsh-hook precmd _omd_load_deferred_features
fi
`)
	}

	// Keep the eager loader for re-sourcing.
	if len(features.OnCommand) > 0 {
		sb.WriteString("\nunset -f _omd_register_oncommand_features\n")
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

# Determine shell root
set -g OMD_SHELL_ROOT (dirname (status --current-filename))

`)

	// Eager loading
	if len(features.Eager) > 0 {
		sb.WriteString("# Load eager features on every source\n")
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

	// A return in a sourced Fish file can also return from the calling function.
	sb.WriteString(`# Register deferred and on-command features once per shell
if not set -q OMD_FISH_LOADED
  set -g OMD_FISH_LOADED 1

`)

	// Fish runs deferred features once on the first interactive prompt.
	if len(features.Defer) > 0 {
		sb.WriteString(`  # Load deferred features in the current shell on first prompt
  if status is-interactive
    function __omd_load_deferred --on-event fish_prompt
      functions -e __omd_load_deferred
`)
		for _, feature := range features.Defer {
			sb.WriteString(fmt.Sprintf(`      set -l feature_file "$OMD_SHELL_ROOT/features/%s.fish"
      if test -r "$feature_file"
        source "$feature_file"
      end
`, feature))
		}
		sb.WriteString("    end\n  end\n\n")
	}

	// Register wrappers that remove themselves before loading so feature-defined
	// functions remain available after the first invocation.
	if len(features.OnCommand) > 0 {
		var wrappers strings.Builder
		writeOnCommandWrappers(&wrappers, "fish", features.OnCommand)
		for line := range strings.SplitSeq(wrappers.String(), "\n") {
			if line != "" {
				sb.WriteString("  " + line + "\n")
			}
		}
	}
	sb.WriteString("end\n")

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

# Shell root is generated as an absolute path because POSIX sh cannot
# reliably discover the path of a sourced script.
OMD_SHELL_ROOT=%s

# Source helper library before the first eager load
if [ "${OMD_POSIX_LOADED:-}" != "1" ] && [ -r "$OMD_SHELL_ROOT/../lib/helpers.sh" ]; then
  . "$OMD_SHELL_ROOT/../lib/helpers.sh"
fi

`, shellSingleQuote(shellRoot)))

	// Eager loading
	if len(features.Eager) > 0 {
		sb.WriteString("# Load eager features on every source\n")
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

	sb.WriteString(`# Register on-command wrappers and load deferred features once per shell
if [ "${OMD_POSIX_LOADED:-}" = "1" ]; then
  return 0 2>/dev/null || exit 0
fi
OMD_POSIX_LOADED=1

`)

	// Deferred features can call the wrappers or replace their functions.
	if len(features.OnCommand) > 0 {
		writeOnCommandWrappers(&sb, "posix", features.OnCommand)
	}

	// POSIX sh has no portable first-prompt event, so defer runs inline once.
	if len(features.Defer) > 0 {
		sb.WriteString(`# POSIX sh has no portable prompt hook. Load deferred features inline
# after eager features and on-command wrappers in interactive shells.
case $- in
  *i*)
`)
		for _, feature := range features.Defer {
			sb.WriteString(fmt.Sprintf(`    feature_file="$OMD_SHELL_ROOT/features/%s.sh"
    if [ -r "$feature_file" ]; then
      . "$feature_file"
    fi
`, feature))
		}
		sb.WriteString("    ;;\n")
		sb.WriteString("esac\n\n")
	}

	return sb.String()
}

// writeOnCommandWrappers loads a feature in the invoking shell, then calls the
// resulting command without bypassing shell functions or aliases.
func writeOnCommandWrappers(sb *strings.Builder, shellName string, features map[string][]string) {
	for index, feature := range slices.Sorted(maps.Keys(features)) {
		commands := features[feature]
		loader := fmt.Sprintf("_omd_load_oncommand_%d", index)
		switch shellName {
		case "bash", "zsh", "posix":
			extension, remove, inspect := "sh", "unset -f", "declare -f"
			if shellName == "zsh" {
				extension, remove, inspect = "zsh", "unfunction", "functions"
			}
			loaderCall := loader + ` "$@" || return $?`
			if shellName != "posix" {
				sb.WriteString("local ")
			}
			sb.WriteString("_omd_oncommand_registered=''\n")
			for _, commandName := range commands {
				if shellName == "posix" && (strings.Contains(commandName, "-") || strings.TrimLeft(commandName, "0123456789") != commandName) {
					fmt.Fprintf(sb, "printf '%%s\\n' %s >&2\n", shellSingleQuote(fmt.Sprintf("oh-my-dot: warning: '%s' is not a portable sh function name; skipping on-command trigger", commandName)))
					continue
				}

				// Keep aliases, functions, builtins and keywords. Delay parsing
				// the wrapper until after that check.
				switch shellName {
				case "bash":
					fmt.Fprintf(sb, `if ! alias -- '%s' >/dev/null 2>&1; then
  case "$(type -t -- '%s')" in
    alias|function|builtin|keyword) ;;
    *)
`, commandName, commandName)
				case "zsh":
					fmt.Fprintf(sb, `case "$(whence -w -- '%s')" in
  *": command"|*": hashed"|*": none"|'')
`, commandName)
				case "posix":
					// Limit the lookup to shell commands by disabling PATH lookup.
					fmt.Fprintf(sb, "if ! PATH=/dev/null command -v '%s' >/dev/null 2>&1; then\n", commandName)
				}

				// Reparse the validated trigger name so the feature's aliases
				// can expand on the first call. Eval text never includes arguments.
				load := loaderCall
				if shellName == "posix" {
					load = fmt.Sprintf("unset -f %s\n  %s", commandName, loaderCall)
				}
				wrapper := fmt.Sprintf(`%s() {
  %s
  if command -v '%s' >/dev/null 2>&1; then
    eval '%s "$@"'
  else
    echo "oh-my-dot: %s command not found after loading feature" >&2
    return 127
  fi
}`, commandName, load, commandName, commandName, commandName)
				fmt.Fprintf(sb, "    eval %s && _omd_oncommand_registered=\"$_omd_oncommand_registered %s\"\n", shellSingleQuote(wrapper), commandName)
				switch shellName {
				case "bash":
					sb.WriteString("      ;;\n  esac\nfi\n")
				case "zsh":
					sb.WriteString("    ;;\nesac\n")
				case "posix":
					sb.WriteString("fi\n")
				}
			}

			// Source the feature in a separate function so `set --` cannot
			// replace the wrapper's arguments.
			loaderStart := fmt.Sprintf("%s() {\n", loader)
			var loaderEnd string
			if shellName == "posix" {
				// POSIX sh cannot inspect functions. Each wrapper removes
				// itself, and the loader records the attempt before sourcing.
				loaderEnd = fmt.Sprintf("  %s() { return 0; }\n", loader)
			} else {
				loaderStart += fmt.Sprintf("  %s %s\n  local _omd_trigger\n  for _omd_trigger in", remove, loader)
				loaderEnd = fmt.Sprintf(`; do
    case "$(builtin %s -- "$_omd_trigger" 2>/dev/null)" in
      *%s*) %s "$_omd_trigger" ;;
    esac
  done
`, inspect, shellSingleQuote(loaderCall), remove)
			}
			loaderEnd += fmt.Sprintf(`  if [ ! -r "$OMD_SHELL_ROOT/features/%s.%s" ]; then
    echo "oh-my-dot: feature '%s' not found" >&2
    return 127
  fi
  . "$OMD_SHELL_ROOT/features/%s.%s"
}`, feature, extension, feature, feature, extension)
			sb.WriteString("if [ -n \"$_omd_oncommand_registered\" ]; then\n")
			if shellName == "posix" {
				sb.WriteString(loaderStart + loaderEnd + "\nfi\nunset _omd_oncommand_registered\n")
			} else {
				fmt.Fprintf(sb, "  eval %s\"$_omd_oncommand_registered\"%s\nfi\n", shellSingleQuote(loaderStart), shellSingleQuote(loaderEnd))
			}
			sb.WriteString("\n")
		case "fish":
			sb.WriteString("begin\n  set -l _omd_oncommand_registered\n")
			for _, commandName := range commands {
				fmt.Fprintf(sb, `  if not functions -q -- %s; and not builtin -q -- %s
    function %s
      %s; or return $status
      set -l trigger %s
      if type -q -- $trigger
        $trigger $argv
      else
        echo "oh-my-dot: %s command not found after loading feature" >&2
        return 127
      end
    end
    set -a _omd_oncommand_registered %s
  end
`, commandName, commandName, commandName, loader, commandName, commandName, commandName)
			}
			fmt.Fprintf(sb, `  if set -q _omd_oncommand_registered[1]
    function %s --inherit-variable _omd_oncommand_registered
      functions -e %s
      for trigger in $_omd_oncommand_registered
        if string match -q -- '*%s; or return $status*' (functions -- $trigger 2>/dev/null)
          functions -e -- $trigger
        end
      end
      set -l feature_file "$OMD_SHELL_ROOT/features/%s.fish"
      if not test -r "$feature_file"
        echo "oh-my-dot: feature '%s' not found" >&2
        return 127
      end
      source "$feature_file"
    end
  end
end

`, loader, loader, loader, feature, feature)

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
