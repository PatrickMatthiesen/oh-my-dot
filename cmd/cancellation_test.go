package cmd

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
)

func TestExecuteCancellation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		cancel bool
	}{
		{"cancelled", interactive.ErrCancelled, true},
		{"wrapped", fmt.Errorf("select files: %w", interactive.ErrCancelled), true},
		{"joined cancellations", errors.Join(interactive.ErrCancelled, fmt.Errorf("choose name: %w", interactive.ErrCancelled)), true},
		{"mixed failure", errors.Join(errors.New("disk full"), interactive.ErrCancelled), false},
		{"real failure", errors.New("disk full"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			viper.Set("initialized", true)
			t.Cleanup(viper.Reset)
			command := &cobra.Command{Use: "cancel-test", RunE: func(*cobra.Command, []string) error { return tc.err }}
			rootCmd.AddCommand(command)
			t.Cleanup(func() { rootCmd.RemoveCommand(command) })
			output, err := captureExecuteOutput(t, []string{"cancel-test"})
			output = stripANSICodes(output)
			if tc.cancel {
				if err != nil || !strings.Contains(output, "Cancelled") || strings.Contains(output, "Error:") || strings.Contains(output, "Usage:") {
					t.Fatalf("error=%v, output=%q", err, output)
				}
			} else if err == nil || !strings.Contains(output, "disk full") {
				t.Fatalf("real error lost: %v, %q", err, output)
			}
		})
	}
}
