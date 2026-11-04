package cmd

import (
	"errors"
	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
	"github.com/spf13/cobra"
)

func TestConfigGuidedInputAndAutomation(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("OMDOT_NON_INTERACTIVE", "")
	oldSelect, oldInput := selectConfigKey, inputConfigValue
	t.Cleanup(func() { selectConfigKey, inputConfigValue = oldSelect, oldInput })
	command := &cobra.Command{}
	command.Flags().Bool("interactive", true, "")
	calls := 0
	selectConfigKey = func(question string, values []string) (int, error) { calls++; return 0, nil }
	inputConfigValue = func(string, string) (string, error) { return "", nil }
	key, value, err := resolveConfigInput(command, nil)
	if err != nil || key != "remote-url" || value != "" || calls != 1 {
		t.Fatalf("blank remote selection %q %q %v calls%d", key, value, err, calls)
	}
	selectConfigKey = func(string, []string) (int, error) { return 0, interactive.ErrCancelled }
	if _, _, err := resolveConfigInput(command, []string{"push-compact"}); !errors.Is(err, interactive.ErrCancelled) {
		t.Fatalf("cancel %v", err)
	}
	calls = 0
	selectConfigKey = func(string, []string) (int, error) { calls++; return 0, nil }
	if _, _, err := resolveConfigInput(command, []string{"push-compact", "ask"}); err != nil || calls != 0 {
		t.Fatalf("complete command prompted: %v", err)
	}
	t.Setenv("CI", "true")
	if _, _, err := resolveConfigInput(command, nil); err == nil || calls != 0 {
		t.Fatal("CI prompted or missing args succeeded")
	}
}
