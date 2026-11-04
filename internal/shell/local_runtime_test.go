package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
)

func snapshotRuntimeTree(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		contents := []byte{}
		if !entry.IsDir() {
			contents, err = os.ReadFile(path)
			if err != nil {
				return err
			}
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files[relative] = fmt.Sprintf("%s:%s", info.Mode(), contents)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return files
}

func TestRuntimePreviewDoesNotMaterializePrivateStateOrIgnoreRules(t *testing.T) {
	root := t.TempDir()
	if err := AddFeatureToShell(root, "bash", "oh-my-posh", "eager", nil, false, map[string]any{"theme": "pure"}); err != nil {
		t.Fatal(err)
	}
	local := &manifest.FeatureManifest{Features: []manifest.FeatureConfig{{Name: "oh-my-posh", Options: map[string]any{"theme": "paradox"}}}}
	if err := manifest.WriteLocalManifest(GetLocalManifestPath(root, "bash"), local); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "omd-shells", ".gitignore")); err != nil {
		t.Fatal(err)
	}
	before := snapshotRuntimeTree(t, root)
	if err := ValidateRuntimeScripts(root, "bash"); err != nil {
		t.Fatal(err)
	}
	if after := snapshotRuntimeTree(t, root); !reflect.DeepEqual(before, after) {
		t.Fatal("runtime preview changed bytes, paths, or file permissions")
	}
	if err := RegenerateInitScript(root, "bash"); err != nil {
		t.Fatal(err)
	}
	before = snapshotRuntimeTree(t, root)
	if err := ValidateRuntimeScripts(root, "bash"); err != nil {
		t.Fatal(err)
	}
	if after := snapshotRuntimeTree(t, root); !reflect.DeepEqual(before, after) {
		t.Fatal("runtime preview changed existing generated state")
	}
}

func TestEffectiveFeatureLookupOnlyReadsAndUsesRuntimeOwnership(t *testing.T) {
	root := t.TempDir()
	if err := AddFeatureToShell(root, "bash", "oh-my-posh", "eager", nil, false, map[string]any{"theme": "pure"}); err != nil {
		t.Fatal(err)
	}
	shared, _ := GetFeatureFilePath(root, "bash", "oh-my-posh")
	private, _ := GetLocalFeatureFilePath(root, "bash", "oh-my-posh")
	for _, test := range []struct {
		name    string
		feature manifest.FeatureConfig
		want    string
	}{
		{"load-only override", manifest.FeatureConfig{Name: "oh-my-posh", Disabled: true}, shared},
		{"equivalent options", manifest.FeatureConfig{Name: "oh-my-posh", Options: map[string]any{"theme": "pure"}}, shared},
		{"private options", manifest.FeatureConfig{Name: "oh-my-posh", Options: map[string]any{"theme": "paradox"}}, private},
	} {
		t.Run(test.name, func(t *testing.T) {
			local := &manifest.FeatureManifest{Features: []manifest.FeatureConfig{test.feature}}
			if err := manifest.WriteLocalManifest(GetLocalManifestPath(root, "bash"), local); err != nil {
				t.Fatal(err)
			}
			before := snapshotRuntimeTree(t, root)
			actual, err := GetEffectiveFeatureFilePath(root, "bash", "oh-my-posh")
			if err != nil || actual != test.want {
				t.Fatalf("effective path %q, %v; want %q", actual, err, test.want)
			}
			if after := snapshotRuntimeTree(t, root); !reflect.DeepEqual(before, after) {
				t.Fatal("ownership lookup changed runtime state")
			}
		})
	}
}

