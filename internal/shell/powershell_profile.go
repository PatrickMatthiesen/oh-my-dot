package shell

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
)

const powerShellProfileTimeout = 30 * time.Second

// PowerShellProfileReport describes one run of the PowerShell console profiles.
type PowerShellProfileReport struct {
	Shell                string          `json:"shell"`                // Shell identifier.
	Executable           string          `json:"executable"`           // Resolved PowerShell executable.
	Edition              string          `json:"edition"`              // PowerShell edition, such as Core or Desktop.
	Version              string          `json:"version"`              // PowerShell version.
	Host                 string          `json:"host"`                 // PowerShell host name.
	TotalMS              float64         `json:"totalMs"`              // Sum of profile file durations in milliseconds.
	Profiles             []ProfileTiming `json:"profiles"`             // Existing console profiles, slowest first.
	AttributionAvailable bool            `json:"attributionAvailable"` // Whether an instrumented loader ran.
	Warnings             []string        `json:"warnings"`             // Notes about the measurement.
}

// ProfileTiming measures one dot-sourced PowerShell profile file.
type ProfileTiming struct {
	Path       string          `json:"path"`       // Absolute profile path.
	DurationMS float64         `json:"durationMs"` // Elapsed milliseconds, including features.
	Features   []FeatureTiming `json:"features"`   // Feature loads attributed to this profile, slowest first.
}

// FeatureTiming measures one dot-sourced oh-my-dot feature.
type FeatureTiming struct {
	Name       string  `json:"name"`       // Feature name from the loader.
	Path       string  `json:"path"`       // Path used by the loader.
	Strategy   string  `json:"strategy"`   // Configured loading strategy.
	DurationMS float64 `json:"durationMs"` // Elapsed milliseconds.
}

// ProfilePowerShell runs the existing console profiles once in a fresh
// PowerShell process. Profile scripts can have their usual side effects.
// If a profile fails, the report includes any completed timings.
func ProfilePowerShell(ctx context.Context) (PowerShellProfileReport, error) {
	_, executable, err := resolvePowerShellExecutable()
	if err != nil {
		return PowerShellProfileReport{}, err
	}
	return profilePowerShell(ctx, executable, nil)
}

