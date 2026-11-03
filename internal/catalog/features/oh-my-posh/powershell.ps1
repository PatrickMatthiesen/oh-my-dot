if (-not (Get-Command oh-my-posh -ErrorAction SilentlyContinue)) {
    return
}

$omdOmpConfig = "{{ if .ConfigFile }}{{ .ConfigFile }}{{ else }}{{ .DefaultConfigPath }}{{ end }}"

if (-not (Test-Path $omdOmpConfig -PathType Leaf)) {
    $themeUrl = "{{ .ThemeURL }}"
    try {
        Invoke-WebRequest -Uri $themeUrl -OutFile $omdOmpConfig -UseBasicParsing -ErrorAction Stop | Out-Null
    }
    catch {
    }
}

{{ if .AutoUpgrade }}# Oh My Posh caches update checks according to its configured upgrade interval.
try {
    if (Test-Path $omdOmpConfig -PathType Leaf) {
        oh-my-posh upgrade --auto --config $omdOmpConfig
    }
    else {
        oh-my-posh upgrade --auto
    }
}
catch {
    Write-Warning "Oh My Posh auto upgrade failed: $_"
}

{{ end }}if (Test-Path $omdOmpConfig -PathType Leaf) {
    oh-my-posh init pwsh --config $omdOmpConfig | Invoke-Expression
}
else {
    oh-my-posh init pwsh | Invoke-Expression
}