func TestPrivateGeneratedOwnershipOnSharedChangeAndReset(t *testing.T) {
	root := t.TempDir()
	if err := AddFeatureToShell(root, "bash", "oh-my-posh", "eager", nil, false, map[string]any{"theme": "pure"}); err != nil {
		t.Fatal(err)
	}
	local := &manifest.FeatureManifest{Features: []manifest.FeatureConfig{{Name: "oh-my-posh", Options: map[string]any{"auto_upgrade": true}}}}
	if err := manifest.WriteLocalManifest(GetLocalManifestPath(root, "bash"), local); err != nil {
		t.Fatal(err)
	}
	if err := RegenerateInitScript(root, "bash"); err != nil {
		t.Fatal(err)
	}
	private, _ := GetLocalFeatureFilePath(root, "bash", "oh-my-posh")
	base, err := manifest.ParseManifest(GetManifestPath(root, "bash"))
	if err != nil {
		t.Fatal(err)
	}
	base.Features[0].Options["theme"] = "paradox"
	if err := manifest.WriteManifest(GetManifestPath(root, "bash"), base); err != nil {
		t.Fatal(err)
	}
	before := snapshotRuntimeTree(t, root)
	if err := ValidateRuntimeScripts(root, "bash"); err != nil {
		t.Fatal(err)
	}
	if after := snapshotRuntimeTree(t, root); !reflect.DeepEqual(before, after) {
		t.Fatal("preview rewrote a stale generated variant")
	}
	if err := RegenerateInitScript(root, "bash"); err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(private)
	if err != nil || !strings.Contains(string(generated), "paradox") {
		t.Fatalf("private variant did not follow shared option change: %v", err)
	}
	custom := append(generated, []byte("\n# private customization\n")...)
	if err := os.WriteFile(private, custom, 0600); err != nil {
		t.Fatal(err)
	}
	base.Features[0].Options["theme"] = "robbyrussell"
	if err := manifest.WriteManifest(GetManifestPath(root, "bash"), base); err != nil {
		t.Fatal(err)
	}
	before = snapshotRuntimeTree(t, root)
	if err := ValidateRuntimeScripts(root, "bash"); err == nil || !strings.Contains(err.Error(), "customized") {
		t.Fatalf("preview accepted stale customized private options: %v", err)
	}
	if after := snapshotRuntimeTree(t, root); !reflect.DeepEqual(before, after) {
		t.Fatal("failed preview changed existing files")
	}
	if err := RegenerateInitScript(root, "bash"); err == nil || !strings.Contains(err.Error(), "customized") {
		t.Fatalf("regeneration accepted stale customized private options: %v", err)
	}
	if after, _ := os.ReadFile(private); string(after) != string(custom) {
		t.Fatal("shared option change overwrote customized private script")
	}
	// Restore the recorded generated contents so clearing the override can safely
	// remove the stale catalog artifact rather than preserving a user edit.
	if err := os.WriteFile(private, generated, 0600); err != nil {
		t.Fatal(err)
	}
	if err := manifest.WriteLocalManifest(GetLocalManifestPath(root, "bash"), &manifest.FeatureManifest{}); err != nil {
		t.Fatal(err)
	}
	if err := RegenerateInitScript(root, "bash"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(private); !os.IsNotExist(err) {
		t.Fatal("obsolete unchanged generated private implementation remains")
	}
}

func TestPrivateInitDispatchLoadsExactlyOneEffectiveConfiguration(t *testing.T) {
	for _, shellName := range []string{"bash", "zsh", "fish", "powershell", "posix"} {
		t.Run(shellName, func(t *testing.T) {
			executable := shellName
			if shellName == "powershell" {
				executable = "pwsh"
			}
			if shellName == "posix" {
				executable = "sh"
			}
			interpreter, err := exec.LookPath(executable)
			if err != nil || isWslBashLauncher(interpreter) {
				t.Skip("native interpreter unavailable; WSL uses a separate filesystem")
			}
			root := t.TempDir()
			if err := InitializeShellDirectory(root, shellName); err != nil {
				t.Fatal(err)
			}
			base := &manifest.FeatureManifest{Features: []manifest.FeatureConfig{{Name: "shared", Strategy: "eager"}}}
			if err := manifest.WriteManifest(GetManifestPath(root, shellName), base); err != nil {
				t.Fatal(err)
			}
			local := &manifest.FeatureManifest{Features: []manifest.FeatureConfig{{Name: "shared", Disabled: true}, {Name: "OMD_PRIVATE_SENTINEL_2048", Strategy: "eager"}}}
			if err := manifest.WriteLocalManifest(GetLocalManifestPath(root, shellName), local); err != nil {
				t.Fatal(err)
			}
			private, err := GetLocalFeatureFilePath(root, shellName, "OMD_PRIVATE_SENTINEL_2048")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(private), 0700); err != nil {
				t.Fatal(err)
			}
			script := `printf 'PRIVATE\n' >> "$OMD_TEST_LOG"` + "\n"
			if shellName == "powershell" {
				script = `Add-Content -LiteralPath $env:OMD_TEST_LOG -Value 'PRIVATE'` + "\n"
			}
			if err := os.WriteFile(private, []byte(script), 0600); err != nil {
				t.Fatal(err)
			}
			shared, _ := GetFeatureFilePath(root, shellName, "shared")
			if err := os.WriteFile(shared, []byte(strings.ReplaceAll(script, "PRIVATE", "SHARED")), 0644); err != nil {
				t.Fatal(err)
			}
			if err := RegenerateInitScript(root, shellName); err != nil {
				t.Fatal(err)
			}
			publicPath, _ := GetInitScriptPath(root, shellName)
			public, err := os.ReadFile(publicPath)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(public), "OMD_PRIVATE_SENTINEL_2048") {
				t.Fatal("private feature name leaked into public init")
			}
			log := filepath.Join(root, "runtime.log")
			var args []string
			switch shellName {
			case "bash":
				args = []string{"--noprofile", "--norc", "-c", `. "$1"; cd "$OMD_SHELL_ROOT" && pwd; cd "$OMD_TEST_ROOT" && pwd`, "test", filepath.ToSlash(publicPath)}
			case "zsh":
				args = []string{"-f", "-c", `. "$1"; cd "$OMD_SHELL_ROOT" && pwd; cd "$OMD_TEST_ROOT" && pwd`, "test", filepath.ToSlash(publicPath)}
			case "fish":
				args = []string{"--no-config", "-c", `source "$argv[1]"; cd "$OMD_SHELL_ROOT"; or exit 1; pwd; cd "$OMD_TEST_ROOT"; or exit 1; pwd`, filepath.ToSlash(publicPath)}
			case "powershell":
				args = []string{"-NoProfile", "-NonInteractive", "-Command", `. $env:OMD_TEST_INIT; (Resolve-Path -LiteralPath $OMD_SHELL_ROOT).ProviderPath; (Resolve-Path -LiteralPath $env:OMD_TEST_ROOT).ProviderPath`}
			case "posix":
				args = []string{"-c", `. "$1"; cd "$OMD_SHELL_ROOT" && pwd; cd "$OMD_TEST_ROOT" && pwd`, "test", filepath.ToSlash(publicPath)}
			}
			command := exec.Command(interpreter, args...)
			command.Env = append(os.Environ(), "OMD_TEST_LOG="+log, "OMD_TEST_INIT="+publicPath, "OMD_TEST_ROOT="+filepath.ToSlash(GetShellDirectory(root, shellName)))
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("effective source: %v\n%s", err, output)
			}
			roots := strings.Split(strings.ReplaceAll(strings.TrimSpace(string(output)), "\r", ""), "\n")
			if len(roots) != 2 {
				t.Fatalf("wrong shell root: %q", output)
			}
			if roots[0] != roots[1] {
				// PowerShell may expand a Windows 8.3 path while the fixture
				// retains its short spelling. Both must identify the same directory.
				actual, actualErr := os.Stat(roots[0])
				expected, expectedErr := os.Stat(roots[1])
				if actualErr != nil || expectedErr != nil || !os.SameFile(actual, expected) {
					t.Fatalf("wrong shell root: %q (path errors: %v, %v)", output, actualErr, expectedErr)
				}
			}
			data, err := os.ReadFile(log)
			if err != nil || strings.TrimSpace(string(data)) != "PRIVATE" {
				t.Fatalf("effective script did not run once: %q %v", data, err)
			}
		})
	}
}

