# PowerShell PSReadLine - Interactive key bindings and editing enhancements

if (-not (Get-Command Set-PSReadLineOption -ErrorAction SilentlyContinue)) {
    Write-Verbose "PSReadLine is not available; skipping powershell-psreadline feature"
    return
}

if ($Host.Name -ne "ConsoleHost" -or [Console]::IsInputRedirected -or [Console]::IsOutputRedirected) {
    Write-Verbose "PSReadLine console features require an interactive console; skipping powershell-psreadline feature"
    return
}

# Shows navigable menu of all options when hitting Tab
try {
    Set-PSReadLineKeyHandler -Key Tab -Function MenuComplete -ErrorAction Stop
    Set-PSReadLineOption -PredictionViewStyle ListView -ErrorAction Stop
}
catch [System.IO.IOException] {
    Write-Verbose "PSReadLine console handle is not available; skipping powershell-psreadline feature"
    return
}

# Insert paired quotes and move cursor between them when appropriate
Set-PSReadLineKeyHandler -Chord '"', "'" `
    -BriefDescription SmartInsertQuote `
    -LongDescription "Insert paired quotes if not already on a quote" `
    -ScriptBlock {
    param($key, $arg)

    $line = $null
    $cursor = $null
    [Microsoft.PowerShell.PSConsoleReadLine]::GetBufferState([ref]$line, [ref]$cursor)

    if ($line.Length -gt $cursor -and $line[$cursor] -eq $key.KeyChar) {
        [Microsoft.PowerShell.PSConsoleReadLine]::SetCursorPosition($cursor + 1)
    }
    else {
        [Microsoft.PowerShell.PSConsoleReadLine]::Insert("$($key.KeyChar)" * 2)
        [Microsoft.PowerShell.PSConsoleReadLine]::GetBufferState([ref]$line, [ref]$cursor)
        [Microsoft.PowerShell.PSConsoleReadLine]::SetCursorPosition($cursor - 1)
    }
}
