# Plan: Shell Framework Implementation Phases

## Overview

This document breaks down the implementation of the shell framework into prioritized phases with clear dependencies. Each phase delivers a working, testable increment of functionality.

---

## Phase 0: Foundation (Prerequisites)

**Goal:** Set up the core infrastructure needed for all subsequent phases.

**Duration Estimate:** 1-2 weeks

### Tasks:

1. **Create Go package structure:**
   ```
   internal/
   ├── shell/          # Core shell framework logic
   │   ├── types.go    # Data structures (FeatureManifest, FeatureConfig)
   │   ├── catalog.go  # Feature catalog management
   │   └── shells.go   # Shell detection and configuration
   ├── manifest/       # JSON manifest management
   │   ├── parse.go    # JSON parsing and validation
   │   ├── merge.go    # Manifest merging logic
   │   └── validate.go # Feature ID and strategy validation
   └── hooks/          # Profile file patching
       ├── markers.go  # Hook marker management
       ├── insert.go   # Hook insertion logic
       └── remove.go   # Hook removal logic
   ```

2. **Define core data structures:**
   ```go
   type FeatureManifest struct {
       Features []FeatureConfig `json:"features"`
   }
   
   type FeatureConfig struct {
       Name      string   `json:"name"`
       Strategy  string   `json:"strategy,omitempty"`
       OnCommand []string `json:"onCommand,omitempty"`
       Disabled  bool     `json:"disabled,omitempty"`
   }
   
   type FeatureMetadata struct {
       Name            string
       Description     string
       DefaultStrategy string
       DefaultCommands []string
       SupportedShells []string
       Category        string
   }
   
   type ShellConfig struct {
       Name        string  // "bash", "zsh", "fish", etc.
       ProfilePath string  // Path to profile file
       InitScript  string  // Path to init script template
       Extension   string  // File extension for features
   }
   ```

3. **Implement basic manifest parsing:**
   - JSON validation and error handling
   - Feature name regex validation
   - Strategy validation (eager/defer/on-command)
   - Unknown field tolerance

4. **Set up testing infrastructure:**
   - Test fixtures for manifests
   - Golden file testing for hooks
   - Mock file system utilities

**Deliverables:**
- ✅ Package structure exists
- ✅ Core types defined
- ✅ Basic manifest parsing works
- ✅ Unit tests pass

**Dependencies:** None

---

## Phase 1: Basic Shell Detection & Directory Structure

**Goal:** Implement shell detection and create the `omd-shells/` directory structure.

**Duration Estimate:** 1 week

### Tasks:

1. **Implement shell detection:**
   - Detect current shell from environment variables (`$SHELL`, `$0`)
   - Resolve default profile paths for each shell
   - Add `--shell` flag support to override detection
   - Handle edge cases (unknown shells, missing profiles)

2. **Create `omdot shell init` command:**
   - Create `cmd/shell_init.go`
   - Generate directory structure:
     ```
     omd-shells/
       lib/helpers.sh
       <shell-name>/
         init.<ext>
         enabled.json (empty features array)
         features/
     ```
   - Support `--shell` flag (or all shells if not specified)
   - Support `--repo` flag for custom repo path
   - Make operation idempotent

3. **Add shell configuration to viper config:**
   - Store repo root path
   - Store enabled shells
   - Default to `$HOME/dotfiles`

4. **Write tests:**
   - Shell detection unit tests
   - Directory creation tests
   - Idempotency tests

**Deliverables:**
- ✅ `omdot shell init` creates proper directory structure
- ✅ Shell detection works reliably
- ✅ Configuration is stored correctly
- ✅ Tests pass

**Dependencies:** Phase 0

**Decision Required:** Shell detection policy - auto-detect vs. require explicit shell selection

---

## Phase 2: Hook Management (Critical Path)

**Goal:** Implement hook insertion and removal in shell profile files.

**Duration Estimate:** 2 weeks

### Tasks:

1. **Implement hook marker logic:**
   - Define marker format: `# >>> oh-my-dot shell >>>`
   - Detect existing hooks
   - Extract hook content between markers

