package catalog

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOhMyPoshAutoUpgradeTemplates(t *testing.T) {
	for _, shellName := range []string{"bash", "zsh", "fish", "powershell"} {
		for _, enabled := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/enabled=%t", shellName, enabled), func(t *testing.T) {
				template, err := GetFeatureTemplate("oh-my-posh", shellName)
				if err != nil {
					t.Fatal(err)
				}
				options := map[string]any{}
				if enabled {
					options["auto_upgrade"] = true
				}
				rendered, err := RenderFeatureTemplate(template, "oh-my-posh", shellName, options)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(rendered, "oh-my-posh upgrade --auto") != enabled {
					t.Fatalf("auto upgrade enabled = %t, rendered feature:\n%s", enabled, rendered)
				}
				if strings.Contains(rendered, "--force") || strings.Contains(rendered, "enable upgrade") {
					t.Fatal("feature must respect native update intervals and avoid persistent global opt-in")
				}
			})
		}
	}
}

func TestOhMyPoshAutoUpgradeShellExecution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stub executable uses a POSIX shell; template coverage runs on Windows")
	}
	for _, shellName := range []string{"bash", "zsh", "fish", "powershell"} {
		for _, enabled := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/enabled=%t", shellName, enabled), func(t *testing.T) {
				executable := shellName
				args := []string{"-f"}
				switch shellName {
				case "bash":
					args = []string{"--noprofile", "--norc"}
				case "fish":
					args = []string{"--no-config"}
				case "powershell":
					executable = "pwsh"
					args = []string{"-NoProfile", "-NonInteractive", "-File"}
				}
				interpreter, err := exec.LookPath(executable)
				if err != nil {
					t.Skipf("%s unavailable", executable)
				}
				dir := t.TempDir()
				logPath := filepath.Join(dir, "calls")
				configPath := filepath.Join(dir, "custom theme.omp.json")
				if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
					t.Fatal(err)
				}
				stub := "#!/bin/sh\nprintf '%s\\n' \"$@\" >> \"$OMD_TEST_POSH_LOG\"\nif [ \"$1\" = upgrade ]; then exit 1; fi\n"
				if err := os.WriteFile(filepath.Join(dir, "oh-my-posh"), []byte(stub), 0755); err != nil {
					t.Fatal(err)
				}
				template, err := GetFeatureTemplate("oh-my-posh", shellName)
				if err != nil {
					t.Fatal(err)
				}
				rendered, err := RenderFeatureTemplate(template, "oh-my-posh", shellName, map[string]any{"config_file": configPath, "auto_upgrade": enabled})
				if err != nil {
					t.Fatal(err)
				}
				scriptPath := filepath.Join(dir, "feature"+GetShellExtension(shellName))
				if err := os.WriteFile(scriptPath, []byte(rendered), 0644); err != nil {
					t.Fatal(err)
				}
				command := exec.Command(interpreter, append(args, scriptPath)...)
				command.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"), "OMD_TEST_POSH_LOG="+logPath)
				if output, err := command.CombinedOutput(); err != nil {
					t.Fatalf("feature execution: %v\n%s", err, output)
				}
				calls, err := os.ReadFile(logPath)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(calls), "init\n") {
					t.Fatalf("prompt initialization did not continue after upgrade failure: %s", calls)
				}
				wantUpgrade := "upgrade\n--auto\n--config\n" + configPath + "\n"
				if strings.Contains(string(calls), wantUpgrade) != enabled {
					t.Fatalf("unexpected upgrade arguments (enabled=%t): %s", enabled, calls)
				}
			})
		}
	}
}
