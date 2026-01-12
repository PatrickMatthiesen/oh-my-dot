# Shell Framework Specification

This directory contains the complete specification for oh-my-dot's integrated shell framework feature.

## Overview

The shell framework provides seamless shell profile integration with modular feature management. Users can add shell features (aliases, prompts, completions) that are automatically applied alongside their dotfiles.

## Key Features

- **Unified workflow** - Single `omdot apply` command for dotfiles and shell hooks
- **Smart defaults** - Intelligent shell selection based on feature compatibility
- **Auto-initialization** - Shell directories created on-demand when adding features
- **Auto-cleanup** - Hooks removed automatically when last feature is removed
- **Multi-shell support** - bash, zsh, fish, PowerShell, POSIX sh
- **Lazy loading** - Eager, defer, and on-command load strategies
- **Local experimentation** - Optional gitignored overrides with security validation

## Documents

Read in this order:

### 1. **00-quick-reference.md** - Start Here
Quick reference guide with 30-second overview, CLI commands, and smart behaviors.

### 2. **01-design.md** - Primary Specification
Complete integrated design including:
- Updated CLI design
- Workflow examples
- Updated implementation phases
- Key behavior changes
- File structure

### 3. **02-decisions.md** - All Decisions Finalized
Records all design decisions including:
- Git commit behavior (stage, don't auto-commit)
- Git push integration (yes, unified)
- Non-interactive mode handling (require --shell flag)
- Complete CLI reference

### 4. **03-implementation-roadmap.md** - Detailed Phases
Original 8-phase implementation roadmap:
- Phase breakdown with dependencies
- Duration estimates (14-16 weeks total)
- Critical path for MVP (7 weeks)
- Risk mitigation strategies

### 5. **04-code-examples.md** - Implementation Templates
Reusable code patterns including:
- Hook insertion patterns for all shells
- Init script templates
- Load strategy implementations
- Security validation examples
- Feature template examples

### 6. **05-code-validation.md** - Go Code Verification
Validation of all Go code examples in the spec:
- Data structure definitions
- Type validations
- Test examples

## Quick Start

### For Implementers

1. Read `00-quick-reference.md` for overview
2. Read `01-design.md` for complete specification
3. Reference `04-code-examples.md` while coding
4. Follow phases in `03-implementation-roadmap.md`

### For Reviewers

1. Read `00-quick-reference.md` for context
2. Review `01-design.md` for design decisions
3. Check `02-decisions.md` for rationale

## Timeline

- **MVP**: 8 weeks (Phases 0-4) - Basic feature management with eager loading
- **Full feature set**: 14-17 weeks - Includes defer/on-command loading, security validation

## Status

**Current Phase**: 📝 Specification Complete - Ready for Implementation

All design decisions finalized and documented. Ready to begin Phase 0 (Foundation).

## New CLI Commands

```bash
# Core commands (modified)
omdot apply                   # Apply dotfiles + shell hooks (unified)
omdot apply --no-shell        # Skip shell hooks
omdot push                    # Commit + push everything
omdot doctor                  # Diagnose dotfiles + shell integration

# Feature commands (new)
omdot feature add -i                    # Browse catalog interactively
omdot feature add <feature>         # Add with smart shell selection
omdot feature add <feature> --shell bash
omdot feature add <feature> --all
omdot feature remove <feature>      # Remove with auto-cleanup
omdot feature enable <feature>
omdot feature disable <feature>
omdot feature list [--shell <name>] # List with file paths
omdot feature info <feature>
```

## Example Workflow

```bash
# First-time setup
omdot init github.com/user/dotfiles
omdot add ~/.gitconfig ~/.vimrc
omdot feature add -i               # Browse and select features interactively
omdot apply                         # Applies everything
omdot push

# Daily usage
omdot feature add kubectl-completion
omdot apply
omdot feature list                  # See features with file paths
omdot doctor
```

## Architecture

### File Structure

```
~/dotfiles/
├── .gitconfig, .vimrc, etc.  # Dotfiles
└── omd-shells/                # Shell framework (created on-demand)
    ├── lib/helpers.sh
    ├── bash/
    │   ├── init.sh
    │   ├── enabled.json
    │   └── features/
    │       ├── core-aliases.sh
    │       └── git-prompt.sh
    └── zsh/
        ├── init.zsh
        ├── enabled.json
        └── features/
            └── git-prompt.zsh
```

### Key Design Principles

1. **Zero cognitive overhead** - One apply command for everything
2. **Progressive disclosure** - Simple cases simple, complex cases possible  
3. **Auto-maintenance** - Initialization and cleanup handled automatically
4. **Smart defaults** - Works without flags most of the time
5. **Security first** - Strict validation for local overrides

## Related Documentation

- Migration notes: `MIGRATION.md` (explains reorganization from original spec)
- Main project README: `../../../README.md`

## Questions?

All design decisions are documented in `02-decisions.md`. For implementation details, see `01-design.md` and `04-code-examples.md`.