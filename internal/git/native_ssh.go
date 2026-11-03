package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/storage/memory"
)

// IsSSHRemote reports whether a URL uses SSH, including Git's scp-style syntax.
func IsSSHRemote(remoteURL string) bool {
	if strings.HasPrefix(remoteURL, "ssh://") {
		return true
	}
	return !strings.Contains(remoteURL, "://") && strings.Contains(remoteURL, ":") && !filepathLike(remoteURL)
}

func filepathLike(value string) bool {
	return strings.HasPrefix(value, "/") || strings.HasPrefix(value, "./") || strings.HasPrefix(value, "../") || (len(value) > 1 && value[1] == ':' && ((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')))
}

// runRemoteGit uses Git's own SSH configuration, including IdentityFile,
// ProxyCommand, core.sshCommand, GIT_SSH_COMMAND, and Windows agent selection.
// It never retries with another protocol or changes authentication settings.
func runRemoteGit(path string, timeout time.Duration, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = path
	command.WaitDelay = time.Second
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=never", "GIT_SSH_COMMAND="+batchSSHCommand(path))
	var stderr strings.Builder
	command.Stderr = &stderr
	output, err := command.Output()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("Git remote operation timed out: %w", ctx.Err())
	}
	if err != nil {
		return nil, fmt.Errorf("Git remote operation: %w\n%s", err, strings.TrimSpace(stderr.String()))
	}
	return output, nil
}

// CloneSSH clones through native Git, accepting a reachable empty remote.
func CloneSSH(path, remoteURL string) (*gogit.Repository, error) {
	if _, err := runRemoteGit("", 2*time.Minute, "clone", "--", remoteURL, path); err != nil {
		return nil, err
	}
	return GetRepository(path)
}

// FetchSSH fetches through native Git without changing the configured origin.
func FetchSSH(path, remoteURL string, refspecs ...string) error {
	args := []string{"fetch", "--", remoteURL}
	args = append(args, refspecs...)
	_, err := runRemoteGit(path, 2*time.Minute, args...)
	return err
}

// ListRemote checks read access using the same SSH client as native Git.
// HTTPS and local remotes retain the embedded transport and credential helpers.
func ListRemote(path, remoteURL string, timeoutSeconds int) ([]*plumbing.Reference, error) {
	if !IsSSHRemote(remoteURL) {
		remote := gogit.NewRemote(memory.NewStorage(), &config.RemoteConfig{Name: "origin", URLs: []string{remoteURL}})
		return remote.List(&gogit.ListOptions{Timeout: timeoutSeconds, ClientOptions: RemoteClientOptions(path, remoteURL)})
	}
	output, err := runRemoteGit(path, time.Duration(timeoutSeconds)*time.Second, "ls-remote", "--symref", "--", remoteURL)
	if err != nil {
		return nil, err
	}
	return parseRemoteRefs(string(output))
}

func parseRemoteRefs(output string) ([]*plumbing.Reference, error) {
	var refs []*plumbing.Reference
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		value, name, ok := strings.Cut(line, "\t")
		if !ok {
			return nil, fmt.Errorf("invalid remote reference advertisement")
		}
		if target, symbolic := strings.CutPrefix(value, "ref: "); symbolic {
			refs = append(refs, plumbing.NewSymbolicReference(plumbing.ReferenceName(name), plumbing.ReferenceName(target)))
		} else {
			if len(value) != 40 {
				return nil, fmt.Errorf("invalid remote object ID")
			}
			refs = append(refs, plumbing.NewHashReference(plumbing.ReferenceName(name), plumbing.NewHash(value)))
		}
	}
	return refs, nil
}

// Preserve the user's SSH command while disabling terminal authentication/trust
// prompts. Git executes this trusted configuration through its own shell.
func batchSSHCommand(path string) string {
	command := os.Getenv("GIT_SSH_COMMAND")
	if command == "" {
		configCommand := exec.Command("git", "config", "--get", "core.sshCommand")
		configCommand.Dir = path
		if output, err := configCommand.Output(); err == nil {
			command = strings.TrimSpace(string(output))
		}
	}
	if command == "" {
		if executable := os.Getenv("GIT_SSH"); executable != "" {
			command = "'" + strings.ReplaceAll(filepath.ToSlash(executable), "'", "'\"'\"'") + "'"
		} else {
			command = "ssh"
		}
	}
	variant := os.Getenv("GIT_SSH_VARIANT")
	if variant == "" {
		configCommand := exec.Command("git", "config", "--get", "ssh.variant")
		configCommand.Dir = path
		if output, err := configCommand.Output(); err == nil {
			variant = strings.TrimSpace(string(output))
		}
	}
	if variant == "plink" || variant == "tortoiseplink" || strings.Contains(strings.ToLower(command), "plink") {
		return command + " -batch"
	}
	return command + " -oBatchMode=yes"
}
