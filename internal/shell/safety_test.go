package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
)

func TestShellCleanupPreservesLocalAndCustomContent(t *testing.T) {
	for _, scenario := range []string{"local feature", "local override", "custom file", "empty generated"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			if err := AddFeatureToShell(root, "bash", "git-prompt", "eager", nil, false, nil); err != nil {
				t.Fatal(err)
			}
			preserved := ""
			switch scenario {
			case "local feature", "local override":
				name := "private"
				if scenario == "local override" {
					name = "git-prompt"
				}
				local := &manifest.FeatureManifest{Features: []manifest.FeatureConfig{{Name: name, Strategy: "eager"}}}
				if err := manifest.WriteManifest(GetLocalManifestPath(root, "bash"), local); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(GetLocalManifestPath(root, "bash"), 0600); err != nil {
					t.Fatal(err)
				}
				preserved = filepath.Join(GetFeaturesDirectory(root, "bash"), name+".sh")
				if err := os.WriteFile(preserved, []byte("private-content"), 0600); err != nil {
					t.Fatal(err)
				}
			case "custom file":
				preserved = filepath.Join(GetShellDirectory(root, "bash"), "notes.txt")
				if err := os.WriteFile(preserved, []byte("private-content"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := RemoveFeatureFromShell(root, "bash", "git-prompt"); err != nil {
				t.Fatal(err)
			}
			cleanup, err := NeedsCleanup(root, "bash")
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "empty generated" {
				if !cleanup {
					t.Fatal("unused generated shell was not eligible for cleanup")
				}
				if err := CleanupShellDirectory(root, "bash"); err != nil {
					t.Fatal(err)
				}
				if _, err := os.Stat(GetShellDirectory(root, "bash")); !os.IsNotExist(err) {
					t.Fatal("unused directory remains")
				}
			} else {
				if cleanup {
					t.Fatal("user content eligible for cleanup")
				}
				if err := CleanupShellDirectory(root, "bash"); err == nil {
					t.Fatal("direct cleanup accepted user content")
				}
				data, err := os.ReadFile(preserved)
				if err != nil || string(data) != "private-content" {
					t.Fatalf("user file changed: %q %v", data, err)
				}
			}
		})
	}
}

func TestBashEagerFeaturesReapplyOnSource(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash unavailable")
	}
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "features"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "features", "prompt.sh"), []byte("PS1=restored\n"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "init.sh")
	if err := os.WriteFile(path, []byte(generateBashInit(FeaturesByStrategy{Eager: []string{"prompt"}})), 0600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(bash, "--noprofile", "--norc", "-c", `. "$1"; PS1=reset; . "$1"; printf '%s' "$PS1"`, "test", path).CombinedOutput()
	if err != nil || string(output) != "restored" {
		t.Fatalf("re-source prompt: %s %v", output, err)
	}
}

func TestLazyFeaturesPreserveLoadedFunctions(t *testing.T) {
	for _, shellName := range []string{"bash", "zsh", "fish", "posix", "powershell"} {
		for _, multiple := range []bool{false, true} {
			name := "single"
			if multiple {
				name = "multiple"
			}
			t.Run(shellName+"/"+name, func(t *testing.T) {
				executable := shellName
				if shellName == "posix" {
					executable = "sh"
				}
				if shellName == "powershell" {
					executable = "pwsh"
				}
				binary, err := exec.LookPath(executable)
				if err != nil {
					t.Skip(executable + " unavailable")
				}
				root := t.TempDir()
				directory := GetShellDirectory(root, shellName)
				if err := os.MkdirAll(filepath.Join(directory, "features"), 0700); err != nil {
					t.Fatal(err)
				}
				commands := []string{"omd_first"}
				if multiple {
					commands = append(commands, "omd_second")
				}
				feature := "printf 'loaded\\n'\nomd_first() { printf 'first:%s\\n' \"$1\"; }\nomd_second() { printf 'second:%s\\n' \"$1\"; }\n"
				runner := `. "$1"; omd_first 'one argument'; omd_first again`
				if multiple {
					runner += "; omd_second last"
				}
				extension := ".sh"
				args := []string{"-c", runner, "test"}
				switch shellName {
				case "bash":
					args = append([]string{"--noprofile", "--norc"}, args...)
				case "zsh":
					extension = ".zsh"
					args = append([]string{"-f"}, args...)
				case "fish":
					extension = ".fish"
					feature = "printf 'loaded\\n'\nfunction omd_first; printf 'first:%s\\n' $argv[1]; end\nfunction omd_second; printf 'second:%s\\n' $argv[1]; end\n"
					runner = "source $argv[1]; omd_first 'one argument'; omd_first again"
					if multiple {
						runner += "; omd_second last"
					}
					args = []string{"--no-config", "-c", runner}
				case "powershell":
					extension = ".ps1"
					feature = "Write-Output 'loaded'\nfunction omd_first { Write-Output \"first:$($args[0])\" }\nfunction omd_second { Write-Output \"second:$($args[0])\" }\n"
					runner = ". $args[0]; omd_first 'one argument'; omd_first again"
					if multiple {
						runner += "; omd_second last"
					}
					args = []string{"-NoProfile", "-NonInteractive", "-Command", runner}
				}
				if err := os.WriteFile(filepath.Join(directory, "features", "example"+extension), []byte(feature), 0600); err != nil {
					t.Fatal(err)
				}
				features := FeaturesByStrategy{OnCommand: map[string][]string{"example": commands}}
				var generated string
				switch shellName {
				case "bash":
					generated = generateBashInit(features)
				case "zsh":
					generated = generateZshInit(features)
				case "fish":
					generated = generateFishInit(features)
				case "posix":
					generated = generatePosixInit(root, features)
				case "powershell":
					generated = generatePowerShellInit(features)
				}
				initPath := filepath.Join(directory, "init"+extension)
				if err := os.WriteFile(initPath, []byte(generated), 0600); err != nil {
					t.Fatal(err)
				}
				if shellName == "powershell" {
					runner = ". '" + strings.ReplaceAll(initPath, "'", "''") + "'; omd_first 'one argument'; omd_first again"
					if multiple {
						runner += "; omd_second last"
					}
					args = []string{"-NoProfile", "-NonInteractive", "-Command", runner}
				} else {
					args = append(args, initPath)
				}
				output, err := exec.Command(binary, args...).CombinedOutput()
				want := "loaded\nfirst:one argument\nfirst:again\n"
				if multiple {
					want += "second:last\n"
				}
				if err != nil || strings.ReplaceAll(string(output), "\r\n", "\n") != want {
					t.Fatalf("loaded function behavior:\n%s\nerror %v", output, err)
				}
			})
		}
	}
}

