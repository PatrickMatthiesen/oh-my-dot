package shell

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInitDefersUntilFirstPrompt(t *testing.T) {
	tests := []struct {
		name       string
		shell      string
		setupHooks string
		appendHook string
		listHooks  string
	}{
		{
			name:       "bash scalar",
			shell:      "bash",
			setupHooks: `PROMPT_COMMAND='user_prompt "$?" # keep this comment'`,
			appendHook: `PROMPT_COMMAND="$PROMPT_COMMAND"$'\n'second_prompt`,
			listHooks:  `printf '%s' "${PROMPT_COMMAND-}"`,
		},
		{
			name:       "bash scalar semicolon",
			shell:      "bash",
			setupHooks: `PROMPT_COMMAND='user_prompt "$?" # keep this comment'`,
			appendHook: `PROMPT_COMMAND="$PROMPT_COMMAND; second_prompt"`,
			listHooks:  `printf '%s' "${PROMPT_COMMAND-}"`,
		},
		{
			name:       "bash array",
			shell:      "bash",
			setupHooks: `PROMPT_COMMAND=([2]='user_prompt "$?"' [5]='printf "second-hook\n"')`,
			listHooks:  `declare -p PROMPT_COMMAND`,
		},
		{
			name:  "zsh precmd",
			shell: "zsh",
			setupHooks: `precmd() { printf 'bare-hook\n'; }
precmd_functions=( 'user_prompt' 'second_prompt' )`,
			listHooks: `printf '%s\n' "${precmd_functions[@]}"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executable := findPromptTestShell(t, tt.shell)
			if tt.name == "bash array" {
				check := exec.Command(executable, "--noprofile", "--norc", "-c",
					"(( BASH_VERSINFO[0] > 5 || (BASH_VERSINFO[0] == 5 && BASH_VERSINFO[1] >= 1) ))")
				if err := check.Run(); err != nil {
					t.Skip("PROMPT_COMMAND array processing requires Bash 5.1 or newer")
				}
			}

			dir := writePromptTestInit(t, tt.shell, tt.listHooks)
			rc := `user_prompt() { printf 'user-hook:%s\n' "${1:-0}"; }
second_prompt() { printf 'second-hook\n'; }
` + tt.setupHooks + `
. ./init
printf 'startup:%s:%s\n' "$eager_runs" "${deferred_runs:-0}"
PS1=reset
. ./init
printf 'startup-resource:%s:%s:%s\n' "$eager_runs" "${deferred_runs:-0}" "$PS1"
` + tt.appendHook + "\n"
			args := []string{"-d", "-i"}
			rcName := ".zshrc"
			if tt.shell == "bash" {
				args = []string{"--noprofile", "--rcfile", "./bashrc", "-i"}
				rcName = "bashrc"
			}
			if err := os.WriteFile(filepath.Join(dir, rcName), []byte(rc), 0o600); err != nil {
				t.Fatal(err)
			}
			output := runPromptTestShell(t, executable, dir, args, `deferred_command
PS1=reset; . ./init; printf 'prompt-resource:%s:%s:%s\n' "$eager_runs" "$deferred_runs" "$PS1"
false
exit 0
`)
			want := "startup:1:0\nstartup-resource:2:0:eager> \n"
			if !strings.HasPrefix(output, want) {
				t.Errorf("expected eager re-source and no defer during startup, got:\n%s", output)
			}
			for _, marker := range []string{"deferred\n", "deferred-command\n", "prompt-resource:3:1:eager> \n"} {
				if strings.Count(output, marker) != 1 {
					t.Errorf("expected one %q, got:\n%s", marker, output)
				}
			}
			if strings.Contains(output, "callback-still-registered\n") {
				t.Errorf("deferred callback remained registered during feature sourcing:\n%s", output)
			}
			for _, hook := range []string{"user-hook:", "second-hook\n", "bare-hook\n"} {
				wantCount := 4
				if hook == "bare-hook\n" && tt.shell == "bash" {
					wantCount = 0
				}
				if strings.Count(output, hook) != wantCount {
					t.Errorf("expected %d instances of %q, got:\n%s", wantCount, hook, output)
				}
			}
			if tt.shell == "bash" && !strings.Contains(output, "user-hook:1\n") {
				t.Errorf("Bash user hook did not receive the failed command status:\n%s", output)
			}
		})
	}
}

func TestInitSkipsDeferredFeaturesInNonInteractiveShell(t *testing.T) {
	for _, shellName := range []string{"bash", "zsh"} {
		t.Run(shellName, func(t *testing.T) {
			executable := findPromptTestShell(t, shellName)
			dir := writePromptTestInit(t, shellName, ":")
			output := runPromptTestShell(t, executable, dir, []string{"-c", `
. ./init
. ./init
printf '%s:%s\n' "$eager_runs" "${deferred_runs:-0}"
`}, "")
			if output != "2:0\n" {
				t.Errorf("expected eager features twice and deferred features zero times, got %q", output)
			}
		})
	}
}

func findPromptTestShell(t *testing.T, shellName string) string {
	t.Helper()
	executable, err := exec.LookPath(shellName)
	if err != nil {
		t.Skipf("%s unavailable: %v", shellName, err)
	}
	if isWslBashLauncher(executable) {
		t.Skip("native Bash unavailable; WSL uses a separate filesystem")
	}
	return executable
}

func writePromptTestInit(t *testing.T, shellName, listHooks string) string {
	t.Helper()
	dir := t.TempDir()
	features := FeaturesByStrategy{Eager: []string{"eager"}, Defer: []string{"deferred"}}
	init := generateBashInit(features)
	extension := ".sh"
	if shellName == "zsh" {
		init = generateZshInit(features)
		extension = ".zsh"
	}
	if err := os.Mkdir(filepath.Join(dir, "features"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"init": init,
		"features/eager" + extension: `eager_runs=$(( ${eager_runs:-0} + 1 ))
PS1='eager> '
`,
		"features/deferred" + extension: `case "$(` + listHooks + `)" in
  *_omd_load_deferred_features*) printf 'callback-still-registered\n' ;;
esac
deferred_runs=$(( ${deferred_runs:-0} + 1 ))
deferred_command() { printf 'deferred-command\n'; }
printf 'deferred\n'
`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func runPromptTestShell(t *testing.T, executable, dir string, args []string, input string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = dir
	shellDir := filepath.ToSlash(dir)
	cmd.Env = []string{"HOME=" + shellDir, "ZDOTDIR=" + shellDir, "TERM=dumb", "PATH=" + os.Getenv("PATH")}
	cmd.Stdin = strings.NewReader(input)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("shell failed: %v\nstdout:\n%s\nstderr:\n%s", err, output, stderr.String())
	}
	return strings.ReplaceAll(string(output), "\r\n", "\n")
}
