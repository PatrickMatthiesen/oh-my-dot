# WinGet CommandNotFound - Suggest WinGet packages for missing commands

$wingetCnfModule = Get-Module -ListAvailable -Name Microsoft.WinGet.CommandNotFound

if (-not $wingetCnfModule) {
    Write-Verbose "Microsoft.WinGet.CommandNotFound is not available; skipping winget-command-not-found feature"
    return
}

try {
    Import-Module Microsoft.WinGet.CommandNotFound -ErrorAction Stop
    Write-Verbose "Microsoft.WinGet.CommandNotFound loaded successfully"
}
catch {
    Write-Warning "Failed to import Microsoft.WinGet.CommandNotFound: $_"
}