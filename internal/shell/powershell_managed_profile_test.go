package shell

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
)

func TestProfilePowerShellFeatures(t *testing.T) {
	testPowerShellProfileEditions(t, func(t *testing.T, executable string) {
		repo := filepath.Join(t.TempDir(), "managed repo's PowerShell")
		base := []manifest.FeatureConfig{
			{Name: "eager"},
			{Name: "override", Strategy: "eager"},
			{Name: "disabled", Disabled: true},
			{Name: "disabled-locally"},
			{Name: "missing"},
			{Name: "throws"},
			{Name: "writes-error"},
			{Name: "exits"},
		}
		local := []manifest.FeatureConfig{
			{Name: "override", Strategy: "defer", Options: map[string]any{"fixture": "local"}},
			{Name: "disabled-locally", Disabled: true},
			{Name: "private", Strategy: "on-command", OnCommand: []string{"Invoke-NoSuchFixture"}},
			{Name: "legacy"},
		}
		writeManagedProfileManifests(t, repo, base, local)
		root := GetShellDirectory(repo, "powershell")
		// Windows can spell one directory with either short or long path names.
		// Check a file placed at the expected root instead of comparing strings.
		writePowerShellFixture(t, filepath.Join(root, "managed-root-fixture.txt"), "expected root")
		t.Setenv("OMD_FEATURE_EAGER_LOADED", "1")
		writePowerShellFixture(t, filepath.Join(root, "features", "eager.ps1"), `
if ($env:OMD_FEATURE_EAGER_LOADED -eq '1') { return }
if ([IO.File]::ReadAllText((Join-Path $OMD_SHELL_ROOT 'managed-root-fixture.txt')).Trim() -ne 'expected root') { throw 'wrong shell root' }
$global:omdManagedFixture = 'eager'
Start-Sleep -Milliseconds 250
`)
		for _, name := range []string{"override", "disabled", "disabled-locally"} {
			writePowerShellFixture(t, filepath.Join(root, "features", name+".ps1"), "throw 'wrong script ran'")
		}
		for _, name := range []string{"override", "private"} {
			writePowerShellFixture(t, filepath.Join(root, ".local", "features", name+".ps1"), `
if ([IO.File]::ReadAllText((Join-Path $OMD_SHELL_ROOT 'managed-root-fixture.txt')).Trim() -ne 'expected root') { throw 'wrong shell root' }
if ($null -ne $global:omdManagedFixture) { throw 'feature process was reused' }
$global:omdManagedFixture = 'private'
`)
		}
		writePowerShellFixture(t, filepath.Join(root, "features", "legacy.ps1"), "if ($null -ne $global:omdManagedFixture) { throw 'feature process was reused' }")
		writePowerShellFixture(t, filepath.Join(root, "features", "throws.ps1"), "throw 'terminating fixture fault'")
		writePowerShellFixture(t, filepath.Join(root, "features", "writes-error.ps1"), "Write-Error 'nonterminating fixture fault'; 'script continued'")
		writePowerShellFixture(t, filepath.Join(root, "features", "exits.ps1"), "exit 7")

		report, err := profilePowerShellFeatures(context.Background(), repo, executable)
		if err != nil {
			t.Fatal(err)
		}
		for _, init := range []string{"init.ps1", "init.local.ps1"} {
			if _, err := os.Stat(filepath.Join(root, init)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("unexpected generated init %s: %v", init, err)
			}
		}
		if report.Shell != "powershell" || report.Executable != executable || report.Edition == "" || report.Version == "" || report.Host != "ConsoleHost" ||
			len(report.Profiles) != 0 || len(report.ManagedFeatures) != 8 {
			t.Fatalf("unexpected managed report: %+v", report)
		}
		want := map[string]struct{ strategy, directory, failure string }{
			"eager":        {"eager", "features", ""},
			"override":     {"defer", filepath.Join(".local", "features"), ""},
			"private":      {"on-command", filepath.Join(".local", "features"), ""},
			"legacy":       {"eager", "features", ""},
			"missing":      {"eager", "features", "not found"},
			"throws":       {"eager", "features", "terminating fixture fault"},
			"writes-error": {"eager", "features", "nonterminating fixture fault"},
			"exits":        {"eager", "features", "7"},
		}
		var total float64
		for i, timing := range report.ManagedFeatures {
			expected, ok := want[timing.Name]
			if !ok || timing.Strategy != expected.strategy ||
				timing.Path != filepath.Join(root, expected.directory, timing.Name+".ps1") ||
				(expected.failure == "" && timing.Error != "" || expected.failure != "" && !strings.Contains(timing.Error, expected.failure)) {
				t.Fatalf("unexpected managed feature timing: %+v", timing)
			}
			delete(want, timing.Name)
			if i > 0 && report.ManagedFeatures[i-1].DurationMS < timing.DurationMS {
				t.Fatalf("managed timings are out of rank order: %+v", report.ManagedFeatures)
			}
			switch timing.Name {
			case "eager":
				if timing.DurationMS < 250-powerShellFixtureSleepToleranceMS {
					t.Fatalf("eager script took %.3fms", timing.DurationMS)
				}
			case "missing":
				if timing.DurationMS != 0 {
					t.Fatalf("missing script has a duration: %+v", timing)
				}
			}
			total += timing.DurationMS
		}
		if math.Abs(total-report.TotalMS) > 0.001 {
			t.Fatalf("unexpected total %.3fms, managed sum %.3fms", report.TotalMS, total)
		}
		data, err := json.Marshal(report)
		if err != nil || !strings.Contains(string(data), `"managedFeatures":[`) || strings.Contains(string(data), `"error":""`) {
			t.Fatalf("unexpected JSON: %s, %v", data, err)
		}
	})
}

