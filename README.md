# oh-my-dot

A cross-platform dotfile manager written in go.

Oh-my-dot is a dotfile manager that helps you manage your dotfiles across multiple machines. It is designed to be simple and easy to use, while still being powerful enough to handle complex configurations. It is written in Go and is cross-platform, so you can use it on Windows, macOS, and Linux.

Oh-my-dot is designed to be used with git, so you can easily store your dotfiles in a git repository and share them with others.
It also supports multiple profiles, so you can have different configurations for different machines or users.

## Install manager

### Linux and macOS (Automatic)

Run the install script to automatically download and install the latest version:

```sh
curl -fsSL https://raw.githubusercontent.com/PatrickMatthiesen/oh-my-dot/main/install.sh | bash
```

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

## Init config

### New fresh config

```sh
oh-my-dot init github.com/username/dotfiles
or
oh-my-dot init --remote github.com/username/dotfiles
```

### Existing config

Use the fore flag to force initialization even if the folder is not empty.

```sh
oh-my-dot init -r github.com/username/dotfiles -f /path/to/dotfiles --force
```

## Known issues

### SSH

If you are using SSH to clone your dotfiles, you will need to add your SSH key to the SSH-Agent. This is because Oh-My-Dot uses Git under the hood. You can add your key by running the following command:

```sh
ssh-add ~/.ssh/id_rsa
```

> If you are having issues on Windows you might want to install a newer version of [OpenSSH](https://github.com/PowerShell/Win32-OpenSSH/wiki/Install-Win32-OpenSSH). The one that comes with windows is outdated and requires a bit more work to get going. The newer version says it's in beta but it's only matter of [Microsoft not yet supporting it officially](https://github.com/PowerShell/Win32-OpenSSH/discussions/2136).
