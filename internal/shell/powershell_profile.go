package shell

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
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
// After a failure, the report retains completed profiles and completed features
// from a profile whose failure the child was able to report.
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
	resultPath := filepath.Join(dir, "result.txt")
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

	cmd := exec.Command(executable, "-NoLogo", "-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(encoded))
	// The nil streams use the null device directly. A child started by a
	// profile can inherit those handles without leaving a Go pipe open.
	runErr := runPowerShellProfileCommand(ctx, cmd)
	if errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded) {
		runErr = fmt.Errorf("PowerShell startup profiling interrupted: %w", runErr)
	} else if runErr != nil {
		runErr = fmt.Errorf("run PowerShell startup profiler with %s: %w", executable, runErr)
	}

	data, err := os.ReadFile(resultPath)
	if err != nil {
		return PowerShellProfileReport{}, errors.Join(runErr, fmt.Errorf("read PowerShell startup report: %w", err))
	}
	report, err := decodePowerShellProfileReport(string(data), executable)
	if runErr != nil && err == errIncompletePowerShellReport {
		err = runErr
	} else {
		err = errors.Join(runErr, err)
	}
	if err != nil && len(report.Profiles) == 0 {
		return PowerShellProfileReport{}, err
	}
	if !report.AttributionAvailable {
		report.Warnings = append(report.Warnings, "No instrumented oh-my-dot loader ran. Only profile file timings are available.")
	}
	sort.SliceStable(report.Profiles, func(i, j int) bool {
		return report.Profiles[i].DurationMS > report.Profiles[j].DurationMS
	})
	return report, err
}

var errIncompletePowerShellReport = errors.New("PowerShell did not write a startup report to completion (check execution policy and whether a profile exits PowerShell)")

func decodePowerShellProfileReport(data, executable string) (PowerShellProfileReport, error) {
	report := PowerShellProfileReport{
		Shell: "powershell", Executable: executable, Profiles: []ProfileTiming{}, Warnings: []string{},
	}
	header, data, complete := strings.Cut(data, "\n")
	if !complete {
		return report, errIncompletePowerShellReport
	}
	fields := strings.Split(header, "\t")
	if len(fields) != 4 || fields[0] != "H" {
		return report, fmt.Errorf("invalid PowerShell startup report header")
	}
	var err error
	report.Edition, err = decodePowerShellProfileField(fields[1])
	if err != nil || report.Edition == "" {
		return report, fmt.Errorf("invalid PowerShell edition in startup report")
	}
	report.Version, err = decodePowerShellProfileField(fields[2])
	if err != nil || report.Version == "" {
		return report, fmt.Errorf("invalid PowerShell version in startup report")
	}
	report.Host, err = decodePowerShellProfileField(fields[3])
	if err != nil || report.Host == "" {
		return report, fmt.Errorf("invalid PowerShell host in startup report")
	}
	// Each newline ends a checkpoint. An interrupted write can leave an
	// unfinished record after the last usable sample.
	var failures []error
	for records := 0; ; records++ {
		var line string
		line, data, complete = strings.Cut(data, "\n")
		if !complete {
			if len(failures) == 0 {
				return report, errIncompletePowerShellReport
			}
			return report, errors.Join(append(failures, errIncompletePowerShellReport)...)
		}
		if line == "D" && data == "" {
			return report, errors.Join(failures...)
		}
		fields := strings.Split(line, "\t")
		// A profile record has six fields followed by four fields per feature.
		if len(fields) < 6 || (len(fields)-6)%4 != 0 || (fields[0] != "P" && fields[0] != "F") || records == 4 {
			return report, errors.Join(append(failures, fmt.Errorf("invalid PowerShell startup profile record"))...)
		}
		profile, err := decodePowerShellProfileTiming(fields)
		if err != nil {
			return report, errors.Join(append(failures, err)...)
		}
		errorCount, err := strconv.ParseUint(fields[4], 10, 32)
		if err != nil || (fields[3] != "0" && fields[3] != "1") {
			return report, errors.Join(append(failures, fmt.Errorf("invalid PowerShell startup profile record for %s", profile.Path))...)
		}
		failure, err := decodePowerShellProfileField(fields[5])
		failed := fields[0] == "F"
		if err != nil || failed != (failure != "") {
			return report, errors.Join(append(failures, fmt.Errorf("invalid PowerShell startup profile completion for %s", profile.Path))...)
		}
		report.AttributionAvailable = report.AttributionAvailable || fields[3] == "1"
		if errorCount != 0 {
			report.Warnings = append(report.Warnings, fmt.Sprintf("PowerShell recorded %d error(s) while running %s.", errorCount, profile.Path))
		}
		if !failed || len(profile.Features) != 0 {
			report.Profiles = append(report.Profiles, profile)
			report.TotalMS += profile.DurationMS
		}
		if failed {
			failures = append(failures, fmt.Errorf("PowerShell startup profiling failed: %s: %s", profile.Path, failure))
		}
	}
}

