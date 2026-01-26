# PowerShell Aliases - Common PowerShell aliases and shortcuts
# Cross-platform aliases adapted for PowerShell

# Navigation shortcuts
function .. { Set-Location .. }
function ... { Set-Location ..\.. }
function .... { Set-Location ..\..\.. }

# Directory listing aliases (PowerShell-native equivalents)
# PowerShell's Get-ChildItem (ls/dir) already has color by default
Set-Alias -Name ll -Value Get-ChildItem -Option AllScope -Force
function la { Get-ChildItem -Force }  # Show hidden files

# Git shortcuts
Set-Alias -Name g -Value git -Option AllScope -Force
function gs { git status }
function ga { git add @args }
function gc { git commit @args }
function gca { git commit -a @args }
function gcm { git commit -m @args }
function gp { git push @args }
function gpl { git pull @args }
function gd { git diff @args }
function gco { git checkout @args }
function gb { git branch @args }
function gl { git log --oneline --graph --decorate @args }

# Safety aliases
# PowerShell already prompts for confirmation on dangerous operations by default
# when $ConfirmPreference is set appropriately. The following are shortcuts
# that preserve this behavior while providing familiar Unix-style command names.

# Directory operations
function lsd { Get-ChildItem -Directory }

# Show PATH in readable format
function path {
    $env:Path -split [IO.Path]::PathSeparator
}

# Create parent directories as needed (PowerShell's New-Item -Force already does this)
function New-Directory {
    param([string]$Path)
    New-Item -Path $Path -ItemType Directory -Force
}

# Reload PowerShell profile
function reload {
    . $PROFILE
    Write-Host "Profile reloaded" -ForegroundColor Green
}

# Common Unix-like utilities (if you prefer Unix names)
Set-Alias -Name cat -Value Get-Content -Option AllScope -Force
Set-Alias -Name grep -Value Select-String -Option AllScope -Force

# PowerShell-specific shortcuts
function which {
    param([string]$Command)
    Get-Command $Command -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Source
}

# Quick access to edit profile
function Edit-Profile {
    if (Test-Path $PROFILE) {
        & $env:EDITOR $PROFILE
    }
    else {
        # If no editor set, use notepad as fallback
        notepad $PROFILE
    }
}
Set-Alias -Name ep -Value Edit-Profile -Option AllScope -Force

# Useful PowerShell cmdlet shortcuts
Set-Alias -Name exp -Value explorer -Option AllScope -Force
