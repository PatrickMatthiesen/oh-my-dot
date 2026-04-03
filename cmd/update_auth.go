package cmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const ghAuthTokenTimeout = 10 * time.Second

const allowGHAuthConfigKey = "update.allow-gh-auth"

var runGHAuthTokenCommand = func(ctx context.Context) (string, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, "gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

var hasGitHubCLI = func() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

var confirmGitHubCLIAuthUse = func() (bool, error) {
	fileops.ColorPrintfn(fileops.Cyan, "Hint: use --gh-auth to skip this prompt, or answer yes to remember this choice in config")
	return interactive.Confirm("Use your GitHub CLI authentication for release checks?", false)
}

var confirmRememberGitHubCLIAuth = func() (bool, error) {
	return interactive.Confirm("Always allow GitHub CLI authentication for update checks?", false)
}

var persistAllowGitHubCLIAuth = func(value bool) error {
	viper.Set(allowGHAuthConfigKey, value)
	return viper.WriteConfig()
}

func resolveGitHubAPIToken(ctx context.Context, cmd *cobra.Command) string {
	for _, envVar := range []string{"GITHUB_TOKEN", "GH_TOKEN"} {
		token := strings.TrimSpace(os.Getenv(envVar))
		if token != "" {
			return token
		}
	}

	if !shouldUseGitHubCLIAuth(ctx, cmd) {
		return ""
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, ghAuthTokenTimeout)
	defer cancel()

	token, err := runGHAuthTokenCommand(timeoutCtx)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(token)
}

func shouldUseGitHubCLIAuth(ctx context.Context, cmd *cobra.Command) bool {
	if isGitHubCLIAuthForced(cmd) || viper.GetBool(allowGHAuthConfigKey) {
		return hasGitHubCLI()
	}

	if cmd == nil || !hasGitHubCLI() || !interactive.ShouldPrompt(cmd, false) {
		return false
	}

	allowed, err := confirmGitHubCLIAuthUse()
	if err != nil || !allowed {
		return false
	}

	remember, err := confirmRememberGitHubCLIAuth()
	if err == nil && remember {
		if writeErr := persistAllowGitHubCLIAuth(true); writeErr != nil {
			fileops.ColorPrintfn(fileops.Yellow, "Could not save GitHub CLI auth preference: %v", writeErr)
		}
	}

	return true
}

func isGitHubCLIAuthForced(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}

	forced, err := cmd.Flags().GetBool("gh-auth")
	if err != nil {
		return false
	}

	return forced
}

func printGitHubRateLimitHint(apiToken string, err error) {
	if apiToken != "" || err == nil {
		return
	}

	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(strings.ToLower(err.Error()), "rate limit") {
		return
	}

	fileops.ColorPrintfn(fileops.Yellow,
		"Warning: GitHub API token not found. This may cause update checks to hit GitHub's anonymous rate limits.")
	fileops.ColorPrintfn(fileops.Yellow, `
To fix this, choose one option:

  Option 1: Use GitHub CLI (recommended)
    • Ensure "gh" is installed and authenticated: gh auth login
    • Then allow GitHub CLI authentication in the interactive update prompt
    • Or pass --gh-auth to skip the prompt

  Option 2: Set an environment variable
    • Export GITHUB_TOKEN or GH_TOKEN with a valid GitHub personal access token

To silence this warning:
  • Run: oh-my-dot config update allow-gh-auth true
  • Or pass --gh-auth on update commands
`)

}
