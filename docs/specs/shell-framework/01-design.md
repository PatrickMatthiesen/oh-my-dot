# Updated Plan: Integrated Shell Framework

## Overview

This plan updates the shell framework design to integrate seamlessly with the existing oh-my-dot workflow, eliminating the need for users to remember separate `omdot shell` commands.

---

## Key Design Principles

1. **Zero cognitive overhead** - `omdot apply` just works for everything
2. **Auto-initialization** - Shell directories created on-demand when adding features
3. **Smart cleanup** - Hooks removed automatically when last feature is removed
4. **Intelligent defaults** - Most common case (current shell) requires zero flags
5. **Flattened CLI** - Top-level `feature` commands instead of nested `shell feature`

---

## Updated CLI Design

### Core Commands (Modified)

#### `omdot init <remote>`
**Behavior:** Creates dotfiles repository (existing behavior, unchanged)
- Does NOT create shell framework structure
- Keeps initialization simple and focused on dotfiles

#### `omdot apply`
**Behavior:** Applies dotfiles AND shell hooks (if any exist)
- Applies all dotfile symlinks (existing behavior)
- **NEW:** If `omd-shells/` directory exists with features:
  - Detects which shells have features configured
  - Applies hooks to those shell profiles
  - Idempotent - safe to run multiple times
- **NEW:** `--no-shell` flag to skip shell hook application

**Example output:**
```bash
$ omdot apply
Applying dotfiles... ✓ 15 files linked
Applying shell integration...
  bash: Adding hook to ~/.bashrc ✓
  zsh: Adding hook to ~/.zshrc ✓
Done!
```

#### `omdot doctor`
**Behavior:** Unified health check for dotfiles AND shell integration
- Checks dotfile links (existing)
- **NEW:** Checks shell hooks are present
- **NEW:** Validates manifest files
- **NEW:** Checks for missing feature files
- **NEW:** Verifies local override permissions

**Example output:**
```bash
$ omdot doctor
Checking dotfiles...
  ✓ Repository at ~/dotfiles
  ✓ 15 files tracked
  ✓ All links valid

Checking shell integration...
  ✓ bash: Hook present in ~/.bashrc
  ✓ bash: 3 features enabled (all files found)
  ✓ zsh: Hook present in ~/.zshrc
  ⚠ zsh: Feature 'kubectl' enabled but file missing
  
Run 'omdot feature list --shell zsh' for details
```

---

### New Feature Commands (Top-Level)

#### `omdot feature add <feature>`
**Behavior:** Adds a feature to shell(s)

**Interactive mode with `-i` flag:**
- `omdot feature add -i` - Browse and select features from catalog interactively
- Multi-select interface showing:
  - Feature name and description
  - Category (completion, prompt, alias, tool)
  - Supported shells
  - Default load strategy
- Features sorted by category (alias → completion → prompt → tool), then alphabetically
- After selecting features, prompts for shell selection

**Smart shell selection:**
1. If feature only supports current shell → add to current shell automatically
2. If feature supports multiple shells OR not current shell → interactive prompt
3. Can override with `--shell <name>` or `--all` flags

**Auto-initialization:**
- If shell directory doesn't exist, creates it:
  ```
  omd-shells/<shell>/
    init.<ext>
    enabled.json
    features/
  ```

**Example flows:**
```bash
# Interactive catalog browsing (sorted by category, then alphabetically)
$ omdot feature add -i
? Select features to add:
  ◉ core-aliases (Essential command aliases)
  ◯ directory-shortcuts (Quick navigation)
  ◯ aws-completion (AWS CLI completion)
  ◯ docker-completion (Docker CLI completion)
  ◯ kubectl-completion (Kubernetes CLI completion)
  ◉ git-prompt (Show git branch in prompt)
  ◯ nvm (Node Version Manager)
  ◯ python-venv (Python virtual environment)
  ...
? Select shells:
  ◉ bash (current)
  ◯ zsh
Adding core-aliases to bash... ✓
Adding git-prompt to bash... ✓

# Feature only works in bash, user is in bash
$ omdot feature add kubectl-completion
Adding kubectl-completion to bash... ✓
Run 'omdot apply' to activate

# Feature works in multiple shells
$ omdot feature add git-prompt
? Select shells to add feature to:
  ◉ bash (current)
  ◉ zsh
  ◯ fish (not installed)
Adding git-prompt to bash... ✓
Adding git-prompt to zsh... ✓
Run 'omdot apply' to activate

# Explicit shell
$ omdot feature add nvm --shell bash
Adding nvm to bash... ✓
Run 'omdot apply' to activate
```

**Flags:**
- `-i, --interactive` - Browse and select features from catalog
- `--shell <name>` - Target specific shell
- `--all` - Add to all supported shells (that are initialized)
- `--strategy <eager|defer|on-command>` - Override load strategy
- `--on-command <cmd1,cmd2>` - Set trigger commands
- `--disabled` - Add but don't enable

#### `omdot feature remove <feature>`
**Behavior:** Removes a feature from shell(s)

