if ! command -v oh-my-posh >/dev/null 2>&1; then
  return 0
fi

OMD_OMP_CONFIG="{{ if .ConfigFile }}{{ .ConfigFile }}{{ else }}{{ .DefaultConfigPath }}{{ end }}"

if [ ! -f "$OMD_OMP_CONFIG" ]; then
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "{{ .ThemeURL }}" -o "$OMD_OMP_CONFIG" 2>/dev/null
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$OMD_OMP_CONFIG" "{{ .ThemeURL }}" 2>/dev/null
  fi
fi

{{ if .AutoUpgrade }}# Oh My Posh caches update checks according to its configured upgrade interval.
if [ -f "$OMD_OMP_CONFIG" ]; then
  oh-my-posh upgrade --auto --config "$OMD_OMP_CONFIG" || :
else
  oh-my-posh upgrade --auto || :
fi

{{ end }}if [ -f "$OMD_OMP_CONFIG" ]; then
  eval "$(oh-my-posh init bash --config "$OMD_OMP_CONFIG")"
else
  eval "$(oh-my-posh init bash)"
fi
