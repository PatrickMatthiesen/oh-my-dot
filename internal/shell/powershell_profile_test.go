package shell

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
)

func TestPowerShellProfileMeasuresSharedScopeAndFeatures(t *testing.T) {
	executable, paths := powerShellProfileFixtures(t)
	repo := filepath.Join(filepath.Dir(paths[0]), "managed repo")
	if err := InitializeShellDirectory(repo, "powershell"); err != nil {
		t.Fatal(err)
	}
	base := &manifest.FeatureManifest{Features: []manifest.FeatureConfig{
		{Name: "eager-demo", Strategy: "eager"},
		{Name: "lazy-demo", Strategy: "on-command", OnCommand: []string{"Invoke-OMDFixture"}},
		{Name: "unused-demo", Strategy: "on-command", OnCommand: []string{"Invoke-OMDUnused"}},
	}}
	local := &manifest.FeatureManifest{Features: []manifest.FeatureConfig{
		{Name: "defer-demo", Strategy: "defer"},
	}}
	if err := manifest.WriteManifest(GetManifestPath(repo, "powershell"), base); err != nil {
		t.Fatal(err)
	}
	if err := manifest.WriteLocalManifest(GetLocalManifestPath(repo, "powershell"), local); err != nil {
		t.Fatal(err)
	}
	eager, _ := GetFeatureFilePath(repo, "powershell", "eager-demo")
	deferred, _ := GetLocalFeatureFilePath(repo, "powershell", "defer-demo")
	lazy, _ := GetFeatureFilePath(repo, "powershell", "lazy-demo")
	unused, _ := GetFeatureFilePath(repo, "powershell", "unused-demo")
	writePowerShellFixture(t, eager, `
$omdFixtureOrder += 'E'
Start-Sleep -Milliseconds 30
function Get-OMDFixture { $omdFixtureOrder }
`)
	writePowerShellFixture(t, deferred, `
if ((Get-OMDFixture) -cne '12E') { throw 'eager feature did not share the profile scope' }
$omdFixtureOrder += 'D'
Start-Sleep -Milliseconds 120
`)
	writePowerShellFixture(t, lazy, `
if ((Get-OMDFixture) -cne '12ED3') { throw 'trigger did not inherit the profile scope' }
Start-Sleep -Milliseconds 40
function Invoke-OMDFixture { "loaded-$($args[0])" }
`)
	writePowerShellFixture(t, unused, "throw 'uninvoked wrapper loaded its feature'")
	if err := RegenerateInitScript(repo, "powershell"); err != nil {
		t.Fatal(err)
	}
	initPath, _ := GetInitScriptPath(repo, "powershell")
	writePowerShellFixture(t, paths[0], `
$omdFixtureOrder = '1'
'unrelated success output'
Write-Warning 'unrelated warning'
Write-Error 'unrelated nonterminating error' -ErrorAction Continue
[Console]::WriteLine('direct stdout')
[Console]::Error.WriteLine('direct stderr')
if ($null -ne [Console]::ReadLine()) { throw 'profiling stdin was open' }
`)
	writePowerShellFixture(t, paths[1], `
if ($omdFixtureOrder -cne '1') { throw 'first profile did not run in shared scope' }
$omdFixtureOrder += '2'
. `+powerShellLiteral(initPath))
	writePowerShellFixture(t, paths[2], `
if ($omdFixtureOrder -cne '12ED') { throw 'features did not share the profile scope' }
$omdFixtureOrder += '3'
if ((Invoke-OMDFixture first) -cne 'loaded-first') { throw 'wrapper lost the function or its argument' }
`)
	writePowerShellFixture(t, paths[3], `
if ($omdFixtureOrder -cne '12ED3') { throw 'profiles ran out of order' }
if ((Invoke-OMDFixture next) -cne 'loaded-next') { throw 'wrapper failed to promote the loaded function' }
. `+powerShellLiteral(initPath))

	report, err := profilePowerShell(context.Background(), executable, &paths)
	if err != nil {
		t.Fatal(err)
	}
	if report.Shell != "powershell" || report.Executable != executable || report.Edition == "" || report.Version == "" || report.Host != "ConsoleHost" || !report.AttributionAvailable || len(report.Profiles) != 4 || len(report.Warnings) != 0 {
		t.Fatalf("unexpected report: %+v", report)
	}
	var total float64
	byPath := make(map[string][]FeatureTiming)
	for i, profile := range report.Profiles {
		if i > 0 && report.Profiles[i-1].DurationMS < profile.DurationMS {
			t.Fatalf("profiles are out of rank order: %+v", report.Profiles)
		}
		total += profile.DurationMS
		byPath[profile.Path] = profile.Features
	}
	if math.Abs(total-report.TotalMS) > 0.001 || total < 190 {
		t.Fatalf("unexpected total %.3f ms, profile sum %.3f ms", report.TotalMS, total)
	}
	if len(byPath[paths[0]]) != 0 || len(byPath[paths[3]]) != 0 {
		t.Fatalf("recorded timings without feature loads: %+v", byPath)
	}
	features := byPath[paths[1]]
	if len(features) != 2 || features[0].DurationMS < features[1].DurationMS {
		t.Fatalf("expected two ranked features in second profile: %+v", features)
	}
	features = append(features, byPath[paths[2]]...)
	if len(features) != 3 {
		t.Fatalf("expected three timed features: %+v", features)
	}
	want := map[string]struct {
		path, strategy string
		minMS          float64
	}{
		"eager-demo": {eager, "eager", 30},
		"defer-demo": {deferred, "defer", 120},
		"lazy-demo":  {lazy, "on-command", 40},
	}
	for _, feature := range features {
		expected := want[feature.Name]
		actualFile, actualErr := os.Stat(feature.Path)
		expectedFile, expectedErr := os.Stat(expected.path)
		if feature.Strategy != expected.strategy || feature.DurationMS < expected.minMS ||
			actualErr != nil || expectedErr != nil || !os.SameFile(actualFile, expectedFile) {
			t.Fatalf("unexpected feature timing: %+v, want %+v", feature, expected)
		}
		delete(want, feature.Name)
	}
}

