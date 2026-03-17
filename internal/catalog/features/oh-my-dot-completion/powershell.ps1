# oh-my-dot Shell Completion for PowerShell
# Enables command-line completion for oh-my-dot (omdot) commands

# Register completions for every available command name. Cobra binds the
# completer to the command name used to generate the script.
foreach ($commandName in @("omdot", "oh-my-dot")) {
    if (-not (Get-Command $commandName -ErrorAction SilentlyContinue)) {
        continue
    }

    try {
        $completionScript = & $commandName completion powershell 2>$null | Out-String
        if (-not [string]::IsNullOrWhiteSpace($completionScript)) {
            Invoke-Expression $completionScript
        }
    }
    catch {
        continue
    }
}