2. **Implement hook insertion:**
   - Generate shell-specific hook content
   - Find insertion point (end of file)
   - Check for existing hooks (idempotency)
   - Handle file creation if profile doesn't exist
   - Preserve file permissions

3. **Implement hook removal:**
   - Find marker blocks
   - Remove exactly the marked content
   - Leave rest of file intact

4. **Create `omdot shell apply` command:**
   - Insert hooks into appropriate profile files
   - Support `--shell` flag
   - Dry-run mode (`--dry-run`)
   - Backup existing files before modification

5. **Create `omdot shell unapply` command:**
   - Remove hook blocks
   - Support `--shell` flag
   - Leave `omd-shells/` directory intact

6. **Handle bash login shim:**
   - Detect if `.bash_profile` sources `.bashrc`
   - Add shim only if needed
   - Use separate markers for login shim

7. **Write comprehensive tests:**
   - Golden file tests for each shell's hook format
   - Idempotency tests (apply twice = same result)
   - Removal tests (apply + unapply = clean)
   - Edge cases (missing files, corrupted hooks)

**Deliverables:**
- ✅ `omdot shell apply` adds hooks correctly
- ✅ `omdot shell unapply` removes hooks cleanly
- ✅ Operations are idempotent
- ✅ All shells supported (bash, zsh, fish, PowerShell, POSIX)
- ✅ Tests pass with golden files

**Dependencies:** Phase 1

---

## Phase 3: Basic Feature Management (Eager Loading Only)

**Goal:** Implement feature add/remove/enable/disable with eager loading strategy only.

**Duration Estimate:** 2 weeks

### Tasks:

1. **Implement feature file operations:**
   - Create template feature files
   - Delete feature files with confirmation
   - Validate feature IDs against regex

2. **Implement manifest operations:**
   - Add features to `enabled.json`
   - Remove features from `enabled.json`
   - Enable/disable features (set `disabled` flag)
   - Preserve array ordering
   - Pretty-print JSON with 2-space indent

3. **Create CLI commands:**
   - `omdot shell feature add <feature> [--shell <name>]`
   - `omdot shell feature remove <feature> [--shell <name>] [--force]`
   - `omdot shell feature enable <feature> [--shell <name>]`
   - `omdot shell feature disable <feature> [--shell <name>]`
   - `omdot shell feature list [--shell <name>]`

4. **Create init script templates:**
   - POSIX init.sh with guard variable
   - bash init.sh with guard variable
   - zsh init.zsh with guard variable
   - fish init.fish with guard variable
   - PowerShell init.ps1 with guard variable
   - Implement eager loading only (defer/on-command come later)

5. **Add basic feature catalog:**
   - Embed 3-5 example features in binary
   - Implement catalog lookup
   - Default to "eager" for unknown features

6. **Write tests:**
   - Feature add/remove tests
   - Manifest JSON operations tests
   - Init script parsing tests
   - Catalog lookup tests

**Deliverables:**
- ✅ Users can add, remove, enable, disable features
- ✅ Init scripts load eager features correctly
- ✅ `omdot shell feature list` shows feature status
- ✅ Basic catalog works
- ✅ Tests pass

**Dependencies:** Phase 2

---

## Phase 4: Feature Catalog & CLI Integration

**Goal:** Complete the feature catalog system and improve CLI UX.

**Duration Estimate:** 1 week

### Tasks:

1. **Expand feature catalog:**
   - Add 10-15 common features with metadata
   - Include default strategies for each
   - Add descriptions and categories
   - Document supported shells per feature

2. **Implement `omdot shell feature info <feature>`:**
   - Show catalog metadata
   - Show current user configuration (if enabled)
   - Display default vs. overridden strategy
   - Show trigger commands for on-command features

3. **Enhance `omdot shell feature list`:**
   - Show strategy (eager/defer/on-command)
   - Indicate catalog default vs. user override
   - Show trigger commands
   - Mark local-only features
   - Add color coding for status

4. **Add strategy override flags:**
   - `--strategy <eager|defer|on-command>` for add/enable commands
   - `--on-command <cmd1,cmd2>` for on-command features
   - Validate strategy values
   - Validate command names

5. **Write tests:**
   - Catalog query tests
   - Info command output tests
   - Strategy override tests

