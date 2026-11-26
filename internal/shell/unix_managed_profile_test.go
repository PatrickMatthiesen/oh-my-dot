package shell

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
)

func TestProfileUnixShellFeatures(t *testing.T) {
	for _, shellName := range []string{"bash", "zsh", "fish", "posix"} {
		t.Run(shellName, func(t *testing.T) {
			executable, err := resolveUnixProfileExecutable(shellName)
			if err != nil {
				t.Skip(err)
			}
			repo := filepath.Join(t.TempDir(), "managed repo's shell")
			root := GetShellDirectory(repo, shellName)
			shared := []manifest.FeatureConfig{
				{Name: "eager"},
				{Name: "override"},
				{Name: "disabled", Disabled: true},
				{Name: "disabled-locally"},
				{Name: "missing"},
				{Name: "directory"},
				{Name: "returns"},
				{Name: "exits"},
				{Name: "exits-clean"},
			}
			local := []manifest.FeatureConfig{
				{Name: "override", Strategy: "defer", Options: map[string]any{"fixture": "local"}},
				{Name: "disabled-locally", Disabled: true},
				{Name: "private", Strategy: "on-command", OnCommand: []string{"fixture-trigger"}},
				{Name: "legacy"},
			}
			writeUnixProfileManifests(t, repo, shellName, shared, local)

			config := filepath.Join(t.TempDir(), "config")
			startup := filepath.Join(config, ".zshenv")
			writeUnixProfileFixture(t, startup, "export OMD_PROFILE_UNEXPECTED_STARTUP=1\n")
			writeUnixProfileFixture(t, filepath.Join(config, "fish", "config.fish"), "set -gx OMD_PROFILE_UNEXPECTED_STARTUP 1\n")
			t.Setenv("ZDOTDIR", filepath.ToSlash(config))
			t.Setenv("XDG_CONFIG_HOME", filepath.ToSlash(config))
			t.Setenv("BASH_ENV", filepath.ToSlash(startup))
			t.Setenv("ENV", filepath.ToSlash(startup))
			t.Setenv("OMD_PROFILE_UNEXPECTED_STARTUP", "")
			t.Setenv("OMD_FEATURE_EAGER_LOADED", "1")

			configuredShell, _ := GetShellConfig(shellName)
			extension := configuredShell.Extension
			eagerPath := filepath.Join(root, "features", "eager"+extension)
			helperPath := filepath.Join(root, "..", "lib", "helpers.sh")
			writeUnixProfileFixture(t, helperPath, "omd_profile_fixture_helper() { return 0; }\n")
			eager := fmt.Sprintf(`
case $- in *i*) exit 30 ;; esac
test "$OMD_FEATURE_EAGER_LOADED" != 1 || return 0
test -z "$OMD_PROFILE_UNEXPECTED_STARTUP" || exit 31
test "$OMD_SHELL_ROOT" = %s || exit 32
omd_profile_fixture_helper || exit 33
OMD_PROFILE_FIXTURE=1
sleep 0.20
`, shellSingleQuote(filepath.ToSlash(root)))
			if shellName == "bash" || shellName == "zsh" {
				eager = "local omd_fixture_local=1 || exit 34\n" + eager
			}
			isolated := "test -z \"${OMD_PROFILE_FIXTURE-}\" || exit 35\n"
			if shellName == "fish" {
				eager = fmt.Sprintf(`
status is-interactive; and exit 30
test "$OMD_FEATURE_EAGER_LOADED" = 1; and return 0
test -z "$OMD_PROFILE_UNEXPECTED_STARTUP"; or exit 31
test "$OMD_SHELL_ROOT" = %s; or exit 32
functions -q omd_profile_fixture_helper; and exit 33
set -g OMD_PROFILE_FIXTURE 1
sleep 0.20
`, fishProfileLiteral(root))
				isolated = "set -q OMD_PROFILE_FIXTURE; and exit 35\nreturn 0\n"
			}
			writeUnixProfileFixture(t, eagerPath, eager)
			for _, name := range []string{"override", "disabled", "disabled-locally"} {
				writeUnixProfileFixture(t, filepath.Join(root, "features", name+extension), "exit 36\n")
			}
			if err := os.Mkdir(filepath.Join(root, "features", "directory"+extension), 0700); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"override", "private"} {
				writeUnixProfileFixture(t, filepath.Join(root, ".local", "features", name+extension), isolated)
			}
			writeUnixProfileFixture(t, filepath.Join(root, "features", "legacy"+extension), isolated)
			writeUnixProfileFixture(t, filepath.Join(root, "features", "returns"+extension), "sleep 0.06\nreturn 8\n")
			writeUnixProfileFixture(t, filepath.Join(root, "features", "exits"+extension), "exit 7\n")
			writeUnixProfileFixture(t, filepath.Join(root, "features", "exits-clean"+extension), "exit 0\n")

			report, err := ProfileUnixShellFeatures(t.Context(), shellName, repo)
			if err != nil {
				t.Fatal(err)
			}
			if report.Shell != shellName || report.Executable != executable || len(report.Profiles) != 0 {
				t.Fatalf("unexpected managed report: %+v", report)
			}
			want := map[string]struct {
				strategy string
				private  bool
				failure  string
				minMS    float64
			}{
				"eager":       {"eager", false, "", 180},
				"override":    {"defer", true, "", 0},
				"private":     {"on-command", true, "", 0},
				"legacy":      {"eager", false, "", 0},
				"missing":     {"eager", false, "not found", 0},
				"directory":   {"eager", false, "not a regular file", 0},
				"returns":     {"eager", false, "exit status 8", 40},
				"exits":       {"eager", false, "exit status 7", 0},
				"exits-clean": {"eager", false, "before the feature timing finished", 0},
			}
			if shellName == "fish" {
				// Fish exits the sourced file with status 0 and resumes its caller.
				expected := want["exits-clean"]
				expected.failure = ""
				want["exits-clean"] = expected
			}
			var sum float64
			for i, timing := range report.ManagedFeatures {
				expected, ok := want[timing.Name]
				if !ok {
					t.Fatalf("unexpected or duplicate feature: %+v", timing)
				}
				delete(want, timing.Name)
				directory := filepath.Join(root, "features")
				if expected.private {
					directory = filepath.Join(root, ".local", "features")
				}
				if timing.Path != filepath.Join(directory, timing.Name+extension) || timing.Strategy != expected.strategy ||
					timing.DurationMS < expected.minMS || (expected.failure == "" && timing.Error != "") ||
					!strings.Contains(timing.Error, expected.failure) {
					t.Errorf("unexpected timing: %+v", timing)
				}
				if i > 0 && timing.DurationMS > report.ManagedFeatures[i-1].DurationMS {
					t.Errorf("timings out of order: %+v", report.ManagedFeatures)
				}
				sum += timing.DurationMS
			}
			if len(want) != 0 || math.Abs(sum-report.TotalMS) > 0.001 {
				t.Fatalf("missing timings %v or incorrect total: %+v", want, report)
			}
			init, err := GetInitScriptPath(repo, shellName)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(init); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("unexpected generated init %s: %v", init, err)
			}
		})
	}
}

