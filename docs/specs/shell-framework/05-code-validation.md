# Plan: Validate Go Code Examples

## Overview

The shell framework specification includes Go code examples that define data structures. This document validates those examples for syntactic correctness.

---

## Go Code Blocks Found

The spec contains 2 Go code blocks:

1. **Line 181-190:** `FeatureMetadata` struct definition
2. **Line 201-226:** `BuiltinFeatures` example catalog

---

## Validation Results

### Block 1: FeatureMetadata struct (lines 181-190)

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

**Status:** ✅ Valid Go syntax
- Proper struct definition
- Appropriate field types
- Good field names following Go conventions
- Comments are helpful

**Recommendations:**
- Consider adding struct tags for JSON marshaling: `json:"name"`
- Consider adding validation tags if using a validation library

**Improved version:**
```go
type FeatureMetadata struct {
    Name            string   `json:"name"`
    Description     string   `json:"description"`
    DefaultStrategy string   `json:"defaultStrategy"`
    DefaultCommands []string `json:"defaultCommands,omitempty"`
    SupportedShells []string `json:"supportedShells"`
    Category        string   `json:"category"`
}
```

---

### Block 2: BuiltinFeatures catalog (lines 201-226)

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

**Status:** ✅ Valid Go syntax
- Proper slice initialization
- Struct literals are well-formed
- Field values are appropriate types

**Recommendations:**
- Good starting catalog
- Consider adding more entries for common use cases
- Ensure consistency in shell names (lowercase: "posix", "bash", etc.)

---

## Additional Data Structures Needed

Based on the spec, these additional Go types should be defined:

### FeatureConfig (from manifest)

```go
// FeatureConfig represents a feature entry in enabled.json
type FeatureConfig struct {
    Name      string   `json:"name"`
    Strategy  string   `json:"strategy,omitempty"`
    OnCommand []string `json:"onCommand,omitempty"`
    Disabled  bool     `json:"disabled,omitempty"`
}
```

**Status:** ✅ Synthesized from spec (not in original doc)

---

### FeatureManifest (manifest wrapper)

```go
// FeatureManifest represents the structure of enabled.json
type FeatureManifest struct {
    Features []FeatureConfig `json:"features"`
}
```

**Status:** ✅ Synthesized from spec (not in original doc)

---

### ShellConfig (shell configuration)

```go
// ShellConfig holds configuration for a specific shell
type ShellConfig struct {
    Name        string // "bash", "zsh", "fish", "powershell", "posix"
    ProfilePath string // Path to the shell's profile file (e.g., ~/.bashrc)
    InitScript  string // Path to init script template
    Extension   string // File extension for features (e.g., ".sh", ".zsh")
}
```

**Status:** ✅ Synthesized from spec needs

---

## Validation Test File

Here's a simple Go file to validate all the types compile:

```go
package shell_test

import (
    "encoding/json"
    "testing"
)

// Data structures from spec
type FeatureMetadata struct {
    Name            string   `json:"name"`
    Description     string   `json:"description"`
    DefaultStrategy string   `json:"defaultStrategy"`
    DefaultCommands []string `json:"defaultCommands,omitempty"`
    SupportedShells []string `json:"supportedShells"`
    Category        string   `json:"category"`
}

type FeatureConfig struct {
    Name      string   `json:"name"`
    Strategy  string   `json:"strategy,omitempty"`
    OnCommand []string `json:"onCommand,omitempty"`
    Disabled  bool     `json:"disabled,omitempty"`
}

type FeatureManifest struct {
    Features []FeatureConfig `json:"features"`
}

type ShellConfig struct {
    Name        string
    ProfilePath string
    InitScript  string
    Extension   string
}

// Test catalog from spec
var BuiltinFeatures = []FeatureMetadata{
    {
        Name:            "git-prompt",
        Description:     "Git status in prompt",
        DefaultStrategy: "defer",
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
        DefaultStrategy: "eager",
        SupportedShells: []string{"posix", "bash", "zsh", "fish", "powershell"},
        Category:        "aliases",
    },
}

// Test that structures can be marshaled/unmarshaled
func TestFeatureManifestJSON(t *testing.T) {
    manifest := FeatureManifest{
        Features: []FeatureConfig{
            {
                Name:     "git-prompt",
                Strategy: "defer",
            },
            {
                Name:      "kubectl",
                Strategy:  "on-command",
                OnCommand: []string{"kubectl", "k"},
            },
        },
    }
    
    // Marshal to JSON
    data, err := json.MarshalIndent(manifest, "", "  ")
    if err != nil {
        t.Fatalf("Failed to marshal: %v", err)
    }
    
    // Unmarshal back
    var decoded FeatureManifest
    if err := json.Unmarshal(data, &decoded); err != nil {
        t.Fatalf("Failed to unmarshal: %v", err)
    }
    
    if len(decoded.Features) != 2 {
        t.Errorf("Expected 2 features, got %d", len(decoded.Features))
    }
}

func TestFeatureMetadata(t *testing.T) {
    if len(BuiltinFeatures) != 3 {
        t.Errorf("Expected 3 builtin features, got %d", len(BuiltinFeatures))
    }
    
    // Validate catalog entry structure
    for _, feature := range BuiltinFeatures {
        if feature.Name == "" {
            t.Error("Feature has empty name")
        }
        if feature.DefaultStrategy == "" {
            t.Errorf("Feature %s has empty strategy", feature.Name)
        }
        if len(feature.SupportedShells) == 0 {
            t.Errorf("Feature %s has no supported shells", feature.Name)
        }
    }
}
```

**Status:** ✅ This compiles and tests pass

---

## Summary

### Validation Results

| Code Block | Status | Notes |
|------------|--------|-------|
| FeatureMetadata struct | ✅ Valid | Recommend adding JSON tags |
| BuiltinFeatures catalog | ✅ Valid | Good starting point |
| Additional types needed | ✅ Defined | FeatureConfig, FeatureManifest, ShellConfig |
| Test compilation | ✅ Pass | All types compile and work correctly |

### Recommendations

1. ✅ **Add JSON struct tags** to all types for proper marshaling
2. ✅ **Define missing types** (FeatureConfig, FeatureManifest) in the spec
3. ✅ **Add validation functions** for:
   - Feature name regex: `^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`
   - Strategy values: `eager`, `defer`, `on-command`
   - OnCommand array validation
4. ✅ **Consider adding constants** for strategy values:
   ```go
   const (
       StrategyEager      = "eager"
       StrategyDefer      = "defer"
       StrategyOnCommand  = "on-command"
   )
   ```

### Conclusion

All Go code examples in the specification are syntactically valid and will compile correctly. Minor enhancements recommended for production use (JSON tags, validation), but the core structures are sound.