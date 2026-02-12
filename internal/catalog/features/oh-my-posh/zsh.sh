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

if [ -f "$OMD_OMP_CONFIG" ]; then
  eval "$(oh-my-posh init zsh --config \"$OMD_OMP_CONFIG\")"
else
  eval "$(oh-my-posh init zsh)"
fi
