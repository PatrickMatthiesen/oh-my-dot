# Spec: Shell Feature Framework (oh-my-dot)

Status: Draft
Owner: Patrick
Feature: Shell profile integration + modular “features” managed by oh-my-dot (omdot)
Last updated: 2026-01-11

## 0. Goals (what this must achieve)

1. **Single-hook integration**: oh-my-dot modifies user shell startup files by adding exactly one managed hook line (plus an identifiable comment block) per supported shell/profile target.
2. **User-owned + portable**: all managed shell configuration lives inside the user’s dotfiles repo (version-controlled), not in `/etc` and not in random generated locations.
3. **Multi-shell**: support bash, zsh, POSIX sh (`.profile`), fish, and PowerShell.
4. **Modular features**: features are independent files. Users can add/remove/enable/disable features via CLI.
5. **Deterministic load order**: feature execution order is stable and predictable.
6. **No double-loading**: init scripts must be idempotent (guards) and safe across chained shell startup flows.
7. **Secure local overrides**: optional “local experimentation” file can be gitignored but must be treated as sensitive input and validated before use.
8. **Idempotent operations**: running any command repeatedly produces the same result without duplicating hooks/entries.
9. **Safe failure modes**: missing features or misconfigurations must not crash the user’s shell.
10. **Lazy loading**: features support configurable load strategies (eager, deferred, on-demand) to optimize shell startup performance.

Non-goals:

- Managing system-wide profiles.
- Implementing every possible shell. (Allow extension later.)
- Syncing secret material; secrets remain the user’s responsibility.

## 1. Terminology

- **Repo root**: user’s dotfiles repository root directory (e.g. `~/dotfiles`).
- **Shell root**: directory inside repo containing shell integration assets: `ohm-shells/`.
- **Init script**: per-shell entry script sourced by the user’s shell profile. Responsible for loading enabled features.
- **Feature**: a self-contained script file that adds behavior (prompt, aliases, env, integrations).
- **Feature catalog**: internal registry (embedded in oh-my-dot binary) containing metadata about available features including default load strategies.
- **Enabled manifest**: a JSON file (`enabled.json`) listing which features to load, in what order, with optional per-feature overrides.
- **Local override manifest**: optional gitignored JSON manifest (`enabled.local.json`) that can add/remove features locally without committing.

## 2. On-disk Layout (source of truth)

All files below are within the dotfiles repo root.

```file System
~/<dotfiles repo>/omd-shells/
  lib/
    helpers.sh
  posix/
    init.sh
    enabled.json
    enabled.local.json   (optional, gitignored)
    features/
      <feature-id>.sh
  bash/
    init.sh
    enabled.json
    enabled.local.json   (optional, gitignored)
    features/
      <feature-id>.sh
  zsh/
    init.zsh
    enabled.json
    enabled.local.json   (optional, gitignored)
    features/
      <feature-id>.zsh
  fish/
    init.fish
    enabled.json
    enabled.local.json   (optional, gitignored)
    features/
      <feature-id>.fish
  powershell/
    init.ps1
    enabled.json
    enabled.local.json   (optional, gitignored)
    features/
      <feature-id>.ps1
```

Notes:

- `lib/helpers.sh` must be POSIX-sh compatible and may be sourced by `posix/init.sh` and `bash/init.sh`. zsh/fish/PowerShell have their own helper idioms.
- Features are shell-specific. Do not place zsh-only features under `posix/`.
- The file extension in `features/` may be shell-specific as shown above. (Alternative: allow `.sh` everywhere, but then loader must match correct extension.)

## 3. Manifest Format

### 3.1 enabled.json (tracked)

- JSON format (UTF-8 encoded).
- Top-level object with `features` array.
- Each feature entry is an object with at minimum a `name` key.

#### Schema

```json
{
  "features": [
    {
      "name": "feature-id",
      "strategy": "eager|defer|on-command",
      "onCommand": ["cmd1", "cmd2"],
      "disabled": false
    }
  ]
}
```

