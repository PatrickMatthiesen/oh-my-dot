# oh-my-dot

A cross-platform dotfile manager with an advanced shell framework written in Go.

Oh-my-dot helps you manage your dotfiles across multiple machines with git integration and provides a powerful shell framework for managing shell configurations (aliases, prompts, completions) across bash, zsh, fish, PowerShell, and POSIX sh.

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

Features can be loaded with different strategies to optimize shell startup:

#### Eager Loading (Default)
Loads immediately during shell startup:
```sh
oh-my-dot feature add git-prompt --strategy eager
```

#### Deferred Loading
Loads in background for interactive shells (faster startup):
```sh
oh-my-dot feature add kubectl-completion --strategy defer
```

#### On-Command Loading
Lazy loads when specific commands are invoked:
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

# Enable/disable features
oh-my-dot feature enable git-prompt
oh-my-dot feature disable kubectl --shell bash

# Show feature info
oh-my-dot feature info git-prompt
```

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

### Health Checks

Validate your shell framework setup:

```sh
# Check all shells
oh-my-dot doctor

# Check specific shell
oh-my-dot doctor --shell bash

# Auto-fix issues
oh-my-dot doctor --fix
```

The doctor checks:
- Directory structure
- Manifest validity
- Feature file existence
- Profile hooks installation
- Local override security
- Init script syntax

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

The init script supports all loading strategies (eager, defer, on-command) with PowerShell-native syntax.

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

- `oh-my-dot feature add [-i] <feature>` - Add shell feature
- `oh-my-dot feature remove [-i] <feature>` - Remove shell feature
- `oh-my-dot feature list` - List all features
- `oh-my-dot feature enable <feature>` - Enable feature
- `oh-my-dot feature disable <feature>` - Disable feature
- `oh-my-dot feature info <feature>` - Show feature details

### Utility Commands

- `oh-my-dot doctor [--fix]` - Health check and diagnostics
- `oh-my-dot completion <shell>` - Generate shell completion
- `oh-my-dot version` - Show version information

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

## License

This project is licensed under the MIT License - see the LICENSE file for details.
