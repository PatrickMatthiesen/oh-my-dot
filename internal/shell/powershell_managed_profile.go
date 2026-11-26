package shell

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/manifest"
)

// ProfilePowerShellFeatures sources each enabled PowerShell feature once in
// its own process. Feature failures are recorded on their individual timings.
func ProfilePowerShellFeatures(ctx context.Context, repoPath string) (PowerShellProfileReport, error) {
	return profilePowerShellFeatures(ctx, repoPath, "")
}

func profilePowerShellFeatures(ctx context.Context, repoPath, executable string) (PowerShellProfileReport, error) {
	ctx, cancel := context.WithTimeout(ctx, shellProfileTimeout)
	defer cancel()

	report := PowerShellProfileReport{Shell: "powershell", Profiles: []ProfileTiming{}, Warnings: []string{}}
	if strings.TrimSpace(repoPath) == "" {
		return report, fmt.Errorf("oh-my-dot repository path is not configured")
	}
	repoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return report, fmt.Errorf("resolve oh-my-dot repository path: %w", err)
	}
	if err := managedPowerShellProfileContextError(ctx); err != nil {
		return report, err
	}
	const shellName = "powershell"
	sharedPath := GetManifestPath(repoPath, shellName)
	base, err := manifest.ParseManifest(sharedPath)
	if errors.Is(err, fs.ErrNotExist) {
		return report, managedPowerShellProfileContextError(ctx)
	}
	if err != nil {
		return report, fmt.Errorf("load shared PowerShell manifest: %w", err)
	}
	local, err := strictLocalManifest(GetLocalManifestPath(repoPath, shellName))
	if err != nil {
		return report, err
	}
	shellRoot := GetShellDirectory(repoPath, shellName)
	for _, feature := range manifest.MergeManifests(base, local).Features {
		if ctx.Err() != nil {
			break
		}
		if feature.Disabled {
			continue
		}
		if executable == "" {
			_, executable, err = resolvePowerShellExecutable()
			if err != nil {
				return report, err
			}
		}
		report.Executable = executable
		timing := ManagedFeatureTiming{FeatureTiming: FeatureTiming{
			Name: feature.Name, Strategy: feature.Strategy,
		}}
		if timing.Strategy == "" {
			timing.Strategy = "eager"
		}
		timing.Path, err = effectiveFeatureFilePath(repoPath, shellName, base, feature)
		if err == nil {
			var metadata PowerShellProfileReport
			metadata, err = measurePowerShellFeature(ctx, executable, shellRoot, &timing)
			if report.Version == "" {
				report.Edition, report.Version, report.Host = metadata.Edition, metadata.Version, metadata.Host
			}
		}
		if err != nil {
			timing.Error = err.Error()
		}
		report.ManagedFeatures = append(report.ManagedFeatures, timing)
		report.TotalMS += timing.DurationMS
	}
	sort.SliceStable(report.ManagedFeatures, func(i, j int) bool {
		return report.ManagedFeatures[i].DurationMS > report.ManagedFeatures[j].DurationMS
	})
	return report, managedPowerShellProfileContextError(ctx)
}

func managedPowerShellProfileContextError(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("PowerShell feature profiling interrupted: %w", err)
	}
	return nil
}

func measurePowerShellFeature(ctx context.Context, executable, shellRoot string, timing *ManagedFeatureTiming) (PowerShellProfileReport, error) {
	script := powerShellProfileHeaderScript +
		"$OMD_SHELL_ROOT = " + powerShellLiteral(shellRoot) + "\n" +
		"$global:__omdManagedPath = " + powerShellLiteral(timing.Path) + "\n" +
		powerShellManagedProfileScript
	data, runErr, err := runPowerShellProfileScript(ctx, executable, script, managedShellProfileEnv())
	if err != nil {
		return PowerShellProfileReport{}, err
	}
	report, records, err := decodePowerShellProfileHeader(string(data), executable)
	if err != nil {
		if runErr != nil && err == errIncompletePowerShellReport {
			return PowerShellProfileReport{}, runErr
		}
		return PowerShellProfileReport{}, errors.Join(runErr, err)
	}
	line, remaining, complete := strings.Cut(records, "\n")
	if !complete {
		if runErr != nil {
			return report, runErr
		}
		return report, fmt.Errorf("PowerShell did not write a feature timing to completion")
	}
	fields := strings.Split(line, "\t")
	if len(fields) != 3 || fields[0] != "M" || remaining != "" {
		return report, errors.Join(runErr, fmt.Errorf("invalid PowerShell managed feature record"))
	}
	ticks, durationErr := strconv.ParseUint(fields[1], 10, 64)
	failure, failureErr := decodePowerShellProfileField(fields[2])
	if durationErr != nil || failureErr != nil {
		return report, errors.Join(runErr, fmt.Errorf("invalid PowerShell managed feature timing"))
	}
	timing.DurationMS = float64(ticks) / 10000
	if failure != "" {
		return report, errors.Join(runErr, errors.New(failure))
	}
	return report, runErr
}

const powerShellManagedProfileScript = `
$global:__omdManagedWatch = [System.Diagnostics.Stopwatch]::new()
$__omdManagedFailure = ''
if (-not [System.IO.File]::Exists($global:__omdManagedPath)) {
  $__omdManagedFailure = 'Feature script was not found or is not a file.'
} else {
  # The nested pipeline collects errors without changing PowerShell's normal
  # error handling inside the feature. Start timing after pipeline setup.
  $global:__omdManagedReturned = $false
  $__omdManagedPipeline = [PowerShell]::Create([System.Management.Automation.RunspaceMode]::CurrentRunspace)
  $null = $__omdManagedPipeline.AddScript('
    $global:__omdManagedWatch.Start()
    . $global:__omdManagedPath
    $global:__omdManagedSucceeded = $?
    $global:__omdManagedWatch.Stop()
    $global:__omdManagedReturned = $true
  ', $false)
  try {
    $null = $__omdManagedPipeline.Invoke()
  } catch {
    $global:__omdManagedWatch.Stop()
    $__omdManagedFailure = $_.Exception.GetBaseException().Message
  }
  $global:__omdManagedWatch.Stop()
  if (-not $__omdManagedFailure) {
    if ($__omdManagedPipeline.Streams.Error.Count -gt 0) {
      $__omdManagedFailure = $__omdManagedPipeline.Streams.Error[0].ToString()
    } elseif (-not $global:__omdManagedReturned) {
      $__omdManagedFailure = 'Feature script stopped before the measurement finished.'
    } elseif (-not $global:__omdManagedSucceeded) {
      $__omdManagedFailure = "Feature script failed (last exit code: $LASTEXITCODE)."
    }
  }
  $__omdManagedPipeline.Dispose()
}
if ($__omdManagedFailure.Length -gt 512) {
  $__omdManagedFailure = $__omdManagedFailure.Substring(0, 512)
}
$__omdManagedFields = @(
  'M'
  $global:__omdManagedWatch.Elapsed.Ticks
  & $__omdStartupEncode $__omdManagedFailure
)
[System.IO.File]::AppendAllText($__omdStartupResultPath, ($__omdManagedFields -join [char]9) + [char]10)
`
