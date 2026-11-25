package shell

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

func generatePowerShellInit(features FeaturesByStrategy) string {
	var sb strings.Builder
	sb.WriteString(`# oh-my-dot shell framework - PowerShell init script
# Auto-generated - do not edit manually

$OMD_SHELL_ROOT = $PSScriptRoot

`)
	hasLazyFeatures := len(features.Defer) > 0 || len(features.OnCommand) > 0
	if len(features.Eager) > 0 || !hasLazyFeatures {
		writePowerShellProfilerHandshake(&sb, "")
	}

	// Dot-source eager features in the caller's scope every time.
	for _, feature := range features.Eager {
		fmt.Fprintf(&sb, `$__omdEagerFile = Microsoft.PowerShell.Management\Join-Path $PSScriptRoot %s
if (Microsoft.PowerShell.Management\Test-Path -LiteralPath $__omdEagerFile -PathType Leaf) {
`, powerShellFeaturePath(feature))
		writePowerShellProfiledSource(&sb, "  ", powerShellQuote(feature), "$__omdEagerFile", "'eager'", ". $__omdEagerFile")
		fmt.Fprintf(&sb, `} else {
  Microsoft.PowerShell.Utility\Write-Warning %s
}

`, powerShellQuote("oh-my-dot: feature '"+feature+"' not found"))
	}
	if len(features.Eager) > 0 {
		sb.WriteString("Microsoft.PowerShell.Utility\\Remove-Variable __omdEagerFile\n\n")
	}
	if !hasLazyFeatures {
		return sb.String()
	}

	sb.WriteString(`# Keep registration state alive even if init was sourced in a local scope.
& {
  param($root)
  if (Microsoft.PowerShell.Utility\Get-Variable OMD_POWERSHELL_LOADED -Scope Global -ValueOnly -ErrorAction Ignore) {
    return
  }
  $global:OMD_POWERSHELL_LOADED = $true

`)
	if len(features.Eager) == 0 {
		writePowerShellProfilerHandshake(&sb, "  ")
	}
	sb.WriteString(`  $loadFeature = {
    param($file, $feature, $root, $strategy)
    $output = [System.Collections.Generic.List[object]]::new()
    $module = Microsoft.PowerShell.Core\New-Module -Name "oh-my-dot-$feature" -ArgumentList $file,$output,$root,$feature,$strategy -ErrorAction Stop -ScriptBlock {
      param($__omdFeatureFile, $__omdFeatureOutput, $OMD_SHELL_ROOT, $__omdFeatureName, $__omdFeatureStrategy)
`)
	writePowerShellProfiledSource(&sb, "      ", "$__omdFeatureName", "$__omdFeatureFile", "$__omdFeatureStrategy",
		`. $__omdFeatureFile | Microsoft.PowerShell.Core\ForEach-Object { $__omdFeatureOutput.Add($_) }`)
	sb.WriteString(`      Microsoft.PowerShell.Core\Export-ModuleMember -Function * -Alias *
    }
    Microsoft.PowerShell.Core\Import-Module -ModuleInfo $module -Global -DisableNameChecking -ErrorAction Stop

    # Event and lookup actions have no caller success stream.
    foreach ($value in $output) { Microsoft.PowerShell.Utility\Write-Host -Object $value }
  }

`)
	if len(features.Defer) > 0 {
		writePowerShellIdleAction(&sb, features.Defer)
	}
	if len(features.OnCommand) > 0 {
		writePowerShellLookupAction(&sb, features.OnCommand)
	}
	sb.WriteString("} $PSScriptRoot\n")
	return sb.String()
}

// The return marker excludes callbacks after a profile finishes. Variable
// lookup needs no cmdlets and short-circuits before method calls in restricted
// language modes or reads of undefined variables under strict mode.
const powerShellProfilerActive = `$ExecutionContext.SessionState.LanguageMode -eq 'FullLanguage' -and
    $ExecutionContext.SessionState.PSVariable.GetValue('global:__omdStartupProfiler') -is [hashtable] -and
    $null -ne $global:__omdStartupProfiler['Features'] -and
    $false -eq $ExecutionContext.SessionState.PSVariable.GetValue('global:__omdStartupProfileReturned')`

func writePowerShellProfilerHandshake(sb *strings.Builder, indent string) {
	fmt.Fprintf(sb, "%sif (%s) {\n%s  $global:__omdStartupProfiler.AttributionAvailable = $true\n%s}\n\n",
		indent, strings.ReplaceAll(powerShellProfilerActive, "\n", "\n"+indent), indent, indent)
}

