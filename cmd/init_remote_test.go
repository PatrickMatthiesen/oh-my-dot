package cmd

import (
	"fmt"
	"testing"

	"github.com/spf13/cobra"
)

func TestSelectForcedInitRemote(t *testing.T) {
	oldConfirm, oldInput := confirmInitRemote, inputInitRemote
	t.Cleanup(func() { confirmInitRemote, inputInitRemote = oldConfirm, oldInput })
	tests := []struct {
		name                                 string
		explicit, interactive, reuse, cancel bool
		requested, saved, input, want        string
		wantError                            bool
	}{
		{name: "explicit replacement", explicit: true, requested: "new", saved: "old", want: "new"},
		{name: "confirm reuse", interactive: true, reuse: true, saved: "old", want: "old"},
		{name: "replace saved URL", interactive: true, saved: "old", input: "new", want: "new"},
		{name: "ask without saved URL", interactive: true, input: "new", want: "new"},
		{name: "noninteractive requires URL", saved: "old", wantError: true},
		{name: "cancel", interactive: true, saved: "old", cancel: true, wantError: true},
		{name: "empty replacement", interactive: true, saved: "old", wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command := &cobra.Command{}
			command.Flags().Bool("interactive", tt.interactive, "")
			command.Flags().Bool("no-interactive", !tt.interactive, "")
			confirmInitRemote = func(question string) (bool, error) {
				if tt.cancel {
					return false, fmt.Errorf("cancelled")
				}
				return tt.reuse, nil
			}
			inputInitRemote = func(question, defaultValue string) (string, error) { return tt.input, nil }
			got, err := selectForcedInitRemote(command, tt.requested, tt.saved, tt.explicit)
			if got != tt.want || (err != nil) != tt.wantError {
				t.Fatalf("selection = %q, %v; want %q, error %v", got, err, tt.want, tt.wantError)
			}
		})
	}
}
