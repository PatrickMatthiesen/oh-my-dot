#!/usr/bin/env sh
# oh-my-dot Shell Completion
# Enables command-line completion for oh-my-dot (omdot) commands

# Check if oh-my-dot/omdot is installed
if command -v omdot >/dev/null 2>&1 || command -v oh-my-dot >/dev/null 2>&1; then
    # Determine which command name is available
    OMD_CMD="omdot"
    if ! command -v omdot >/dev/null 2>&1; then
        OMD_CMD="oh-my-dot"
    fi
    
    if [ -n "$BASH_VERSION" ]; then
        # Bash completion
        source <($OMD_CMD completion bash)
    elif [ -n "$ZSH_VERSION" ]; then
        # Zsh completion
        source <($OMD_CMD completion zsh)
    fi
fi
