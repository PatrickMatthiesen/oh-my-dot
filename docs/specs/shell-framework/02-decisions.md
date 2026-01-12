# Final Decisions: Integrated Shell Framework

## Overview

This document records the final decisions made for the integrated shell framework design, resolving all remaining open questions.

---

## Confirmed Design Decisions

### 1. ✅ Git Commit Behavior

**Question:** Should `omdot feature add` auto-commit to git, or wait for user to commit manually?

**Decision:** Wait for user to commit manually

**Rationale:**
- Consistent with existing oh-my-dot behavior
- Users may want to add multiple features before committing
- Gives users control over commit granularity
- Follows existing pattern where `omdot push` handles commits

**Implementation:**
```bash
$ omdot feature add git-prompt
Adding git-prompt to bash... ✓
Created: omd-shells/bash/features/git-prompt.sh
Updated: omd-shells/bash/enabled.json

Changes staged for commit. Run 'omdot push' when ready.

$ omdot feature add kubectl
Adding kubectl to bash... ✓
Created: omd-shells/bash/features/kubectl.sh
Updated: omd-shells/bash/enabled.json

$ omdot push
Committing changes...
  - Added 2 shell features (git-prompt, kubectl)
  - Modified enabled.json
Pushing to remote... ✓
```

---

### 2. ✅ Git Push Integration

**Question:** Should shell framework changes auto-push with `omdot push`, or require separate git operations?

**Decision:** Yes, integrate with existing `omdot push`

**Rationale:**
- Shell features are part of the dotfiles configuration
- Should be treated the same as regular dotfiles
- Users expect `omdot push` to sync everything
- Simplifies workflow - one command to sync all changes

**Implementation:**
- `omdot push` detects changes in both dotfiles and `omd-shells/` directory
- Generates appropriate commit message mentioning shell features
- Single push operation for all changes

**Example commit messages:**
```
Added shell features: git-prompt, kubectl (bash)

Added .gitconfig and .vimrc
Updated shell features: enabled git-prompt (zsh)

Removed shell feature: docker-completion (bash)
```

---

### 3. ✅ Non-Interactive Environment Behavior

**Question:** Should non-interactive shells (CI, scripts) skip feature selection prompts or error?

**Decision:** Require explicit `--shell` flag in non-interactive environments

**Rationale:**
- Prevents hanging in CI/CD pipelines
- Clear error message guides user to correct usage
- Explicit is better than implicit in automated contexts
- Matches best practices for CLI tools in scripts

**Implementation:**

**Interactive shell (terminal):**
```bash
$ omdot feature add git-prompt
? Select shells to add feature to:
  ◉ bash (current)
  ◯ zsh
```

**Non-interactive shell (script/CI):**
```bash
$ omdot feature add git-prompt
Error: Cannot prompt for shell selection in non-interactive mode
Please specify target shell(s):
  --shell bash          Add to specific shell
  --all                 Add to all supported shells
  
Example: omdot feature add git-prompt --shell bash
```

**Script-friendly usage:**
```bash
#!/bin/bash
# CI/CD script
omdot feature add core-aliases --shell bash
omdot feature add git-prompt --shell bash --shell zsh
omdot feature add kubectl --all
omdot apply
```

**Detection logic:**
```go
func isInteractive() bool {
    // Check if stdin is a terminal
    return term.IsTerminal(int(os.Stdin.Fd()))
}

func selectShells(feature FeatureMetadata) ([]string, error) {
    if !isInteractive() {
        return nil, fmt.Errorf("cannot prompt in non-interactive mode, use --shell flag")
    }
    // Show interactive prompt
    return showShellSelector(feature)
}
```

---

## Summary of All Decisions

### Core Design Decisions (from earlier planning)

| Decision | Choice | Source Document |
|----------|--------|----------------|
| CLI structure | Flattened (`omdot feature` not `omdot shell feature`) | User requirement |
| Apply behavior | Unified command for dotfiles + shell hooks | User requirement |
| Auto-initialization | On-demand when adding first feature | User requirement |
| Auto-cleanup | Remove hooks when last feature removed | User requirement |
| Shell selection | Smart defaults based on feature compatibility | User requirement |
| File extensions | Shell-specific (`.sh`, `.zsh`, `.fish`, `.ps1`) | From design analysis |
| Missing features | Warn and continue (don't crash shell) | From design analysis |
| Local overrides | Enabled with strict security validation | From design analysis |

### Git Integration Decisions (confirmed today)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Auto-commit on feature add | ❌ No | User control over commit granularity |
| Push integration | ✅ Yes | Simplify workflow, one push for everything |
| Non-interactive mode | Require `--shell` flag | Prevent hanging, explicit is better |

---

## Implementation Notes

### Phase 2 Updates (Feature Management Core)

Add these behaviors:

1. **Staging changes, not committing:**
   - `omdot feature add` stages files with git
   - Displays message: "Changes staged for commit. Run 'omdot push' when ready."
   - Allow multiple feature operations before commit

2. **Push command integration:**
   - Modify existing `omdot push` to detect shell framework changes
   - Generate descriptive commit messages that include feature changes
   - Single atomic commit for both dotfiles and shell features

3. **Non-interactive detection:**
   - Add `isInteractive()` helper function
   - Check before showing any prompts
   - Return clear error with usage examples when `--shell` required

### Testing Requirements

Add tests for:
- ✅ Non-interactive mode detection
- ✅ Error messages in non-interactive mode
- ✅ `--shell` flag behavior in scripts
- ✅ Git staging (not committing) on feature add
- ✅ Push integration with shell framework changes
- ✅ Commit message generation with feature mentions

---

## Updated CLI Reference

### Complete Command List

```bash
# Core commands (modified)
omdot init <remote>                 # Initialize dotfiles repo
omdot apply                         # Apply dotfiles + shell hooks
omdot apply --no-shell              # Apply only dotfiles
omdot push                          # Commit and push (dotfiles + features)
omdot doctor                        # Diagnose dotfiles + shell integration

# Feature commands (new)
omdot feature add -i                        # Browse catalog interactively
omdot feature add <feature>         # Add feature (interactive or --shell)
omdot feature add <feature> --shell <name>  # Add to specific shell
omdot feature add <feature> --all   # Add to all supported shells
omdot feature remove <feature>      # Remove feature (with cleanup)
omdot feature enable <feature>      # Enable disabled feature
omdot feature disable <feature>     # Disable without removing
omdot feature list                  # List all features with paths
omdot feature list --shell <name>   # List features for specific shell
omdot feature info <feature>        # Show feature details

# Existing commands (unchanged)
omdot add <file>                    # Add dotfile
omdot remove <file>                 # Remove dotfile
omdot list                          # List dotfiles
omdot config                        # Show configuration
omdot version                       # Show version
omdot update                        # Self-update
```

---

## Ready for Implementation

All decisions have been finalized:
- ✅ Core design decisions resolved
- ✅ CLI structure defined
- ✅ Git integration behavior confirmed
- ✅ Non-interactive mode handling specified
- ✅ Testing requirements identified

**Next step:** Begin implementation following the phased roadmap in `03-implementation-roadmap.md`