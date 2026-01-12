# Quick Reference: Shell Framework Implementation

## 📋 Documentation Index

Read in this order:

1. **`README.md`** - Overview and feature introduction
2. **`01-design.md`** ⭐ - **PRIMARY SPEC** - Complete integrated design
3. **`02-decisions.md`** - All decisions finalized and confirmed
4. **`04-code-examples.md`** - Code patterns and templates for implementation
5. **`03-implementation-roadmap.md`** - Detailed 8-phase roadmap

Supporting documents:
- `05-code-validation.md` - Go code validation results
- `MIGRATION.md` - Documentation reorganization summary

---

## 🎯 The New Design in 30 Seconds

**Single command workflow:**
```bash
omdot feature add git-prompt    # Auto-creates shell structure
omdot apply                     # Applies dotfiles + shell hooks
omdot push                      # Commits and pushes everything
```

**Key features:**
- ✅ Unified `omdot apply` (handles dotfiles + shell hooks)
- ✅ Flattened CLI (`omdot feature` not `omdot shell feature`)
- ✅ Auto-initialization (creates shell dirs on-demand)
- ✅ Auto-cleanup (removes hooks when last feature removed)
- ✅ Smart defaults (intelligently selects shell)

---

## 📊 Implementation Roadmap

**Timeline:** 14-17 weeks (~4 months)

**MVP:** Phases 0-4 (~8 weeks) = Basic feature management with eager loading

| Phase | Focus | Duration | Status |
|-------|-------|----------|--------|
| 0 | Foundation (packages, types, parsing) | 1-2 weeks | 📝 Planned |
| 1 | Feature Catalog (metadata system) | 1 week | 📝 Planned |
| 2 | Feature Management (add/remove/list) | 2 weeks | 📝 Planned |
| 3 | Hook Management (integrate into apply) | 2 weeks | 📝 Planned |
| 4 | Init Scripts (eager loading only) | 2 weeks | 📝 Planned |
| 5 | Load Strategies (defer, on-command) | 2-3 weeks | 📝 Planned |
| 6 | Local Overrides (security validation) | 1-2 weeks | 📝 Planned |
| 7 | Unified Doctor (diagnostics) | 1 week | 📝 Planned |
| 8 | Polish & Documentation | 1-2 weeks | 📝 Planned |

---

## 🛠️ New CLI Commands

### Core Commands (Modified)

```bash
omdot init <remote>           # Initialize dotfiles repo (unchanged)
omdot apply                   # Apply dotfiles + shell hooks (NEW: unified)
omdot apply --no-shell        # Skip shell hooks (NEW: flag)
omdot push                    # Commit + push everything (NEW: includes features)
omdot doctor                  # Diagnose everything (NEW: includes shell checks)
```

### Feature Commands (New)

```bash
omdot feature add -i                         # Browse and select from catalog
omdot feature add <feature>              # Add with smart shell selection
omdot feature add <feature> --shell bash # Add to specific shell
omdot feature add <feature> --all        # Add to all supported shells
omdot feature remove <feature>           # Remove with auto-cleanup
omdot feature enable <feature>           # Enable disabled feature
omdot feature disable <feature>          # Disable without removing
omdot feature list                       # List all features with file paths
omdot feature list --shell bash          # List features for specific shell
omdot feature info <feature>             # Show feature details
```

### Removed Commands

```bash
# These no longer exist:
omdot shell init              # Auto-init on feature add
omdot shell apply             # Integrated into apply
omdot shell unapply           # Auto-cleanup on feature remove
omdot shell feature *         # Flattened to omdot feature *
omdot shell doctor            # Integrated into doctor
```

---

## 💡 Smart Behaviors

### Shell Selection Logic

When running `omdot feature add <feature>`:

1. **Feature supports only current shell** → Add automatically
   ```bash
   $ omdot feature add bash-specific-thing
   Adding bash-specific-thing to bash... ✓
   ```

2. **Feature supports multiple shells** → Interactive prompt
   ```bash
   $ omdot feature add git-prompt
   ? Select shells to add feature to:
     ◉ bash (current)
     ◯ zsh
     ◯ fish
   ```

