package catalog

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOhMyPoshCustomPathIsLiteralAtShellStartup(t *testing.T) {
	for _, shellName := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shellName, func(t *testing.T) {
			if runtime.GOOS == "windows" && shellName != "powershell" {
				t.Skip("POSIX probe requires a native Unix interpreter")
			}
			executable := shellName
			if shellName == "powershell" {
				executable = "pwsh"
			}
			binary, err := exec.LookPath(executable)
			if err != nil {
				t.Skipf("%s unavailable", executable)
			}
			root := t.TempDir()
			name := "theme $HOME $(touch OMD_INJECTED) `touch OMD_BACKTICK` ' \" \\ [literal] (touch OMD_FISH).json"
			if runtime.GOOS == "windows" {
				name = "theme $HOME $(New-Item OMD_INJECTED) `x ' [literal].json"
			}
			config := filepath.Join(root, name)
			if err := os.WriteFile(config, []byte("{}"), 0600); err != nil {
				t.Fatal(err)
			}
			source, err := GetFeatureTemplate("oh-my-posh", shellName)
			if err != nil {
				t.Fatal(err)
			}
			script, err := RenderFeatureTemplate(source, "oh-my-posh", shellName, map[string]any{"config_file": config})
			if err != nil {
				t.Fatal(err)
			}
			log := filepath.Join(root, "arguments")
			args := []string{"--noprofile", "--norc"}
			switch shellName {
			case "zsh":
				args = []string{"-f"}
			case "fish":
				args = []string{"--no-config"}
			case "powershell":
				args = []string{"-NoProfile", "-NonInteractive", "-File"}
				script = "$ErrorActionPreference = 'Stop'\nfunction global:oh-my-posh { [IO.File]::WriteAllText($env:OMD_PATH_LOG, (ConvertTo-Json -InputObject @($args) -Compress)) }\n" + script
			}
			if shellName != "powershell" {
				probe := "#!/bin/sh\nprintf '%s\\000' \"$@\" > \"$OMD_PATH_LOG\"\n"
				if err := os.WriteFile(filepath.Join(root, "oh-my-posh"), []byte(probe), 0755); err != nil {
					t.Fatal(err)
				}
			}
			path := filepath.Join(root, "feature"+GetShellExtension(shellName))
			if err := os.WriteFile(path, []byte(script), 0600); err != nil {
				t.Fatal(err)
			}
			command := exec.Command(binary, append(args, path)...)
			command.Dir = root
			command.Env = append(os.Environ(), "PATH="+root+string(os.PathListSeparator)+os.Getenv("PATH"), "OMD_PATH_LOG="+log)
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("startup failed: %q %v", output, err)
			}
			data, err := os.ReadFile(log)
			if err != nil {
				t.Fatal(err)
			}
			var received []string
			if shellName == "powershell" {
				if err := json.Unmarshal(data, &received); err != nil {
					t.Fatal(err)
				}
			} else {
				received = strings.Split(strings.TrimSuffix(string(data), "\x00"), "\x00")
			}
			if len(received) != 4 || received[2] != "--config" || received[3] != config {
				t.Fatalf("literal path changed: %q, expected %q", received, config)
			}
			for _, marker := range []string{"OMD_INJECTED", "OMD_BACKTICK", "OMD_FISH"} {
				if _, err := os.Stat(filepath.Join(root, marker)); !os.IsNotExist(err) {
					t.Fatalf("filename executed shell syntax: %s", marker)
				}
			}
		})
	}
}

func TestOhMyPoshRejectsExecutableThemeFromSavedOptions(t *testing.T) {
	for _, shellName := range []string{"bash", "zsh", "fish", "powershell"} {
		source, err := GetFeatureTemplate("oh-my-posh", shellName)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := RenderFeatureTemplate(source, "oh-my-posh", shellName, map[string]any{"theme": "$(touch OMD_THEME_INJECTED)"}); err == nil {
			t.Fatalf("saved theme becomes shell syntax in %s", shellName)
		}
	}
}
