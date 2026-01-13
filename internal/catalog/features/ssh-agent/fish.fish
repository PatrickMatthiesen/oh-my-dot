# SSH Agent Management for Fish
# Automatically starts ssh-agent and loads SSH keys if not already running

if not set -q SSH_AUTH_SOCK
    set -l SSH_ENV "$HOME/.ssh/agent-env.fish"
    
    if test -f $SSH_ENV
        source $SSH_ENV > /dev/null
        # Check if the agent is still running
        if not kill -0 $SSH_AGENT_PID 2>/dev/null
            # Agent died, start a new one
            eval (ssh-agent -c) > /dev/null
            echo "set -x SSH_AUTH_SOCK $SSH_AUTH_SOCK;" > $SSH_ENV
            echo "set -x SSH_AGENT_PID $SSH_AGENT_PID;" >> $SSH_ENV
        end
    else
        # No agent info file, start agent
        eval (ssh-agent -c) > /dev/null
        echo "set -x SSH_AUTH_SOCK $SSH_AUTH_SOCK;" > $SSH_ENV
        echo "set -x SSH_AGENT_PID $SSH_AGENT_PID;" >> $SSH_ENV
    end
    
    # Add default SSH keys
    ssh-add 2>/dev/null
end
