# Shell Framework Examples & Templates

## Overview

This document provides practical, reusable examples for implementing the shell framework. It consolidates the shell-specific patterns to reduce redundancy while maintaining clarity.

---

## 1. Hook Insertion Patterns

### Hook Block Structure

All hooks use the same marker pattern:

```
# >>> oh-my-dot shell >>>
<shell-specific hook content>
# <<< oh-my-dot shell <<<
```

### Generic Hook Template (POSIX-compatible shells)

**Pattern:** bash, POSIX sh, zsh (with minor variations)

```bash
# >>> oh-my-dot shell >>>
if [ -r "$HOME/dotfiles/omd-shells/<SHELL>/init.<EXT>" ]; then
  . "$HOME/dotfiles/omd-shells/<SHELL>/init.<EXT>"
fi
# <<< oh-my-dot shell <<<
```

**Variations:**
- **POSIX sh**: Uses `. ` (dot space)
- **bash**: Uses `. ` (dot space) or `source`
- **zsh**: Prefers `source` but `. ` works

### Shell-Specific Hook Examples

#### bash (~/.bashrc)

```bash
# >>> oh-my-dot shell >>>
if [ -r "$HOME/dotfiles/omd-shells/bash/init.sh" ]; then
  . "$HOME/dotfiles/omd-shells/bash/init.sh"
fi
# <<< oh-my-dot shell <<<
```

#### zsh (~/.zshrc)

```zsh
# >>> oh-my-dot shell >>>
if [ -r "$HOME/dotfiles/omd-shells/zsh/init.zsh" ]; then
  source "$HOME/dotfiles/omd-shells/zsh/init.zsh"
fi
# <<< oh-my-dot shell <<<
```

#### fish (~/.config/fish/config.fish)

```fish
# >>> oh-my-dot shell >>>
if test -r "$HOME/dotfiles/omd-shells/fish/init.fish"
  source "$HOME/dotfiles/omd-shells/fish/init.fish"
end
# <<< oh-my-dot shell <<<
```

#### PowerShell ($PROFILE)

```powershell
# >>> oh-my-dot shell >>>
$omdInit = Join-Path $HOME "dotfiles/omd-shells/powershell/init.ps1"
if (Test-Path $omdInit) {
  . $omdInit
}
# <<< oh-my-dot shell <<<
```

### Bash Login Shim (~/.bash_profile)

Special case to ensure `.bashrc` is sourced in login shells:

```bash
# >>> oh-my-dot bash login >>>
if [ -r "$HOME/.bashrc" ]; then
  . "$HOME/.bashrc"
fi
# <<< oh-my-dot bash login <<<
```

**Rules:**
- Only add if `.bash_profile` doesn't already source `.bashrc`
- Use different markers than main hook
- Don't overwrite existing `.bash_profile` content

---

## 2. Init Script Templates

### Guard Variable Pattern

All init scripts use a guard variable to prevent double-loading:

```bash
# Generic pattern (POSIX)
if [ "${OMD_<SHELL>_LOADED:-}" = "1" ]; then
  return 0 2>/dev/null || exit 0
fi
OMD_<SHELL>_LOADED=1
```

### Complete Init Script Template (bash)

```bash
# oh-my-dot shell framework - bash init script
# Auto-generated - do not edit manually

# Guard against double-loading
if [ "${OMD_BASH_LOADED:-}" = "1" ]; then
  return 0
fi
OMD_BASH_LOADED=1

# Determine shell root
OMD_SHELL_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source helper library
if [ -r "$OMD_SHELL_ROOT/../lib/helpers.sh" ]; then
  . "$OMD_SHELL_ROOT/../lib/helpers.sh"
fi

# Read and parse enabled.json
_omd_features=()
# (JSON parsing logic here - see section 3)

# Read and merge enabled.local.json (if safe)
# (Local override logic here - see section 4)

# Load features by strategy
_omd_load_eager_features() {
  for feature in "${_omd_eager_features[@]}"; do
    local feature_file="$OMD_SHELL_ROOT/features/${feature}.sh"
    if [ -r "$feature_file" ]; then
      . "$feature_file"
    else
      echo "oh-my-dot: warning: feature '$feature' not found" >&2
    fi
  done
}

_omd_load_deferred_features() {
  if [[ $- == *i* ]]; then
    for feature in "${_omd_deferred_features[@]}"; do
      local feature_file="$OMD_SHELL_ROOT/features/${feature}.sh"
      [ -r "$feature_file" ] && { . "$feature_file"; } &
    done
  fi
}

_omd_register_oncommand_features() {
  # (On-command wrapper logic here - see section 5)
}

# Execute loading
_omd_load_eager_features
_omd_register_oncommand_features
_omd_load_deferred_features

# Cleanup
unset -f _omd_load_eager_features _omd_load_deferred_features _omd_register_oncommand_features
```

