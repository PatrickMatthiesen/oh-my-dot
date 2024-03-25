# oh-my-dot

A cross-platform dotfile manager written in go.

Oh-my-dot is a dotfile manager that helps you manage your dotfiles across multiple machines. It is designed to be simple and easy to use, while still being powerful enough to handle complex configurations. It is written in Go and is cross-platform, so you can use it on Windows, macOS, and Linux.

Oh-my-dot is designed to be used with git, so you can easily store your dotfiles in a git repository and share them with others.
It also supports multiple profiles, so you can have different configurations for different machines or users.

## Install manager

### Release (manual)

1. Find the latest release matching your CPU architecture and Operating system.
2. Download the executable to a persistent folder.
   > A folder you know won't change or randomly be deleted
   >
   > I recommend putting it under `$HOME/oh-my-dot`
3. Add the folder to your PATH
4. Start using oh-my-dot

### CLI (Automatic) TO BE IMPLIMENTED

1. Run in a terminal

```sh
winget install oh-my-dot
apt install oh-my-dot
brew install oh-my-dot
```

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

If you are using SSH to clone your dotfiles, you will need to add your SSH key to the ssh-agent. You can do this by running the following command:

```sh
ssh-add ~/.ssh/id_rsa
```

> If you are having issues on Windows you might want to install a newer version of [OpenSSH](https://github.com/PowerShell/Win32-OpenSSH/wiki/Install-Win32-OpenSSH). The one that comes with windows is outdated and requires a bit more work to get going. The newer version says it's in beta but it's only matter of [Microsoft not yet supporting it officially](https://github.com/PowerShell/Win32-OpenSSH/discussions/2136).
