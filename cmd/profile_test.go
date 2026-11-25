package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
)

func TestProfileJSON(t *testing.T) {
	profileFailure := errors.New("profile replay interrupted")
	report := shell.PowerShellProfileReport{
		Shell: "powershell", Executable: "pwsh", Edition: "Core", Version: "7.5.0", Host: "ConsoleHost",
		TotalMS: 125.25, AttributionAvailable: true, Warnings: []string{"example warning"},
		Profiles: []shell.ProfileTiming{{
			Path: "profile.ps1", DurationMS: 125.25,
			Features: []shell.FeatureTiming{{Name: "test", Path: "test.ps1", Strategy: "eager", DurationMS: 12.5}},
		}},
	}
	tests := []struct {
		name         string
		args         []string
		installed    bool
		wantStatus   string
		wantShell    string
		wantProbe    bool
		wantRun      bool
		wantReport   bool
		returnReport shell.PowerShellProfileReport
		returnError  error
	}{
		{
			name: "default profiles installed PowerShell", installed: true,
			wantStatus: "ok", wantShell: "powershell", wantProbe: true, wantRun: true, wantReport: true, returnReport: report,
		},
		{
			name: "explicit selection", args: []string{"--shell", "PowerShell"}, installed: true,
			wantStatus: "ok", wantShell: "powershell", wantProbe: true, wantRun: true, wantReport: true, returnReport: report,
		},
		{
			name: "unsupported shell", args: []string{"--shell", "bash"}, installed: true,
			wantStatus: "unsupported", wantShell: "bash",
		},
		{
			name: "empty selection", args: []string{"--shell="}, installed: true,
			wantStatus: "unsupported",
		},
		{
			name:       "no installed profilers",
			wantStatus: "unavailable", wantShell: "powershell", wantProbe: true,
		},
		{
			name: "requested PowerShell missing", args: []string{"--shell", "powershell"},
			wantStatus: "unavailable", wantShell: "powershell", wantProbe: true,
		},
		{
			name: "partial failure", installed: true, returnReport: report, returnError: profileFailure,
			wantStatus: "partial", wantShell: "powershell", wantProbe: true, wantRun: true, wantReport: true,
		},
		{
			name: "failure before timing", installed: true, returnError: profileFailure,
			wantStatus: "failed", wantShell: "powershell", wantProbe: true, wantRun: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			probes, runs := 0, 0
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			command := newProfileCommand(func(name string) bool {
				probes++
				if name != "powershell" {
					t.Fatalf("availability probe used shell %q", name)
				}
				return tt.installed
			}, func(got context.Context) (shell.PowerShellProfileReport, error) {
				runs++
				cancel()
				if !errors.Is(got.Err(), context.Canceled) {
					t.Fatal("profiler context did not inherit command cancellation")
				}
				return tt.returnReport, tt.returnError
			})
			command.SetOut(&stdout)
			command.SetErr(&stderr)
			command.SetArgs(append([]string{"--json"}, tt.args...))
			err := command.ExecuteContext(ctx)
			if (err == nil) != (tt.wantStatus == "ok") {
				t.Fatalf("ExecuteContext returned %v for status %q", err, tt.wantStatus)
			}
			if tt.returnError != nil && !errors.Is(err, tt.returnError) {
				t.Fatalf("ExecuteContext returned %v, want wrapped %v", err, tt.returnError)
			}
			if probes > 1 || (probes == 1) != tt.wantProbe || runs > 1 || (runs == 1) != tt.wantRun {
				t.Fatalf("unexpected availability probes %d or profiler runs %d", probes, runs)
			}

			var document profileDocument
			decoder := json.NewDecoder(&stdout)
			if err := decoder.Decode(&document); err != nil {
				t.Fatalf("decode stdout: %v", err)
			}
			if err := decoder.Decode(new(any)); err != io.EOF {
				t.Fatalf("stdout contains extra data: %v", err)
			}
			if document.SchemaVersion != 1 || len(document.Results) != 1 {
				t.Fatalf("unexpected document: %+v", document)
			}
			result := document.Results[0]
			if result.Shell != tt.wantShell || result.Status != tt.wantStatus || (result.Report != nil) != tt.wantReport {
				t.Fatalf("unexpected result: %+v", result)
			}
			if err != nil && (result.Error != err.Error() || !strings.Contains(stderr.String(), result.Error)) {
				t.Fatalf("missing error in report or stderr: result=%+v, stderr=%q", result, stderr.String())
			}
			if result.Report != nil && (result.Report.TotalMS != 125.25 || len(result.Report.Profiles) != 1 ||
				len(result.Report.Profiles[0].Features) != 1 || result.Report.Profiles[0].Features[0].DurationMS != 12.5) {
				t.Fatalf("report lost timings: %+v", result.Report)
			}
		})
	}
}

func TestProfileTextDisplaysTimingsAndMeasuredSuggestions(t *testing.T) {
	report := shell.PowerShellProfileReport{
		Version: "7.5.0", Edition: "Core", Host: "ConsoleHost", Executable: "pwsh",
		TotalMS: 520, AttributionAvailable: true, Warnings: []string{"example warning"},
		Profiles: []shell.ProfileTiming{
			{Path: "user.ps1", DurationMS: 250, Features: []shell.FeatureTiming{
				{Name: "oh-my-posh", Path: "posh.ps1", Strategy: "eager", DurationMS: 175},
				{Name: "another-feature", Path: "another.ps1", Strategy: "defer", DurationMS: 10},
			}},
			{Path: "host.ps1", DurationMS: 150, Features: []shell.FeatureTiming{
				{Name: "fast-feature", Path: "fast.ps1", Strategy: "eager", DurationMS: 5},
			}},
			{Path: "other.ps1", DurationMS: 120, Features: []shell.FeatureTiming{
				{Name: "slow-feature", Path: "slow.ps1", Strategy: "eager", DurationMS: 115},
			}},
		},
	}
	var out bytes.Buffer
	if err := writeProfileText(&out, profileResult{Status: "partial", Report: &report}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"PowerShell 7.5.0 (Core, ConsoleHost)",
		"Partial result",
		"Total profile time: 520.00 ms",
		"  250.00 ms  user.ps1\n    175.00 ms  oh-my-posh (eager)  posh.ps1",
		"Inspect the Oh My Posh theme and cache configuration.",
		"Check whether auto_upgrade runs during startup.",
		"  150.00 ms  host.ps1\n    Inspect imports and commands in this profile.",
		"Consider defer if this feature can wait until the shell is idle.",
		"Warning: example warning",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output missing %q: %s", want, out.String())
		}
	}

	out.Reset()
	report.AttributionAvailable = false
	report.Warnings = nil
	report.Profiles = report.Profiles[1:]
	if err := writeProfileText(&out, profileResult{Status: "ok", Report: &report}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Managed feature timings are unavailable for this run.") ||
		strings.Contains(out.String(), "Oh My Posh") || strings.Contains(out.String(), "apply") {
		t.Fatalf("unexpected output for unavailable attribution: %s", out.String())
	}
}

func TestProfileDoesNotRequireAnInitializedRepository(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	command, _, err := rootCmd.Find([]string{"profile"})
	if err != nil || command.Name() != "profile" {
		t.Fatalf("profile is not registered: command=%v, err=%v", command, err)
	}
	if err := rootCmd.PersistentPreRunE(command, nil); err != nil {
		t.Fatalf("profile requires an initialized repository: %v", err)
	}
}