### Shell-Specific Guard Variables

| Shell | Guard Variable | Pattern |
|-------|---------------|---------|
| POSIX | `OMD_POSIX_LOADED` | `if [ "${OMD_POSIX_LOADED:-}" = "1" ]` |
| bash | `OMD_BASH_LOADED` | `if [ "${OMD_BASH_LOADED:-}" = "1" ]` |
| zsh | `OMD_ZSH_LOADED` | `if [[ -n "$OMD_ZSH_LOADED" ]]` |
| fish | `OMD_FISH_LOADED` | `if set -q OMD_FISH_LOADED` |
| PowerShell | `$OMD_POWERSHELL_LOADED` | `if ($global:OMD_POWERSHELL_LOADED)` |

---

## 3. Load Strategy Implementations

### Eager Loading (All Shells)

**Concept:** Source immediately during init script execution.

#### bash/POSIX
```bash
for feature in "${_omd_eager_features[@]}"; do
  local feature_file="$OMD_SHELL_ROOT/features/${feature}.sh"
  if [ -r "$feature_file" ]; then
    . "$feature_file"
  fi
done
```

#### zsh
```zsh
for feature in $_omd_eager_features; do
  local feature_file="$OMD_SHELL_ROOT/features/${feature}.zsh"
  [[ -r "$feature_file" ]] && source "$feature_file"
done
```

#### fish
```fish
for feature in $omd_eager_features
  set feature_file "$OMD_SHELL_ROOT/features/$feature.fish"
  test -r "$feature_file"; and source "$feature_file"
end
```

#### PowerShell
```powershell
foreach ($feature in $omdEagerFeatures) {
  $featureFile = Join-Path $env:OMD_SHELL_ROOT "features/$feature.ps1"
  if (Test-Path $featureFile) {
    . $featureFile
  }
}
```

---

### Defer Loading (Shell-Specific)

**Concept:** Schedule loading after interactive prompt appears.

#### bash
```bash
# Only in interactive shells
if [[ $- == *i* ]]; then
  _omd_deferred_features=(git-prompt kubectl-completion)
  
  for feature in "${_omd_deferred_features[@]}"; do
    local feature_file="$OMD_SHELL_ROOT/features/${feature}.sh"
    [ -r "$feature_file" ] && { source "$feature_file"; } &
  done
  
  unset _omd_deferred_features
fi
```

#### zsh
```zsh
# Background sourcing with job control
if [[ -o interactive ]]; then
  typeset -a _omd_deferred_features=(git-prompt kubectl-completion)
  
  for feature in $_omd_deferred_features; do
    local feature_file="$OMD_SHELL_ROOT/features/${feature}.zsh"
    [[ -r "$feature_file" ]] && source "$feature_file" &!
  done
  
  unset _omd_deferred_features
fi
```

#### fish
```fish
# Use fish_prompt event (one-time)
function __omd_load_deferred --on-event fish_prompt
  # Remove this function after first run
  functions -e __omd_load_deferred
  
  # Load deferred features in background
  for feature in git-prompt kubectl-completion
    set feature_file "$OMD_SHELL_ROOT/features/$feature.fish"
    test -r "$feature_file"; and source "$feature_file" &
  end
end
```

#### PowerShell
```powershell
# Use background jobs for deferred loading
if ($Host.UI.RawUI) {  # Interactive shell check
  $deferredFeatures = @('git-prompt', 'posh-git')
  
  foreach ($feature in $deferredFeatures) {
    $featureFile = Join-Path $env:OMD_SHELL_ROOT "features/$feature.ps1"
    if (Test-Path $featureFile) {
      Start-Job -ScriptBlock { param($f) . $f } -ArgumentList $featureFile | Out-Null
    }
  }
}
```

---

### On-Command Loading (Wrapper Pattern)

**Concept:** Create wrapper function that loads feature on first invocation.

#### Generic Wrapper Pattern

All shells follow this pattern:
1. Define function with command name
2. Unset/remove the wrapper function
3. Source the feature file
4. Execute the actual command (if it now exists)