func TestPowerShellProfileWithoutAttribution(t *testing.T) {
	executable, paths := powerShellProfileFixtures(t)
	for _, tt := range []struct {
		name, script string
		profiles     int
	}{
		{name: "no profiles"},
		{name: "ordinary profile", script: "'profile output'", profiles: 1},
		{name: "guarded loader", script: "$global:OMD_POWERSHELL_LOADED = $true\n" + generatePowerShellInit(FeaturesByStrategy{}), profiles: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.script != "" {
				writePowerShellFixture(t, paths[0], tt.script)
			}
			report, err := profilePowerShell(context.Background(), executable, &paths)
			if err != nil {
				t.Fatal(err)
			}
			if report.AttributionAvailable || len(report.Profiles) != tt.profiles || len(report.Warnings) == 0 || strings.Contains(strings.Join(report.Warnings, " "), "stale") {
				t.Fatalf("unexpected report: %+v", report)
			}
			data, err := json.Marshal(report)
			if err != nil {
				t.Fatal(err)
			}
			wantArray := `"profiles":[]`
			if tt.profiles != 0 {
				wantArray = `"features":[]`
			}
			if !strings.Contains(string(data), wantArray) {
				t.Fatalf("report lacks %s in %s", wantArray, data)
			}
		})
	}
}

func TestPowerShellProfileFailures(t *testing.T) {
	executable, paths := powerShellProfileFixtures(t)
	for _, tt := range []struct{ name, script, want string }{
		{"profile throw", "throw 'fixture error'", paths[0] + ": fixture error"},
		{"process exit", "[Environment]::Exit(0)", "did not write a startup report"},
		{"profile break", "break", "did not write a startup report"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			writePowerShellFixture(t, paths[0], tt.script)
			_, err := profilePowerShell(context.Background(), executable, &paths)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
	t.Run("partial report", func(t *testing.T) {
		writePowerShellFixture(t, paths[0], "'successful profile'")
		writePowerShellFixture(t, paths[1], "throw 'fixture error'")
		writePowerShellFixture(t, paths[2], "throw 'did not stop after failure'")
		report, err := profilePowerShell(context.Background(), executable, &paths)
		if err == nil || !strings.Contains(err.Error(), paths[1]+": fixture error") {
			t.Fatalf("expected the second profile to fail, got %v", err)
		}
		if len(report.Profiles) != 1 || report.Profiles[0].Path != paths[0] {
			t.Fatalf("expected the completed profile timing, got %+v", report)
		}
	})
	t.Run("cancelled or timed out", func(t *testing.T) {
		writePowerShellFixture(t, paths[0], "Start-Sleep -Seconds 60")
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		_, err := profilePowerShell(ctx, executable, &paths)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected deadline error, got %v", err)
		}
	})
}

func powerShellProfileFixtures(t *testing.T) (string, [4]string) {
	t.Helper()
	executable, err := exec.LookPath("pwsh")
	if err != nil {
		t.Skip("pwsh is unavailable")
	}
	root := t.TempDir()
	return executable, [4]string{
		filepath.Join(root, "first profile.ps1"),
		filepath.Join(root, "second's profile.ps1"),
		filepath.Join(root, "third profile.ps1"),
		filepath.Join(root, "fourth profile.ps1"),
	}
}

func writePowerShellFixture(t *testing.T, path, script string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(script+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
}
