if (-not (Get-Command oh-my-posh -ErrorAction SilentlyContinue)) {
    return
}

$omdOmpConfig = {{ if .ConfigFile }}{{ literal .ConfigFile }}{{ else }}"{{ .DefaultConfigPath }}"{{ end }}

{{ if not .ConfigFile }}if (-not ([System.IO.File]::Exists($omdOmpConfig))) {
    New-Item -ItemType Directory -Path "$OMD_SHELL_ROOT/.local/cache" -Force -ErrorAction SilentlyContinue | Out-Null
    $themeUrl = {{ literal .ThemeURL }}
    try {
        Invoke-WebRequest -Uri $themeUrl -OutFile $omdOmpConfig -UseBasicParsing -ErrorAction Stop | Out-Null
    }
    catch {
    }
}

{{ end }}{{ if .AutoUpgrade }}# Oh My Posh caches update checks according to its configured upgrade interval.
try {
    if ([System.IO.File]::Exists($omdOmpConfig)) {
        oh-my-posh upgrade --auto --config $omdOmpConfig
    }
    else {
        oh-my-posh upgrade --auto
    }
}
catch {
    Write-Warning "Oh My Posh auto upgrade failed: $_"
}

{{ end }}if ([System.IO.File]::Exists($omdOmpConfig)) {
    oh-my-posh init pwsh --config $omdOmpConfig | Invoke-Expression
}
else {
    oh-my-posh init pwsh | Invoke-Expression
}