#### bash
```bash
# For feature "kubectl" with trigger command "kubectl"
kubectl() {
  unset -f kubectl
  local feature_file="$OMD_SHELL_ROOT/features/kubectl.sh"
  
  if [ -r "$feature_file" ]; then
    source "$feature_file"
  fi
  
  if command -v kubectl >/dev/null 2>&1; then
    command kubectl "$@"
  else
    echo "oh-my-dot: kubectl command not found after loading feature" >&2
    return 127
  fi
}
```

**Multiple trigger commands (e.g., nvm, node, npm):**
```bash
# Main feature loader
__omd_load_nvm() {
  local feature_file="$OMD_SHELL_ROOT/features/nvm.sh"
  [ -r "$feature_file" ] && source "$feature_file"
}

nvm() { __omd_load_nvm; unset -f nvm node npm __omd_load_nvm; command nvm "$@"; }
node() { __omd_load_nvm; unset -f nvm node npm __omd_load_nvm; command node "$@"; }
npm() { __omd_load_nvm; unset -f nvm node npm __omd_load_nvm; command npm "$@"; }
```

#### zsh
```zsh
# For feature "kubectl" with trigger "kubectl"
kubectl() {
  unfunction kubectl
  local feature_file="$OMD_SHELL_ROOT/features/kubectl.zsh"
  
  [[ -r "$feature_file" ]] && source "$feature_file"
  
  if (( $+commands[kubectl] )); then
    kubectl "$@"
  else
    echo "oh-my-dot: kubectl command not found after loading feature" >&2
    return 127
  fi
}
```

**Multiple triggers:**
```zsh
__omd_load_nvm() {
  local feature_file="$OMD_SHELL_ROOT/features/nvm.zsh"
  [[ -r "$feature_file" ]] && source "$feature_file"
}

nvm() { __omd_load_nvm; unfunction nvm node npm __omd_load_nvm; nvm "$@"; }
node() { __omd_load_nvm; unfunction nvm node npm __omd_load_nvm; node "$@"; }
npm() { __omd_load_nvm; unfunction nvm node npm __omd_load_nvm; npm "$@"; }
```

#### fish
```fish
# For feature "kubectl" with trigger "kubectl"
function kubectl
  functions -e kubectl
  set feature_file "$OMD_SHELL_ROOT/features/kubectl.fish"
  
  test -r "$feature_file"; and source "$feature_file"
  
  if type -q kubectl
    command kubectl $argv
  else
    echo "oh-my-dot: kubectl command not found after loading feature" >&2
    return 127
  end
end
```

**Multiple triggers:**
```fish
function __omd_load_nvm
  set feature_file "$OMD_SHELL_ROOT/features/nvm.fish"
  test -r "$feature_file"; and source "$feature_file"
end

function nvm
  __omd_load_nvm
  functions -e nvm node npm __omd_load_nvm
  command nvm $argv
end

function node
  __omd_load_nvm
  functions -e nvm node npm __omd_load_nvm
  command node $argv
end

function npm
  __omd_load_nvm
  functions -e nvm node npm __omd_load_nvm
  command npm $argv
end
```

#### PowerShell
```powershell
# For feature "terraform" with triggers "terraform, tf"
function terraform {
  Remove-Item Function:terraform -ErrorAction SilentlyContinue
  Remove-Item Function:tf -ErrorAction SilentlyContinue
  
  $featureFile = Join-Path $env:OMD_SHELL_ROOT "features/terraform.ps1"
  if (Test-Path $featureFile) {
    . $featureFile
  }
  
  if (Get-Command terraform -ErrorAction SilentlyContinue) {
    & terraform @args
  } else {
    Write-Warning "oh-my-dot: terraform command not found after loading feature"
    exit 127
  }
}

# Alias for short command
Set-Alias tf terraform
```

---

## 4. Security Validation (Local Overrides)

### Unix Permission Checking (bash example)

```bash
_omd_validate_local_manifest() {
  local manifest_file="$1"
  
  # File must exist and be a regular file
  if [ ! -f "$manifest_file" ] || [ -L "$manifest_file" ]; then
    return 1
  fi
  
  # Check ownership (must be current user)
  if ! stat -c '%U' "$manifest_file" 2>/dev/null | grep -q "^$USER$"; then
    echo "oh-my-dot: warning: $manifest_file not owned by current user, ignoring" >&2
    return 1
  fi
  
  # Check permissions (not group or world writable)
  local perms=$(stat -c '%a' "$manifest_file" 2>/dev/null)
  if [ "${perms:1:1}" -ge 2 ] || [ "${perms:2:1}" -ge 2 ]; then
    echo "oh-my-dot: warning: $manifest_file is group/world writable, ignoring" >&2
    return 1
  fi
  
  return 0
}

# Usage
local_manifest="$OMD_SHELL_ROOT/enabled.local.json"
if [ -f "$local_manifest" ] && _omd_validate_local_manifest "$local_manifest"; then
  # Safe to read
  # (JSON parsing here)
fi
```

