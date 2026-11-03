package git

import (
	"github.com/spf13/viper"
	"os"
	"runtime"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
)

// DisplaySSHAgentError displays a helpful error message when SSH agent is not configured
// exitOnError: if true, exits the program; if false, just displays a warning
func DisplaySSHAgentError(exitOnError bool) {
	fileops.ColorPrintln("⚠ "+sshAgentHelp(runtime.GOOS), fileops.Yellow)

	if exitOnError {
		os.Exit(1)
	}
}

// CheckRemoteAccessWithHelp checks remote push permissions and provides helpful error messages
// exitOnError: if true, exits on error; if false, displays warning and continues
func CheckRemoteAccessWithHelp(exitOnError bool) {
	hasRemote, err := HasOriginRemote()
	if err != nil {
		if exitOnError {
			fileops.ColorPrintfn(fileops.Red, "Error: %s", err)
			os.Exit(1)
		}
		fileops.ColorPrintfn(fileops.Yellow, "Warning: Unable to inspect remote configuration: %s", err)
		return
	}
	if !hasRemote {
		if exitOnError {
			fileops.ColorPrintln("Error: no remote 'origin' configured", fileops.Red)
			fileops.ColorPrintln("Configure a remote repository before using this command.", fileops.Yellow)
			os.Exit(1)
		}
		return
	}

	if err := CheckRemotePushPermission(); err != nil {
		fileops.ColorPrintfn(fileops.Yellow, "Remote access failed: %s\n%s", err, RemoteErrorHelp(err, viper.GetString("remote-url")))
		if exitOnError {
			os.Exit(1)
		}
	}
}

// sshAgentHelp describes recovery without claiming an unreachable agent is absent.
func sshAgentHelp(platform string) string {
	if platform == "windows" {
		return `SSH agent unavailable; SSH authentication cannot continue.
Check the service in PowerShell: Get-Service ssh-agent
If installed, run in an administrator PowerShell:
  Set-Service -Name ssh-agent -StartupType Manual
  Start-Service ssh-agent
Then load your key in your normal PowerShell:
  ssh-add "$HOME\.ssh\id_ed25519"
Replace the key path with your existing private key.
If the service is missing, install the OpenSSH Client in administrator PowerShell:
  Add-WindowsCapability -Online -Name OpenSSH.Client~~~~0.0.1.0
Then start the service, load your key, and retry the command.`
	}
	return `SSH agent unavailable; SSH authentication cannot continue.
Check that ssh-agent and ssh-add are installed and your shell can reach the agent.
In bash or zsh, start an agent and load your existing key:
  eval "$(ssh-agent -s)"
  ssh-add ~/.ssh/id_ed25519
Replace the key path with your existing private key.
If the commands are missing, install your operating system's OpenSSH client package.
Then retry the command from the shell connected to the agent.`
}