#### Field definitions

- **name** (required, string): Feature identifier
  - Must match regex: `^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`
  - No `/` allowed (prevents path traversal)
  - Max length 128 chars

- **strategy** (optional, string): Load strategy override
  - Valid values: `"eager"`, `"defer"`, `"on-command"`
  - If omitted, uses the feature's default strategy from the internal feature catalog
  - If feature not in catalog, defaults to `"eager"`

- **onCommand** (optional, array of strings): Trigger commands for `on-command` strategy
  - Required if `strategy` is `"on-command"`
  - Ignored for other strategies
  - Each command must be a valid command name (no spaces, paths, or special chars)

- **disabled** (optional, boolean): Whether to skip loading this feature
  - Defaults to `false`
  - Allows keeping feature in manifest without loading it
  - CLI should use this instead of removing features when disabling

#### Example manifest

```json
{
  "features": [
    {
      "name": "core-env"
    },
    {
      "name": "essential-aliases"
    },
    {
      "name": "git-prompt",
      "strategy": "defer"
    },
    {
      "name": "kubectl-completion",
      "strategy": "on-command",
      "onCommand": ["kubectl", "k"]
    },
    {
      "name": "nvm",
      "onCommand": ["nvm", "node", "npm"]
    },
    {
      "name": "experimental-feature",
      "disabled": true
    }
  ]
}
```

#### Ordering

- Array order in `features` determines the registration order.
- Eager features load in array order.
- Deferred features are scheduled in array order but execute asynchronously.
- On-command features are registered in array order.
- The CLI must preserve array ordering unless explicitly reordered by user.

#### Validation

- Init scripts must validate JSON structure before parsing.
- Invalid JSON must cause a warning and fallback to empty feature list (do not crash shell).
- Unknown fields should be ignored (forward compatibility).
- Invalid feature names must be skipped with a warning.

### 3.2 Internal Feature Catalog

oh-my-dot maintains an internal catalog of known features with metadata including recommended load strategies. This catalog is embedded in the oh-my-dot binary and updated with new releases.

#### Catalog entry structure

```go
type FeatureMetadata struct {
    Name            string   // Feature identifier
    Description     string   // Human-readable description
    DefaultStrategy string   // "eager", "defer", or "on-command"
    DefaultCommands []string // Default trigger commands if on-command
    SupportedShells []string // Which shells support this feature
    Category        string   // e.g., "prompt", "completion", "integration"
}
```

#### Usage

- When user enables a feature without specifying strategy: use catalog's `DefaultStrategy`
- When user specifies strategy in `enabled.json`: user's choice overrides catalog default
- Features not in catalog default to `"eager"` strategy
- Catalog provides sensible defaults so users don't need to tune every feature

#### Example catalog entries

```go
var BuiltinFeatures = []FeatureMetadata{
    {
        Name:            "git-prompt",
        Description:     "Git status in prompt",
        DefaultStrategy: "defer",  // Heavy, can defer
        SupportedShells: []string{"bash", "zsh", "fish"},
        Category:        "prompt",
    },
    {
        Name:            "kubectl-completion",
        Description:     "Kubernetes CLI completions",
        DefaultStrategy: "on-command",
        DefaultCommands: []string{"kubectl"},
        SupportedShells: []string{"bash", "zsh", "fish"},
        Category:        "completion",
    },
    {
        Name:            "core-aliases",
        Description:     "Essential command aliases",
        DefaultStrategy: "eager",  // Lightweight, always needed
        SupportedShells: []string{"posix", "bash", "zsh", "fish", "powershell"},
        Category:        "aliases",
    },
}
```

#### CLI integration

- `omdot shell feature add <feature>`: Uses catalog's DefaultStrategy and DefaultCommands
- `omdot shell feature add <feature> --strategy defer`: Overrides catalog default
- `omdot shell feature info <feature>`: Shows catalog metadata including recommended strategy
- `omdot shell feature list`: Shows effective strategy (catalog default or user override)

