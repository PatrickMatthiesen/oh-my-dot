package shell

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
)

func TestPowerShellProfileMeasuresSharedScopeAndFeatures(t *testing.T) {
	testPowerShellProfileEditions(t, testPowerShellProfileMeasuresSharedScopeAndFeatures)
}

func testPowerShellProfileMeasuresSharedScopeAndFeatures(t *testing.T, executable string) {
	paths := powerShellProfileFixtures(t)
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
Invoke-MissingOMDEagerFixtureCommand
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
Invoke-MissingOMDProfileFixtureCommand
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
	if report.Shell != "powershell" || report.Executable != executable || report.Edition == "" || report.Version == "" || report.Host != "ConsoleHost" || !report.AttributionAvailable || len(report.Profiles) != 4 {
		t.Fatalf("unexpected report: %+v", report)
	}
	wantWarnings := []string{
		"PowerShell recorded 2 error(s) while running " + paths[0] + ".",
		"PowerShell recorded 1 error(s) while running " + paths[1] + ".",
	}
	if !slices.Equal(report.Warnings, wantWarnings) {
		t.Fatalf("unexpected warnings: %v", report.Warnings)
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
	testPowerShellProfileEditions(t, testPowerShellProfileWithoutAttribution)
}

func testPowerShellProfileWithoutAttribution(t *testing.T, executable string) {
	paths := powerShellProfileFixtures(t)
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

func TestPowerShellProfileAvoidsObserverModuleImports(t *testing.T) {
	testPowerShellProfileEditions(t, func(t *testing.T, executable string) {
		paths := powerShellProfileFixtures(t)
		const checkModules = `
if (Get-Module Microsoft.PowerShell.Management, Microsoft.PowerShell.Utility) {
  throw 'the loader or profiler imported a module'
}
`
		initPath := filepath.Join(filepath.Dir(paths[0]), "init.ps1")
		writePowerShellFixture(t, initPath, generatePowerShellInit(FeaturesByStrategy{}))
		writePowerShellFixture(t, paths[0], checkModules+`
Set-StrictMode -Version Latest
$PSModuleAutoLoadingPreference = 'None'
function Split-Path {
  param([switch]$Parent, [string]$Path)
  [IO.Path]::GetDirectoryName($Path)
}
. `+powerShellLiteral(initPath)+"\n"+checkModules)
		writePowerShellFixture(t, paths[1], checkModules+`
if ($PSModuleAutoLoadingPreference -cne 'None') { throw 'profiler changed autoload preference' }
`)
		cmd := exec.Command(executable, "-NoLogo", "-NoProfile", "-NonInteractive", "-File", paths[0])
		if output, err := cmd.CombinedOutput(); err != nil || len(output) != 0 {
			t.Fatalf("inactive loader with strict mode and no autoload: %v\n%s", err, output)
		}
		report, err := profilePowerShell(context.Background(), executable, &paths)
		if err != nil || !report.AttributionAvailable || len(report.Profiles) != 2 || len(report.Warnings) != 0 {
			t.Fatalf("active loader with strict mode and no autoload: %+v, %v", report, err)
		}
	})
}

func TestPowerShellProfileFailures(t *testing.T) {
	testPowerShellProfileEditions(t, testPowerShellProfileFailures)
}

func testPowerShellProfileFailures(t *testing.T, executable string) {
	for _, tt := range []struct{ name, script, want string }{
		{"profile throw", "throw 'fixture error'", ": fixture error"},
		{"process exit", "[Environment]::Exit(0)", "did not write a startup report"},
		{"profile break", "break", "profile stopped before it finished running"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			paths := powerShellProfileFixtures(t)
			writePowerShellFixture(t, paths[0], tt.script)
			report, err := profilePowerShell(context.Background(), executable, &paths)
			if err == nil || !strings.Contains(err.Error(), tt.want) || len(report.Profiles) != 0 {
				t.Fatalf("expected %q and no timings, got %+v, %v", tt.want, report, err)
			}
		})
	}
	t.Run("continues after throw and parse failure", func(t *testing.T) {
		paths := powerShellProfileFixtures(t)
		writePowerShellFixture(t, paths[0], "$omdFailureFixture = 'first'")
		writePowerShellFixture(t, paths[1], "$omdFailureFixture += '-second'; throw 'fixture error'")
		writePowerShellFixture(t, paths[2], "if (")
		writePowerShellFixture(t, paths[3], "if ($omdFailureFixture -cne 'first-second') { throw 'profile state was lost' }")
		report, err := profilePowerShell(context.Background(), executable, &paths)
		if err == nil || !strings.Contains(err.Error(), paths[1]+": fixture error") ||
			!strings.Contains(err.Error(), paths[2]+": could not parse profile: ") || errors.Is(err, errIncompletePowerShellReport) {
			t.Fatalf("expected both profile failures in a complete report, got %v", err)
		}
		var completedPaths []string
		for _, profile := range report.Profiles {
			completedPaths = append(completedPaths, profile.Path)
		}
		slices.Sort(completedPaths)
		if !slices.Equal(completedPaths, []string{paths[0], paths[3]}) {
			t.Fatalf("expected the first and fourth profile timings, got %+v", report)
		}
	})
	t.Run("nested parse and in-profile security errors are warnings", func(t *testing.T) {
		paths := powerShellProfileFixtures(t)
		nestedPath := filepath.Join(filepath.Dir(paths[0]), "nested.ps1")
		writePowerShellFixture(t, nestedPath, "if (")
		writePowerShellFixture(t, paths[0], ". "+powerShellLiteral(nestedPath)+`
Write-Error -Exception ([System.Management.Automation.PSSecurityException]::new('fixture security error')) -ErrorId UnauthorizedAccess -ErrorAction Continue
$omdNestedFixture = $true`)
		writePowerShellFixture(t, paths[1], "if (-not $omdNestedFixture) { throw 'profile did not continue' }")
		report, err := profilePowerShell(context.Background(), executable, &paths)
		wantWarning := "PowerShell recorded 2 error(s) while running " + paths[0] + "."
		if err != nil || len(report.Profiles) != 2 || !slices.Contains(report.Warnings, wantWarning) {
			t.Fatalf("expected two timings and a warning, got %+v, %v", report, err)
		}
	})
	t.Run("feature before profile throw", func(t *testing.T) {
		paths := powerShellProfileFixtures(t)
		root := filepath.Dir(paths[0])
		initPath := filepath.Join(root, "init.ps1")
		featurePath := filepath.Join(root, "features", "completed.ps1")
		writePowerShellFixture(t, initPath, generatePowerShellInit(FeaturesByStrategy{Eager: []string{"completed"}}))
		writePowerShellFixture(t, featurePath, "Start-Sleep -Milliseconds 30")
		writePowerShellFixture(t, paths[0], ". "+powerShellLiteral(initPath)+"\nthrow 'after completed feature'")
		report, err := profilePowerShell(context.Background(), executable, &paths)
		if err == nil || !strings.Contains(err.Error(), "after completed feature") {
			t.Fatalf("expected the profile throw, got %v", err)
		}
		if len(report.Profiles) != 1 || report.Profiles[0].Path != paths[0] || !report.AttributionAvailable {
			t.Fatalf("expected the failed profile with attribution, got %+v", report)
		}
		profile := report.Profiles[0]
		if len(profile.Features) != 1 || profile.Features[0].Name != "completed" ||
			profile.Features[0].DurationMS < 30 || profile.DurationMS < profile.Features[0].DurationMS ||
			report.TotalMS != profile.DurationMS {
			t.Fatalf("expected the completed feature and elapsed profile timing, got %+v", report)
		}
	})
	t.Run("hard exit after completed and failed profiles", func(t *testing.T) {
		paths := powerShellProfileFixtures(t)
		writePowerShellFixture(t, paths[0], "'successful profile'")
		writePowerShellFixture(t, paths[1], "throw 'fixture error'")
		writePowerShellFixture(t, paths[2], "[Environment]::Exit(7)")
		report, err := profilePowerShell(context.Background(), executable, &paths)
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) || exitErr.ExitCode() != 7 || !strings.Contains(err.Error(), paths[1]+": fixture error") {
			t.Fatalf("expected the profile failure and process exit code 7, got %v", err)
		}
		if len(report.Profiles) != 1 || report.Profiles[0].Path != paths[0] {
			t.Fatalf("expected the completed profile timing, got %+v", report)
		}
	})
	t.Run("cancelled or timed out", func(t *testing.T) {
		paths := powerShellProfileFixtures(t)
		writePowerShellFixture(t, paths[0], "Start-Sleep -Seconds 60")
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		_, err := profilePowerShell(ctx, executable, &paths)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected deadline error, got %v", err)
		}
	})
	t.Run("cancelled after completed profile", func(t *testing.T) {
		paths := powerShellProfileFixtures(t)
		startedPath := filepath.Join(filepath.Dir(paths[0]), "second-profile-started")
		writePowerShellFixture(t, paths[0], "'successful profile'")
		writePowerShellFixture(t, paths[1], "[IO.File]::WriteAllText("+powerShellLiteral(startedPath)+", '')\nStart-Sleep -Seconds 60")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		type result struct {
			report PowerShellProfileReport
			err    error
		}
		done := make(chan result, 1)
		go func() {
			report, err := profilePowerShell(ctx, executable, &paths)
			done <- result{report, err}
		}()
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case got := <-done:
				if !errors.Is(got.err, context.Canceled) || len(got.report.Profiles) != 1 || got.report.Profiles[0].Path != paths[0] {
					t.Fatalf("expected one completed profile and context cancellation, got %+v, %v", got.report, got.err)
				}
				return
			case <-ticker.C:
				if _, err := os.Stat(startedPath); err == nil {
					cancel()
				}
			}
		}
	})
}

