# Security

This document describes the security features and configuration options in oh-my-dot.

## Path Validation

oh-my-dot validates all file and path inputs from feature options to prevent security vulnerabilities.

### Path Traversal Protection

All file and path inputs are checked for path traversal attempts (e.g., `../../../etc/passwd`) **before** path normalization. This prevents attackers from escaping the intended directory structure.

Examples of blocked paths:

- `../../../etc/passwd`
- `..`
- `some/path/../../../escape`
- `C:\Users\Test\..\..\..\Windows\System32` (Windows)

Legitimate relative and absolute paths are still allowed:

- `~/dotfiles/config.txt`
- `/home/user/.bashrc`
- `C:\Users\Username\Documents\file.txt`

### Restricting Paths to Home Directory

By default, oh-my-dot allows file and path options to reference any accessible location on the filesystem. This provides flexibility for users who need to reference system-wide configuration files (e.g., `/etc/profile.d/custom.sh`).

However, organizations or security-conscious users can optionally restrict all file/path options to the user's home directory by setting a configuration option.

#### Enabling Home Directory Restriction

Set the `restrict-paths-to-home` configuration option to `true`:

```bash
# View current setting
oh-my-dot config get restrict-paths-to-home

# Enable restriction (only allow paths within home directory)
oh-my-dot config set restrict-paths-to-home true

# Disable restriction (allow system-wide paths) - this is the default
oh-my-dot config set restrict-paths-to-home false
```

#### Configuration File

The setting is stored in `~/.oh-my-dot/config.json`:

```json
{
  "restrict-paths-to-home": true
}
```

#### Behavior

- **When disabled (default)**: Users can reference any accessible path, including system-wide configuration files
- **When enabled**: All file and path options must be within the user's home directory

Example error when restriction is enabled and a system path is used:

```error
Error: path must be within home directory (restrict-paths-to-home is enabled): /etc/profile
```

#### Use Cases

**For standard users (default: unrestricted):**

- Reference system-wide shell configurations
- Use dotfiles stored in custom locations
- Access shared configuration files

**For organizations (enable restriction):**

- Enforce security policy that prevents users from binding to system files
- Ensure all user configurations are within their home directory
- Prevent accidental or malicious system file references

## String Validation

All string inputs are validated to prevent shell injection attacks:

- Null bytes are rejected
- Command substitution patterns are detected (`$(...)`, `` `...` ``)
- Shell metacharacters in dangerous contexts are flagged
- Path traversal in string inputs is blocked

## Integer and Boolean Validation

- Integer inputs are validated against min/max constraints
- Boolean inputs accept multiple formats: `true/false`, `yes/no`, `1/0`

## Enum Validation

Enum options are restricted to predefined valid values from the feature catalog.

## Best Practices

1. **Keep `restrict-paths-to-home` disabled by default** unless your security policy requires it
2. **Use absolute paths** when possible for clarity
3. **Use tilde expansion** (`~/`) for paths within your home directory
4. **Review feature options** from untrusted sources before enabling them
5. **Report security issues** through GitHub Security Advisories

## Reporting Security Issues

If you discover a security vulnerability, please report it via [GitHub Security Advisories](https://github.com/PatrickMatthiesen/oh-my-dot/security/advisories).

**Do not** open public issues for security vulnerabilities.