3. **Feature doesn't support current shell** → Interactive prompt with other options

4. **Non-interactive mode** → Require explicit `--shell` flag
   ```bash
   $ omdot feature add git-prompt  # In script/CI
   Error: Cannot prompt in non-interactive mode
   Use: omdot feature add git-prompt --shell bash
   ```

### Auto-Cleanup Logic

When running `omdot feature remove <feature>`:

1. Feature removed from `enabled.json`
2. Feature file deleted from `features/` directory
3. **If last feature in shell:**
   - Hook removed from shell profile (e.g., `~/.bashrc`)
   - Shell directory removed (`omd-shells/bash/`)
   - Changes staged for commit
4. User runs `omdot apply` to sync changes

### Auto-Initialization Logic

When running `omdot feature add <feature>` (first time for a shell):

1. Detects shell directory doesn't exist
2. Creates structure:
   ```
   omd-shells/bash/
     init.sh
     enabled.json
     features/
   ```
3. Adds feature to `enabled.json`
4. Creates feature file
5. Stages changes for commit
6. User runs `omdot apply` to activate

---

## 🎨 File Structure

```
~/dotfiles/
├── .git/
├── .gitconfig                    # Tracked dotfiles
├── .vimrc
└── omd-shells/                   # Created on first feature add
    ├── lib/
    │   └── helpers.sh
    ├── bash/                     # Created when bash feature added
    │   ├── init.sh
    │   ├── enabled.json
    │   ├── enabled.local.json    # Optional, gitignored
    │   └── features/
    │       ├── core-aliases.sh
    │       └── git-prompt.sh
    └── zsh/                      # Created when zsh feature added
        ├── init.zsh
        ├── enabled.json
        └── features/
            └── git-prompt.zsh
```

**Note:** Only shells with features exist in `omd-shells/`

---

## ✅ All Decisions Finalized

### User Requirements
- ✅ Integrated apply (no separate `shell apply`)
- ✅ Flattened CLI structure (`feature` not `shell feature`)
- ✅ Auto-init on feature add
- ✅ Auto-cleanup on last feature remove

### Git Integration
- ✅ Feature operations stage changes (don't auto-commit)
- ✅ `omdot push` includes shell framework changes
- ✅ Descriptive commit messages mention features

### Shell Selection
- ✅ Smart defaults based on feature compatibility
- ✅ Interactive prompts when multiple options
- ✅ Explicit `--shell` flag required in non-interactive mode

### Technical
- ✅ Shell-specific file extensions (`.sh`, `.zsh`, `.fish`, `.ps1`)
- ✅ Warn and continue on missing features (don't crash shell)
- ✅ Local overrides enabled with strict security validation

---

## 🚀 Quick Start for Implementation

1. **Read the design:**
   - Primary spec: `01-design.md`
   - All decisions: `02-decisions.md`

2. **Reference the roadmap:**
   - Implementation phases: `03-implementation-roadmap.md`

3. **Use code templates:**
   - Code examples: `04-code-examples.md`

4. **Start Phase 0:**
   - Create `internal/shell/`, `internal/catalog/`, `internal/hooks/` packages
   - Define data structures (FeatureMetadata, FeatureConfig, FeatureManifest)
   - Implement basic manifest parsing
   - Set up testing infrastructure

---

## 📞 Questions?

All major decisions are finalized. If you need clarification on any aspect:
- Check `01-design.md` first (most comprehensive)
- Check `02-decisions.md` for confirmed choices
- Check `04-code-examples.md` for code patterns

---

## 📦 Deliverables

When fully implemented, users will have:

✅ Seamless shell integration (one `apply` command)
✅ Feature catalog with 10-15 common features
✅ Smart shell selection
✅ Lazy loading strategies (eager, defer, on-command)
✅ Local experimentation support (enabled.local.json)
✅ Comprehensive diagnostics (`omdot doctor`)
✅ Multi-shell support (bash, zsh, fish, PowerShell, POSIX)
✅ Auto-cleanup and maintenance

**Result:** Professional, maintainable, user-friendly shell profile management integrated seamlessly into oh-my-dot!