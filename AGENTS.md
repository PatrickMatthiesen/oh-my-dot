# AGENTS.md - Developer Guide for AI Coding Agents

This file provides coding guidelines and commands for AI agents working on the oh-my-dot codebase.

## Project Overview

**oh-my-dot** is a cross-platform dotfile manager written in Go that uses git for version control. It includes a shell framework feature for managing shell configurations (aliases, prompts, completions) across bash, zsh, fish, PowerShell, and POSIX sh.

- **Language**: Go 1.25.0
- **CLI Framework**: Cobra + Viper
- **Main Binary**: `oh-my-dot`
- **Module**: `github.com/PatrickMatthiesen/oh-my-dot`

## Build, Test, and Lint Commands

### Build

```bash
# Development build with version info (recommended)
# puts binary in ./build/oh-my-dot and has a canary version
# on linux the shell path needs to be updated when testing dev builds
bun run build.ts

# Development build to custom directory
bun run build.ts --out=./custom-dir/

# Manual build (quick, but missing Version/CommitHash injection)
go build -o build/oh-my-dot

# Production release build (done automatically by GoReleaser in CI)
# GoReleaser ldflags include:
#   -s -w                        (strip symbol table + DWARF debug info; smaller binaries)
#   -X .../cmd.Version=...       (set Version)
#   -X .../cmd.CommitHash=...    (set CommitHash)
```

### Test

Remember to run tests for new features or bug fixes.

### Lint and Format

```bash
# Format code (Note: Some files may not be formatted yet)
gofmt -w .

# Check formatting without writing
gofmt -l .

# Run go vet
go vet ./...

# Tidy dependencies
go mod tidy
```

## Code Style Guidelines

### Package Organization

```filetree
oh-my-dot/
├── cmd/              # Cobra commands (feature.go, apply.go, etc.)
├── internal/         # Internal packages
│   ├── catalog/      # Feature catalog metadata
│   ├── fileops/      # File system operations and colored output
│   ├── git/          # Git operations
│   ├── hooks/        # Shell profile hook management
│   ├── interactive/  # Interactive prompts (bubbletea)
│   ├── manifest/     # JSON manifest parsing/validation
│   ├── shell/        # Shell detection and operations
│   └── symlink/      # Symlink management
├── tests/            # Integration tests
└── docs/             # Documentation and specs
```

### Import Conventions

**Order imports in three groups (separated by blank lines):**

1. Standard library imports
2. Third-party imports
3. Local project imports

```go
package shell

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/spf13/cobra"
    "github.com/spf13/viper"

    "github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
    "github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
)
```

### Naming Conventions

- **Files**: Use snake_case for file names (e.g., `manifest_test.go`)
- **Packages**: Use lowercase, single-word names (e.g., `package shell`)
- **Exported Types**: Use PascalCase (e.g., `FeatureMetadata`, `ShellConfig`)
- **Unexported Types**: Use camelCase (e.g., `featureOption`)
- **Functions**: Use PascalCase for exported, camelCase for unexported
- **Constants**: Use PascalCase or UPPER_SNAKE_CASE for exported constants
- **Variables**: Use camelCase for locals, PascalCase for package-level exported

### Type Definitions

Always add comments for exported types with struct field descriptions:

```go
// FeatureConfig represents a single feature configuration in enabled.json
type FeatureConfig struct {
    Name      string   `json:"name"`
    Strategy  string   `json:"strategy,omitempty"`  // "eager", "defer", or "on-command"
    OnCommand []string `json:"onCommand,omitempty"` // Commands that trigger on-command loading
    Disabled  bool     `json:"disabled,omitempty"`  // If true, feature is disabled
}
```

### Error Handling

**Always wrap errors with context using `fmt.Errorf` with `%w`:**

```go
// Good
if err := manifest.ParseManifest(path); err != nil {
    return fmt.Errorf("failed to parse manifest: %w", err)
}

// Bad
if err != nil {
    return err
}
```

**For user-facing errors in commands, use colored output:**

```go
if err != nil {
    fileops.ColorPrintfn(fileops.Red, "Error: %v", err)
    os.Exit(1)
}
```

### Function Documentation

