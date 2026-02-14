# Feature Options - User Guide

## Overview

Some oh-my-dot features support configurable options that allow you to customize their behavior during setup. When adding a feature with options, you'll be prompted interactively to configure it according to your preferences.

## Adding Features with Options

### Interactive Mode

When you add a feature that has configurable options, oh-my-dot will guide you through an interactive configuration process:

```bash
$ omdot feature add oh-my-posh

Adding oh-my-posh to bash...

⚙️  Feature Configuration

❯ Theme (required) - Oh My Posh theme to use
  Select a theme for your prompt:
  
  > agnoster
    paradox
    powerlevel10k_rainbow
    robbyrussell
    jandedobbeleer
    atomic
    dracula
    pure

❯ Configure Config File? (default: <nil>)
  [No]  [Yes]

❯ Configure Auto Upgrade? (default: false)
  [No]  [Yes]

✓ oh-my-posh configured successfully

Run 'omdot apply' to activate the feature
```

### Option Types

Features can have different types of options:

#### 1. **String Options**

Simple text input with validation.

```
Example: Enter a name or identifier
```

#### 2. **Integer Options**

Numeric values with optional min/max constraints.

```
Example: SSH Agent Key Lifetime (seconds) (min: 60, max: 86400)
```

#### 3. **Boolean Options**

Yes/No choices.

```
Example: Auto Upgrade? [No] [Yes]
```

#### 4. **Enum Options**

Select from a predefined list of values.

```
Example: Select a theme:
  > agnoster
    paradox
    robbyrussell
```

#### 5. **File/Path Options**

Path to a file or directory with validation.

```
Example: SSH Private Key (must exist, file only)
  > ~/.ssh/id_rsa
```

## Required vs Optional Options

### Required Options

Required options **must** be configured before the feature can be added. You'll be prompted to provide a value.

### Optional Options

Optional options can be skipped. If you skip an optional option, the feature's default value (if any) will be used.

For optional options, you'll first be asked if you want to configure them:

```
❯ Configure Config File? (default: <nil>)
  [No]  [Yes]
```

- Select **No** or press Enter to skip and use the default
- Select **Yes** to provide a custom value

## Option Validation

All user input is validated for security and correctness:

### String Validation

- Maximum length enforced
- Shell injection prevention (dangerous characters rejected)
- Null bytes removed

### Integer Validation

- Must be a valid number
- Range checking (min/max)

### Boolean Validation

- Accepts: true, false, 1, 0, yes, no, y, n

### Enum Validation

- Value must be from the predefined list

### File/Path Validation

- Path existence check (if required)
- File vs directory validation
- Symlink safety checks
- Path must be within home directory (security)

## Viewing Configured Options

After adding a feature, you can view its configuration including option values:

```bash
$ omdot feature list

bash:
  ✓ oh-my-posh (eager)
    ~/.ohdot/omd-shells/bash/features/oh-my-posh.sh

$ omdot feature info oh-my-posh

oh-my-posh
  Category: prompt
  Description: Oh My Posh prompt engine with customizable themes
  Default Strategy: eager
  Supported Shells: [bash zsh fish powershell]

Current Configuration:
  bash: enabled (eager)
```

## Modifying Options

Currently, to change feature options after initial setup, you need to:

1. Remove the feature: `omdot feature remove oh-my-posh`
2. Re-add it with new options: `omdot feature add oh-my-posh`

## Example: SSH Agent Feature

Here's another example of a feature with various option types:

```bash
$ omdot feature add ssh-agent

Adding ssh-agent to bash...

⚙️  Feature Configuration

❯ SSH Key File (optional, default: ~/.ssh/id_rsa) - Path to your SSH private key
  > ~/.ssh/id_ed25519

❯ Key Lifetime (optional, default: 3600) - How long should SSH keys remain cached? (min: 60, max: 86400)
  > 7200

❯ Auto Start Agent (optional, default: true) - Automatically start SSH agent if not running?
  [Yes]  [No]

✓ ssh-agent configured successfully
```

## Non-Interactive Mode

If you're using oh-my-dot in scripts or CI/CD pipelines (non-TTY), feature options are resolved without prompts:

- Optional options use defaults when available
- Required options use defaults when available
- Required options without defaults fail fast with an error

```bash
# In CI or non-TTY environment
$ omdot feature add oh-my-posh

Adding oh-my-posh to bash...
Error: failed to resolve feature options in non-interactive mode: required option 'Theme' has no default and cannot be resolved in non-interactive mode
```

Use `--option key=value` to provide explicit values (repeat the flag for multiple options):

```bash
omdot feature add oh-my-posh --option theme=agnoster
```

This also works in interactive mode, where provided options are pre-filled and skipped.

## Security Considerations

### Input Sanitization

All user input is sanitized to prevent:

- Shell command injection
- Path traversal attacks
- Null byte attacks

### Safe Defaults

- File/path options are unrestricted by default to support standard system-path workflows
- Home-only path enforcement is available via `restrict-paths-to-home`
- Dangerous characters in strings are escaped or rejected
- Path existence and permissions are validated

## Troubleshooting

### "Validation failed" Error

If you see a validation error, check:

- For file paths: ensure the path exists when required and, if home restriction is enabled, is within your home directory
- For integers: ensure the value is within the min/max range
- For enums: select a value from the provided list

### Skipping Interactive Prompts

Prompts are skipped automatically in non-TTY environments:

```bash
omdot feature add oh-my-posh
```

## Feature Authors

If you're creating features with configurable options, see [docs/specs/feature-options/README.md](../specs/feature-options/README.md) for technical documentation.

## Available Features with Options

To see which features support options, use the catalog browser:

```bash
omdot feature add -i
```

Features with configuration options will show a ⚙️ indicator (future enhancement).

## Feedback

Found a bug or have suggestions for feature options? Please report at:
<https://github.com/PatrickMatthiesen/oh-my-dot/issues>
