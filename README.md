# oh-my-dot

A cross-platform dotfile manager with an advanced shell framework written in Go.

Oh-my-dot helps you manage your dotfiles across multiple machines with git integration and provides a powerful shell framework for managing shell configurations (aliases, prompts, completions) across bash, zsh, fish, PowerShell, and POSIX sh.

See [CLI workflows and usability decisions](docs/cli-usability.md) for guided setup,
headless provisioning, JSON inspection, preview behavior, and safety limits.

## Features

- **Cross-Platform**: Works on Windows, macOS, and Linux
- **Git Integration**: Store and sync your dotfiles with git
- **Shell Framework**: Manage shell features with advanced loading strategies
  - Eager loading for instant availability
  - Deferred loading for faster shell startup
  - On-command loading for lazy evaluation
- **Local Overrides**: Per-machine customizations with security validation
- **Health Checks**: Built-in `doctor` command to validate configuration
- **Interactive Mode**: Browse and manage features interactively

## Install

### Linux and macOS (Automatic)

Run the install script to automatically download and install the latest version:

```sh
curl -fsSL https://raw.githubusercontent.com/PatrickMatthiesen/oh-my-dot/main/install.sh | bash
```

> **Security Note**: For added security, you can download and inspect the script before running it:
> ```sh
> curl -fsSL https://raw.githubusercontent.com/PatrickMatthiesen/oh-my-dot/main/install.sh -o install.sh
> # Review the script
> cat install.sh
> # Run it
> bash install.sh
> ```

This script will:
- Detect your OS and architecture automatically
- Download the latest release from GitHub
- Install the binary to `~/.oh-my-dot/bin`
- Create a symlink in `~/.local/bin`
- Add the binary to your current session's PATH

If you need to install a specific version, you can set the `OH_MY_DOT_VERSION` environment variable:

```sh
curl -fsSL https://raw.githubusercontent.com/PatrickMatthiesen/oh-my-dot/main/install.sh | OH_MY_DOT_VERSION=v0.0.25 bash
```

### Windows (Winget)

```sh
winget install PatrickMatthiesen.oh-my-dot
```

### Manual Installation

1. Find the latest release matching your CPU architecture and Operating system.
2. Download the executable to a persistent folder.
   > A folder you know won't change or randomly be deleted
   >
   > I recommend putting it under `$HOME/oh-my-dot`
3. Add the folder to your PATH
4. Start using oh-my-dot

## Quick Start

### Initialize Repository

```sh
# New fresh config
oh-my-dot init github.com/username/dotfiles

# Or with explicit remote
oh-my-dot init --remote github.com/username/dotfiles

# Existing config (use --force to override)
oh-my-dot init -r github.com/username/dotfiles -f /path/to/dotfiles --force
```

### Apply Dotfiles

```sh
oh-my-dot apply
```

This will:
- Link your dotfiles to their target locations
- Install shell hooks for the shell framework

### Shell Completion

Enable tab completion for your shell:

```sh
# Bash
echo 'source <(oh-my-dot completion bash)' >> ~/.bashrc

# Zsh
echo 'source <(oh-my-dot completion zsh)' >> ~/.zshrc

# Fish
oh-my-dot completion fish > ~/.config/fish/completions/oh-my-dot.fish

# PowerShell
oh-my-dot completion powershell >> $PROFILE
```

## Shell Framework

The shell framework allows you to manage shell configurations as modular features with advanced loading strategies.

### Adding Features

#### Interactive Mode (Recommended)

```sh
oh-my-dot feature add -i
```

Browse the catalog and select features to add. The interactive mode automatically detects your current shell, places it at the top of the shell selection list, and pre-selects it for your convenience.

**Supported Shells:** bash, zsh, fish, PowerShell, and POSIX sh

#### Direct Addition

```sh
# Add to current shell (auto-detected)
oh-my-dot feature add git-prompt

# Add to specific shell
oh-my-dot feature add kubectl-completion --shell bash

# Add to all supported shells
oh-my-dot feature add core-aliases --all
```

When no `--shell` flag is provided, oh-my-dot tries to detects your current shell and adds the feature to it.

### Loading Strategies

Choose when a feature loads with `--strategy`. Run `oh-my-dot apply` to regenerate existing init scripts after an upgrade.

#### Eager Loading (Default)
Loads during shell startup and reapplies when you source the init script again.
```sh
oh-my-dot feature add git-prompt --strategy eager
```