func TestProfilePowerShellFeaturesManifest(t *testing.T) {
	t.Run("unconfigured", func(t *testing.T) {
		_, err := ProfilePowerShellFeatures(context.Background(), " ")
		if err == nil || !strings.Contains(err.Error(), "repository path is not configured") {
			t.Fatalf("expected a configuration error, got %v", err)
		}
	})
	t.Run("missing", func(t *testing.T) {
		report, err := ProfilePowerShellFeatures(context.Background(), t.TempDir())
		if err != nil || len(report.ManagedFeatures) != 0 || report.TotalMS != 0 {
			t.Fatalf("expected no managed features, got %+v, %v", report, err)
		}
	})
	t.Run("unsafe local manifest", func(t *testing.T) {
		repo := t.TempDir()
		writeManagedProfileManifests(t, repo, []manifest.FeatureConfig{{Name: "shared"}}, nil)
		if err := os.Mkdir(GetLocalManifestPath(repo, "powershell"), 0700); err != nil {
			t.Fatal(err)
		}
		_, err := ProfilePowerShellFeatures(context.Background(), repo)
		if err == nil || !strings.Contains(err.Error(), "not a regular file") {
			t.Fatalf("expected private manifest validation error, got %v", err)
		}
	})
}

func TestProfilePowerShellFeaturesCanceled(t *testing.T) {
	testPowerShellProfileEditions(t, func(t *testing.T, executable string) {
		repo := t.TempDir()
		writeManagedProfileManifests(t, repo, []manifest.FeatureConfig{{Name: "complete"}, {Name: "waiting"}, {Name: "never"}}, nil)
		root := GetFeaturesDirectory(repo, "powershell")
		marker := filepath.Join(repo, "waiting")
		writePowerShellFixture(t, filepath.Join(root, "complete.ps1"), "'completed'")
		writePowerShellFixture(t, filepath.Join(root, "waiting.ps1"),
			"[System.IO.File]::WriteAllText("+powerShellLiteral(marker)+", 'waiting')\nStart-Sleep -Seconds 60")
		writePowerShellFixture(t, filepath.Join(root, "never.ps1"), "throw 'started after cancellation'")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		type result struct {
			report PowerShellProfileReport
			err    error
		}
		done := make(chan result, 1)
		go func() {
			report, err := profilePowerShellFeatures(ctx, repo, executable)
			done <- result{report, err}
		}()
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		deadline := time.After(10 * time.Second)
		for ctx.Err() == nil {
			select {
			case <-ticker.C:
				if _, err := os.Stat(marker); err == nil {
					cancel()
				}
			case got := <-done:
				t.Fatalf("profiler returned before cancellation: %+v, %v", got.report, got.err)
			case <-deadline:
				t.Fatal("timed out waiting for the feature to start")
			}
		}
		select {
		case got := <-done:
			if !errors.Is(got.err, context.Canceled) || len(got.report.ManagedFeatures) != 2 ||
				got.report.ManagedFeatures[0].Name != "complete" || got.report.ManagedFeatures[0].Error != "" ||
				got.report.ManagedFeatures[1].Name != "waiting" || !strings.Contains(got.report.ManagedFeatures[1].Error, "context canceled") {
				t.Fatalf("expected partial timings and cancellation, got %+v, %v", got.report, got.err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("profiler did not return after cancellation")
		}
	})
}

func writeManagedProfileManifests(t *testing.T, repo string, base, local []manifest.FeatureConfig) {
	t.Helper()
	if err := os.MkdirAll(GetShellDirectory(repo, "powershell"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := manifest.WriteManifest(GetManifestPath(repo, "powershell"), &manifest.FeatureManifest{Features: base}); err != nil {
		t.Fatal(err)
	}
	if local != nil {
		if err := manifest.WriteLocalManifest(GetLocalManifestPath(repo, "powershell"), &manifest.FeatureManifest{Features: local}); err != nil {
			t.Fatal(err)
		}
	}
}
