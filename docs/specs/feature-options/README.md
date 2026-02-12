# Feature Options Framework

## Overview

This specification defines a framework for feature creators to declare configurable input requirements for shell features. This enables structured, type-safe, and secure collection of user preferences during feature setup.

## Motivation

Currently, features cannot request configuration input from users during setup. This limitation prevents features like oh-my-posh from prompting users to select themes, or SSH agent from asking which keys to load.

The Feature Options Framework solves this by:
- Providing structured metadata for feature input requirements
- Offering interactive prompts with type validation
- Ensuring security through input sanitization
- Supporting multiple input types (string, int, bool, enum, file/path)
- Enabling template-based feature generation with user values

## Architecture

### Core Components

```
┌─────────────────────────────────────────────────────┐
│                  Feature Catalog                     │
│  ┌────────────────────────────────────────────────┐ │
│  │ FeatureMetadata                                │ │
│  │  - Name, Description, Category                 │ │
│  │  - Options []OptionMetadata                    │ │
│  └────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│              Interactive Prompts                     │
│  ┌────────────────────────────────────────────────┐ │
│  │ PromptForOptions()                             │ │
│  │  - Type-specific prompts                       │ │
│  │  - Validation                                  │ │
│  │  - Error handling                              │ │
│  └────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│               Input Validation                       │
│  ┌────────────────────────────────────────────────┐ │
│  │ Type Validators:                               │ │
│  │  - String (sanitization, injection prevention) │ │
│  │  - Int (range checking)                        │ │
│  │  - Bool (true/false/1/0)                       │ │
│  │  - Enum (from predefined list)                 │ │
│  │  - File/Path (existence, safety checks)        │ │
│  └────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│              Manifest Storage                        │
│  ┌────────────────────────────────────────────────┐ │
│  │ FeatureConfig                                  │ │
│  │  - Name, Strategy, OnCommand                   │ │
│  │  - Options map[string]any              │ │
│  └────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│           Template Generation                        │
│  ┌────────────────────────────────────────────────┐ │
│  │ Feature file creation with:                    │ │
│  │  - Variable substitution                       │ │
│  │  - User option values                          │ │
│  └────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────┘
```

## Data Structures

### OptionMetadata

```go
// OptionType represents the type of a feature option
type OptionType string

const (
    OptionTypeString OptionType = "string"
    OptionTypeInt    OptionType = "int"
    OptionTypeBool   OptionType = "bool"
    OptionTypeEnum   OptionType = "enum"
    OptionTypeFile   OptionType = "file"
    OptionTypePath   OptionType = "path"
)

// OptionMetadata defines a configurable option for a feature
type OptionMetadata struct {
    Name         string      // Internal identifier (e.g., "theme_name")
    DisplayName  string      // Human-readable label (e.g., "Theme Name")
    Description  string      // Help text for the user
    Type         OptionType  // Data type
    Required     bool        // Whether this option is mandatory
    Default      any // Default value (used if user skips optional field)
    
    // Type-specific constraints
    EnumValues   []string    // Valid values for enum type
    IntMin       *int        // Minimum value for int type
    IntMax       *int        // Maximum value for int type
    PathMustExist bool       // For file/path: must the path already exist?
    FileOnly     bool        // For path: restrict to files only (no directories)
    
    // Validation
    Validator    func(any) error // Custom validation function
}
```

### Extended FeatureMetadata

```go
// FeatureMetadata contains metadata about a feature from the catalog
type FeatureMetadata struct {
    Name            string
    Description     string
    Category        string
    DefaultStrategy string
    DefaultCommands []string
    SupportedShells []string
    
    // NEW: Configurable options
    Options         []OptionMetadata
}
```

### Extended FeatureConfig

```go
// FeatureConfig represents a single feature configuration in enabled.json
type FeatureConfig struct {
    Name      string
    Strategy  string
    OnCommand []string
    Disabled  bool
    
    // NEW: User-provided option values
    // Stored as map for flexibility with JSON serialization
    Options   map[string]any `json:"options,omitempty"`
}
```

## Input Types and Validation

### String Type

**Validation:**
- Maximum length enforcement (default: 500 characters)
- Shell injection prevention: escape or reject dangerous characters
- Pattern matching support (regex validation)