func writePowerShellProfiledSource(sb *strings.Builder, indent, name, path, strategy, source string) {
	// Keep the source statement in its original scope and error handling
	// context. The stopwatch and report entry exist only during profiling.
	block := fmt.Sprintf(`if (%s) {
  $__omdStartupFeature = @{ name = %s; path = %s; strategy = %s }
  $__omdStartupFeatureWatch = [System.Diagnostics.Stopwatch]::StartNew()
  %s
  $__omdStartupFeature.durationMs = $__omdStartupFeatureWatch.Elapsed.TotalMilliseconds
  $global:__omdStartupProfiler.Features.Add($__omdStartupFeature)
  $ExecutionContext.SessionState.PSVariable.Remove('__omdStartupFeature')
  $ExecutionContext.SessionState.PSVariable.Remove('__omdStartupFeatureWatch')
} else {
  %s
}`, powerShellProfilerActive, name, path, strategy, source, source)
	sb.WriteString(indent + strings.ReplaceAll(block, "\n", "\n"+indent) + "\n")
}

func powerShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func powerShellFeaturePath(feature string) string {
	return powerShellQuote(`features\` + feature + ".ps1")
}

func writePowerShellIdleAction(sb *strings.Builder, features []string) {
	sb.WriteString(`  # The console raises OnIdle when it waits for input.
  $global:__omdPowerShellIdle = @{ Root = $root; Load = $loadFeature; Features = @(
`)
	for _, feature := range features {
		fmt.Fprintf(sb, "    @{ Name = %s; Path = %s }\n", powerShellQuote(feature), powerShellFeaturePath(feature))
	}
	sb.WriteString(`  ) }
  $null = Microsoft.PowerShell.Utility\Register-EngineEvent -SourceIdentifier PowerShell.OnIdle -MaxTriggerCount 1 -Action {
    try {
      $idle = $global:__omdPowerShellIdle
      Microsoft.PowerShell.Utility\Remove-Variable __omdPowerShellIdle -Scope Global
      foreach ($feature in $idle.Features) {
        $file = Microsoft.PowerShell.Management\Join-Path $idle.Root $feature.Path
        if (Microsoft.PowerShell.Management\Test-Path -LiteralPath $file -PathType Leaf) {
          try { & $idle.Load $file $feature.Name $idle.Root 'defer' }
          catch { Microsoft.PowerShell.Utility\Write-Warning "oh-my-dot: feature '$($feature.Name)' failed: $_" }
        }
      }
    } finally {
      Microsoft.PowerShell.Core\Remove-Job -Id $EventSubscriber.Action.Id -Force
    }
  }

`)
}

func writePowerShellLookupAction(sb *strings.Builder, features map[string][]string) {
	sb.WriteString("  $pending = @{}\n")
	for _, feature := range slices.Sorted(maps.Keys(features)) {
		var triggers []string
		for _, trigger := range features[feature] {
			triggers = append(triggers, powerShellQuote(trigger))
		}
		fmt.Fprintf(sb, `  $entry = @{ Name = %s; Path = %s; Triggers = @(%s) }
  foreach ($trigger in $entry.Triggers) { $pending[$trigger] = $entry }
`, powerShellQuote(feature), powerShellFeaturePath(feature), strings.Join(triggers, ", "))
	}
	sb.WriteString(`  $invokeCommand = $ExecutionContext.InvokeCommand
  $lookup = @{ Previous = $invokeCommand.PreCommandLookupAction; Handler = $null }
  $lookup.Handler = [System.EventHandler[System.Management.Automation.CommandLookupEventArgs]]{
    param($name, $eventArgs)
    if ($lookup.Previous) { $lookup.Previous.Invoke($name, $eventArgs) }
    if ($eventArgs.StopSearch -or -not $pending.ContainsKey($name)) { return }

    $feature = $pending[$name]
    foreach ($trigger in $feature.Triggers) { $pending.Remove($trigger) }
    # Restore the original hook when there are no triggers left, unless
    # another hook has taken ownership of this property.
    if ($pending.Count -eq 0 -and $invokeCommand.PreCommandLookupAction -eq $lookup.Handler) {
      $invokeCommand.PreCommandLookupAction = $lookup.Previous
    }

    $file = Microsoft.PowerShell.Management\Join-Path $root $feature.Path
    if (-not (Microsoft.PowerShell.Management\Test-Path -LiteralPath $file -PathType Leaf)) {
      $eventArgs.CommandScriptBlock = { $global:LASTEXITCODE = 127 }
      Microsoft.PowerShell.Utility\Write-Warning "oh-my-dot: feature '$($feature.Name)' not found"
      return
    }
    try { & $loadFeature $file $feature.Name $root 'on-command' }
    catch {
      $eventArgs.CommandScriptBlock = { $global:LASTEXITCODE = 1 }
      Microsoft.PowerShell.Utility\Write-Warning "oh-my-dot: feature '$($feature.Name)' failed: $_"
    }
    # Let PowerShell resolve the command after loading. This preserves native
    # argument parsing, pipeline input, aliases, and command-not-found behavior.
  }.GetNewClosure()
  $invokeCommand.PreCommandLookupAction = $lookup.Handler

`)
}
