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

# Common Unix-like utilities (if you prefer Unix names and signatures)
Set-Alias -Name cat -Value Get-Content -Option AllScope -Force

function head {
    param(
        [Alias('n')]
        [int]$Count = 10,
        [Parameter(ValueFromRemainingArguments = $true)]
        [string[]]$Path
    )

    if (-not $Path) {
        throw 'head requires at least one file path'
    }

    Get-Content -Path $Path -TotalCount $Count
}

function tail {
    param(
        [Alias('n')]
        [int]$Count = 10,
        [Alias('f')]
        [switch]$Follow,
        [Parameter(ValueFromRemainingArguments = $true)]
        [string[]]$Path
    )

    if (-not $Path) {
        throw 'tail requires at least one file path'
    }

    if ($Follow) {
        Get-Content -Path $Path -Tail $Count -Wait
        return
    }

    Get-Content -Path $Path -Tail $Count
}

function less {
    param(
        [Parameter(ValueFromRemainingArguments = $true)]
        [string[]]$Path
    )

    if (-not $Path) {
        throw 'less requires at least one file path'
    }

    Get-Content -Path $Path | Out-Host -Paging
}

function find {
    param(
        [string]$Path = '.',
        [string]$Name,
        [switch]$File,
        [switch]$Directory
    )

    $items = Get-ChildItem -Path $Path -Recurse -Force -ErrorAction SilentlyContinue

    if ($File) {
        $items = $items | Where-Object { -not $_.PSIsContainer }
    }

    if ($Directory) {
        $items = $items | Where-Object { $_.PSIsContainer }
    }

    if ($Name) {
        $items = $items | Where-Object { $_.Name -like $Name }
    }

    $items | Select-Object -ExpandProperty FullName
}

function touch {
    param(
        [Parameter(ValueFromRemainingArguments = $true)]
        [string[]]$Path
    )

    if (-not $Path) {
        throw 'touch requires at least one file path'
    }

    foreach ($item in $Path) {
        if (Test-Path $item) {
            (Get-Item $item).LastWriteTime = Get-Date
            continue
        }

        New-Item -Path $item -ItemType File -Force | Out-Null
    }
}

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