**Security Considerations:**
- Strip null bytes
- Escape shell metacharacters: `$ ` ( ) { } [ ] | & ; < > \` " ' \n \r \t
- Reject if suspicious patterns detected (e.g., `$(...)`, backticks)

**Example:**
```go
{
    Name:        "prompt_color",
    DisplayName: "Prompt Color",
    Description: "Color code for your shell prompt",
    Type:        OptionTypeString,
    Required:    false,
    Default:     "blue",
    Validator: func(v any) error {
        s := v.(string)
        if len(s) > 20 {
            return fmt.Errorf("color name too long")
        }
        return nil
    },
}
```

### Int Type

**Validation:**
- Parse as integer
- Range checking (min/max)
- Reject non-numeric input

**Example:**
```go
{
    Name:        "ssh_agent_lifetime",
    DisplayName: "SSH Agent Key Lifetime (seconds)",
    Description: "How long should SSH keys remain cached?",
    Type:        OptionTypeInt,
    Required:    false,
    Default:     3600,
    IntMin:      ptr(60),     // At least 1 minute
    IntMax:      ptr(86400),  // At most 24 hours
}
```

### Bool Type

**Validation:**
- Accept: true, false, 1, 0, yes, no, y, n (case-insensitive)
- Reject any other value

**Example:**
```go
{
    Name:        "auto_update",
    DisplayName: "Auto Update",
    Description: "Automatically update tool when starting shell?",
    Type:        OptionTypeBool,
    Required:    false,
    Default:     true,
}
```

### Enum Type

**Validation:**
- Value must be from predefined list
- Case-sensitive by default

**Example:**
```go
{
    Name:        "oh_my_posh_theme",
    DisplayName: "Oh My Posh Theme",
    Description: "Select a theme for your prompt",
    Type:        OptionTypeEnum,
    Required:    true,
    EnumValues:  []string{"agnoster", "paradox", "powerlevel10k_rainbow", "robbyrussell"},
}
```

### File Type

**Validation:**
- Absolute path resolution
- Existence check (if PathMustExist is true)
- Ensure it's a file, not a directory (if FileOnly is true)
- Symlink safety: resolve and validate target
- Path traversal prevention: reject `..` in paths

**Security Considerations:**
- Restrict to specific directories (e.g., only allow ~/.config/...)
- Reject paths outside user's home directory
- Check file permissions

**Example:**
```go
{
    Name:          "ssh_key_file",
    DisplayName:   "SSH Private Key",
    Description:   "Path to your SSH private key",
    Type:          OptionTypeFile,
    Required:      false,
    Default:       "~/.ssh/id_rsa",
    PathMustExist: true,
    FileOnly:      true,
}
```

### Path Type

Similar to File, but may be a directory.

**Example:**
```go
{
    Name:          "project_directory",
    DisplayName:   "Project Directory",
    Description:   "Path to your projects folder",
    Type:          OptionTypePath,
    Required:      false,
    Default:       "~/projects",
    PathMustExist: false,
}
```

## Interactive Prompt Flow

### User Experience

When adding a feature with options:

```
$ omdot feature add oh-my-posh

Adding oh-my-posh to bash...

⚙️  Feature Configuration

❯ Oh My Posh Theme (required)
  Select a theme for your prompt
  
  > agnoster
    paradox
    powerlevel10k_rainbow
    robbyrussell

❯ Config File Path (optional, default: ~/.poshthemes/default.json)
  Path to Oh My Posh configuration file
  
  > ~/.poshthemes/default.json
  
  (Press Enter to use default, Esc to cancel)

✓ oh-my-posh configured successfully

