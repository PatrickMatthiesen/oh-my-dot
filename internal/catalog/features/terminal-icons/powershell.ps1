# Terminal-Icons - File and folder icons for PowerShell

$terminalIconsModule = Get-Module -ListAvailable -Name Terminal-Icons

if (-not $terminalIconsModule) {
    Write-Host "Terminal-Icons module not found. Installing..." -ForegroundColor Yellow

    try {
        Install-Module Terminal-Icons -Scope CurrentUser -AllowClobber -ErrorAction Stop
        Write-Host "Terminal-Icons installed successfully" -ForegroundColor Green
    }
    catch {
        Write-Warning "Failed to install Terminal-Icons: $_"
        Write-Host "To install manually, run: Install-Module Terminal-Icons -Scope CurrentUser" -ForegroundColor Yellow
        return
    }
}

try {
    Import-Module Terminal-Icons -ErrorAction Stop
    Write-Verbose "Terminal-Icons loaded successfully"
}
catch {
    Write-Warning "Failed to import Terminal-Icons: $_"
}