### 3.4 Load Strategies

Three load strategies are supported for optimizing shell startup performance:

#### eager (default)

- Feature is sourced immediately during init script execution.
- Use for: essential environment setup, critical aliases, lightweight configurations.
- Blocks shell startup until feature loads.

#### defer

- Feature loading is deferred until after the prompt is displayed.
- Use for: heavy completions, non-critical integrations, prompt themes.
- Does not block interactive shell availability.
- Implementation varies by shell:
  - **bash/POSIX**: Schedule via `eval` after script completes (if interactive)
  - **zsh**: Use `zsh-defer` pattern or background sourcing with `source <file> &!`
  - **fish**: Use `--on-event fish_prompt` trigger (first prompt only)
  - **PowerShell**: Use `Register-EngineEvent` or simple background job

#### on-command

- Feature loads only when a specified command is invoked for the first time.
- Requires trigger command(s) specified after second colon: `feature:on-command:cmd1,cmd2`
- Use for: language-specific tools (nvm, rbenv), completions for rarely-used commands, heavy integrations.
- Implementation: Create lightweight wrapper functions for each trigger command that:
  1. Source the feature file
  2. Unset the wrapper function
  3. Execute the actual command (if it now exists) with original arguments
- Most efficient for commands not used in every session.

Example manifest with mixed strategies:

```
# Critical - load immediately
core-env:eager
essential-aliases:eager

# Heavy but always needed - defer to speed up prompt
kubectl-completion:defer
git-prompt:defer

# Only load if used
nvm:on-command:nvm,node,npm
pyenv:on-command:pyenv,python
tf:on-command:terraform,tf
```

### 3.3 enabled.local.json (optional, gitignored)

- Same JSON format and validation as `enabled.json`.
- Used for local experimentation and overrides.
- Merge semantics:

**Effective features = enabled.json + enabled.local.json**

1. Start with features array from `enabled.json`
2. For each feature in `enabled.local.json`:
   - If feature name exists in tracked list: merge/override properties (strategy, onCommand, disabled)
   - If feature name does not exist: append to effective list
3. Array order from `enabled.json` is preserved; new features from local are appended

Meaning:

- Local manifest can override strategy for any tracked feature
- Local manifest can add new features not in tracked list
- Local manifest can disable tracked features by setting `"disabled": true`
- Local manifest cannot reorder features from tracked list

Example override:

```json
// enabled.json (tracked)
{
  "features": [
    {"name": "git-prompt"},
    {"name": "kubectl"}
  ]
}

// enabled.local.json (gitignored)
{
  "features": [
    {
      "name": "git-prompt",
      "strategy": "defer"  // Override to defer
    },
    {
      "name": "kubectl",
      "disabled": true  // Disable locally
    },
    {
      "name": "my-experiment"  // Add locally
    }
  ]
}

// Effective result:
// 1. git-prompt (defer strategy)
// 2. kubectl (skipped - disabled)
// 3. my-experiment (eager by default)
```

## 4. Supported Shell Profile Targets (default)

The CLI must support selecting targets explicitly; default targets are:

- **POSIX sh**: `~/.profile`
- **bash**:
  - Primary: `~/.bashrc`
  - Optional shim: `~/.bash_profile` may be modified ONLY to ensure `.bashrc` is sourced for login shells (see Hook rules).
- **zsh**: `~/.zshrc`
- **fish**: `~/.config/fish/config.fish`
- **PowerShell**: `$PROFILE` (CurrentUserCurrentHost)

Do not modify:

- `/etc/*` files
- zsh `~/.zshenv` by default

## 5. Hooking Rules (single-hook + idempotent)

### 5.1 Hook block markers

All hook insertions MUST be surrounded by markers so uninstall is possible.

Example marker format (exact strings):

- Begin: `# >>> oh-my-dot shell >>>`
- End:   `# <<< oh-my-dot shell <<<`

The block content must be stable and idempotent.

