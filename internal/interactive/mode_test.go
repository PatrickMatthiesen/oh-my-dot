package interactive

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestInteractionMode(t *testing.T) {
	flags := []struct {
		name          string
		interactive   bool
		noInteractive bool
	}{
		{name: "automatic"},
		{name: "force interactive", interactive: true},
		{name: "force noninteractive", noInteractive: true},
		// Cobra rejects this combination in real commands. Mode selection keeps
		// its existing precedence even when called before flag validation.
		{name: "both flags", interactive: true, noInteractive: true},
	}
	environments := []struct {
		name string
		ci   string
		omd  string
	}{
		{name: "no environment"},
		{name: "CI", ci: "true"},
		{name: "OMDOT_NON_INTERACTIVE", omd: "1"},
		{name: "both environment variables", ci: "true", omd: "1"},
		{name: "nonempty false still disables", ci: "false", omd: "false"},
	}
	for _, flag := range flags {
		for _, environment := range environments {
			for _, stdinTTY := range []bool{false, true} {
				for _, stdoutTTY := range []bool{false, true} {
					name := fmt.Sprintf("%s/%s/stdin=%t/stdout=%t", flag.name, environment.name, stdinTTY, stdoutTTY)
					t.Run(name, func(t *testing.T) {
						t.Setenv("CI", environment.ci)
						t.Setenv("OMDOT_NON_INTERACTIVE", environment.omd)
						original := isTerminal
						t.Cleanup(func() { isTerminal = original })
						isTerminal = func(fd uintptr) bool {
							switch fd {
							case os.Stdin.Fd():
								return stdinTTY
							case os.Stdout.Fd():
								return stdoutTTY
							default:
								t.Fatalf("unexpected terminal descriptor %d", fd)
								return false
							}
						}
						cmd := &cobra.Command{Use: "test"}
						cmd.Flags().Bool("interactive", flag.interactive, "")
						cmd.Flags().Bool("no-interactive", flag.noInteractive, "")
						want := ModeAuto
						if flag.interactive {
							want = ModeInteractive
						} else if flag.noInteractive || environment.ci != "" || environment.omd != "" || !stdinTTY || !stdoutTTY {
							want = ModeNonInteractive
						}
						if got := GetMode(cmd); got != want {
							t.Errorf("GetMode() = %v, want %v", got, want)
						}
						err := ValidateMode(cmd)
						wantError := flag.interactive && (!stdinTTY || !stdoutTTY)
						if (err != nil) != wantError {
							t.Errorf("ValidateMode() error = %v, want error %t", err, wantError)
						}
						if wantError && err != nil {
							for _, hint := range []string{"stdin", "stdout", "--no-interactive", "terminal"} {
								if !strings.Contains(err.Error(), hint) {
									t.Errorf("validation error %q lacks %q", err, hint)
								}
							}
						}
						for _, hasInfo := range []bool{false, true} {
							wantPrompt := want == ModeInteractive || (want == ModeAuto && !hasInfo)
							if got := ShouldPrompt(cmd, hasInfo); got != wantPrompt {
								t.Errorf("ShouldPrompt(hasRequiredInfo=%t) = %t, want %t", hasInfo, got, wantPrompt)
							}
						}
					})
				}
			}
		}
	}
}
