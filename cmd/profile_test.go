package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
)

func TestProfileJSON(t *testing.T) {
	profileFailure := errors.New("profile replay interrupted")
	featureFailure := errors.New("feature failed to load")
	fullReport := shell.ShellProfileReport{
		Shell: "powershell", Executable: "pwsh", Edition: "Core", Version: "7.5.0", Host: "ConsoleHost",
		TotalMS: 125.25, AttributionAvailable: true, Warnings: []string{"example warning"},
		Profiles: []shell.ProfileTiming{{
			Path: "profile.ps1", DurationMS: 125.25,
			Features: []shell.FeatureTiming{{Name: "test", Path: "test.ps1", Strategy: "eager", DurationMS: 12.5}},
		}},
	}
	managedReport := shell.ShellProfileReport{
		Shell: "powershell", Executable: "pwsh", Edition: "Core", Version: "7.5.0", Host: "ConsoleHost",
		TotalMS: 12.5,
		ManagedFeatures: []shell.ManagedFeatureTiming{{
			FeatureTiming: shell.FeatureTiming{Name: "test", Path: "test.ps1", Strategy: "defer", DurationMS: 12.5},
		}},
	}
	partialManagedReport := shell.ShellProfileReport{
		ManagedFeatures: []shell.ManagedFeatureTiming{{
			FeatureTiming: shell.FeatureTiming{Name: "broken", Path: "broken.ps1", Strategy: "eager"},
			Error:         featureFailure.Error(),
		}},
	}
	tests := []struct {
		name         string
		args         []string
		installed    bool
		wantStatus   string
		wantShell    string
		wantMode     string
		wantProbe    bool
		wantManaged  bool
		wantFull     bool
		wantReport   bool
		returnReport shell.ShellProfileReport
		returnError  error
	}{
		{
			name: "default profiles managed features", installed: true,
			wantStatus: "ok", wantShell: "powershell", wantMode: "managed", wantProbe: true, wantManaged: true, wantReport: true, returnReport: managedReport,
		},
		{
			name: "explicit selection", args: []string{"--shell", "PowerShell"}, installed: true,
			wantStatus: "ok", wantShell: "powershell", wantMode: "managed", wantProbe: true, wantManaged: true, wantReport: true, returnReport: managedReport,
		},
		{
			name: "full profiles PowerShell without reading the repository", args: []string{"--full", "--shell", "PowerShell"}, installed: true,
			wantStatus: "ok", wantShell: "powershell", wantMode: "full", wantProbe: true, wantFull: true, wantReport: true, returnReport: fullReport,
		},
		{
			name: "unsupported shell", args: []string{"--shell", "Unknown"}, installed: true,
			wantStatus: "unsupported", wantShell: "Unknown", wantMode: "managed",
		},
		{
			name: "empty selection", args: []string{"--full", "--shell="}, installed: true,
			wantStatus: "unsupported", wantMode: "full",
		},
		{
			name: "no installed profilers", wantProbe: true,
		},
		{
			name: "requested PowerShell missing", args: []string{"--shell", "powershell"},
			wantStatus: "unavailable", wantShell: "powershell", wantMode: "managed", wantProbe: true,
		},
		{
			name: "no enabled managed features", installed: true,
			wantStatus: "ok", wantShell: "powershell", wantMode: "managed", wantProbe: true, wantManaged: true, wantReport: true,
		},
		{
			name: "partial managed report retains feature failure", installed: true, returnReport: partialManagedReport,
			wantStatus: "partial", wantShell: "powershell", wantMode: "managed", wantProbe: true, wantManaged: true, wantReport: true,
		},
		{
			name: "partial managed report retains timings after interruption", installed: true, returnReport: managedReport, returnError: context.DeadlineExceeded,
			wantStatus: "partial", wantShell: "powershell", wantMode: "managed", wantProbe: true, wantManaged: true, wantReport: true,
		},
		{
			name: "partial full report retains timings", args: []string{"--full"}, installed: true, returnReport: fullReport, returnError: profileFailure,
			wantStatus: "partial", wantShell: "powershell", wantMode: "full", wantProbe: true, wantFull: true, wantReport: true,
		},
		{
			name: "failure before timing", installed: true, returnError: featureFailure,
			wantStatus: "failed", wantShell: "powershell", wantMode: "managed", wantProbe: true, wantManaged: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			probes, repoReads, managedRuns, fullRuns := 0, 0, 0, 0
			repoPath := t.TempDir()
			command := newProfileCommand(func(name string) bool {
				probes++
				return name == "powershell" && tt.installed
			}, func() string {
				repoReads++
				return repoPath
			}, func(_ context.Context, name, path string) (shell.ShellProfileReport, error) {
				managedRuns++
				if name != "powershell" {
					t.Fatalf("managed profiler used shell %q", name)
				}
				if path != repoPath {
					t.Fatalf("managed profiler got repository path %q, want %q", path, repoPath)
				}
				return tt.returnReport, tt.returnError
			}, func(_ context.Context, name string) (shell.ShellProfileReport, error) {
				fullRuns++
				if name != "powershell" {
					t.Fatalf("full profiler used shell %q", name)
				}
				return tt.returnReport, tt.returnError
			})
			command.SetOut(&stdout)
			command.SetErr(&stderr)
			command.SetArgs(append([]string{"--json"}, tt.args...))
			err := command.Execute()
			if (err == nil) != (tt.wantStatus == "ok") {
				t.Fatalf("ExecuteContext returned %v for status %q", err, tt.wantStatus)
			}
			if tt.returnError != nil && !errors.Is(err, tt.returnError) {
				t.Fatalf("ExecuteContext returned %v, want wrapped %v", err, tt.returnError)
			}
			wantProbes := 0
			if tt.wantProbe {
				wantProbes = 5
				if command.Flags().Changed("shell") {
					wantProbes = 1
				}
			}
			if probes != wantProbes ||
				managedRuns > 1 || (managedRuns == 1) != tt.wantManaged ||
				fullRuns > 1 || (fullRuns == 1) != tt.wantFull || repoReads != managedRuns {
				t.Fatalf("unexpected availability probes %d, repo path reads %d, managed runs %d, or full runs %d", probes, repoReads, managedRuns, fullRuns)
			}

			if strings.Contains(stdout.String(), `"managedFeatures":`) != (len(tt.returnReport.ManagedFeatures) > 0) {
				t.Fatalf("unexpected managedFeatures field in %s", stdout.String())
			}
			var document profileDocument
			decoder := json.NewDecoder(&stdout)
			if err := decoder.Decode(&document); err != nil {
				t.Fatalf("decode stdout: %v", err)
			}
			if err := decoder.Decode(new(any)); err != io.EOF {
				t.Fatalf("stdout contains extra data: %v", err)
			}
			if document.SchemaVersion != 1 {
				t.Fatalf("unexpected document: %+v", document)
			}
			if tt.wantStatus == "" {
				if document.Results == nil || len(document.Results) != 0 || !strings.Contains(stderr.String(), "no supported shell executables found") {
					t.Fatalf("expected an empty results array and a diagnostic: %+v, %q", document, stderr.String())
				}
				return
			}
			if len(document.Results) != 1 {
				t.Fatalf("unexpected document: %+v", document)
			}
			result := document.Results[0]
			if result.Shell != tt.wantShell || result.Mode != tt.wantMode || result.Status != tt.wantStatus || (result.Report != nil) != tt.wantReport {
				t.Fatalf("unexpected result: %+v", result)
			}
			if err != nil && (result.Error != err.Error() || !strings.Contains(stderr.String(), result.Error)) {
				t.Fatalf("missing error in report or stderr: result=%+v, stderr=%q", result, stderr.String())
			}
			if result.Report != nil && !reflect.DeepEqual(*result.Report, tt.returnReport) {
				t.Fatalf("report lost timings: %+v", result.Report)
			}
		})
	}
}