func TestProfileUnixShellFeaturesManifest(t *testing.T) {
	t.Run("missing shared manifest", func(t *testing.T) {
		report, err := ProfileUnixShellFeatures(t.Context(), "posix", t.TempDir())
		if err != nil || len(report.ManagedFeatures) != 0 || report.TotalMS != 0 || report.Executable != "" {
			t.Fatalf("expected empty report, got %+v, %v", report, err)
		}
	})
	t.Run("unsafe private manifest", func(t *testing.T) {
		repo := t.TempDir()
		writeUnixProfileManifests(t, repo, "posix", []manifest.FeatureConfig{{Name: "shared"}}, nil)
		if err := os.Mkdir(GetLocalManifestPath(repo, "posix"), 0700); err != nil {
			t.Fatal(err)
		}
		_, err := ProfileUnixShellFeatures(t.Context(), "posix", repo)
		if err == nil || !strings.Contains(err.Error(), "not a regular file") {
			t.Fatalf("expected private manifest validation error, got %v", err)
		}
	})
}

func TestProfileUnixShellFeaturesCanceled(t *testing.T) {
	const shellName = "bash"
	if _, err := resolveUnixProfileExecutable(shellName); err != nil {
		t.Skip(err)
	}
	repo := t.TempDir()
	shared := []manifest.FeatureConfig{{Name: "complete"}, {Name: "waiting"}, {Name: "never"}}
	writeUnixProfileManifests(t, repo, shellName, shared, nil)
	root := GetFeaturesDirectory(repo, shellName)
	marker := filepath.Join(repo, "waiting")
	writeUnixProfileFixture(t, filepath.Join(root, "complete.sh"), "sleep 0.05\n")
	writeUnixProfileFixture(t, filepath.Join(root, "waiting.sh"),
		": > "+shellSingleQuote(filepath.ToSlash(marker))+"\nsleep 60\n")
	writeUnixProfileFixture(t, filepath.Join(root, "never.sh"), "exit 99\n")

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	type result struct {
		report ShellProfileReport
		err    error
	}
	done := make(chan result, 1)
	go func() {
		report, err := ProfileUnixShellFeatures(ctx, shellName, repo)
		done <- result{report, err}
	}()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.After(5 * time.Second)
	for ctx.Err() == nil {
		select {
		case <-ticker.C:
			if _, err := os.Stat(marker); err == nil {
				cancel()
			}
		case got := <-done:
			t.Fatalf("profiler returned before cancellation: %+v, %v", got.report, got.err)
		case <-deadline:
			t.Fatal("feature did not start")
		}
	}
	select {
	case got := <-done:
		timings := got.report.ManagedFeatures
		if !errors.Is(got.err, context.Canceled) || len(timings) != 2 ||
			timings[0].Name != "complete" || timings[0].Error != "" || timings[0].DurationMS < 30 ||
			timings[1].Name != "waiting" || !strings.Contains(timings[1].Error, "context canceled") {
			t.Fatalf("expected partial timings and cancellation, got %+v, %v", got.report, got.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("profiler did not return after cancellation")
	}
}

func writeUnixProfileManifests(t *testing.T, repo, shellName string, shared, local []manifest.FeatureConfig) {
	t.Helper()
	if err := os.MkdirAll(GetShellDirectory(repo, shellName), 0700); err != nil {
		t.Fatal(err)
	}
	if err := manifest.WriteManifest(GetManifestPath(repo, shellName), &manifest.FeatureManifest{Features: shared}); err != nil {
		t.Fatal(err)
	}
	if local != nil {
		if err := manifest.WriteLocalManifest(GetLocalManifestPath(repo, shellName), &manifest.FeatureManifest{Features: local}); err != nil {
			t.Fatal(err)
		}
	}
}

func writeUnixProfileFixture(t *testing.T, path, script string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
}
