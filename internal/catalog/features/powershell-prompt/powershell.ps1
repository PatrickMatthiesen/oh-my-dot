# PowerShell Prompt - Custom prompt with git status
# Displays current directory, git branch, and repository status in your PowerShell prompt

# Cache whether the current session is elevated (recomputed only once)
$Script:IsAdministrator = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

# Parse git branch name
function Get-GitBranch {
    try {
        $branch = git branch --show-current 2>$null
        if ($branch) {
            return $branch
        }
    }
    catch {
        return $null
    }
    return $null
}

# Check if repository has uncommitted changes
function Get-GitDirty {
    try {
        $status = git status --porcelain 2>$null
        if ($status) {
            return $true
        }
    }
    catch {
        return $false
    }
    return $false
}

# Custom prompt function
function prompt {
    # Get current location
    $location = Get-Location
    
    # Use ~ for home directory
    $homePattern = [regex]::Escape($HOME)
    $displayPath = $location.Path -replace "^$homePattern", "~"
    
    # Build prompt parts
    $promptParts = @()
    
    # Add user@hostname (optional, comment out if you prefer minimal)
    # $promptParts += "$env:USERNAME@$env:COMPUTERNAME"
    
    # Add current path in cyan
    $promptParts += "`e[36m$displayPath`e[0m"
    
    # Add git info if in a git repository
    $branch = Get-GitBranch
    if ($branch) {
        $isDirty = Get-GitDirty
        if ($isDirty) {
            # Red for dirty repo
            $promptParts += "`e[31m($branch*)`e[0m"
        }
        else {
            # Green for clean repo
            $promptParts += "`e[32m($branch)`e[0m"
        }
    }
    
    # Join parts and add prompt character
    $promptString = $promptParts -join " "
    
    # Return prompt string with newline and prompt character
    # Use different character for admin (requires elevation check)
    $promptChar = if ($Script:IsAdministrator) {
        "`e[31m#`e[0m"  # Red # for admin
    }
    else {
        "$"  # Regular $ for user
    }
    
    return "$promptString`n$promptChar "
}