### 5.2 Hook content per shell

#### POSIX sh (`~/.profile`)

Insert:

```sh
# >>> oh-my-dot shell >>>
if [ -r "$HOME/dotfiles/ohm-shells/posix/init.sh" ]; then
  . "$HOME/dotfiles/ohm-shells/posix/init.sh"
fi
# <<< oh-my-dot shell <<<
```

#### bash (`~/.bashrc`)

Insert:

```bash
# >>> oh-my-dot shell >>>
if [ -r "$HOME/dotfiles/ohm-shells/bash/init.sh" ]; then
  . "$HOME/dotfiles/ohm-shells/bash/init.sh"
fi
# <<< oh-my-dot shell <<<
```

#### bash login shim (`~/.bash_profile`) (optional)

If `.bash_profile` exists (or is being created by the user), ensure it sources `.bashrc` without duplicating.

Preferred minimal insertion (separate block markers allowed):
```bash
# >>> oh-my-dot bash login >>>
if [ -r "$HOME/.bashrc" ]; then
  . "$HOME/.bashrc"
fi
# <<< oh-my-dot bash login <<<
```

Rules:

- Do not overwrite existing `.bash_profile` contents.
- Do not add if `.bash_profile` already sources `.bashrc` (heuristic acceptable; must be idempotent).

#### zsh (`~/.zshrc`)

Insert:

```zsh
# >>> oh-my-dot shell >>>
if [ -r "$HOME/dotfiles/ohm-shells/zsh/init.zsh" ]; then
  source "$HOME/dotfiles/ohm-shells/zsh/init.zsh"
fi
# <<< oh-my-dot shell <<<
```

#### fish (`~/.config/fish/config.fish`)

Insert:

```fish
# >>> oh-my-dot shell >>>
if test -r "$HOME/dotfiles/ohm-shells/fish/init.fish"
  source "$HOME/dotfiles/ohm-shells/fish/init.fish"
end
# <<< oh-my-dot shell <<<
```

#### PowerShell (`$PROFILE`)

Insert:

```powershell
# >>> oh-my-dot shell >>>
$omdInit = Join-Path $HOME "dotfiles/ohm-shells/powershell/init.ps1"
if (Test-Path $omdInit) {
  . $omdInit
}
# <<< oh-my-dot shell <<<
```

### 5.3 Hook location

- Insert at end of file by default.
- If markers exist, do not add a second block; update the block if needed.

### 5.4 Repo path configuration

Default repo path is `$HOME/dotfiles`. The CLI must support overriding repo root path, and hook must reflect that path.
(Implementation detail: store repo root path in omdot config; do not hardcode.)

## 6. Init Script Behavior (loader contract)

Each init script must:

1. Be idempotent using a guard variable:
   - POSIX: `OMD_POSIX_LOADED`
   - bash: `OMD_BASH_LOADED`
   - zsh: `OMD_ZSH_LOADED`
   - fish: `OMD_FISH_LOADED`
   - PowerShell: `$global:OMD_POWERSHELL_LOADED`

2. Determine its shell root directory relative to repo root (path is known by hook) or by computing based on script location.

3. Read and parse enabled manifests:
   - Required: `enabled.json`
   - Optional: `enabled.local.json` (only if safe; see Security)
   - Parse JSON and validate structure
   - If JSON is invalid, warn and treat as empty manifest

4. Merge manifests to get effective feature list:
   - Apply local overrides to tracked features
   - Append local-only features
   - For each feature, extract:
     - Feature ID (name)
     - Load strategy (from manifest or catalog default)
     - Trigger commands (for on-command strategy)
     - Disabled flag
   - Skip features with `disabled: true`
   - Validate feature IDs. Invalid entries must be skipped with a warning (printed once per session).

