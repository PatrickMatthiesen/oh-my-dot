package cmd

import (
	"context"
	"fmt"
	gitops "github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/spf13/viper"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
	"github.com/spf13/cobra"
)

var prepareInitCredentials = gitops.PrepareRemoteCredentials

var selectInitAuth = interactive.PromptSelect
var initAuthOutput = func(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Output()
}
var runInitGH = func(args ...string) error {
	command := exec.Command("gh", args...)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("GitHub CLI setup: %w", err)
	}
	return nil
}

// ResolveInitRemote selects one GitHub authentication method before connecting.
// Existing helper credentials and gh's saved SSH preference avoid extra prompts.
// Other hosts and explicit SSH URLs retain the supplied transport.
func ResolveInitRemote(cmd *cobra.Command, remoteURL string) (string, error) {
	sshURL, ok := githubSSHURL(remoteURL)
	if !ok {
		return remoteURL, nil
	}
	if protocol, err := initAuthOutput("gh", "config", "get", "git_protocol", "--host", "github.com"); err == nil && strings.TrimSpace(string(protocol)) == "ssh" {
		fmt.Println("Using SSH for GitHub, as configured in gh.")
		return sshURL, nil
	}
	credentialPath := viper.GetString("repo-path")
	if info, err := os.Stat(credentialPath); err != nil || !info.IsDir() {
		credentialPath = filepath.Dir(credentialPath)
	}
	if prepareInitCredentials(credentialPath, remoteURL) {
		return remoteURL, nil
	}
	if !interactive.ShouldPrompt(cmd, false) {
		return remoteURL, nil
	}
	methods := []string{"HTTPS (public repository or existing Git credentials)", "SSH (use your existing SSH keys)"}
	_, ghErr := exec.LookPath("gh")
	if ghErr == nil {
		methods = append(methods, "GitHub CLI (sign in with gh and configure HTTPS authentication)")
	}
	choice, err := selectInitAuth("How should oh-my-dot connect to GitHub?", methods)
	if err != nil {
		return "", fmt.Errorf("authentication selection cancelled: %w", err)
	}
	switch choice {
	case 0:
		return remoteURL, nil
	case 1:
		gitops.ForgetPreparedCredentials(remoteURL)
		return sshURL, nil
	case 2:
		if ghErr != nil {
			return "", fmt.Errorf("GitHub CLI is not installed")
		}
		if _, err := initAuthOutput("gh", "auth", "status", "--active", "--hostname", "github.com"); err != nil {
			if err := runInitGH("auth", "login", "--hostname", "github.com", "--git-protocol", "https", "--web"); err != nil {
				return "", err
			}
		}
		if err := runInitGH("auth", "setup-git", "--hostname", "github.com"); err != nil {
			return "", err
		}
		gitops.ForgetPreparedCredentials(remoteURL)
		return remoteURL, nil
	default:
		return "", fmt.Errorf("invalid authentication selection")
	}
}

func githubSSHURL(raw string) (string, bool) {
	endpoint, err := url.Parse(raw)
	if err != nil || endpoint.Scheme != "https" || endpoint.Host != "github.com" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return "", false
	}
	path := strings.TrimSuffix(strings.Trim(endpoint.Path, "/"), ".git")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", false
	}
	for _, part := range parts {
		for _, r := range part {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("-_.", r)) {
				return "", false
			}
		}
	}
	return "git@github.com:" + path + ".git", true
}