**Smart cleanup:**
- If removing last feature from a shell:
  - Removes the hook from profile file
  - Removes the `omd-shells/<shell>/` directory
  - Commits removal to git

**Example:**
```bash
$ omdot feature remove kubectl-completion
? Remove from which shells?
  ◉ bash
  ◯ zsh
Removing kubectl-completion from bash... ✓

Cleaning up bash shell integration:
  - No features remaining
  - Removing hook from ~/.bashrc ✓
  - Removing omd-shells/bash/ directory ✓
  
Run 'omdot apply' to sync changes
```

**Flags:**
- `--shell <name>` - Remove from specific shell
- `--all` - Remove from all shells
- `--force` - Skip confirmation

#### `omdot feature enable <feature>`
**Behavior:** Enables a feature (sets `disabled: false`)

**Example:**
```bash
$ omdot feature enable docker-completion
Enabling docker-completion in bash... ✓
Run 'omdot apply' to activate
```

**Flags:**
- `--shell <name>` - Target specific shell
- `--strategy <eager|defer|on-command>` - Override strategy
- `--on-command <cmd1,cmd2>` - Set trigger commands

#### `omdot feature disable <feature>`
**Behavior:** Disables a feature (sets `disabled: true`)

**Example:**
```bash
$ omdot feature disable docker-completion
Disabling docker-completion in bash... ✓
Run 'omdot apply' to sync changes
```

**Flags:**
- `--shell <name>` - Target specific shell
- `--all` - Disable in all shells

#### `omdot feature list [--shell <name>]`
**Behavior:** Lists features for shell(s) with their file locations

**Example:**
```bash
$ omdot feature list
bash:
  ✓ core-aliases (eager, catalog default)
    ~/dotfiles/omd-shells/bash/features/core-aliases.sh
  ✓ git-prompt (defer, overridden)
    ~/dotfiles/omd-shells/bash/features/git-prompt.sh
  ✓ kubectl (on-command: kubectl, catalog default)
    ~/dotfiles/omd-shells/bash/features/kubectl.sh
  ✗ docker-completion (disabled)
    ~/dotfiles/omd-shells/bash/features/docker-completion.sh

zsh:
  ✓ git-prompt (defer, catalog default)
    ~/dotfiles/omd-shells/zsh/features/git-prompt.zsh
  ✓ nvm (on-command: nvm,node,npm, catalog default)
    ~/dotfiles/omd-shells/zsh/features/nvm.zsh
  
# Specific shell
$ omdot feature list --shell bash
```

#### `omdot feature info <feature>`
**Behavior:** Shows detailed information about a feature

**Example:**
```bash
$ omdot feature info kubectl-completion

kubectl-completion
  Category: completion
  Description: Kubernetes CLI completions
  Default Strategy: on-command
  Default Commands: kubectl
  Supported Shells: bash, zsh, fish
  
Current Configuration:
  bash: enabled (on-command: kubectl,k)
  zsh: not installed
```

---

## Updated Workflow Examples

### First-Time Setup

```bash
# Initialize oh-my-dot
$ omdot init github.com/user/dotfiles

# Add some dotfiles
$ omdot add ~/.gitconfig ~/.vimrc

# Add shell features (auto-creates shell structure)
$ omdot feature add core-aliases
Adding core-aliases to bash... ✓

# Apply everything at once
$ omdot apply
Applying dotfiles... ✓ 2 files linked
Applying shell integration...
  bash: Adding hook to ~/.bashrc ✓
Done! Restart your shell or run: source ~/.bashrc
```

### Daily Usage

```bash
# Add more features
$ omdot feature add kubectl-completion
$ omdot feature add git-prompt

# Apply once - handles both dotfiles and shells
$ omdot apply

# Check status
$ omdot feature list

# Diagnose issues
$ omdot doctor
```

### Cleanup

```bash
# Remove a feature
$ omdot feature remove kubectl-completion

# If it was the last feature, hooks are auto-removed
$ omdot apply
Applying dotfiles... ✓ 15 files linked
Shell integration:
  bash: No features configured, removing hook... ✓
Done!
```

---

## Updated Implementation Phases

### Phase 0: Foundation (1-2 weeks)
**Unchanged from original plan**
- Package structure
- Data types
- Manifest parsing
- Testing infrastructure

### Phase 1: Feature Catalog (1 week)
**Moved earlier - needed for feature detection**
- Implement feature catalog in binary
- Add 10-15 common features with metadata
- Catalog lookup and validation
- Supported shells per feature

### Phase 2: Feature Management Core (2 weeks)
**Replaces old "Shell Detection" phase**
- Implement `omdot feature add` with auto-init
- Implement `omdot feature remove` with auto-cleanup
- Implement `omdot feature enable/disable`
- Implement `omdot feature list`
- Implement `omdot feature info`
- Interactive shell selection
- Smart defaults (current shell auto-select)

### Phase 3: Hook Management (2 weeks)
**Integrated into apply command**
- Hook insertion/removal logic
- Detect shells with features
- **Modify `omdot apply`** to apply hooks automatically
- Add `--no-shell` flag to apply
- Make hooks idempotent