5. Process features by load strategy:

   **Eager features** (strategy = `eager` or no strategy specified):
   - Source immediately in manifest order from `features/`:
     - POSIX: `features/<id>.sh`
     - bash: `features/<id>.sh`
     - zsh: `features/<id>.zsh`
     - fish: `features/<id>.fish`
     - PowerShell: `features/<id>.ps1`

   **Deferred features** (strategy = `defer`):
   - Only in interactive shells:
     - **bash**: Use `eval "source <path>" &` or `{ source <path>; } &` scheduled after prompt
     - **zsh**: Use background sourcing `source <path> &!` or schedule via `precmd` hook (one-time)
     - **fish**: Use `function --on-event fish_prompt` with one-time execution guard
     - **PowerShell**: Use `Register-EngineEvent PowerShell.OnIdle` or simple `Start-Job` pattern
   - In non-interactive shells: treat as eager (fallback)

   **On-command features** (strategy = `on-command:<commands>`):
   - For each trigger command in the comma-separated list:
     - Define a wrapper function with the command name
     - Wrapper must:
       1. Source the feature file
       2. Remove/unset the wrapper function itself
       3. If the command now exists (from the sourced feature), execute it with all original arguments
       4. If command still doesn't exist, print warning
   - Shell-specific wrapper patterns:
     - **bash/POSIX**: `function cmd() { unset -f cmd; source <path>; command cmd "$@"; }`
     - **zsh**: Similar to bash but use `unfunction` instead of `unset -f`
     - **fish**: Use `functions -e` to erase wrapper before executing
     - **PowerShell**: Use `Remove-Item Function:` to remove wrapper

6. Missing feature files:
   - Must not crash the shell.
   - Must warn (once per missing feature per session).
   - Continue loading remaining features.
   - For on-command wrappers: create wrapper that warns on invocation instead of sourcing.

7. No recursion:
   - Init scripts must not source user profile files (`~/.profile`, `~/.bashrc`, etc.).

8. Minimal side effects when non-interactive:
   - POSIX `init.sh` must be safe in non-interactive shells. Prefer env setup only.
   - Deferred and on-command loading should be skipped or adapted for non-interactive contexts.
   - bash/zsh may check for interactive shells and avoid heavy prompt setup in non-interactive contexts.

## 7. Security Requirements (local override and sourcing safety)

### 7.1 Local override file is “data-only”

- `enabled.local.json` is NOT a script. It is a JSON data file with the same schema as `enabled.json`.
- Init scripts must use a safe JSON parser (not `eval` or `source`).

### 7.2 Validate before use

Init script must apply strict validation:

- Only accept feature IDs matching regex in §3.
- Do not interpret paths.
- Do not eval.

### 7.3 Permission checks (Unix)

Before reading `enabled.local.json` on Unix-like systems, init must ensure:

- File exists and is a regular file (not symlink).
- Owned by current user.
- Not group-writable and not world-writable.

If checks fail:

- Ignore `enabled.local.json`.
- Print a warning once per session indicating it was ignored due to unsafe permissions.

(Exact implementation may vary per shell; correctness > portability. If a shell cannot perform the checks reliably, it may fall back to ignoring the local file with a warning.)

### 7.4 Windows (PowerShell)

- If ACL checks are implemented: ensure file is not writable by “Everyone” and not writable by other users.
- If not implemented: still support `enabled.local.json` but print a warning that ACLs are not verified (or default to disabled local override on Windows; pick one behavior and document it).

## 8. CLI Requirements (omdot)

### 8.1 Commands (minimum)

- `omdot shell init [--shell <name>] [--repo <path>]`
  - Creates directory structure + init scripts + empty enabled.json for specified shells.
  - Initializes enabled.json with empty features array: `{"features":[]}`
  - Does not auto-edit user profiles unless explicitly requested (see apply).
- `omdot shell apply [--shell <name>] [--repo <path>]`
  - Inserts hook blocks into the appropriate shell profile file(s).
  - Idempotent.
- `omdot shell unapply [--shell <name>]`
  - Removes hook blocks only (does not delete repo files).