#### Deferred Loading
Bash, Zsh, and Fish load deferred features once, before the first interactive prompt. POSIX sh has no portable prompt hook, so it loads deferred features at the end of interactive init. These shells skip deferred features in non-interactive sessions.

PowerShell loads deferred features once when the session first idles after startup.
```sh
oh-my-dot feature add kubectl-completion --strategy defer
```

#### On-Command Loading
The first trigger invocation loads the feature and forwards the arguments to the resulting command. If enabled features share a trigger, the first one in the effective manifest owns it. In Bash, Zsh, Fish, and POSIX sh, existing aliases and functions take precedence over wrappers.

POSIX sh trigger names must start with a letter or underscore and contain only letters, digits, and underscores. Any sibling wrappers the feature leaves in place remove themselves on their next call.

PowerShell loads the feature during command lookup, which can also happen when you run `Get-Command`. PowerShell's deferred and on-command scripts run in modules that export their functions and aliases. Use `$global:` for variables that must be visible outside the feature.

Load completion features eagerly or with `defer` if you want completion to be ready before you invoke the command. A trigger run in a subshell loads the feature in that subshell.
```sh
oh-my-dot feature add nvm --strategy on-command --on-command nvm,node,npm
```

### Managing Features

```sh
# List features
oh-my-dot feature list
oh-my-dot feature list --shell bash

# Remove features
oh-my-dot feature remove -i              # Interactive
oh-my-dot feature remove git-prompt      # Direct
oh-my-dot feature remove kubectl --all   # From all shells

# Update/refresh features
oh-my-dot feature update -i                      # Interactive
oh-my-dot feature update powershell-aliases      # Direct
oh-my-dot feature update powershell-aliases --shell powershell

# Enable/disable features
oh-my-dot feature enable git-prompt
oh-my-dot feature disable kubectl --shell bash

# Show feature info
oh-my-dot feature info git-prompt
oh-my-dot feature info oh-my-posh --shell powershell --json

# Change saved options without removing a feature
oh-my-dot feature configure oh-my-posh --shell powershell --option theme=pure

# Keep a device-only difference; remove it to restore shared settings
oh-my-dot feature configure oh-my-posh --shell powershell --local --option theme=paradox
oh-my-dot feature remove oh-my-posh --shell powershell --local
```

In a terminal, commands with missing selections open a picker. Complete arguments
run directly. Scripts can use `--no-interactive` to require explicit inputs.
Customized feature scripts require `--force` before configure replaces them.

### Feature Files

Features are stored in `omd-shells/<shell>/features/`:

```sh
# Example: omd-shells/bash/features/git-prompt.sh
# Add your custom shell code here
parse_git_branch() {
  git branch 2>/dev/null | sed -e '/^[^*]/d' -e 's/* \(.*\)/(\1)/'
}

export PS1="\u@\h \W \$(parse_git_branch) $ "
```

### Local Overrides

Create `enabled.local.json` for per-machine customizations:

```sh
# omd-shells/bash/enabled.local.json
{
  "features": [
    {
      "name": "git-prompt",
      "strategy": "eager"
    },
    {
      "name": "work-vpn",
      "strategy": "on-command",
      "onCommand": ["vpn-connect"]
    }
  ]
}
```

**Security Requirements:**
- File must be owned by current user
- File must not be group or world writable
- File must be a regular file (not a symlink)

Invalid local overrides are automatically ignored with warnings.

### Organizing dotfiles

Use `add FILE` to store a dotfile. If its filename is taken, add automatically
includes parent directory names and reports the chosen path. Use `--as tool/config`
for an exact name. Bare `add` opens a picker in a terminal. `list` shows full paths;
`remove tool/config` selects one entry. Folder names organize storage and do not activate groups.
See the [nested file naming guide](docs/specs/nested-files/README.md) for examples,
Windows path rules, and an isolated test configuration.

### Health Checks

Validate your repository and shell framework setup:

```sh
# Check all shells
oh-my-dot doctor

# Check specific shell
oh-my-dot doctor --shell bash

# Auto-fix issues
oh-my-dot doctor --fix
```

For GitHub HTTPS repositories, run `gh auth setup-git` after `gh auth login`.
Clone, push, pull, and remote health checks use Git's configured HTTPS credential
helpers. The `allow-gh-auth` setting applies only to release update checks.
SSH operations use native Git and its SSH configuration, keys, and agents.
For a GitHub HTTPS URL, init uses a configured Git credential helper or gh's saved
SSH preference. Otherwise, interactive init lets you choose HTTPS, SSH, or
GitHub CLI setup before connecting. It saves the selected remote URL.

