# Terminal-Icons - File and folder icons for PowerShell

$terminalIconsModule = Get-Module -ListAvailable -Name Terminal-Icons

if (-not $terminalIconsModule) {
    # Setup guidance is shown by feature add/configure, apply and doctor.
    # Opening a shell must not start a package-manager transaction.
    return
}

try {
    $terminalIconsImportMutex = [System.Threading.Mutex]::new($false, "oh-my-dot-terminal-icons-import")
    $terminalIconsLockTaken = $false
    $terminalIconsLockTaken = $terminalIconsImportMutex.WaitOne([TimeSpan]::FromSeconds(10))

    if (-not $terminalIconsLockTaken) {
        Write-Warning "Timed out waiting to import Terminal-Icons"
        return
    }

    Import-Module Terminal-Icons -ErrorAction Stop
    Write-Verbose "Terminal-Icons loaded successfully"
}
catch {
    Write-Warning "Failed to import Terminal-Icons: $_"
}
finally {
    if ($terminalIconsLockTaken) {
        $terminalIconsImportMutex.ReleaseMutex()
    }

    if ($terminalIconsImportMutex) {
        $terminalIconsImportMutex.Dispose()
    }
}
