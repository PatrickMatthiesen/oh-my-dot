package shell

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestPowerShellInitEagerAndDeferredLifecycle(t *testing.T) {
	features := FeaturesByStrategy{Eager: []string{"eager"}, Defer: []string{"deferred"}}
	testPowerShellInit(t, features, map[string]string{
		"eager": `
$eagerRuns++
function prompt { 'eager> ' }
if ($OMD_SHELL_ROOT -ne (Split-Path -Parent $PSScriptRoot)) { throw 'wrong shell root' }
`,
		"deferred": `
$global:OmdDeferredRuns++
$script:omdIdleState = 'deferred'
function Omd-Deferred { $omdIdleState }
Set-Alias Omd-Idle Omd-Deferred
`,
	}, `
$global:OmdDeferredRuns = 0
function Test-LocalSource {
  $eagerRuns = 0
  . $fixtureInit
  if ($eagerRuns -ne 1 -or (prompt) -ne 'eager> ') { throw 'eager did not run in caller scope' }
  function prompt { 'reset> ' }
  . $fixtureInit
  if ($eagerRuns -ne 2 -or (prompt) -ne 'eager> ') { throw 'eager did not run again in caller scope' }
}
Test-LocalSource
Start-Sleep -Milliseconds 800
if ($OmdDeferredRuns -ne 0) { throw 'deferred feature ran without an interactive console' }
if (@(Get-EventSubscriber -SourceIdentifier PowerShell.OnIdle).Count -ne 1) { throw 'idle action registered more than once' }

# Exercise the action and cleanup independently of the host's idle behavior.
$null = New-Event -SourceIdentifier PowerShell.OnIdle
$deadline = (Get-Date).AddSeconds(3)
while (@(Get-Job).Count -and (Get-Date) -lt $deadline) { Start-Sleep -Milliseconds 50 }
if ($OmdDeferredRuns -ne 1 -or (Omd-Deferred) -ne 'deferred' -or (Omd-Idle) -ne 'deferred') {
  throw 'idle action did not export the feature into the current session'
}
if (@(Get-EventSubscriber -SourceIdentifier PowerShell.OnIdle -ErrorAction Ignore).Count -or @(Get-Job).Count) {
  throw 'idle action did not clean up'
}
Test-LocalSource
if (@(Get-EventSubscriber -SourceIdentifier PowerShell.OnIdle -ErrorAction Ignore).Count) {
  throw 're-sourcing registered another idle action'
}
`)
}