Run 'omdot apply' to activate the feature
```

### Implementation

```go
// PromptForOptions collects user input for feature options
func PromptForOptions(metadata catalog.FeatureMetadata) (map[string]any, error) {
    values := make(map[string]any)
    
    for _, opt := range metadata.Options {
        // Skip if optional and user wants to use default
        if !opt.Required {
            useDefault, err := interactive.Confirm(
                fmt.Sprintf("Configure %s? (default: %v)", opt.DisplayName, opt.Default),
                false,
            )
            if err != nil {
                return nil, err
            }
            if !useDefault {
                values[opt.Name] = opt.Default
                continue
            }
        }
        
        // Prompt based on type
        var value any
        var err error
        
        switch opt.Type {
        case OptionTypeString:
            value, err = promptString(opt)
        case OptionTypeInt:
            value, err = promptInt(opt)
        case OptionTypeBool:
            value, err = promptBool(opt)
        case OptionTypeEnum:
            value, err = promptEnum(opt)
        case OptionTypeFile, OptionTypePath:
            value, err = promptPath(opt)
        default:
            return nil, fmt.Errorf("unsupported option type: %s", opt.Type)
        }
        
        if err != nil {
            return nil, err
        }
        
        // Validate
        if err := validateOption(opt, value); err != nil {
            return nil, fmt.Errorf("validation failed for %s: %w", opt.Name, err)
        }
        
        values[opt.Name] = value
    }
    
    return values, nil
}
```

## Template Variable Substitution

Feature files can use option values through template variables:

### Template Syntax

Use Go template syntax in feature files:

```bash
#!/usr/bin/env bash
# oh-my-dot feature: oh-my-posh
# Oh My Posh prompt theme

{{if .Options.oh_my_posh_theme}}
export POSH_THEME="{{.Options.oh_my_posh_theme}}"
{{end}}

{{if .Options.config_file}}
eval "$(oh-my-posh init bash --config {{.Options.config_file}})"
{{else}}
eval "$(oh-my-posh init bash)"
{{end}}
```

### Template Rendering

```go
// RenderFeatureTemplate renders a feature template with option values
func RenderFeatureTemplate(templatePath string, config manifest.FeatureConfig) (string, error) {
    tmpl, err := template.ParseFiles(templatePath)
    if err != nil {
        return "", err
    }
    
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, config); err != nil {
        return "", err
    }
    
    return buf.String(), nil
}
```

## Security Considerations

### Input Sanitization

All user input must be sanitized before:
1. Writing to manifest files
2. Substituting into templates
3. Executing shell commands

### Shell Injection Prevention

**Dangerous patterns to reject:**
- Command substitution: `$(...)`, `` `...` ``
- Pipes and redirects: `|`, `>`, `<`, `>>`
- Command separators: `;`, `&&`, `||`
- Background execution: `&`
- Glob expansion: `*`, `?`

**Safe escaping strategy:**
- Use shell-safe quoting for all interpolated values
- Prefer whitelisting over blacklisting
- Validate before escaping

### Path Traversal Prevention

**Restrictions:**
- Reject relative paths containing `..`
- Resolve all paths to absolute
- Ensure paths stay within safe boundaries (e.g., home directory)
- Validate symlink targets

### File Permission Checks

For file/path options:
- Verify read/write permissions
- Check ownership (should be current user)
- Reject world-writable files in sensitive contexts

## Non-Interactive Mode

For CI/CD and automation:

```bash
# In non-TTY environments, prompts are skipped automatically
omdot feature add oh-my-posh

# Required options without defaults fail fast
omdot feature add oh-my-posh

# Provide explicit values via flags
omdot feature add oh-my-posh \
  --option theme=agnoster \
  --option config_file=~/.poshthemes/custom.json
```

## Command-Line Flag Override

Specify options via CLI flags:

```go
featureAddCmd.Flags().StringSlice("option", nil, "Set feature option (key=value)")
```

Parse format: `--option key=value`

## Example: oh-my-posh Feature

### Catalog Entry

```go
"oh-my-posh": {
    Name:            "oh-my-posh",
    Description:     "Oh My Posh prompt engine",
    Category:        "prompt",
    DefaultStrategy: "eager",
    SupportedShells: []string{"bash", "zsh", "fish", "powershell"},
    Options: []OptionMetadata{
        {
            Name:        "theme",
            DisplayName: "Theme",
            Description: "Oh My Posh theme to use",
            Type:        OptionTypeEnum,
            Required:    true,
            EnumValues: []string{
                "agnoster",
                "paradox",
                "powerlevel10k_rainbow",
                "robbyrussell",
                "jandedobbeleer",
            },
        },
        {
            Name:          "config_file",
            DisplayName:   "Config File",
            Description:   "Path to Oh My Posh configuration file",
            Type:          OptionTypeFile,
            Required:      false,
            PathMustExist: true,
            FileOnly:      true,
        },
        {
            Name:        "auto_upgrade",
            DisplayName: "Auto Upgrade",
            Description: "Automatically upgrade Oh My Posh on shell start",
            Type:        OptionTypeBool,
            Required:    false,
            Default:     false,
        },
    },
},
```

### Generated Feature File

```bash
#!/usr/bin/env bash
# oh-my-dot feature: oh-my-posh
# Oh My Posh prompt engine

