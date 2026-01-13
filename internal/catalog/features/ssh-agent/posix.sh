# SSH Agent Management for POSIX sh
# Automatically starts ssh-agent and loads SSH keys if not already running

if [ -z "$SSH_AUTH_SOCK" ]; then
    # Check if agent info file exists and is valid
    SSH_ENV="$HOME/.ssh/agent-env"
    
    if [ -f "$SSH_ENV" ]; then
        . "$SSH_ENV" > /dev/null
        # Check if the agent is still running
        if ! kill -0 "$SSH_AGENT_PID" 2>/dev/null; then
            # Agent died, start a new one
            eval "$(ssh-agent -s)" > /dev/null
            echo "export SSH_AUTH_SOCK=$SSH_AUTH_SOCK" > "$SSH_ENV"
            echo "export SSH_AGENT_PID=$SSH_AGENT_PID" >> "$SSH_ENV"
        fi
    else
        # No agent info file, start agent
        eval "$(ssh-agent -s)" > /dev/null
        echo "export SSH_AUTH_SOCK=$SSH_AUTH_SOCK" > "$SSH_ENV"
        echo "export SSH_AGENT_PID=$SSH_AGENT_PID" >> "$SSH_ENV"
    fi
    
    # Add default SSH keys
    ssh-add 2>/dev/null
fi