func TestProfileSelectsInstalledShells(t *testing.T) {
	failure := errors.New("startup failed")
	for _, tt := range []struct {
		name         string
		args         []string
		missing      string
		failing      string
		wantShells   []string
		wantStatuses []string
		wantMode     string
		wantReads    int
	}{
		{
			name:    "default omits unavailable shells and continues after failure",
			missing: "zsh", failing: "bash",
			wantShells: []string{"bash", "fish", "powershell", "posix"}, wantStatuses: []string{"failed", "ok", "ok", "ok"},
			wantMode: profileModeManaged, wantReads: 1,
		},
		{
			name:       "full runs all installed shells",
			args:       []string{"--full"},
			wantShells: []string{"bash", "zsh", "fish", "powershell", "posix"}, wantStatuses: []string{"ok", "ok", "ok", "ok", "ok"},
			wantMode: profileModeFull,
		},
		{
			name:       "explicit selection accepts a case-insensitive sh alias",
			args:       []string{"--shell", "SH"},
			wantShells: []string{"posix"}, wantStatuses: []string{"ok"},
			wantMode: profileModeManaged, wantReads: 1,
		},
		{
			name: "explicit unavailable selection uses the canonical name",
			args: []string{"--shell", "FISH"}, missing: "fish",
			wantShells: []string{"fish"}, wantStatuses: []string{"unavailable"},
			wantMode: profileModeManaged,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var ran []string
			var repoReads int
			run := func(name, mode string) (shell.ShellProfileReport, error) {
				if mode != tt.wantMode {
					t.Fatalf("profiler ran in %s mode, want %s", mode, tt.wantMode)
				}
				ran = append(ran, name)
				if name == tt.failing {
					return shell.ShellProfileReport{}, failure
				}
				return shell.ShellProfileReport{Shell: name, TotalMS: 23.5}, nil
			}
			command := newProfileCommand(func(name string) bool {
				return name != tt.missing
			}, func() string {
				repoReads++
				return "repo"
			}, func(_ context.Context, name, path string) (shell.ShellProfileReport, error) {
				if path != "repo" {
					t.Fatalf("managed profiler used path %q", path)
				}
				return run(name, profileModeManaged)
			}, func(_ context.Context, name string) (shell.ShellProfileReport, error) {
				return run(name, profileModeFull)
			})
			command.SetOut(&stdout)
			command.SetErr(io.Discard)
			command.SetArgs(append([]string{"--json"}, tt.args...))
			err := command.Execute()
			unavailable := tt.wantStatuses[0] == "unavailable"
			if (err != nil) != (tt.failing != "" || unavailable) || (tt.failing != "" && !errors.Is(err, failure)) {
				t.Fatalf("unexpected command error: %v", err)
			}
			wantRuns := tt.wantShells
			if unavailable {
				wantRuns = nil
			}
			if !slices.Equal(ran, wantRuns) || repoReads != tt.wantReads {
				t.Fatalf("ran %v and read repo path %d times, want %v and %d reads", ran, repoReads, wantRuns, tt.wantReads)
			}
			var document profileDocument
			if err := json.Unmarshal(stdout.Bytes(), &document); err != nil || len(document.Results) != len(tt.wantShells) {
				t.Fatalf("unexpected JSON document: %s (%v)", stdout.String(), err)
			}
			for i, result := range document.Results {
				if result.Shell != tt.wantShells[i] || result.Mode != tt.wantMode || result.Status != tt.wantStatuses[i] ||
					(result.Error != "") != (result.Status != "ok") {
					t.Fatalf("unexpected result: %+v", result)
				}
			}
		})
	}
}

func TestProfileStopsOnCommandCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var out bytes.Buffer
	command := newProfileCommand(func(string) bool { return true }, func() string { return "repo" },
		func(got context.Context, name, _ string) (shell.ShellProfileReport, error) {
			if name != "bash" {
				t.Fatalf("started %s after cancellation", name)
			}
			cancel()
			if !errors.Is(got.Err(), context.Canceled) {
				t.Fatal("profiler context did not inherit command cancellation")
			}
			return shell.ShellProfileReport{TotalMS: 2}, got.Err()
		}, nil)
	command.SetOut(&out)
	command.SetErr(io.Discard)
	command.SetArgs([]string{"--json"})
	if err := command.ExecuteContext(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	var document profileDocument
	if err := json.Unmarshal(out.Bytes(), &document); err != nil || len(document.Results) != 1 ||
		document.Results[0].Status != "partial" || document.Results[0].Report == nil || document.Results[0].Report.TotalMS != 2 {
		t.Fatalf("expected collected timing, got %s (%v)", out.String(), err)
	}
}

func TestProfileTextDisplaysTimingsAndMeasuredSuggestions(t *testing.T) {
	report := shell.ShellProfileReport{
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
	if err := writeProfileText(&out, profileResult{Shell: "powershell", Mode: profileModeFull, Status: "partial", Report: &report}); err != nil {
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
	if err := writeProfileText(&out, profileResult{Shell: "powershell", Mode: profileModeFull, Status: "ok", Report: &report}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Managed feature timings are unavailable for this run.") ||
		strings.Contains(out.String(), "Oh My Posh") || strings.Contains(out.String(), "apply") {
		t.Fatalf("unexpected output for unavailable attribution: %s", out.String())
	}
}

func TestManagedProfileText(t *testing.T) {
	report := shell.ShellProfileReport{
		Version: "7.5.0", Edition: "Core", Host: "ConsoleHost", Executable: "pwsh",
		TotalMS: 120.25,
		ManagedFeatures: []shell.ManagedFeatureTiming{
			{FeatureTiming: shell.FeatureTiming{Name: "slow", Path: "slow.ps1", Strategy: "eager", DurationMS: 120}},
			{
				FeatureTiming: shell.FeatureTiming{Name: "broken", Path: "broken.ps1", Strategy: "on-command", DurationMS: 0.25},
				Error:         "feature failed to load",
			},
		},
	}
	var out bytes.Buffer
	if err := writeProfileText(&out, profileResult{Shell: "powershell", Mode: profileModeManaged, Status: "partial", Report: &report}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Partial result",
		"Total managed feature time: 120.25 ms",
		"  120.00 ms  slow (eager)  slow.ps1",
		"    Inspect imports and external commands in this feature.",
		"  0.25 ms  broken (on-command)  broken.ps1\n    Error: feature failed to load",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output missing %q: %s", want, out.String())
		}
	}
	for _, unwanted := range []string{"Consider defer", "profile file timings", "Managed feature timings are unavailable"} {
		if strings.Contains(out.String(), unwanted) {
			t.Errorf("output contains %q: %s", unwanted, out.String())
		}
	}

	out.Reset()
	emptyReport := shell.ShellProfileReport{}
	if err := writeProfileText(&out, profileResult{Shell: "fish", Mode: profileModeManaged, Status: "ok", Report: &emptyReport}); err != nil {
		t.Fatal(err)
	}
	want := "fish managed feature profiling\nTotal managed feature time: 0.00 ms\nNo enabled managed fish features to profile.\n"
	if out.String() != want {
		t.Fatalf("unexpected empty report: %s", out.String())
	}
}

func TestUnixFullProfileText(t *testing.T) {
	report := shell.ShellProfileReport{Executable: "/bin/bash", TotalMS: 34.25}
	var out bytes.Buffer
	if err := writeProfileText(&out, profileResult{Shell: "bash", Mode: profileModeFull, Status: "ok", Report: &report}); err != nil {
		t.Fatal(err)
	}
	want := "bash startup profiling\nExecutable: /bin/bash\nTotal startup time (including process launch): 34.25 ms\n"
	if out.String() != want {
		t.Fatalf("unexpected startup report: %s", out.String())
	}
}

func TestProfileRootPreRunAllowsUninitializedRepository(t *testing.T) {
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
