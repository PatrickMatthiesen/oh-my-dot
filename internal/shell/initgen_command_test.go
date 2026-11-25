package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
)

func TestOnCommandInit(t *testing.T) {
	for _, shellName := range []string{"bash", "zsh", "posix", "fish"} {
		t.Run(shellName, func(t *testing.T) {
			executable := shellName
			if shellName == "posix" {
				executable = "sh"
			}
			binary, err := exec.LookPath(executable)
			if err != nil || isWslBashLauncher(binary) {
				t.Skip(executable + " unavailable as a native interpreter")
			}

			root := filepath.Join(t.TempDir(), "repo with spaces")
			config, _ := GetShellConfig(shellName)
			shellDir := GetShellDirectory(root, shellName)
			featureDir := filepath.Join(shellDir, "features")
			if err := os.MkdirAll(featureDir, 0700); err != nil {
				t.Fatal(err)
			}

			feature := `printf 'loaded with <%s>\n' "$1"
set -- 'replaced by feature'
omd_print_args() { printf '<%s>\n' "$@"; }
alias omd_first=omd_print_args
omd_sibling() { printf 'sibling:%s\n' "$1"; }
`
			failure := `omd_fail() { printf 'unexpected redispatch\n'; }
return 42
`
			hyphen := `omd-hyphen() { printf 'hyphen trigger\n'; }` + "\n"
			runner := `omd_existing() { printf 'existing\n'; }
alias omd_alias='printf "alias preserved\n"'
. "$OMD_TEST_INIT"
omd_replaced() { printf 'replacement:%s\n' "$1"; }
if :; then printf 'builtin and keyword preserved\n'; fi
omd_existing
eval omd_alias
omd_first 'two words' '' '*' '"; echo unexpected; #' '$(echo literal)'
omd_sibling again
omd_replaced kept
omd_existing
eval omd_alias
omd_missing_file
printf 'missing feature status:%s\n' "$?"
command -v omd_missing_file >/dev/null || printf 'missing file wrapper removed\n'
command -v omd_missing_sibling >/dev/null || printf 'missing sibling wrapper removed\n'
omd_missing_trigger
printf 'missing trigger status:%s\n' "$?"
omd_fail
printf 'failed source status:%s\n' "$?"
command -v omd_fail_sibling >/dev/null || printf 'failed source sibling wrapper removed\n'
`

			var flags []string
			switch shellName {
			case "bash":
				flags = []string{"--noprofile", "--norc", "-O", "expand_aliases"}
			case "zsh":
				flags = []string{"-f"}
			case "posix":
				// Git for Windows can start sh as Bash with alias expansion disabled.
				runner = `if [ -n "${BASH_VERSION:-}" ]; then
    set -o posix
fi
` + runner
			case "fish":
				flags = []string{"--no-config"}
				feature = `printf 'loaded\n'
set argv 'replaced by feature'
function omd_print_args
    printf '<%s>\n' $argv
end
alias omd_first omd_print_args
function omd_sibling
    printf 'sibling:%s\n' $argv[1]
end
`
				failure = `function omd_fail
    printf 'unexpected redispatch\n'
end
return 42
`
				hyphen = `function omd-hyphen; printf 'hyphen trigger\n'; end` + "\n"
				runner = `function omd_existing; printf 'existing\n'; end
alias omd_alias 'printf "alias preserved\n"'
source "$OMD_TEST_INIT"
function omd_replaced; printf 'replacement:%s\n' $argv[1]; end
if true; printf 'builtin and keyword preserved\n'; end
omd_existing
omd_alias
omd_first 'two words' '' '*' '"; echo unexpected; #' '$(echo literal)'
omd_sibling again
omd_replaced kept
omd_existing
omd_alias
omd_missing_file
printf 'missing feature status:%s\n' $status
functions -q omd_missing_file; or printf 'missing file wrapper removed\n'
functions -q omd_missing_sibling; or printf 'missing sibling wrapper removed\n'
omd_missing_trigger
printf 'missing trigger status:%s\n' $status
omd_fail
printf 'failed source status:%s\n' $status
functions -q omd_fail_sibling; or printf 'failed source sibling wrapper removed\n'
`
			}

			want := `builtin and keyword preserved
existing
alias preserved
loaded
<two words>
<>
<*>
<"; echo unexpected; #>
<$(echo literal)>
sibling:again
replacement:kept
existing
alias preserved
oh-my-dot: feature 'absent' not found
missing feature status:127
missing file wrapper removed
missing sibling wrapper removed
oh-my-dot: omd_missing_trigger command not found after loading feature
missing trigger status:127
failed source status:42
failed source sibling wrapper removed
`
			if shellName == "posix" {
				want = "oh-my-dot: warning: 'omd-hyphen' is not a portable sh function name; skipping on-command trigger\n" + want
				// POSIX sh removes each wrapper on its own call. After a failed
				// load, the remaining wrappers resolve without sourcing again.
				runner += `command -v omd_missing_sibling >/dev/null && printf 'missing sibling wrapper still installed\n'
omd_missing_sibling
printf 'missing sibling status:%s\n' "$?"
command -v omd_missing_sibling >/dev/null || printf 'missing sibling wrapper removed\n'
command -v omd_fail_sibling >/dev/null && printf 'failed source sibling wrapper still installed\n'
omd_fail_sibling
printf 'failed source sibling status:%s\n' "$?"
command -v omd_fail_sibling >/dev/null || printf 'failed source sibling wrapper removed\n'
`
				want = strings.Replace(want, "missing sibling wrapper removed\n", "", 1)
				want = strings.Replace(want, "failed source sibling wrapper removed\n", "", 1)
				want += `missing sibling wrapper still installed
oh-my-dot: omd_missing_sibling command not found after loading feature
missing sibling status:127
missing sibling wrapper removed
failed source sibling wrapper still installed
oh-my-dot: omd_fail_sibling command not found after loading feature
failed source sibling status:127
failed source sibling wrapper removed
`
			} else {
				runner += "omd-hyphen\n"
				want += "hyphen trigger\n"
			}
			if shellName != "fish" {
				want = strings.Replace(want, "loaded\n", "loaded with <two words>\n", 1)
			}

			for name, body := range map[string]string{"example": feature, "no-trigger": "true\n", "failing": failure, "hyphen": hyphen} {
				if err := os.WriteFile(filepath.Join(featureDir, name+config.Extension), []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
			}

			base := &manifest.FeatureManifest{Features: []manifest.FeatureConfig{
				{Name: "example", Strategy: "on-command", OnCommand: []string{"omd_first", "omd_sibling", "omd_replaced", "omd_existing", "omd_alias", "printf", "if"}},
				{Name: "absent", Strategy: "on-command", OnCommand: []string{"omd_missing_file", "omd_missing_sibling"}},
				{Name: "no-trigger", Strategy: "on-command", OnCommand: []string{"omd_missing_trigger"}},
				{Name: "failing", Strategy: "on-command", OnCommand: []string{"omd_fail", "omd_fail_sibling"}},
				{Name: "hyphen", Strategy: "on-command", OnCommand: []string{"omd-hyphen"}},
			}}
			content, err := generateManifestInit(root, shellName, manifest.MergeManifests(base, nil))
			if err != nil {
				t.Fatal(err)
			}
			initPath := filepath.Join(shellDir, config.InitScript)
			if err := os.WriteFile(initPath, []byte(content), 0600); err != nil {
				t.Fatal(err)
			}

			cmd := exec.Command(binary, append(flags, "-c", runner)...)
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "OMD_TEST_INIT="+filepath.ToSlash(initPath))
			output, err := cmd.CombinedOutput()
			if got := strings.ReplaceAll(string(output), "\r\n", "\n"); err != nil || got != want {
				t.Fatalf("shell exit: %v\noutput:\n%s\nwant:\n%s", err, got, want)
			}
		})
	}
}
