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

if (Test-Path $omdOmpConfig -PathType Leaf) {
    oh-my-posh init pwsh --config $omdOmpConfig | Invoke-Expression
}
else {
    oh-my-posh init pwsh | Invoke-Expression
}