Initialization reports clone failures without leaving a partial repository.
`init --force` confirms whether to reuse the current remote or asks for a new URL.
An explicit URL skips the prompt; non-interactive forced initialization requires
an explicit URL. Forced initialization validates the selected remote, updates
origin, and fetches and checks out its default branch when recovering an empty
repository. Existing commits and files are preserved. Remote initialization
accepts reachable empty remotes; unreachable remotes still fail.
Normal init asks for a URL once. Leave it blank for a local repository.
Use `oh-my-dot config set remote-url <url>` later to update both origin and
the saved setting. Use `oh-my-dot config set remote-url ""` to remove origin.
`status` is an alias for `doctor`.

The doctor checks:

- Git repository and effective commit identity
- Origin remote configuration and read access (warnings only)
- Directory structure
- Manifest validity
- Feature file existence
- Profile hooks installation
- Local override security
- Init script syntax

Repository checks run even without shell features. Missing Git identity is an
error with instructions for setting your name and email; `doctor --fix` does not
choose an identity for you. Missing or inaccessible remotes produce warnings in
doctor. Local `add` and `remove` operations do not contact the remote. Remote read
access does not guarantee push permission.

### PowerShell Support

Oh-my-dot fully supports PowerShell (both Windows PowerShell 5.1 and PowerShell Core 7+):

**Auto-Detection:** When running `feature add -i` in PowerShell, your current shell is automatically detected and pre-selected.

**PowerShell-Specific Features:**

- `powershell-prompt` - Custom prompt with git status
- `powershell-aliases` - Common PowerShell aliases and shortcuts
- `posh-git` - Git integration for PowerShell

**Profile Integration:**

```powershell
# PowerShell profile ($PROFILE)
. "$HOME\dotfiles\omd-shells\powershell\init.ps1"
```

### Profile shell startup

```sh
oh-my-dot profile
oh-my-dot profile --shell bash
oh-my-dot profile --json
oh-my-dot profile --shell zsh --full
```

`profile` measures enabled oh-my-dot features in your configured dotfiles
repository. It uses device-local overrides and runs each feature in its own
noninteractive shell with user startup files disabled. Zsh still runs its
system `zshenv` during process launch. Managed profiling includes deferred
and on-command features. The total sums standalone feature timings and
excludes process launch. Features that depend on another feature or your
profile can behave differently during startup. On Bash, Zsh, Fish, and
POSIX sh, very fast loads can round to zero at the file system's timestamp
resolution.

The command profiles installed Bash, Zsh, Fish, PowerShell, and POSIX sh.
Use `--shell` to select one. Use `sh` or `posix` for POSIX sh. The profiler
prefers PowerShell 7 and falls back to Windows PowerShell 5.1 on Windows.

Use `--full` to measure shell startup without an oh-my-dot repository. Bash,
Zsh, and Fish use interactive non-login startup. POSIX sh uses interactive
login startup to include `.profile`. These shells report one total for
process launch and startup, without a terminal or a first prompt.

PowerShell replays its console profile files in startup order and reports
each file's timing, excluding process launch and the first prompt. Feature
timings from an instrumented loader appear under the file that loaded them.
Those timings are included in the file's time. If your generated PowerShell
loader predates profiling, run `oh-my-dot apply` to regenerate it.

Profile and feature scripts can have their usual side effects, such as
writing files or contacting services. The profiler does not write to the
managed files.

Use `--json` for scripts. Standard output contains one document with
`schemaVersion` and a `results` array. Each result identifies the shell, the
mode (`managed` or `full`), and its status (`ok`, `partial`, `failed`,
`unavailable`, or `unsupported`).
Results include a `report` when timings are available and an `error` when
the profiler cannot complete. Each shell has a 30-second deadline and
retains collected results if a script fails or the deadline expires. Any
status other than `ok` returns a nonzero exit code.

## Directory Structure

```
dotfiles/
├── files/                    # Your dotfiles
│   ├── .gitconfig
│   ├── .vimrc
│   └── ...
├── omd-shells/              # Shell framework
│   ├── bash/
│   │   ├── enabled.json           # Base configuration (tracked)
│   │   ├── enabled.local.json     # Local overrides (untracked)
│   │   ├── init.sh               # Auto-generated init script
│   │   ├── features/             # Feature implementations
│   │   │   ├── git-prompt.sh
│   │   │   └── aliases.sh
│   │   └── helpers/              # Shared helper functions
│   ├── zsh/
│   ├── fish/
│   └── powershell/
└── .linkings                # Dotfile link mappings
```