func TestPrivateCatalogVariantPreservesSharedAndCustomContent(t *testing.T) {
	root := t.TempDir()
	if err := AddFeatureToShell(root, "bash", "oh-my-posh", "eager", nil, true, map[string]any{"theme": "pure"}); err != nil {
		t.Fatal(err)
	}
	shared, _ := GetFeatureFilePath(root, "bash", "oh-my-posh")
	original, err := os.ReadFile(shared)
	if err != nil {
		t.Fatal(err)
	}
	privateConfig := filepath.Join(root, "OMD_PRIVATE_CONFIG_2048.json")
	if err := os.WriteFile(privateConfig, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	privateOptions := map[string]any{"theme": "OMD_PRIVATE_THEME_2048", "config_file": privateConfig}
	local := &manifest.FeatureManifest{Features: []manifest.FeatureConfig{{Name: "oh-my-posh", Options: privateOptions}}}
	if err := manifest.WriteLocalManifest(GetLocalManifestPath(root, "bash"), local); err != nil {
		t.Fatal(err)
	}
	if err := RegenerateInitScript(root, "bash"); err != nil {
		t.Fatal(err)
	}
	private, _ := GetLocalFeatureFilePath(root, "bash", "oh-my-posh")
	variant, err := os.ReadFile(private)
	if err != nil || !strings.Contains(string(variant), "OMD_PRIVATE_CONFIG_2048") {
		t.Fatalf("private options missing: %q %v", variant, err)
	}
	after, _ := os.ReadFile(shared)
	if string(after) != string(original) {
		t.Fatal("shared implementation overwritten by private options")
	}
	publicPath, _ := GetInitScriptPath(root, "bash")
	public, _ := os.ReadFile(publicPath)
	if strings.Contains(string(public), "OMD_PRIVATE_THEME_2048") {
		t.Fatal("private options leaked to public init")
	}
	localPath, _ := GetLocalInitScriptPath(root, "bash")
	effective, _ := os.ReadFile(localPath)
	if strings.Contains(string(effective), ".local/features/oh-my-posh.sh") {
		t.Fatal("options-only override re-enabled disabled shared feature")
	}
	local.Features[0].DisabledSet = true
	if err := manifest.WriteLocalManifest(GetLocalManifestPath(root, "bash"), local); err != nil {
		t.Fatal(err)
	}
	if err := RegenerateInitScript(root, "bash"); err != nil {
		t.Fatal(err)
	}
	effective, _ = os.ReadFile(localPath)
	if !strings.Contains(string(effective), ".local/features/oh-my-posh.sh") {
		t.Fatal("explicit local enable was not loaded")
	}
	if bash, err := exec.LookPath("bash"); err == nil && runtime.GOOS != "windows" {
		bin := filepath.Join(root, "bin")
		if err := os.Mkdir(bin, 0700); err != nil {
			t.Fatal(err)
		}
		stub := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$OMD_OPTION_LOG\"\n"
		if err := os.WriteFile(filepath.Join(bin, "oh-my-posh"), []byte(stub), 0700); err != nil {
			t.Fatal(err)
		}
		log := filepath.Join(root, "selected-options.log")
		command := exec.Command(bash, "--noprofile", "--norc", "-c", `. "$1"`, "test", publicPath)
		command.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "OMD_OPTION_LOG="+log)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("source private catalog variant: %v %s", err, output)
		}
		output, err := os.ReadFile(log)
		if err != nil || !strings.Contains(string(output), privateConfig) {
			t.Fatalf("private options not actually selected: %q %v", output, err)
		}
	}
	custom := string(variant) + "\n# user private customization\n"
	if err := os.WriteFile(private, []byte(custom), 0600); err != nil {
		t.Fatal(err)
	}
	if err := RegenerateInitScript(root, "bash"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(private)
	if string(data) != custom {
		t.Fatal("unchanged local configuration overwrote customization")
	}
	if err := manifest.WriteLocalManifest(GetLocalManifestPath(root, "bash"), &manifest.FeatureManifest{}); err != nil {
		t.Fatal(err)
	}
	if err := RegenerateInitScript(root, "bash"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(localPath); !os.IsNotExist(err) {
		t.Fatal("obsolete private dispatch target remains active")
	}
	if data, _ := os.ReadFile(private); string(data) != custom {
		t.Fatal("unused customized private file deleted")
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(private)
		if info.Mode().Perm() != 0600 {
			t.Fatal("private script permission broadened")
		}
	}
}