**Deliverables:**
- ✅ Rich feature catalog embedded in binary
- ✅ `omdot shell feature info` provides detailed information
- ✅ Users can override load strategies
- ✅ Enhanced list command output
- ✅ Tests pass

**Dependencies:** Phase 3

---

## Phase 5: Advanced Load Strategies (Defer & On-Command)

**Goal:** Implement deferred and on-command loading strategies.

**Duration Estimate:** 2-3 weeks

### Tasks:

1. **Implement defer loading in init scripts:**
   - bash: Background sourcing after init
   - zsh: Use `&!` or precmd hook
   - fish: Use `--on-event fish_prompt`
   - PowerShell: Use `Register-EngineEvent` or background job
   - Add interactive shell detection
   - Fallback to eager in non-interactive shells

2. **Implement on-command loading in init scripts:**
   - bash: Function wrappers with `unset -f`
   - zsh: Function wrappers with `unfunction`
   - fish: Function wrappers with `functions -e`
   - PowerShell: Function wrappers with `Remove-Item Function:`
   - Handle missing commands gracefully
   - Support multiple trigger commands per feature

3. **Update manifest handling:**
   - Parse `onCommand` array
   - Validate trigger command names
   - Generate appropriate wrapper code

4. **Add strategy validation:**
   - Require `onCommand` for on-command strategy
   - Warn if on-command used without triggers

5. **Write comprehensive tests:**
   - Defer loading tests (per shell)
   - On-command wrapper tests (per shell)
   - Strategy validation tests
   - Interactive vs. non-interactive behavior tests

**Deliverables:**
- ✅ Defer strategy works in all shells
- ✅ On-command strategy works in all shells
- ✅ Shell startup performance is optimized
- ✅ Tests verify lazy loading behavior
- ✅ Documentation includes load strategy examples

**Dependencies:** Phase 4

---

## Phase 6: Local Overrides & Security

**Goal:** Implement `enabled.local.json` with security validation.

**Duration Estimate:** 1-2 weeks

### Tasks:

1. **Implement manifest merging:**
   - Parse `enabled.local.json` if it exists
   - Merge with `enabled.json` (override strategy, add features, set disabled)
   - Preserve array order from tracked manifest
   - Append local-only features

2. **Implement security validation (Unix):**
   - Check file is regular file (not symlink)
   - Verify ownership matches current user
   - Check permissions (not group/world writable)
   - Warn and skip if validation fails

3. **Implement Windows ACL checking (optional):**
   - Check file is not writable by Everyone
   - Check file is not writable by other users
   - Fall back to warning if ACL checks unavailable

4. **Update init scripts:**
   - Add local override reading logic
   - Add security validation before parsing
   - Add warning messages for security failures
   - Ensure no shell crashes on validation failure

5. **Update `list` command:**
   - Show which features are from local overrides
   - Indicate which properties are overridden

6. **Write tests:**
   - Manifest merging tests
   - Security validation tests (Unix)
   - Permission checking tests
   - Merge ordering tests

**Deliverables:**
- ✅ Local overrides work securely
- ✅ Permission validation prevents unsafe files
- ✅ Merge semantics are correct
- ✅ Tests cover security scenarios

**Dependencies:** Phase 5

---

## Phase 7: Shell Doctor & Diagnostics

**Goal:** Implement health checking and diagnostics.

**Duration Estimate:** 1 week

### Tasks:

1. **Create `omdot shell doctor` command:**
   - Check if hooks are present in profiles
   - Verify `omd-shells/` directory structure exists
   - Validate `enabled.json` format
   - Check local override permissions
   - Verify feature files exist for enabled features
   - Test if init scripts are sourceable (basic syntax check)

2. **Add diagnostic output:**
   - Color-coded status (✓ green, ✗ red, ⚠ yellow)
   - Clear error messages with suggestions
   - Show which features have missing files
   - Report permission issues on local overrides

3. **Add repair suggestions:**
   - Suggest `omdot shell apply` if hooks missing
   - Suggest `omdot shell init` if structure missing
   - Suggest fixing permissions on local overrides

4. **Write tests:**
   - Doctor command tests for various states
   - Missing file detection tests
   - Validation tests

