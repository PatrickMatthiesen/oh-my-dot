package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestResolveGitHubAPIToken(t *testing.T) {
	originalRunner := runGHAuthTokenCommand
	originalHasGitHubCLI := hasGitHubCLI
	originalConfirmUse := confirmGitHubCLIAuthUse
	originalConfirmRemember := confirmRememberGitHubCLIAuth
	originalPersist := persistAllowGitHubCLIAuth
	t.Cleanup(func() {
		runGHAuthTokenCommand = originalRunner
		hasGitHubCLI = originalHasGitHubCLI
		confirmGitHubCLIAuthUse = originalConfirmUse
		confirmRememberGitHubCLIAuth = originalConfirmRemember
		persistAllowGitHubCLIAuth = originalPersist
		viper.Reset()
	})

	newCommand := func(t *testing.T, args ...string) *cobra.Command {
		t.Helper()
		cmd := &cobra.Command{Use: "update"}
		cmd.Flags().Bool("interactive", false, "")
		cmd.Flags().Bool("no-interactive", false, "")
		cmd.Flags().Bool("gh-auth", false, "")
		if len(args) > 0 {
			if err := cmd.ParseFlags(args); err != nil {
				t.Fatalf("ParseFlags() error = %v", err)
			}
		}
		return cmd
	}

	resetState := func() {
		viper.Reset()
		hasGitHubCLI = func() bool { return false }
		confirmGitHubCLIAuthUse = func() (bool, error) { return false, nil }
		confirmRememberGitHubCLIAuth = func() (bool, error) { return false, nil }
		persistAllowGitHubCLIAuth = func(value bool) error { return nil }
	}

	t.Run("prefers GITHUB_TOKEN", func(t *testing.T) {
		resetState()
		t.Setenv("GITHUB_TOKEN", "github-token")
		t.Setenv("GH_TOKEN", "gh-token")
		hasGitHubCLI = func() bool { return true }

		called := false
		runGHAuthTokenCommand = func(ctx context.Context) (string, error) {
			called = true
			return "cli-token", nil
		}

		token := resolveGitHubAPIToken(context.Background(), newCommand(t))
		if token != "github-token" {
			t.Fatalf("resolveGitHubAPIToken() = %q, want %q", token, "github-token")
		}
		if called {
			t.Fatal("expected gh auth token command not to be called")
		}
	})

	t.Run("uses GH_TOKEN when GITHUB_TOKEN is unset", func(t *testing.T) {
		resetState()
		t.Setenv("GITHUB_TOKEN", "")
		t.Setenv("GH_TOKEN", "gh-token")
		hasGitHubCLI = func() bool { return true }

		called := false
		runGHAuthTokenCommand = func(ctx context.Context) (string, error) {
			called = true
			return "cli-token", nil
		}

		token := resolveGitHubAPIToken(context.Background(), newCommand(t))
		if token != "gh-token" {
			t.Fatalf("resolveGitHubAPIToken() = %q, want %q", token, "gh-token")
		}
		if called {
			t.Fatal("expected gh auth token command not to be called")
		}
	})

	t.Run("uses gh auth token when forced by flag", func(t *testing.T) {
		resetState()
		t.Setenv("GITHUB_TOKEN", "")
		t.Setenv("GH_TOKEN", "")
		hasGitHubCLI = func() bool { return true }
		confirmGitHubCLIAuthUse = func() (bool, error) {
			t.Fatal("expected interactive confirmation not to run")
			return false, nil
		}

		runGHAuthTokenCommand = func(ctx context.Context) (string, error) {
			return " cli-token \n", nil
		}

		token := resolveGitHubAPIToken(context.Background(), newCommand(t, "--gh-auth"))
		if token != "cli-token" {
			t.Fatalf("resolveGitHubAPIToken() = %q, want %q", token, "cli-token")
		}
	})

	t.Run("uses gh auth token when config allows it", func(t *testing.T) {
		resetState()
		t.Setenv("GITHUB_TOKEN", "")
		t.Setenv("GH_TOKEN", "")
		hasGitHubCLI = func() bool { return true }
		viper.Set(allowGHAuthConfigKey, true)
		confirmGitHubCLIAuthUse = func() (bool, error) {
			t.Fatal("expected interactive confirmation not to run")
			return false, nil
		}

		runGHAuthTokenCommand = func(ctx context.Context) (string, error) {
			return "cli-token", nil
		}

		token := resolveGitHubAPIToken(context.Background(), newCommand(t))
		if token != "cli-token" {
			t.Fatalf("resolveGitHubAPIToken() = %q, want %q", token, "cli-token")
		}
	})

	t.Run("prompts and remembers gh auth preference", func(t *testing.T) {
		resetState()
		t.Setenv("GITHUB_TOKEN", "")
		t.Setenv("GH_TOKEN", "")
		hasGitHubCLI = func() bool { return true }

		confirmed := 0
		confirmGitHubCLIAuthUse = func() (bool, error) {
			confirmed++
			return true, nil
		}
		confirmRememberGitHubCLIAuth = func() (bool, error) {
			return true, nil
		}

		persisted := false
		persistAllowGitHubCLIAuth = func(value bool) error {
			persisted = value
			return nil
		}

		runGHAuthTokenCommand = func(ctx context.Context) (string, error) {
			return "cli-token", nil
		}

		token := resolveGitHubAPIToken(context.Background(), newCommand(t, "--interactive"))
		if token != "cli-token" {
			t.Fatalf("resolveGitHubAPIToken() = %q, want %q", token, "cli-token")
		}
		if confirmed != 1 {
			t.Fatalf("expected one interactive confirmation, got %d", confirmed)
		}
		if !persisted {
			t.Fatal("expected gh auth preference to be persisted")
		}
	})

	t.Run("returns empty when gh auth token fails", func(t *testing.T) {
		resetState()
		t.Setenv("GITHUB_TOKEN", "")
		t.Setenv("GH_TOKEN", "")
		hasGitHubCLI = func() bool { return true }
		confirmGitHubCLIAuthUse = func() (bool, error) {
			return true, nil
		}
		confirmRememberGitHubCLIAuth = func() (bool, error) {
			return false, nil
		}

		runGHAuthTokenCommand = func(ctx context.Context) (string, error) {
			return "", errors.New("gh unavailable")
		}

		token := resolveGitHubAPIToken(context.Background(), newCommand(t, "--interactive"))
		if token != "" {
			t.Fatalf("resolveGitHubAPIToken() = %q, want empty string", token)
		}
	})
}