func decodePowerShellProfileField(field string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(field)
	return string(decoded), err
}

func decodePowerShellProfileTiming(fields []string) (ProfileTiming, error) {
	path, pathErr := decodePowerShellProfileField(fields[1])
	ticks, durationErr := strconv.ParseUint(fields[2], 10, 64)
	if pathErr != nil || durationErr != nil || !filepath.IsAbs(path) {
		return ProfileTiming{}, fmt.Errorf("invalid PowerShell startup profile timing")
	}
	profile := ProfileTiming{Path: path, DurationMS: float64(ticks) / 10000, Features: []FeatureTiming{}}
	for i := 6; i < len(fields); i += 4 {
		name, nameErr := decodePowerShellProfileField(fields[i])
		path, pathErr := decodePowerShellProfileField(fields[i+1])
		strategy, strategyErr := decodePowerShellProfileField(fields[i+2])
		ticks, durationErr := strconv.ParseUint(fields[i+3], 10, 64)
		if nameErr != nil || pathErr != nil || strategyErr != nil || durationErr != nil || name == "" || path == "" || strategy == "" {
			return ProfileTiming{}, fmt.Errorf("invalid PowerShell feature timing for %s", profile.Path)
		}
		profile.Features = append(profile.Features, FeatureTiming{
			Name: name, Path: path, Strategy: strategy, DurationMS: float64(ticks) / 10000,
		})
	}
	sort.SliceStable(profile.Features, func(i, j int) bool {
		return profile.Features[i].DurationMS > profile.Features[j].DurationMS
	})
	return profile, nil
}

func powerShellLiteral(value string) string {
	quotes := strings.NewReplacer("'", "''", "‘", "‘‘", "’", "’’", "‚", "‚‚", "‛", "‛‛")
	return "'" + quotes.Replace(value) + "'"
}