func TestLazyFeatureFailureReturnsWithoutCallingLoadedCommand(t *testing.T) {
	for _, shellName := range []string{"bash", "zsh", "fish", "posix", "powershell"} {
		t.Run(shellName, func(t *testing.T) {
			executable := shellName
			if shellName == "posix" {
				executable = "sh"
			}
			if shellName == "powershell" {
				executable = "pwsh"
			}
			binary, err := exec.LookPath(executable)
			if err != nil {
				t.Skip(executable + " unavailable")
			}
			root := t.TempDir()
			directory := GetShellDirectory(root, shellName)
			if err := os.MkdirAll(filepath.Join(directory, "features"), 0700); err != nil {
				t.Fatal(err)
			}
			feature := "omd_first() { printf 'should-not-run\\n'; }\nreturn 42\n"
			extension := ".sh"
			runner := `. "$1"; omd_first; result=$?; printf 'status=%s\nafter\n' "$result"`
			args := []string{"-c", runner, "test"}
			switch shellName {
			case "bash":
				args = append([]string{"--noprofile", "--norc"}, args...)
			case "zsh":
				extension = ".zsh"
				args = append([]string{"-f"}, args...)
			case "fish":
				extension = ".fish"
				feature = "function omd_first; echo should-not-run; end\nreturn 42\n"
				runner = "source $argv[1]; omd_first; printf 'status=%s\\nafter\\n' $status"
				args = []string{"--no-config", "-c", runner}
			case "powershell":
				extension = ".ps1"
				feature = "function omd_first { Write-Output 'should-not-run' }\nthrow 'load failed'\n"
			}
			if err := os.WriteFile(filepath.Join(directory, "features", "example"+extension), []byte(feature), 0600); err != nil {
				t.Fatal(err)
			}
			features := FeaturesByStrategy{OnCommand: map[string][]string{"example": {"omd_first"}}}
			var generated string
			switch shellName {
			case "bash":
				generated = generateBashInit(features)
			case "zsh":
				generated = generateZshInit(features)
			case "fish":
				generated = generateFishInit(features)
			case "posix":
				generated = generatePosixInit(root, features)
			case "powershell":
				generated = generatePowerShellInit(features)
			}
			initPath := filepath.Join(directory, "init"+extension)
			if err := os.WriteFile(initPath, []byte(generated), 0600); err != nil {
				t.Fatal(err)
			}
			wantStatus := "status=42"
			if shellName == "powershell" {
				runner = ". '" + strings.ReplaceAll(initPath, "'", "''") + "'; omd_first; Write-Output \"status=$global:LASTEXITCODE\"; Write-Output 'after'"
				args = []string{"-NoProfile", "-NonInteractive", "-Command", runner}
				wantStatus = "status=1"
			} else {
				args = append(args, initPath)
			}
			output, err := exec.Command(binary, args...).CombinedOutput()
			if err != nil || strings.Contains(string(output), "should-not-run") || !strings.Contains(string(output), wantStatus) || !strings.Contains(string(output), "after") {
				t.Fatalf("failed loader behavior: %s %v", output, err)
			}
		})
	}
}
