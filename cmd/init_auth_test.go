package cmd

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
)

func TestGithubSSHURL(t *testing.T) {
	for _, tt := range []struct{ url, want string }{
		{"https://github.com/user/repo", "git@github.com:user/repo.git"},
		{"https://github.com/user/repo.git", "git@github.com:user/repo.git"},
		{"https://other.example/user/repo", ""}, {"https://github.com/user/repo?token=x", ""},
		{"https://github.com/user/repo%0ahost=x", ""}, {"https://github.com/user/repo/extra", ""},
	} {
		t.Run(tt.url, func(t *testing.T) {
			got, ok := githubSSHURL(tt.url)
			if got != tt.want || ok != (tt.want != "") {
				t.Fatalf("got %q, %v", got, ok)
			}
		})
	}
}

func TestInitAuthSelectsOnceWithoutNetworkRetries(t *testing.T) {
	oldOutput, oldSelect, oldPrepare := initAuthOutput, selectInitAuth, prepareInitCredentials
	t.Cleanup(func() { initAuthOutput, selectInitAuth, prepareInitCredentials = oldOutput, oldSelect, oldPrepare })
	for _, tt := range []struct {
		name, protocol, helper string
		interactive            bool
		choice                 int
		want                   string
		prompts                int
	}{
		{"saved ssh", "ssh", "", false, 0, "git@github.com:user/repo.git", 0},
		{"existing helper", "", "manager", true, 1, "https://github.com/user/repo", 0},
		{"user selects ssh", "", "", true, 1, "git@github.com:user/repo.git", 1},
		{"user selects https", "", "", true, 0, "https://github.com/user/repo", 1},
		{"noninteractive retains URL", "", "", false, 0, "https://github.com/user/repo", 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			prepareInitCredentials = func(string, string) bool { return tt.helper != "" }
			initAuthOutput = func(name string, args ...string) ([]byte, error) {
				if name == "gh" && tt.protocol != "" {
					return []byte(tt.protocol), nil
				}
				if name == "git" && tt.helper != "" {
					return []byte(tt.helper), nil
				}
				return nil, errors.New("not configured")
			}
			selectInitAuth = func(_ string, _ []string) (int, error) { calls++; return tt.choice, nil }
			cmd := &cobra.Command{}
			cmd.Flags().Bool("interactive", tt.interactive, "")
			cmd.Flags().Bool("no-interactive", !tt.interactive, "")
			got, err := ResolveInitRemote(cmd, "https://github.com/user/repo")
			if err != nil || got != tt.want || calls != tt.prompts {
				t.Fatalf("got %q, %v, prompts %d", got, err, calls)
			}
		})
	}
}
