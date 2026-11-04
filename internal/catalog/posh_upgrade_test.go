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
				initShell := shellName
				if shellName == "powershell" {
					initShell = "pwsh"
				}
				wantInit := "init\n" + initShell + "\n--config\n" + configPath + "\n"
				if !strings.Contains(string(calls), wantInit) {
					t.Fatalf("initialization did not receive exact config path: %s", calls)
				}
				wantUpgrade := "upgrade\n--auto\n--config\n" + configPath + "\n"
				if strings.Contains(string(calls), wantUpgrade) != enabled {
					t.Fatalf("unexpected upgrade arguments (enabled=%t): %s", enabled, calls)
				}
			})
		}
	}
}

func TestOhMyPoshThemeChangeUsesDistinctManagedCache(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture executable uses POSIX sh")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash unavailable")
	}
	root := t.TempDir()
	shellRoot := filepath.Join(root, "shell root with spaces")
	if err := os.MkdirAll(filepath.Join(shellRoot, "features"), 0700); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(shellRoot, "features", "oh-my-posh.omp.json")
	if err := os.WriteFile(legacy, []byte("legacy cached content"), 0600); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(root, "calls")
	omp := "#!/bin/sh\nprintf '%s\\n' \"$@\" >> \"$OMD_TEST_POSH_LOG\"\nprintf 'PS1=omd_test_prompt\\n'\n"
	curl := "#!/bin/sh\nwhile [ \"$#\" -gt 0 ]; do case \"$1\" in -o) shift; target=$1;; http*) url=$1;; esac; shift; done\nprintf '%s\\n' \"$url\" > \"$target\"\n"
	for name, content := range map[string]string{"oh-my-posh": omp, "curl": curl} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(content), 0700); err != nil {
			t.Fatal(err)
		}
	}
	template, err := GetFeatureTemplate("oh-my-posh", "bash")
	if err != nil {
		t.Fatal(err)
	}
	for _, theme := range []string{"paradox", "pure"} {
		script, err := RenderFeatureTemplate(template, "oh-my-posh", "bash", map[string]any{"theme": theme})
		if err != nil {
			t.Fatal(err)
		}
		file := filepath.Join(root, theme+".sh")
		if err := os.WriteFile(file, []byte(script), 0600); err != nil {
			t.Fatal(err)
		}
		command := exec.Command(bash, "--noprofile", "--norc", "-c", `. "$1"; test "$PS1" = omd_test_prompt`, "_", file)
		command.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "OMD_SHELL_ROOT="+shellRoot, "OMD_TEST_POSH_LOG="+log)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("activate %s theme: %v %s", theme, err, output)
		}
		cache := filepath.Join(shellRoot, ".local", "cache", "oh-my-posh."+theme+".omp.json")
		data, err := os.ReadFile(cache)
		if err != nil || !strings.Contains(string(data), "/"+theme+".omp.json") {
			t.Fatalf("wrong cache for %s: %s %v", theme, data, err)
		}
	}
	data, err := os.ReadFile(legacy)
	if err != nil || string(data) != "legacy cached content" {
		t.Fatal("legacy/custom content was deleted or overwritten")
	}
	calls, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	for _, theme := range []string{"paradox", "pure"} {
		if !strings.Contains(string(calls), filepath.Join(shellRoot, ".local", "cache", "oh-my-posh."+theme+".omp.json")) {
			t.Fatalf("init did not select changed theme: %s", calls)
		}
	}
}
