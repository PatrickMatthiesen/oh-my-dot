if not type -q oh-my-posh
    return
end

set -l omd_omp_config {{ if .ConfigFile }}{{ literal .ConfigFile }}{{ else }}"{{ .DefaultConfigPath }}"{{ end }}

{{ if not .ConfigFile }}if not test -f "$omd_omp_config"
    mkdir -p "$OMD_SHELL_ROOT/.local/cache" 2>/dev/null; or true
    if type -q curl
        curl -fsSL {{ literal .ThemeURL }} -o "$omd_omp_config" 2>/dev/null
    else if type -q wget
        wget -qO "$omd_omp_config" {{ literal .ThemeURL }} 2>/dev/null
    end
end

{{ end }}{{ if .AutoUpgrade }}# Oh My Posh caches update checks according to its configured upgrade interval.
if test -f "$omd_omp_config"
    oh-my-posh upgrade --auto --config "$omd_omp_config"; or true
else
    oh-my-posh upgrade --auto; or true
end

{{ end }}if test -f "$omd_omp_config"
    oh-my-posh init fish --config "$omd_omp_config" | source
else
    oh-my-posh init fish | source
end