### Windows ACL Checking (PowerShell)

```powershell
function Test-LocalManifestSecurity {
  param([string]$ManifestPath)
  
  if (-not (Test-Path $ManifestPath)) {
    return $false
  }
  
  $acl = Get-Acl $ManifestPath
  
  # Check if writable by Everyone or other users
  $unsafeRules = $acl.Access | Where-Object {
    ($_.IdentityReference -eq "Everyone" -or 
     $_.IdentityReference -ne $env:USERNAME) -and
    $_.FileSystemRights -match "Write"
  }
  
  if ($unsafeRules) {
    Write-Warning "oh-my-dot: $ManifestPath has unsafe permissions, ignoring"
    return $false
  }
  
  return $true
}

# Usage
$localManifest = Join-Path $env:OMD_SHELL_ROOT "enabled.local.json"
if ((Test-Path $localManifest) -and (Test-LocalManifestSecurity $localManifest)) {
  # Safe to read
  # (JSON parsing here)
}
```

---

## 5. Feature Template Examples

### Example: Core Aliases (eager)

**File:** `bash/features/core-aliases.sh`

```bash
# oh-my-dot feature: core-aliases
# Essential command aliases
# Strategy: eager

# Safety aliases
alias rm='rm -i'
alias cp='cp -i'
alias mv='mv -i'

# Directory navigation
alias ..='cd ..'
alias ...='cd ../..'
alias ....='cd ../../..'

# Listing aliases
alias ll='ls -lah'
alias la='ls -A'
alias l='ls -CF'

# Git shortcuts
alias gs='git status'
alias gd='git diff'
alias gl='git log --oneline --graph'
alias ga='git add'
alias gc='git commit'
```

### Example: kubectl Completion (on-command)

**File:** `bash/features/kubectl-completion.sh`

```bash
# oh-my-dot feature: kubectl-completion
# Kubernetes CLI completion
# Strategy: on-command (triggers: kubectl, k)

if command -v kubectl >/dev/null 2>&1; then
  source <(kubectl completion bash)
  
  # Short alias
  alias k='kubectl'
  complete -F __start_kubectl k
fi
```

### Example: Git Prompt (defer)

**File:** `zsh/features/git-prompt.zsh`

```zsh
# oh-my-dot feature: git-prompt
# Git status in prompt
# Strategy: defer

autoload -Uz vcs_info
precmd_vcs_info() { vcs_info }
precmd_functions+=( precmd_vcs_info )
setopt prompt_subst

zstyle ':vcs_info:git:*' formats ' %b'
zstyle ':vcs_info:*' enable git

PROMPT='%F{green}%~%f%F{yellow}${vcs_info_msg_0_}%f %# '
```

---

## 6. Error Handling Examples

### Missing Feature File

```bash
# Good: Warn but don't crash
if [ -r "$feature_file" ]; then
  . "$feature_file"
else
  echo "oh-my-dot: warning: feature '$feature_name' not found: $feature_file" >&2
  echo "  Run 'omdot shell doctor' to diagnose issues" >&2
fi
```

### Invalid JSON Manifest

```bash
# Good: Fallback to empty list
if ! _omd_features=$(parse_json "$manifest_file" 2>/dev/null); then
  echo "oh-my-dot: warning: invalid JSON in $manifest_file, using empty feature list" >&2
  _omd_features=()
fi
```

### Command Not Found After Loading

```bash
# Good: Clear error message with exit code
if command -v kubectl >/dev/null 2>&1; then
  command kubectl "$@"
else
  echo "oh-my-dot: kubectl command not found after loading feature" >&2
  echo "  Feature file may be empty or command not installed" >&2
  return 127
fi
```

---

## Summary

This document provides reusable patterns for:
- ✅ Hook insertion across all shells
- ✅ Init script structure with guards
- ✅ Eager, defer, and on-command loading strategies
- ✅ Security validation for local overrides
- ✅ Error handling best practices
- ✅ Feature template examples

Use these patterns as templates when implementing the shell framework.