# Configuration
export POSH_THEME="agnoster"

# Auto-upgrade check
if [ "$POSH_AUTO_UPGRADE" = "true" ]; then
  oh-my-posh upgrade &>/dev/null &
fi

# Initialize Oh My Posh
if [ -f "~/.config/oh-my-posh/config.json" ]; then
  eval "$(oh-my-posh init bash --config ~/.config/oh-my-posh/config.json)"
else
  eval "$(oh-my-posh init bash)"
fi
```

## Edge Cases

### Symbolic Links
- Always resolve symlinks to real paths
- Validate the target of the symlink
- Reject broken symlinks

### Non-UTF8 Filenames
- Attempt to normalize using Unicode NFC
- Reject if normalization fails
- Log warning for user

### Circular Symlinks
- Detect and reject circular symlink chains
- Use OS-level path resolution functions

### World-Writable Files
- Warn or reject world-writable config files
- Especially important for SSH keys, credentials

## Validation Rules by Shell

Some options may need shell-specific validation:

```go
type OptionMetadata struct {
    // ... existing fields ...
    
    // Shell-specific overrides
    ShellValidators map[string]func(any) error
}
```

Example:
```go
{
    Name: "completion_style",
    Type: OptionTypeEnum,
    EnumValues: []string{"inline", "popup", "menu"},
    ShellValidators: map[string]func(any) error{
        "bash": func(v any) error {
            // Bash only supports "menu" style
            if v.(string) != "menu" {
                return fmt.Errorf("bash only supports menu style")
            }
            return nil
        },
    },
}
```

## Testing Strategy

### Unit Tests
- Test each validator independently
- Test injection attempts (security)
- Test edge cases (empty, nil, overflow)

### Integration Tests
- End-to-end feature addition with options
- Template rendering with options
- Manifest persistence and loading

### Security Tests
- Fuzz testing for input validation
- Injection attack scenarios
- Path traversal attempts

## Backwards Compatibility

- Features without options continue to work unchanged
- Options field in FeatureConfig is optional (omitempty)
- Old manifest files without options remain valid

## Future Enhancements

1. **Dynamic Enum Values**: Fetch enum values at runtime (e.g., list themes from filesystem)
2. **Conditional Options**: Show/hide options based on other option values
3. **Option Groups**: Organize related options together
4. **Validation Feedback**: Show inline validation errors during input
5. **Option Presets**: Allow users to save and reuse option sets
6. **Import/Export**: Export options to YAML/JSON for sharing

## Open Questions

### Q1: Should we allow directories for file type?
**Answer**: Create separate `path` type for directories/mixed, keep `file` strictly for files.

### Q2: Path sanitization: allow only within home directory?
**Answer**: By default no (to support standard workflows), with optional enforcement via `restrict-paths-to-home`.

### Q3: Should validation rules be configurable per feature?
**Answer**: Yes, via custom Validator functions in OptionMetadata.

### Q4: Do feature authors need granular control over validation/option prompts per shell?
**Answer**: Yes, via ShellValidators map for shell-specific validation logic.

### Q5: How to handle symlinks?
**Answer**: Always resolve to real path, validate target, reject broken links.

## Implementation Phases

### Phase 1: Core Infrastructure (MVP)
- Define OptionMetadata and extend FeatureMetadata
- Implement basic validators (string, int, bool, enum)
- Extend manifest.FeatureConfig with Options field
- Basic template variable substitution

### Phase 2: Security Hardening
- Shell injection prevention
- Path traversal prevention
- Input sanitization
- Security testing

### Phase 3: File/Path Support
- File picker integration
- Path validation
- Symlink handling
- Permission checks

### Phase 4: Interactive UX
- Type-specific prompts
- Validation feedback
- Error messages
- Help text display

### Phase 5: Polish
- Non-interactive mode
- CLI flag overrides
- Documentation
- Example features

## References

- Go text/template: https://pkg.go.dev/text/template
- OWASP Input Validation: https://cheatsheetseries.owasp.org/cheatsheets/Input_Validation_Cheat_Sheet.html
- Shell Injection Prevention: https://www.owasp.org/index.php/Command_Injection