func TestPowerShellInitLoadsOnLookup(t *testing.T) {
	nativeCommand, nativeCall := "uname", "uname -s"
	if runtime.GOOS == "windows" {
		nativeCommand, nativeCall = "cmd", "cmd /d /c echo native-fallback"
	}
	features := FeaturesByStrategy{OnCommand: map[string][]string{
		"forward":   {"Omd-Forward", "Omd-Second"},
		"collision": {"Omd-Collision"},
		"native":    {nativeCommand},
		"no-target": {"Omd-Absent", "Omd-StillAbsent"},
	}}
	testPowerShellInit(t, features, map[string]string{
		"forward": `
$script:omdPrefix = 'module'
$global:omdForwardRuns++
if (Get-Command Omd-Second -ErrorAction Ignore) { throw 'sibling lookup found a command while loading' }
if ($OMD_SHELL_ROOT -ne (Split-Path -Parent $PSScriptRoot)) { throw 'wrong shell root' }
function Invoke-OmdForward {
  param([Parameter(ValueFromPipeline)]$InputObject, [string]$Name, [switch]$Flag)
  process { "${omdPrefix}:${InputObject}:${Name}:${Flag}" }
}
function Omd-Second { $omdPrefix }
Set-Alias Omd-Forward Invoke-OmdForward
`,
		"collision": `
$global:omdCollisionRuns++
function Omd-Collision { throw 'the pre-existing alias should retain command precedence' }
`,
		"native":    "$global:omdNativeRuns++\n",
		"no-target": "$global:omdAbsentRuns++\nfunction Omd-Helper { 'helper' }\n",
	}, `
$global:omdForwardRuns = 0
$global:omdCollisionRuns = 0
$global:omdNativeRuns = 0
$global:omdAbsentRuns = 0
$global:omdPreviousRuns = 0
$global:omdBlockForward = $true
$expectedNativeOutput = (`+nativeCall+`)
$previous = [System.EventHandler[System.Management.Automation.CommandLookupEventArgs]]{
  param($name, $eventArgs)
  if ($name -eq 'Omd-Previous') { $eventArgs.CommandScriptBlock = { 'previous' } }
  if ($name -eq 'Omd-Forward') {
    $global:omdPreviousRuns++
    if ($global:omdBlockForward) { $eventArgs.CommandScriptBlock = { 'previous override' } }
  }
}
$ExecutionContext.InvokeCommand.PreCommandLookupAction = $previous
Set-Alias Omd-Collision Write-Output
& { . $fixtureInit }
$installed = $ExecutionContext.InvokeCommand.PreCommandLookupAction
. $fixtureInit
if ($ExecutionContext.InvokeCommand.PreCommandLookupAction -ne $installed) { throw 'lookup action registered more than once' }
if ($omdForwardRuns -ne 0) { throw 'on-command feature ran during startup' }
if ((Omd-Previous) -ne 'previous') { throw 'previous lookup action was bypassed' }
if ((Omd-Forward) -ne 'previous override' -or $omdForwardRuns -ne 0) { throw 'previous lookup action lost precedence' }
$global:omdBlockForward = $false
if (('x' | omd-forward -Name 'two words' -Flag) -ne 'module:x:two words:True') { throw 'alias lost pipeline input or parameters' }
if (('y' | Omd-Forward -Name next) -ne 'module:y:next:False') { throw 'alias did not persist' }
if ((Omd-Second) -ne 'module' -or $omdForwardRuns -ne 1) { throw 'feature reloaded or lost its module state' }
if ($omdPreviousRuns -lt 1) { throw 'trigger lookup did not chain to previous action' }
if ((Omd-Collision incumbent) -ne 'incumbent' -or $omdCollisionRuns -ne 1) { throw 'existing alias lost command precedence' }
if ((`+nativeCall+`) -ne $expectedNativeOutput -or $omdNativeRuns -ne 1) { throw 'native fallback failed' }
try { Omd-Absent; throw 'missing target resolved' }
catch [System.Management.Automation.CommandNotFoundException] {}
try { Omd-StillAbsent; throw 'missing sibling resolved' }
catch [System.Management.Automation.CommandNotFoundException] {}
if ((Omd-Helper) -ne 'helper' -or $omdAbsentRuns -ne 1) { throw 'missing target did not leave the feature loaded once' }
if ($ExecutionContext.InvokeCommand.PreCommandLookupAction -ne $previous) { throw 'previous lookup action was not restored' }
`)
}

func testPowerShellInit(t *testing.T, features FeaturesByStrategy, scripts map[string]string, checks string) {
	t.Helper()
	for _, executable := range []string{"pwsh", "powershell"} {
		t.Run(executable, func(t *testing.T) {
			binary, err := exec.LookPath(executable)
			if err != nil {
				t.Skipf("%s unavailable: %v", executable, err)
			}
			dir := filepath.Join(t.TempDir(), "repo [with spaces]")
			featureDir := filepath.Join(dir, "features")
			if err := os.MkdirAll(featureDir, 0o700); err != nil {
				t.Fatal(err)
			}
			for feature, script := range scripts {
				if err := os.WriteFile(filepath.Join(featureDir, feature+".ps1"), []byte(script), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(dir, "init.ps1"), []byte(generatePowerShellInit(features)), 0o600); err != nil {
				t.Fatal(err)
			}
			runner := filepath.Join(dir, "checks.ps1")
			preamble := "$ErrorActionPreference = 'Stop'\n$fixtureInit = Join-Path $PSScriptRoot 'init.ps1'\n"
			if err := os.WriteFile(runner, []byte(preamble+checks), 0o600); err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, binary, "-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", runner)
			cmd.Dir = dir
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%s failed: %v\n%s", executable, err, output)
			}
		})
	}
}
