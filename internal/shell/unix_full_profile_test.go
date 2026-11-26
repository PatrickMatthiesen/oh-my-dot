//go:build unix

package shell

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProfileUnixShellStartupNative(t *testing.T) {
	t.Setenv("BASH_ENV", "")
	t.Setenv("ENV", "")
	bwrap, err := exec.LookPath("bwrap")
	if err != nil {
		t.Skip("bubblewrap is required to isolate native startup tests from system profiles")
	}
	if output, err := exec.Command(bwrap, "--unshare-all", "--ro-bind", "/", "/", "--", "/bin/true").CombinedOutput(); err != nil {
		t.Skipf("bubblewrap isolation is unavailable: %v: %s", err, output)
	}
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep executable is required for timing fixtures")
	}
	for _, tt := range []struct {
		name        string
		bin         string
		startupFile string
		files       map[string]string
	}{
		{"bash", "bash", ".bashrc", map[string]string{
			".bash_profile": "exit 7\n",
			".bashrc":       "case $- in *i*) ;; *) exit 7 ;; esac\n[ ! -t 0 ] || exit 7\nset -C\ntrap 'sleep 1' EXIT\n",
		}},
		{"zsh", "zsh", ".zshrc", map[string]string{
			".zshenv":   "steps=env\n",
			".zprofile": "exit 7\n",
			".zshrc":    "[[ -o interactive && ! -o login && ! -t 0 && $steps = env ]] || exit 7\n",
		}},
		{"fish", "fish", "config/fish/config.fish", map[string]string{
			"config/fish/config.fish": "status is-interactive; and not status is-login; and not test -t 0; and test \"$steps\" = system; or exit 7\n",
		}},
		{"posix", "sh", ".profile", map[string]string{
			".profile": "[ \"$steps\" = system ] && [ ! -t 0 ] || exit 7\ncase $- in *i*) ;; *) exit 7 ;; esac\n",
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			binary, err := exec.LookPath(tt.bin)
			if err != nil {
				t.Skipf("%s executable is unavailable", tt.bin)
			}
			home := filepath.Join(t.TempDir(), `fixture's \ home`)
			for path, contents := range tt.files {
				if path == tt.startupFile {
					contents += "sleep 0.12\nprintf 'startup output'\n"
				}
				writeUnixStartupFixture(t, filepath.Join(home, path), contents, 0600)
			}
			systemProfile := filepath.Join(home, "system-profile")
			writeUnixStartupFixture(t, systemProfile, "steps=system\n", 0600)
			args := []string{bwrap, "--unshare-all", "--die-with-parent", "--dev", "/dev", "--proc", "/proc"}
			for _, libraryDir := range []string{"/lib", "/lib64", "/usr/lib", "/usr/lib64"} {
				args = append(args, "--ro-bind-try", libraryDir, libraryDir)
			}
			if tt.name == "fish" {
				// Fish checks for shipped assets before it sources user configuration.
				fishAssets := filepath.Join(home, "assets-config.fish")
				writeUnixStartupFixture(t, fishAssets, "", 0600)
				fishSystemConfig := filepath.Join(home, "system-config.fish")
				writeUnixStartupFixture(t, fishSystemConfig, "set -g steps system\n", 0600)
				args = append(args, "--ro-bind", fishAssets, "/usr/share/fish/config.fish",
					"--ro-bind", fishSystemConfig, "/etc/fish/config.fish")
			}
			args = append(args, "--bind", home, home, "--ro-bind", binary, "/bin/"+tt.bin,
				"--ro-bind", sleep, "/bin/sleep", "--ro-bind", systemProfile, "/etc/profile", "--clearenv",
				"--setenv", "HOME", home, "--setenv", "ZDOTDIR", home,
				"--setenv", "XDG_CONFIG_HOME", filepath.Join(home, "config"),
				"--setenv", "PATH", "/bin", "--", "/bin/"+tt.bin)
			for i, arg := range args {
				args[i] = shellSingleQuote(arg)
			}
			wrapper := filepath.Join(home, "bin", tt.bin)
			writeUnixStartupFixture(t, wrapper, "#!/bin/sh\nexec "+strings.Join(args, " ")+` "$@"`+"\n", 0700)
			t.Setenv("TMPDIR", home)
			t.Setenv("PATH", filepath.Dir(wrapper))
			start := time.Now()
			report, err := ProfileUnixShellStartup(context.Background(), tt.name)
			wallMS := float64(time.Since(start)) / float64(time.Millisecond)
			if err != nil {
				t.Fatal(err)
			}
			if report.Shell != tt.name || report.Executable != wrapper || len(report.Profiles) != 0 || report.AttributionAvailable {
				t.Fatalf("unexpected startup report: %+v", report)
			}
			if report.TotalMS < 110 || report.TotalMS > wallMS {
				t.Fatalf("startup duration = %.2f ms, want between 110 and %.2f ms", report.TotalMS, wallMS)
			}
			if tt.name == "bash" && wallMS-report.TotalMS < 800 {
				t.Fatalf("startup duration %.2f ms includes shutdown, wall time %.2f ms", report.TotalMS, wallMS)
			}
		})
	}
}

func TestProfileUnixShellStartupMissingMarker(t *testing.T) {
	for _, tt := range []struct {
		name   string
		script string
	}{
		{"exit zero", "exit 0"},
		{"exit nonzero", "exit 6"},
		{"stdout marker", "printf omd-startup-complete"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fixture := filepath.Join(t.TempDir(), "fixture")
			writeUnixStartupFixture(t, fixture, "#!/bin/sh\n"+tt.script+"\n", 0700)
			_, err := profileUnixShellStartup(context.Background(), "bash", fixture)
			if err == nil || !strings.Contains(err.Error(), "exited before writing the startup completion marker") {
				t.Fatalf("profile error = %v, want missing completion marker", err)
			}
		})
	}
}

func TestProfileUnixShellStartupCanceled(t *testing.T) {
	fixture := filepath.Join(t.TempDir(), "fixture")
	writeUnixStartupFixture(t, fixture, "#!/bin/sh\nsleep 10\n", 0700)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := profileUnixShellStartup(ctx, "bash", fixture)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("profile error = %v, want context deadline exceeded", err)
	}
	canceled, stop := context.WithCancel(context.Background())
	stop()
	_, err = profileUnixShellStartup(canceled, "bash", fixture)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("profile error = %v, want context canceled", err)
	}
}

func writeUnixStartupFixture(t *testing.T, path, contents string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), mode); err != nil {
		t.Fatal(err)
	}
}