**Deliverables:**
- ✅ `omdot shell doctor` diagnoses common issues
- ✅ Clear, actionable output
- ✅ Tests cover diagnostic scenarios

**Dependencies:** Phase 6

---

## Phase 8: Polish & Documentation

**Goal:** Final polish, comprehensive documentation, and user testing.

**Duration Estimate:** 1-2 weeks

### Tasks:

1. **Improve error messages:**
   - Consistent formatting
   - Actionable suggestions
   - Context-aware help

2. **Add shell completion:**
   - Cobra shell completion for bash/zsh/fish/PowerShell
   - Complete feature names from catalog
   - Complete shell names

3. **Write comprehensive documentation:**
   - User guide with examples
   - Feature catalog reference
   - Load strategy guide
   - Troubleshooting guide
   - Migration guide from other tools

4. **Create example features:**
   - git-prompt (defer)
   - kubectl-completion (on-command)
   - core-aliases (eager)
   - nvm integration (on-command)
   - docker-completion (on-command)

5. **Perform user testing:**
   - Test on fresh systems
   - Test with existing dotfiles
   - Test all shells on their native platforms
   - Gather feedback and iterate

6. **Write integration tests:**
   - End-to-end workflow tests
   - Multi-shell scenarios
   - Performance benchmarks for startup time

**Deliverables:**
- ✅ Polished user experience
- ✅ Comprehensive documentation
- ✅ Example features available
- ✅ User-tested and refined
- ✅ Ready for release

**Dependencies:** Phase 7

---

## Summary Timeline

| Phase | Duration | Cumulative |
|-------|----------|------------|
| Phase 0: Foundation | 1-2 weeks | 2 weeks |
| Phase 1: Shell Detection | 1 week | 3 weeks |
| Phase 2: Hook Management | 2 weeks | 5 weeks |
| Phase 3: Basic Features | 2 weeks | 7 weeks |
| Phase 4: Feature Catalog | 1 week | 8 weeks |
| Phase 5: Load Strategies | 2-3 weeks | 11 weeks |
| Phase 6: Local Overrides | 1-2 weeks | 13 weeks |
| Phase 7: Shell Doctor | 1 week | 14 weeks |
| Phase 8: Polish & Docs | 1-2 weeks | 16 weeks |

**Total Estimated Time:** 14-16 weeks (~4 months)

---

## Critical Path

The critical path for minimum viable product (MVP):
1. Phase 0 → Phase 1 → Phase 2 → Phase 3

This gives users basic feature management with eager loading, which is already highly useful.

Phases 4-8 add polish, optimization, and advanced features but are not blockers for initial release.

---

## Risk Mitigation

**Risk:** Shell-specific behavior differences
- **Mitigation:** Test each shell independently, golden file tests, manual verification

**Risk:** Breaking existing user shells
- **Mitigation:** Robust guard variables, safe failure modes, comprehensive testing

**Risk:** Security vulnerabilities in local overrides
- **Mitigation:** Strict validation, permission checking, clear warnings

**Risk:** Performance regression in shell startup
- **Mitigation:** Lazy loading strategies, benchmarking, profiling

**Risk:** Cross-platform compatibility issues
- **Mitigation:** Test on Windows, macOS, Linux; use portable patterns

---

## Open Questions for Phase 1

Before starting Phase 1, we need to resolve:

1. **Shell detection policy:** Should `omdot shell apply` auto-detect the current shell, or require explicit `--shell` flag?
   - **Option A:** Auto-detect by default, allow override with `--shell`
   - **Option B:** Require explicit `--shell` for safety
   - **Recommendation:** Option A for better UX, with clear confirmation messages

2. **Multi-shell initialization:** Should `omdot shell init` set up all shells by default, or just the current one?
   - **Option A:** Initialize all supported shells
   - **Option B:** Initialize only current shell (require `--all` for all shells)
   - **Recommendation:** Option B to avoid clutter, with `--all` flag

3. **Repo path detection:** If user already has dotfiles managed by oh-my-dot, should we auto-detect repo path?
   - **Answer:** Yes, use existing config from `~/.oh-my-dot/config.json`