// profilePowerShell accepts fixture paths so tests never source machine profiles.
func profilePowerShell(ctx context.Context, executable string, fixturePaths *[4]string) (PowerShellProfileReport, error) {
	ctx, cancel := context.WithTimeout(ctx, powerShellProfileTimeout)
	defer cancel()

	dir, err := os.MkdirTemp("", "omd-powershell-profile-")
	if err != nil {
		return PowerShellProfileReport{}, fmt.Errorf("create temporary PowerShell report directory: %w", err)
	}
	defer os.RemoveAll(dir)
	resultPath := filepath.Join(dir, "result.json")
	if err := manifest.CreatePrivateFile(resultPath, nil); err != nil {
		return PowerShellProfileReport{}, fmt.Errorf("create temporary PowerShell report: %w", err)
	}

	profilePaths := "@($PROFILE.AllUsersAllHosts, $PROFILE.AllUsersCurrentHost, $PROFILE.CurrentUserAllHosts, $PROFILE.CurrentUserCurrentHost)"
	if fixturePaths != nil {
		var literals []string
		for _, path := range fixturePaths {
			literals = append(literals, powerShellLiteral(path))
		}
		profilePaths = "@(" + strings.Join(literals, ", ") + ")"
	}
	script := "$__omdStartupResultPath = " + powerShellLiteral(resultPath) + "\n" +
		"$__omdStartupProfilePaths = " + profilePaths + "\n" + powerShellProfileScript
	// Both PowerShell editions expect UTF-16LE for -EncodedCommand.
	var encoded []byte
	for _, unit := range utf16.Encode([]rune(script)) {
		encoded = binary.LittleEndian.AppendUint16(encoded, unit)
	}

	cmd := exec.CommandContext(ctx, executable, "-NoLogo", "-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(encoded))
	// The nil streams use the null device directly. A child started by a
	// profile can inherit those handles without leaving a Go pipe open.
	runErr := cmd.Run()
	if err := ctx.Err(); err != nil {
		return PowerShellProfileReport{}, fmt.Errorf("PowerShell startup profiling interrupted: %w", err)
	}
	if runErr != nil {
		return PowerShellProfileReport{}, fmt.Errorf("run PowerShell startup profiler with %s (check execution policy and whether a profile exits PowerShell): %w", executable, runErr)
	}

	data, err := os.ReadFile(resultPath)
	if err != nil {
		return PowerShellProfileReport{}, fmt.Errorf("read PowerShell startup report: %w", err)
	}
	if len(data) == 0 {
		return PowerShellProfileReport{}, fmt.Errorf("PowerShell did not write a startup report (check execution policy and whether a profile exits PowerShell)")
	}
	var result struct {
		Report  *PowerShellProfileReport `json:"report"`
		Failure string                   `json:"failure"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return PowerShellProfileReport{}, fmt.Errorf("decode PowerShell startup report: %w", err)
	}
	report := result.Report
	if report == nil || report.Edition == "" || report.Version == "" || report.Host == "" || report.TotalMS < 0 || report.Profiles == nil || len(report.Profiles) > 4 {
		return PowerShellProfileReport{}, fmt.Errorf("PowerShell returned an incomplete startup report")
	}
	for _, profile := range report.Profiles {
		if profile.Path == "" || profile.DurationMS < 0 || profile.Features == nil {
			return PowerShellProfileReport{}, fmt.Errorf("PowerShell returned an incomplete startup profile timing")
		}
		for _, feature := range profile.Features {
			if feature.Name == "" || feature.Path == "" || feature.Strategy == "" || feature.DurationMS < 0 {
				return PowerShellProfileReport{}, fmt.Errorf("PowerShell returned an incomplete feature timing for %s", profile.Path)
			}
		}
		sort.SliceStable(profile.Features, func(i, j int) bool {
			return profile.Features[i].DurationMS > profile.Features[j].DurationMS
		})
	}
	sort.SliceStable(report.Profiles, func(i, j int) bool {
		return report.Profiles[i].DurationMS > report.Profiles[j].DurationMS
	})
	report.Shell = "powershell"
	report.Executable = executable
	report.Warnings = []string{}
	if !report.AttributionAvailable {
		report.Warnings = append(report.Warnings, "No instrumented oh-my-dot loader ran. Only profile file timings are available.")
	}
	if result.Failure != "" {
		err := fmt.Errorf("PowerShell startup profiling failed: %s", result.Failure)
		if len(report.Profiles) == 0 {
			return PowerShellProfileReport{}, err
		}
		return *report, err
	}
	return *report, nil
}

func powerShellLiteral(value string) string {
	quotes := strings.NewReplacer("'", "''", "‘", "‘‘", "’", "’’", "‚", "‚‚", "‛", "‛‛")
	return "'" + quotes.Replace(value) + "'"
}

const powerShellProfileScript = `
$__omdStartupReport = @{
  edition = $PSVersionTable.PSEdition
  version = $PSVersionTable.PSVersion.ToString()
  host = $Host.Name
  totalMs = 0.0
}
$__omdStartupTimings = [System.Collections.Generic.List[object]]::new()
$__omdStartupFailure = ''
$global:__omdStartupProfiler = @{
  AttributionAvailable = $false
  Features = $null
}
$__omdStartupMeasure = {
  param($__omdStartupPath)
  if (-not (Microsoft.PowerShell.Management\Test-Path -LiteralPath $__omdStartupPath -PathType Leaf)) {
    return
  }
  $global:__omdStartupProfiler.Features = [System.Collections.Generic.List[object]]::new()
  $__omdStartupWatch = [System.Diagnostics.Stopwatch]::StartNew()
  . $__omdStartupPath
  $__omdStartupWatch.Stop()
  $__omdStartupDuration = $__omdStartupWatch.Elapsed.TotalMilliseconds
  $__omdStartupReport.totalMs += $__omdStartupDuration
  $__omdStartupTimings.Add(@{
    path = $__omdStartupPath
    durationMs = $__omdStartupDuration
    features = $global:__omdStartupProfiler.Features.ToArray()
  })
}
try {
  # Dot-source in one scope without a loop that a profile's break could target.
  . $__omdStartupMeasure $__omdStartupProfilePaths[0]
  . $__omdStartupMeasure $__omdStartupProfilePaths[1]
  . $__omdStartupMeasure $__omdStartupProfilePaths[2]
  . $__omdStartupMeasure $__omdStartupProfilePaths[3]
} catch {
  $__omdStartupFailure = "${__omdStartupPath}: $($_.Exception.Message)"
}
$__omdStartupReport.profiles = $__omdStartupTimings.ToArray()
$__omdStartupReport.attributionAvailable = $global:__omdStartupProfiler.AttributionAvailable
$__omdStartupResult = @{ report = $__omdStartupReport; failure = $__omdStartupFailure }
[System.IO.File]::WriteAllText($__omdStartupResultPath, (Microsoft.PowerShell.Utility\ConvertTo-Json -InputObject $__omdStartupResult -Depth 8 -Compress))
`