- `omdot shell feature add <feature> [--shell <name>] [--strategy <strategy>] [--on-command <cmds>]`
  - Adds feature template file to `features/` if not present.
  - Adds feature to `enabled.json` unless `--disabled` is passed.
  - `--strategy`: Override load strategy from catalog default (values: `eager`, `defer`, `on-command`)
  - `--on-command`: Override trigger commands (comma-separated, for use with `--strategy on-command` or when overriding on-command feature's commands)
  - If no flags provided: uses catalog's default strategy and commands
  - Overwrites existing feature file if `--force` is passed.
- `omdot shell feature remove <feature> [--shell <name>]`
  - Deletes feature file from `features/` if present.
  - Removes from enabled.list if present.
  - If `--force` not passed, must confirm with user before deleting.
- `omdot shell feature enable <feature> [--shell <name>] [--strategy <strategy>] [--on-command <cmds>]`
  - Adds feature to `enabled.json` if not present, or sets `disabled: false` if present.
  - If feature already enabled, updates strategy/commands if flags provided.
  - `--strategy`: Override load strategy from catalog default
  - `--on-command`: Override trigger commands
  - If no flags: uses catalog defaults or keeps existing overrides
- `omdot shell feature disable <feature> [--shell <name>]`
  - Sets `"disabled": true` in `enabled.json` (does not remove entry).
  - Keeps feature file in `features/` directory.
- `omdot shell feature list [--shell <name>]`
  - Shows installed features with:
    - Enabled status
    - Load strategy (showing if from catalog default or user override)
    - Trigger commands (if on-command)
    - Local override indicator
  - Example output:
    ```
    ✓ core-aliases (eager, catalog default)
    ✓ git-prompt (defer, overridden) [local override]
    ✓ kubectl (on-command: kubectl, catalog default)
    ✗ docker-completion (disabled)
    ✓ my-experiment (eager, catalog default) [local only]
    ```
- `omdot shell feature info <feature>`
  - Shows detailed information about a feature from catalog:
    - Description
    - Default strategy and commands
    - Supported shells
    - Category
    - Current user configuration (if enabled)
- `omdot shell doctor`
  - Checks: hook present, repo files exist, manifests valid, local override permissions safe.

### 8.2 Feature identifiers

The `<feature>` argument maps to a feature ID, which determines the filename.

- The CLI must reject feature IDs not matching the regex in §3.

### 8.3 Shell selection

Supported shell identifiers:

- `posix`, `bash`, `zsh`, `fish`, `powershell`

Default behavior:

- If `--shell` not provided, operate on “current shell” if detectable, else require explicit `--shell`.

## 9. Feature Template Contract

A feature file:

- Must be valid for its shell.
- Must not exit the shell on error; it should fail gracefully.
- Must not assume interactive context unless it checks.
- Should be idempotent where possible (avoid duplicating PATH entries, re-defining functions dangerously, etc.).

Recommended (not required):

- Features should log using helper functions so warnings are consistent.

## 10. Idempotency & Uninstall

- Applying hooks multiple times must not duplicate blocks.
- Removing hooks must remove exactly the marked blocks and nothing else.
- Features enable/disable operations must be no-op if already in desired state.

## 11. Testing Requirements (Go)

Minimum automated tests:

- Manifest parsing: 
  - Valid JSON structure with features array
  - Invalid JSON handled gracefully (warning, empty list)
  - Feature name validation (regex, length)
  - Load strategy parsing: eager, defer, on-command
  - Invalid strategies rejected with warning
  - Trigger command validation for on-command
  - Unknown fields ignored (forward compatibility)
- Feature catalog:
  - Catalog lookups return correct defaults
  - Missing features default to eager
  - Catalog data integrity (valid strategies, shells)
- Merge behavior with enabled.local.json:
  - Strategy override from local manifest
  - Disabled flag from local manifest
  - Local-only features appended
  - Order preservation from tracked manifest
- Hook patching: insertion, update, removal using golden files for each shell profile target.
- Path handling: repo path override reflected in hook content.
- Feature enable/disable: 
  - JSON updates preserve structure and ordering
  - Disable sets flag rather than removing entry
  - Strategy updates preserve feature position
  - On-command trigger commands properly formatted
- CLI commands:
  - Add uses catalog defaults when no flags provided
  - Enable/disable updates disabled flag correctly
  - Info command shows catalog metadata

## 12. Open Decisions (must be resolved before implementation)

3. Shell detection policy for default `omdot shell apply`: auto-detect vs require explicit shell.

---

## Appendix A: Guard variable examples

POSIX `posix/init.sh` must begin with:

```sh
if [ "${OMD_POSIX_LOADED:-}" = "1" ]; then
  return 0 2>/dev/null || exit 0
fi
OMD_POSIX_LOADED=1
```

bash/zsh/fish/PowerShell analogs should follow their idioms.

---

## Appendix B: Error handling rules

- Warnings should go to stderr.
- Warnings should be rate-limited (e.g., print once per feature per session).
- Init scripts should never cause a non-zero exit of an interactive shell session.

---

## Appendix C: Load Strategy Implementation Examples

### bash on-command wrapper

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

### bash defer loading

```bash
# In init.sh, schedule deferred features
if [[ $- == *i* ]]; then
  # Interactive shell - defer heavy features
  _omd_deferred_features=(
    "$OMD_SHELL_ROOT/features/git-prompt.sh"
    "$OMD_SHELL_ROOT/features/kubectl-completion.sh"
  )
  
  # Load after init completes
  for _feat in "${_omd_deferred_features[@]}"; do
    [ -r "$_feat" ] && { source "$_feat"; } &
  done
  unset _omd_deferred_features _feat
fi
```

### zsh on-command wrapper

```zsh
# For feature "nvm" with triggers "nvm,node,npm"
nvm() {
  unfunction nvm
  local feature_file="$OMD_SHELL_ROOT/features/nvm.zsh"
  [[ -r "$feature_file" ]] && source "$feature_file"
  if (( $+commands[nvm] )); then
    nvm "$@"
  else
    echo "oh-my-dot: nvm command not found after loading feature" >&2
    return 127
  fi
}

# Same wrapper for node and npm
node() { nvm; unfunction node; node "$@"; }
npm() { nvm; unfunction npm; npm "$@"; }
```

### fish on-command wrapper

```fish
# For feature "docker" with trigger "docker"
function docker
  functions -e docker
  set feature_file "$OMD_SHELL_ROOT/features/docker.fish"
  test -r "$feature_file"; and source "$feature_file"
  if type -q docker
    command docker $argv
  else
    echo "oh-my-dot: docker command not found after loading feature" >&2
    return 127
  end
end
```

### fish defer loading

```fish
# In init.fish, use fish_prompt event for deferred load
function __omd_load_deferred --on-event fish_prompt
  # Remove this function after first run
  functions -e __omd_load_deferred
  
  # Load deferred features
  for feature in git-prompt kubectl-completion
    set feature_file "$OMD_SHELL_ROOT/features/$feature.fish"
    test -r "$feature_file"; and source "$feature_file" &
  end
end
```

### PowerShell on-command wrapper

```powershell
# For feature "terraform" with triggers "terraform,tf"
function terraform {
  Remove-Item Function:terraform -ErrorAction SilentlyContinue
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

Set-Alias tf terraform
```

### PowerShell defer loading

```powershell
# In init.ps1, use background jobs for deferred features
if ($Host.UI.RawUI) {
  # Interactive shell
  $deferredFeatures = @(
    (Join-Path $env:OMD_SHELL_ROOT "features/git-prompt.ps1"),
    (Join-Path $env:OMD_SHELL_ROOT "features/posh-git.ps1")
  )
  
  foreach ($feature in $deferredFeatures) {
    if (Test-Path $feature) {
      Start-Job -ScriptBlock { param($f) . $f } -ArgumentList $feature | Out-Null
    }
  }
}
```
