# Posh-Git - Git integration for PowerShell
# Provides git tab completion and enhanced prompt for PowerShell
# This is a wrapper that installs/imports the posh-git module

# Check if posh-git module is installed
$poshGitInstalled = Get-Module -ListAvailable -Name posh-git

if (-not $poshGitInstalled) {
    Write-Host "posh-git module not found. Installing..." -ForegroundColor Yellow
    
    try {
        # Install posh-git from PowerShell Gallery
        # -Scope CurrentUser doesn't require admin privileges
        Install-Module posh-git -Scope CurrentUser -AllowClobber -ErrorAction Stop
        Write-Host "posh-git installed successfully" -ForegroundColor Green
    }
    catch {
        Write-Warning "Failed to install posh-git: $_"
        Write-Host "To install manually, run: Install-Module posh-git -Scope CurrentUser" -ForegroundColor Yellow
        return
    }
}

# Import posh-git module
try {
    Import-Module posh-git -ErrorAction Stop
    
    # Optional: Configure posh-git settings
    # Customize these settings as needed
    
    # Show status for the current repository
    $GitPromptSettings.DefaultPromptAbbreviateHomeDirectory = $true
    
    # Customize branch color (optional)
    # $GitPromptSettings.BranchColor = [ConsoleColor]::Cyan
    
    # Show stash count (optional)
    # $GitPromptSettings.EnableStashStatus = $true
    
    Write-Verbose "posh-git loaded successfully"
}
catch {
    Write-Warning "Failed to import posh-git: $_"
    Write-Host "posh-git may not be installed correctly. Try running: Install-Module posh-git -Scope CurrentUser -Force" -ForegroundColor Yellow
}

# Note: posh-git automatically integrates with your prompt
# If you have a custom prompt function, posh-git will enhance it
# For full customization, see: https://github.com/dahlbyk/posh-git