func TestPowerShellProfileRestrictedExecutionPolicy(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("requires Windows PowerShell execution policy")
	}
	executable, err := exec.LookPath("powershell.exe")
	if err != nil {
		t.Skip("Windows PowerShell is unavailable")
	}
	t.Setenv("PSExecutionPolicyPreference", "Restricted")
	cmd := exec.Command(executable, "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", "Get-ExecutionPolicy")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("read effective execution policy: %v\n%s", err, output)
	}
	if policy := strings.TrimSpace(string(output)); policy != "Restricted" {
		t.Skipf("effective execution policy is %q instead of process-scoped Restricted", policy)
	}
	paths := powerShellProfileFixtures(t)
	writePowerShellFixture(t, paths[0], "throw 'restricted fixture ran'")
	report, err := profilePowerShell(context.Background(), executable, &paths)
	if err == nil || !strings.Contains(err.Error(), paths[0]+": authorization blocked the profile: ") ||
		errors.Is(err, errIncompletePowerShellReport) || len(report.Profiles) != 0 {
		t.Fatalf("expected the blocked profile diagnostic and no timings, got %+v, %v", report, err)
	}
}

func testPowerShellProfileEditions(t *testing.T, test func(*testing.T, string)) {
	t.Helper()
	names := []string{"pwsh"}
	if runtime.GOOS == "windows" {
		names = append(names, "powershell.exe")
		// Use Process scope for the child running disposable local scripts.
		t.Setenv("PSExecutionPolicyPreference", "Bypass")
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			executable, err := exec.LookPath(name)
			if err != nil {
				t.Skipf("%s is unavailable", name)
			}
			test(t, executable)
		})
	}
}

func powerShellProfileFixtures(t *testing.T) [4]string {
	t.Helper()
	root := t.TempDir()
	return [4]string{
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
