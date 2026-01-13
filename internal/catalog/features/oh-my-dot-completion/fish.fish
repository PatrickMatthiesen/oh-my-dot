#!/usr/bin/env fish
# oh-my-dot Shell Completion for Fish
# Enables command-line completion for oh-my-dot (omdot) commands

# Check if oh-my-dot/omdot is installed
if command -v omdot >/dev/null 2>&1
    # Generate and source Fish completions
    omdot completion fish | source
else if command -v oh-my-dot >/dev/null 2>&1
    # Fallback to oh-my-dot command name
    oh-my-dot completion fish | source
end