## Configuration

Oh-my-dot uses a `.oh-my-dot.yaml` config file:

```yaml
repo-path: ~/dotfiles
remote: github.com/username/dotfiles
```

## Commands Reference

### Core Commands

- `oh-my-dot init` - Initialize dotfiles repository
- `oh-my-dot apply` - Apply dotfiles and shell integration
- `oh-my-dot push` - Commit and push changes to git
- `oh-my-dot pull` - Pull changes from git
- `oh-my-dot status` - Show repository status

### Feature Commands

- `oh-my-dot feature add [feature]` - Add a shell feature or browse in a terminal
- `oh-my-dot feature configure [feature]` - Change saved options
- `oh-my-dot feature remove [feature]` - Remove a shell feature
- `oh-my-dot feature update [feature]` - Refresh a catalog implementation
- `oh-my-dot feature list` - List all features
- `oh-my-dot feature enable <feature>` - Enable feature
- `oh-my-dot feature disable <feature>` - Disable feature
- `oh-my-dot feature info <feature>` - Show feature details

### Utility Commands

- `oh-my-dot doctor [--fix]` - Health check and diagnostics
- `oh-my-dot profile [--shell powershell] [--json]` - Measure shell profile startup
- `oh-my-dot completion <shell>` - Generate shell completion
- `oh-my-dot version` - Show version information

### Automatic Remote Update Checks

oh-my-dot performs a lightweight remote sync check (no full fetch) on:

- `oh-my-dot add`
- `oh-my-dot remove`
- `oh-my-dot apply`
- `oh-my-dot push`
- `oh-my-dot doctor`

When updates are detected, oh-my-dot shows a non-blocking notice and suggests `oh-my-dot pull`.

Sync state messages:

- **Remote ahead**: prompts you to run `oh-my-dot pull`
- **Diverged**: warns that pull may require conflict resolution
- **Local ahead**: suggests running `oh-my-dot push`

### Global Flags

- `-i, --interactive` - Force interactive mode
- `--no-interactive` - Disable all prompts (for CI/scripting)

## Examples

### Basic Workflow

```sh
# Initialize
oh-my-dot init github.com/username/dotfiles

# Add features interactively
oh-my-dot feature add -i

# Apply changes
oh-my-dot apply

# Commit and push
oh-my-dot push
```

### Advanced Feature Management

```sh
# Add git prompt with eager loading
oh-my-dot feature add git-prompt --strategy eager --shell bash

# Add kubectl completion with deferred loading
oh-my-dot feature add kubectl --strategy defer

# Add nvm with on-command loading
oh-my-dot feature add nvm --strategy on-command --on-command nvm,node,npm

# Create local override for work machine
cat > ~/dotfiles/omd-shells/bash/enabled.local.json << 'EOF'
{
  "features": [
    {
      "name": "work-aliases",
      "strategy": "eager"
    }
  ]
}
EOF
chmod 600 ~/dotfiles/omd-shells/bash/enabled.local.json

# Validate setup
oh-my-dot doctor --fix
```

### Troubleshooting

```sh
# Check health
oh-my-dot doctor

# Fix common issues
oh-my-dot doctor --fix

# Re-apply shell hooks
oh-my-dot apply

# Check git status
oh-my-dot status
```

## Known Issues

### SSH

If you are using SSH to clone your dotfiles, you will need to add your SSH key to the SSH-Agent. This is because oh-my-dot uses Git under the hood:

```sh
ssh-add ~/.ssh/id_rsa
```

> If you are having issues on Windows you might want to install a newer version of [OpenSSH](https://github.com/PowerShell/Win32-OpenSSH/wiki/Install-Win32-OpenSSH). The one that comes with Windows is outdated and requires a bit more work to get going.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Security

Oh-my-dot includes security features to protect against path traversal and other vulnerabilities. See [docs/SECURITY.md](docs/SECURITY.md) for details on:

- Path traversal protection
- Configurable home directory restrictions
- Input validation

To report security vulnerabilities, please use [GitHub Security Advisories](https://github.com/PatrickMatthiesen/oh-my-dot/security/advisories).

## License

This project is licensed under the MIT License - see the LICENSE file for details.