Document all exported functions with comments:

```go
// AddFeatureToShell adds a feature to a specific shell
// It creates the shell directory if needed, updates the manifest, generates
// the feature file template, and regenerates the init script.
func AddFeatureToShell(repoPath, shellName, featureName string, strategy string, onCommand []string, disabled bool) error {
    // Implementation
}
```

### Testing Patterns

Use table-driven tests with subtests:

```go
func TestValidateFeatureName(t *testing.T) {
    tests := []struct {
        name      string
        input     string
        wantError bool
    }{
        {"valid simple name", "git-prompt", false},
        {"empty name", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateFeatureName(tt.input)
            if (err != nil) != tt.wantError {
                t.Errorf("ValidateFeatureName(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
            }
        })
    }
}
```

## Important Implementation Notes

### Shell Framework Architecture

The shell framework consists of:

- **Manifest** (`enabled.json`): JSON config with features and strategies
- **Init Scripts**: Generated per-shell scripts that load features
- **Feature Files**: User-editable shell scripts in `omd-shells/<shell>/features/`
- **Hooks**: Integration points in shell profile files

**Init scripts are auto-generated** - regenerate them after any manifest changes:

```go
// Always regenerate after modifying manifest
if err := RegenerateInitScript(repoPath, shellName); err != nil {
    return fmt.Errorf("failed to regenerate init script: %w", err)
}
```

### Loading Strategies

- **eager**: Source immediately during shell startup (Phase 4 - IMPLEMENTED)
- **defer**: Background load for interactive shells (Phase 5 - TODO)
- **on-command**: Lazy-load when command is invoked (Phase 5 - TODO)

### Init Script Re-sourcing Behavior

When `.bashrc` is re-sourced, the init script guard allows **eager features to re-run** to handle cases where `.bashrc` resets environment variables like `PS1`. This is important for features that modify the prompt or environment.

**Eager features that don't need re-running must include their own guard:**

```bash
# Example: Expensive or state-dependent eager feature
if [ "${OMD_FEATURE_MYFEATURE_LOADED:-}" = "1" ]; then
  return 0
fi
export OMD_FEATURE_MYFEATURE_LOADED=1

# Rest of feature implementation...
```

**Use feature-level guards for:**

- Expensive operations (network calls, heavy computation)
- State-dependent setup (starting daemons, checking system state)
- One-time initialization that shouldn't repeat

**Don't use feature-level guards for:**

- Prompt modifications (PS1, RPROMPT) - need to re-apply on `.bashrc` re-source
- Environment variable modifications (PATH, EDITOR) - should be re-applied
- Function/alias definitions - harmless to redefine, but guard is optional

### Key Files to Understand

- `internal/manifest/manifest.go` - Feature manifest parsing/validation
- `internal/shell/initgen.go` - Init script generation (Phase 4)
- `internal/shell/operations.go` - Shell operations (add/remove/enable/disable features)
- `internal/hooks/hooks.go` - Profile file hook insertion
- `internal/catalog/catalog.go` - Pre-defined feature metadata
- `cmd/feature.go` - Feature management commands
- `cmd/apply.go` - Apply dotfiles and shell hooks

## Common Tasks

### Adding a New Feature to Catalog

Edit `internal/catalog/catalog.go` and add to the `Catalog` map.
Then create template files in `internal/catalog/features/<feature>/` for each shell.

### Creating a New Command

Add a new file in `cmd/` following the pattern of existing commands.

### Regenerating Init Scripts

Init scripts are automatically regenerated when features are added/removed/enabled/disabled via `shell.RegenerateInitScript()`.

## CI/CD

- **GitHub Actions**: `.github/workflows/test.yml` runs tests on Ubuntu and Windows
- **Release**: `.goreleaser.yml` builds cross-platform binaries
- Tests must pass on both Ubuntu and Windows

## Documentation

Specs are in `docs/specs/<feature>/`. Key shell framework spec is `docs/specs/shell-framework/README.md`.

Always refer to specs when implementing new shell framework features.

## Feature template

When creating new shell features don't add shebangs (e.g. `#!/bin/bash`) at the top of feature files. The init scripts handle shell detection and sourcing appropriately.
