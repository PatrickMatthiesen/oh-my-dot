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
        if omd_completion_output="$($OMD_CMD completion bash 2>/dev/null)"; then
            source /dev/stdin <<<"$omd_completion_output"
        fi
    elif [ -n "$ZSH_VERSION" ]; then
        # Zsh completion
        if omd_completion_output="$($OMD_CMD completion zsh 2>/dev/null)"; then
            source /dev/stdin <<<"$omd_completion_output"
        fi
    fi
fi