### Phase 4: Init Script Generation (2 weeks)
**Eager loading only**
- Generate init scripts per shell
- Implement guard variables
- Load eager features
- Missing feature warnings
- Testing per shell

### Phase 5: Advanced Load Strategies (2-3 weeks)
**Defer and on-command**
- Implement defer loading
- Implement on-command wrappers
- Strategy validation
- Shell-specific implementations
- Performance testing

### Phase 6: Local Overrides & Security (1-2 weeks)
**Security validation**
- enabled.local.json support
- Permission checking
- Manifest merging
- Security warnings

### Phase 7: Doctor Command (1 week)
**Unified diagnostics**
- Integrate shell checks into `omdot doctor`
- Hook validation
- Manifest validation
- Missing file detection
- Permission checking

### Phase 8: Polish & Documentation (1-2 weeks)
**Final polish**
- Error messages
- Documentation
- Example features
- User testing

**Updated Timeline:** 14-17 weeks (~4 months)

---

## Key Behavior Changes

### Automatic Hook Application

**Old design:**
```bash
omdot apply              # Apply dotfiles
omdot shell apply        # Apply shell hooks (separate command)
```

**New design:**
```bash
omdot apply              # Apply BOTH dotfiles and shell hooks
omdot apply --no-shell   # Skip shell hooks if needed
```

### Automatic Cleanup

**Old design:**
```bash
omdot shell feature remove git-prompt
omdot shell unapply      # Remove hooks manually
```

**New design:**
```bash
omdot feature remove git-prompt
# If last feature removed, hooks auto-removed
omdot apply              # Syncs the removal
```

### On-Demand Initialization

**Old design:**
```bash
omdot shell init --shell bash      # Initialize first
omdot shell feature add git-prompt # Then add features
```

**New design:**
```bash
omdot feature add git-prompt
# Auto-creates omd-shells/bash/ if needed
# No separate init command
```

### Intelligent Shell Selection

**Old design:**
```bash
omdot shell feature add git-prompt --shell bash
omdot shell feature add git-prompt --shell zsh
```

**New design:**
```bash
omdot feature add git-prompt
# Interactive prompt if multiple shells supported
# Auto-selects if only current shell supported
```

---

## Updated File Structure

```
~/dotfiles/
├── .git/
├── .gitconfig          # Tracked dotfiles
├── .vimrc
└── omd-shells/         # Shell framework (created on-demand)
    ├── lib/
    │   └── helpers.sh
    └── bash/           # Created when first feature added
        ├── init.sh
        ├── enabled.json
        └── features/
            ├── core-aliases.sh
            └── git-prompt.sh
```

**Note:** `omd-shells/` only exists if user has added at least one feature

---

## Migration from Original Spec

### Commands Removed
- ❌ `omdot shell init` → Auto-init on first `feature add`
- ❌ `omdot shell apply` → Integrated into `omdot apply`
- ❌ `omdot shell unapply` → Auto-cleanup on last `feature remove`
- ❌ `omdot shell feature *` → Flattened to `omdot feature *`
- ❌ `omdot shell doctor` → Integrated into `omdot doctor`

### Commands Moved/Renamed
- `omdot shell feature add` → `omdot feature add`
- `omdot shell feature remove` → `omdot feature remove`
- `omdot shell feature enable` → `omdot feature enable`
- `omdot shell feature disable` → `omdot feature disable`
- `omdot shell feature list` → `omdot feature list`
- `omdot shell feature info` → `omdot feature info`

### Behavior Changes
- ✅ `omdot apply` now applies shell hooks automatically
- ✅ `omdot doctor` now checks shell integration
- ✅ Auto-creates shell directories when needed
- ✅ Auto-removes hooks when last feature removed
- ✅ Smart shell selection based on feature compatibility

---

## Benefits of New Design

1. **Simpler mental model** - One `apply` command for everything
2. **Less typing** - `feature add` vs `shell feature add`
3. **Auto-cleanup** - No orphaned hooks or directories
4. **Smart defaults** - Works without flags 80% of the time
5. **Progressive disclosure** - Simple cases are simple, complex cases possible
6. **Consistent with dotfile workflow** - Same patterns throughout

---

## Next Steps

1. ✅ Review and approve this updated design
2. See `03-implementation-roadmap.md` for detailed implementation phases
3. See `02-decisions.md` for all finalized decisions
4. Begin implementation with Phase 0 from the roadmap

---

## Related Documents

- **All decisions finalized:** `02-decisions.md` (answers the questions below)
- **Implementation roadmap:** `03-implementation-roadmap.md`
- **Code examples:** `04-code-examples.md`

## Original Open Questions (Now Resolved)

These questions have been answered in `02-decisions.md`:

1. **Commit behavior:** Wait for user to commit manually ✅
2. **Push behavior:** Include shell framework changes in `omdot push` ✅
3. **Interactive mode:** Require explicit `--shell` flag in non-interactive mode ✅