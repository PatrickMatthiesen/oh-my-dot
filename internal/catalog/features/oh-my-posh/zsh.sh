if ! command -v oh-my-posh >/dev/null 2>&1; then
  return 0
fi

OMD_OMP_CONFIG={{ if .ConfigFile }}{{ literal .ConfigFile }}{{ else }}"{{ .DefaultConfigPath }}"{{ end }}

{{ if not .ConfigFile }}if [ ! -f "$OMD_OMP_CONFIG" ]; then
  mkdir -p "$OMD_SHELL_ROOT/.local/cache" 2>/dev/null || :
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL {{ literal .ThemeURL }} -o "$OMD_OMP_CONFIG" 2>/dev/null
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$OMD_OMP_CONFIG" {{ literal .ThemeURL }} 2>/dev/null
  fi
fi

{{ end }}{{ if .AutoUpgrade }}# Oh My Posh caches update checks according to its configured upgrade interval.
if [ -f "$OMD_OMP_CONFIG" ]; then
  oh-my-posh upgrade --auto --config "$OMD_OMP_CONFIG" || :
else
  oh-my-posh upgrade --auto || :
fi

{{ end }}if [ -f "$OMD_OMP_CONFIG" ]; then
  eval "$(oh-my-posh init zsh --config "$OMD_OMP_CONFIG")"
else
  eval "$(oh-my-posh init zsh)"
fi
