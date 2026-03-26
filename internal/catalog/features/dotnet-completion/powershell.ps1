# dotnet-completion - Native PowerShell argument completion for dotnet CLI

if (-not (Get-Command dotnet -ErrorAction SilentlyContinue)) {
    Write-Verbose "dotnet is not installed; skipping dotnet-completion feature"
    return
}

Register-ArgumentCompleter -Native -CommandName dotnet -ScriptBlock {
    param($commandName, $wordToComplete, $cursorPosition)

    dotnet complete --position $cursorPosition "$wordToComplete" 2>$null | ForEach-Object {
        [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
    }
}