// The child writes tab-separated records with base64 text fields and elapsed
// TimeSpan ticks. .NET I/O and encoding avoid importing modules or changing
// preferences before the measured profiles run.
const powerShellProfileScript = `
$__omdStartupEncode = {
  param([string]$Text)
  [Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes($Text))
}
$__omdStartupHeader = @(
  'H'
  & $__omdStartupEncode $PSVersionTable.PSEdition
  & $__omdStartupEncode $PSVersionTable.PSVersion.ToString()
  & $__omdStartupEncode $Host.Name
)
[System.IO.File]::AppendAllText($__omdStartupResultPath, ($__omdStartupHeader -join [char]9) + [char]10)
$global:__omdStartupProfiler = @{
  AttributionAvailable = $false
  Features = $null
}
$__omdStartupInvocation = '. $global:__omdStartupProfilePath; $global:__omdStartupProfileReturned = $true'
foreach ($__omdStartupPath in $__omdStartupProfilePaths) {
  if (-not [System.IO.File]::Exists($__omdStartupPath)) {
    continue
  }
  $global:__omdStartupProfiler.Features = [System.Collections.Generic.List[object]]::new()
  $global:__omdStartupProfilePath = $__omdStartupPath
  $global:__omdStartupProfileReturned = $false
  $__omdStartupFailure = ''
  # The nested pipeline shares session state. Its script has no try/catch
  # around the dot-source, so statement-terminating errors can continue.
  $__omdStartupPipeline = [PowerShell]::Create([System.Management.Automation.RunspaceMode]::CurrentRunspace)
  $null = $__omdStartupPipeline.AddScript($__omdStartupInvocation, $false).AddCommand('Microsoft.PowerShell.Core\Out-Default')
  $__omdStartupWatch = [System.Diagnostics.Stopwatch]::StartNew()
  try {
    $null = $__omdStartupPipeline.Invoke()
    $__omdStartupWatch.Stop()
  } catch {
    $__omdStartupWatch.Stop()
    $__omdStartupFailure = $_.Exception.GetBaseException().Message
  }
  if (-not $global:__omdStartupProfileReturned -and -not $__omdStartupFailure) {
    $__omdStartupFailure = 'profile stopped before it finished running'
  }
  $global:__omdStartupProfileReturned = $true
  # The return marker can run even when PowerShell could not load the file.
  # Check where the error originated so errors in scripts the profile calls
  # remain ordinary profile warnings.
  if (-not $__omdStartupFailure) {
    foreach ($__omdStartupError in $__omdStartupPipeline.Streams.Error) {
      $__omdStartupException = $__omdStartupError.Exception
      $__omdStartupSource = $__omdStartupError.InvocationInfo
      if ($null -eq $__omdStartupSource) {
        continue
      }
      $__omdStartupFromRunner = -not $__omdStartupSource.ScriptName -and $__omdStartupSource.Line -ceq $__omdStartupInvocation
      if ($__omdStartupException -is [System.Management.Automation.ScriptRequiresException] -and $__omdStartupFromRunner) {
        $__omdStartupFailure = 'profile requirements were not met: ' + $__omdStartupException.Message
        break
      }
      if ($__omdStartupException -is [System.Management.Automation.ParseException] -and
          ($__omdStartupFromRunner -or $__omdStartupSource.ScriptName -ceq $__omdStartupPath) -and
          $null -ne $__omdStartupException.Errors -and
          $__omdStartupException.Errors.Count -ne 0 -and
          $__omdStartupException.Errors[0].Extent.File -ceq $__omdStartupPath) {
        $__omdStartupFailure = 'could not parse profile: ' + $__omdStartupException.Message
        break
      }
      # Write-Error can report its caller's invocation from inside a profile.
      if ($__omdStartupException -is [System.Management.Automation.PSSecurityException] -and
          $__omdStartupFromRunner -and $__omdStartupError.FullyQualifiedErrorId -ceq 'UnauthorizedAccess') {
        $__omdStartupFailure = 'authorization blocked the profile: ' + $__omdStartupException.Message
        break
      }
    }
  }
  if ($__omdStartupFailure.Length -gt 512) {
    $__omdStartupFailure = $__omdStartupFailure.Substring(0, 512)
  }
  $__omdStartupFields = @(
    if ($__omdStartupFailure) { 'F' } else { 'P' }
    & $__omdStartupEncode $__omdStartupPath
    $__omdStartupWatch.Elapsed.Ticks
    [int]$global:__omdStartupProfiler.AttributionAvailable
    $__omdStartupPipeline.Streams.Error.Count
    & $__omdStartupEncode $__omdStartupFailure
    foreach ($__omdStartupFeature in $global:__omdStartupProfiler.Features) {
      & $__omdStartupEncode $__omdStartupFeature.name
      & $__omdStartupEncode $__omdStartupFeature.path
      & $__omdStartupEncode $__omdStartupFeature.strategy
      [long]($__omdStartupFeature.durationMs * [TimeSpan]::TicksPerMillisecond)
    }
  )
  $__omdStartupPipeline.Dispose()
  [System.IO.File]::AppendAllText($__omdStartupResultPath, ($__omdStartupFields -join [char]9) + [char]10)
}
[System.IO.File]::AppendAllText($__omdStartupResultPath, 'D' + [char]10)
`
