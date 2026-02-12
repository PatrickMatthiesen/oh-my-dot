if not type -q oh-my-posh
    return
end

set -l omd_omp_config "{{ if .ConfigFile }}{{ .ConfigFile }}{{ else }}$OMD_SHELL_ROOT/features/oh-my-posh.omp.json{{ end }}"

if not test -f "$omd_omp_config"
    if type -q curl
        curl -fsSL "{{ .ThemeURL }}" -o "$omd_omp_config" 2>/dev/null
    else if type -q wget
        wget -qO "$omd_omp_config" "{{ .ThemeURL }}" 2>/dev/null
    end
end

if test -f "$omd_omp_config"
    oh-my-posh init fish --config "$omd_omp_config" | source
else
    oh-my-posh init fish | source
end