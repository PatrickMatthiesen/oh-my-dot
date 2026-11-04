package git

import (
	"context"
	"errors"
	"net"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// RemoteErrorHelp gives recovery instructions for the failed stage, without
// recommending agent setup for a connection that never reached authentication.
func RemoteErrorHelp(err error, remoteURL string) string {
	return remoteErrorHelp(err, remoteURL, runtime.GOOS, windowsAgentState)
}

func remoteErrorHelp(err error, remoteURL, platform string, agentState func() string) string {
	if err == nil {
		return ""
	}
	message := strings.ToLower(err.Error())
	var networkError net.Error
	switch {
	case errors.Is(err, context.DeadlineExceeded) || errors.As(err, &networkError) && networkError.Timeout() || strings.Contains(message, "deadline exceeded") || strings.Contains(message, "timed out") || strings.Contains(message, "i/o timeout"):
		return "The connection timed out before remote access could be verified. Check your network, VPN, firewall, and the remote host/port. Retry with git ls-remote origin from the repository to compare. Agent configuration will not fix a TCP timeout."
	case strings.Contains(message, "key mismatch") || strings.Contains(message, "host identification has changed"):
		return "The SSH host key changed. Verify the new fingerprint with your Git host before changing known_hosts; do not disable host key checking."
	case strings.Contains(message, "known_hosts") || strings.Contains(message, "host key"):
		return "SSH host trust could not be verified.\n  Run " + sshHelpCommand(remoteURL) + " in your terminal.\n  Compare the fingerprint with your Git host's published fingerprint before accepting it, then retry.\n  Do not disable host key checking."
	case IsSSHAgentError(err):
		if platform == "windows" {
			switch agentState() {
			case "missing":
				return "The Windows OpenSSH agent service is not installed.\n  In administrator PowerShell:\n    Add-WindowsCapability -Online -Name OpenSSH.Client~~~~0.0.1.0\n  Then enable/start ssh-agent and load your existing key with ssh-add."
			case "Stopped":
				return "The Windows OpenSSH agent service is stopped.\n  In administrator PowerShell:\n    Set-Service -Name ssh-agent -StartupType Manual\n    Start-Service ssh-agent\n  In your normal PowerShell:\n    ssh-add \"$HOME\\.ssh\\id_ed25519\"\n  Replace the path with your existing private key."
			case "Running":
				return "The Windows SSH agent service is running, but authentication could not reach a usable key.\n  In your normal PowerShell, run ssh-add -l, then ssh-add with your existing private key if needed.\n  Check that Git uses the SSH client connected to this agent."
			}
		}
		return sshAgentHelp(platform)
	case IsSSHRemote(remoteURL) && (strings.Contains(message, "permission denied") || strings.Contains(message, "authentication")):
		return "SSH authentication was rejected. Run ssh-add -l to inspect loaded keys and " + sshHelpCommand(remoteURL) + " to verify your account. Confirm that account has access to this repository."
	case strings.HasPrefix(remoteURL, "https://") && (strings.Contains(message, "401") || strings.Contains(message, "403") || strings.Contains(message, "authentication") || strings.Contains(message, "repository not found")):
		return "HTTPS access was rejected. The repository may be private, the URL may be incorrect, or your account may lack access.\n  Configure Git's credential helper and sign in. For GitHub CLI, run gh auth login, then gh auth setup-git.\n  If you use SSH keys, select SSH during init or set an SSH origin with oh-my-dot config set remote-url."
	default:
		return "Check the remote URL, network connection, and repository access for your account."
	}
}

func sshHelpCommand(remoteURL string) string {
	user, host, port := "git", "", ""
	if strings.HasPrefix(remoteURL, "ssh://") {
		if endpoint, err := url.Parse(remoteURL); err == nil {
			host, port = endpoint.Hostname(), endpoint.Port()
			if endpoint.User != nil {
				user = endpoint.User.Username()
			}
		}
	} else {
		host, _, _ = strings.Cut(remoteURL, ":")
		if prefix, suffix, ok := strings.Cut(host, "@"); ok {
			user, host = prefix, suffix
		}
	}
	if host == "" {
		return "ssh -T git@<your-git-host>"
	}
	for _, value := range []string{user, host, port} {
		if strings.HasPrefix(value, "-") {
			return "ssh -T git@<your-git-host>"
		}
		for _, r := range value {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._-:", r)) {
				return "ssh -T git@<your-git-host>"
			}
		}
	}
	command := "ssh -T "
	if port != "" {
		command += "-p " + port + " "
	}
	return command + user + "@" + host
}

func windowsAgentState() string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "$service = Get-Service -Name ssh-agent -ErrorAction SilentlyContinue; if ($null -eq $service) { 'missing' } else { $service.Status.ToString() }")
	output, err := command.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}
