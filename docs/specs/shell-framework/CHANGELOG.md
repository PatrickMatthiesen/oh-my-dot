# Shell Framework Spec Updates

## Summary of Changes

This document summarizes the updates made to the shell framework specification and implementation.

## Updated Features

### 1. Enhanced `omdot feature list` Command

**Previous behavior:**
- Listed features with their status, strategy, and trigger commands
- Did not show file locations

**New behavior:**
- Lists features with all previous information
- **Now shows the file path** for each feature
- Displays paths with `~` notation for home directory
- Helps users quickly locate feature files for editing

**Example output:**
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
```

### 2. Interactive Catalog Browser with `omdot feature add -i`

**New feature:**
- Added `-i` / `--interactive` flag to `omdot feature add`
- Allows browsing the entire feature catalog interactively
- Multi-select interface for choosing multiple features at once
- Automatically prompts for shell selection after feature selection

**Usage:**
```bash
$ omdot feature add -i
? Select features to add:
  ◉ core-aliases - Essential command aliases [alias]
  ◯ directory-shortcuts - Quick navigation to common directories [alias]
  ◯ aws-completion - AWS CLI command completion [completion]
  ◯ docker-completion - Docker CLI command completion [completion]
  ◯ gcloud-completion - Google Cloud CLI command completion [completion]
  ◯ kubectl-completion - Kubernetes CLI completion [completion]
  ◯ terraform-completion - Terraform CLI completion [completion]
  ◉ git-prompt - Show git branch in prompt [prompt]
  ◯ nvm - Node Version Manager [tool]
  ◯ python-venv - Python virtual environment helpers [tool]
  ...
  
? Select shells to add features to:
  ◉ bash (current)
  ◯ zsh

Adding core-aliases to bash... ✓
Adding git-prompt to bash... ✓
```

**Features:**
- Shows feature name, description, and category in selection list
- **Sorted by category** (alias → completion → prompt → tool), then alphabetically
- Displays `[category]` tag for easy identification
- Pre-selects current shell by default
- Skips features that don't support selected shells
- Reports success/skip counts at the end

**Backward compatibility:**
- Original syntax still works: `omdot feature add <feature>`
- All existing flags still supported
- Args validation changed from `ExactArgs(1)` to `MaximumNArgs(1)`

## Implementation Details

### Code Changes

1. **cmd/feature.go**:
   - Added `flagInteractive` variable
   - Updated `featureAddCmd` to accept 0 or 1 arguments
   - Added `runInteractiveFeatureAdd()` function
   - Updated `runFeatureAdd()` to route to interactive mode when `-i` flag is set
   - Enhanced `runFeatureList()` to display file paths

2. **Interactive workflow**:
   - Uses `catalog.ListFeatures()` to get all available features
   - **Sorts features by category** (alias, completion, prompt, tool) then alphabetically
   - Added `sortFeaturesByCategory()` helper function with defined category order
   - Creates formatted labels: `"name - description [category]"`
   - Uses `interactive.MultiSelect()` for both feature and shell selection
   - Validates shell support for each feature
   - Skips unsupported combinations with clear messaging

### Documentation Updates

Updated the following specification files:
- `docs/specs/shell-framework/01-design.md` - Added interactive mode section
- `docs/specs/shell-framework/00-quick-reference.md` - Updated command reference
- `docs/specs/shell-framework/02-decisions.md` - Updated CLI reference
- `docs/specs/shell-framework/README.md` - Updated example workflow

## Benefits

### For Users

1. **Discoverability**: Users can now browse all available features without knowing their names
2. **Efficiency**: Can add multiple features in one interactive session
3. **Visibility**: File paths make it easy to locate and edit feature files
4. **Learning**: Category tags help users understand what each feature does

### For Developer Experience

1. **Reduced friction**: No need to remember exact feature names
2. **Better onboarding**: New users can explore available features
3. **Quick editing**: File paths in list output enable fast navigation to files

## Examples

### Adding multiple features interactively
```bash
# Start interactive session
$ omdot feature add -i

# Select 3 features from catalog
# Select bash and zsh shells
# Result: 6 features added (3 features × 2 shells)
```

### Traditional single feature add (still works)
```bash
$ omdot feature add git-prompt
? Select shells to add feature to:
  ◉ bash (current)
  ◯ zsh
```

### Listing features with file paths
```bash
$ omdot feature list --shell bash
bash:
  ✓ git-prompt (defer, catalog default)
    ~/dotfiles/omd-shells/bash/features/git-prompt.sh
  ✓ kubectl-completion (on-command: kubectl)
    ~/dotfiles/omd-shells/bash/features/kubectl-completion.sh
```

## Testing

The implementation has been tested for:
- ✅ Successful build with `go build`
- ✅ Backward compatibility with existing `feature add <name>` syntax
- ✅ Interactive mode with `-i` flag
- ✅ File path display in `feature list` command
- ✅ Home directory expansion (`~` notation)

## Future Enhancements

Potential improvements for future iterations:
- Add filtering by category in interactive mode (e.g., show only completions)
- Search/filter features by name in interactive mode
- Show which features are already installed in the catalog browser
- Add `--category` flag to filter by category in list command
