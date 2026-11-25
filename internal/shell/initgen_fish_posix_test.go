package shell

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFishInitLoadsDeferredFeaturesOnFirstPrompt(t *testing.T) {
	fish, err := exec.LookPath("fish")
	if err != nil {
		t.Skip("fish unavailable")
	}
	root := t.TempDir()
	features := FeaturesByStrategy{
		Eager:     []string{"eager"},
		Defer:     []string{"deferred"},
		OnCommand: map[string][]string{"lazy": {"omd_test_lazy"}},
	}
	writeInitTestFiles(t, root, map[string]string{
		"init.fish":              generateFishInit(features),
		"features/eager.fish":    "set -f omd_test_value restored\n",
		"features/deferred.fish": "echo deferred-load\nfunction omd_test_deferred; echo deferred-call; end\n",
		"features/lazy.fish":     "echo lazy-load\nfunction omd_test_lazy; echo lazy-call; end\n",
	})
	// Emit fish_prompt explicitly to exercise the real event handler without a terminal.
	runner := `
set -g omd_test_init $argv[1]
set -e OMD_FISH_LOADED
function omd_test_source
  set -f omd_test_value reset
  source "$omd_test_init"; or exit 1
  echo "eager:$omd_test_value"
end
omd_test_source
omd_test_lazy; or exit 1
omd_test_source
if functions -q omd_test_deferred
  exit 1
end
echo before-prompt
emit fish_prompt
if status is-interactive
  omd_test_deferred; or exit 1
else if functions -q omd_test_deferred
  exit 1
end
functions -e omd_test_deferred
emit fish_prompt
omd_test_source
emit fish_prompt
if functions -q omd_test_deferred
  exit 1
end
omd_test_lazy; or exit 1
echo done
`
	for _, tc := range []struct {
		name     string
		option   string
		deferred string
	}{
		{name: "interactive", option: "-ic", deferred: "deferred-load\ndeferred-call\n"},
		{name: "noninteractive", option: "-c"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.CommandContext(t.Context(), fish, "--no-config", tc.option, runner, filepath.Join(root, "init.fish"))
			want := "eager:restored\nlazy-load\nlazy-call\neager:restored\nbefore-prompt\n" +
				tc.deferred + "eager:restored\nlazy-call\ndone\n"
			checkInitTestOutput(t, cmd, want)
		})
	}
}

func TestPosixInitLoadsDeferredFeaturesInlineOnce(t *testing.T) {
	dash, err := exec.LookPath("dash")
	if err != nil || isWslBashLauncher(dash) {
		t.Skip("dash unavailable as a native interpreter")
	}
	repo := t.TempDir()
	root := GetShellDirectory(repo, "posix")
	features := FeaturesByStrategy{
		Eager: []string{"eager"},
		Defer: []string{"deferred", "missing"},
		OnCommand: map[string][]string{
			"lazy":     {"omd_test_lazy"},
			"override": {"omd_test_override"},
		},
	}
	writeInitTestFiles(t, root, map[string]string{
		"init.sh":           generatePosixInit(repo, features),
		"../lib/helpers.sh": "echo helpers\n",
		"features/eager.sh": `printf 'eager:%s\n' "$2"` + "\n",
		"features/lazy.sh":  "echo lazy-load\nomd_test_lazy() { echo lazy-call; }\n",
		"features/deferred.sh": `echo deferred
omd_test_lazy
omd_test_override() { echo override; }
`,
	})
	runner := `
unset OMD_POSIX_LOADED
set -- "$1" first
. "$1"
echo after-init
omd_test_lazy
set -- "$1" second
. "$1"
omd_test_lazy
case $- in
  *i*) omd_test_override ;;
esac
echo done
`
	for _, tc := range []struct {
		name   string
		option string
		want   string
	}{
		{
			name:   "interactive",
			option: "-eic",
			want:   "helpers\neager:first\ndeferred\nlazy-load\nlazy-call\nafter-init\nlazy-call\neager:second\nlazy-call\noverride\ndone\n",
		},
		{
			name:   "noninteractive",
			option: "-ec",
			want:   "helpers\neager:first\nafter-init\nlazy-load\nlazy-call\neager:second\nlazy-call\ndone\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.CommandContext(t.Context(), dash, tc.option, runner, "test", filepath.Join(root, "init.sh"))
			cmd.Env = append(os.Environ(), "ENV=")
			checkInitTestOutput(t, cmd, tc.want)
		})
	}
}

func writeInitTestFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func checkInitTestOutput(t *testing.T, cmd *exec.Cmd, want string) {
	t.Helper()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil || string(output) != want {
		t.Fatalf("shell output = %q, want %q; error = %v\nstderr:\n%s", output, want, err, &stderr)
	}
}
