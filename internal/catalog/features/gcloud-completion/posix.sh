#!/usr/bin/env sh
# Google Cloud CLI (gcloud) Command Completion
# Enables gcloud shell completions for faster command-line usage

# Check if gcloud is installed
if command -v gcloud >/dev/null 2>&1; then
    # Try to find gcloud installation directory
    GCLOUD_SDK_PATH=""
    
    # Common installation paths
    if [ -d "$HOME/google-cloud-sdk" ]; then
        GCLOUD_SDK_PATH="$HOME/google-cloud-sdk"
    elif [ -d "/usr/local/Caskroom/google-cloud-sdk/latest/google-cloud-sdk" ]; then
        GCLOUD_SDK_PATH="/usr/local/Caskroom/google-cloud-sdk/latest/google-cloud-sdk"
    elif [ -d "/opt/google-cloud-sdk" ]; then
        GCLOUD_SDK_PATH="/opt/google-cloud-sdk"
    elif [ -d "/usr/lib/google-cloud-sdk" ]; then
        GCLOUD_SDK_PATH="/usr/lib/google-cloud-sdk"
    fi
    
    # Load completion if found
    if [ -n "$GCLOUD_SDK_PATH" ]; then
        if [ -n "$BASH_VERSION" ] && [ -f "$GCLOUD_SDK_PATH/completion.bash.inc" ]; then
            . "$GCLOUD_SDK_PATH/completion.bash.inc"
        elif [ -n "$ZSH_VERSION" ] && [ -f "$GCLOUD_SDK_PATH/completion.zsh.inc" ]; then
            . "$GCLOUD_SDK_PATH/completion.zsh.inc"
        fi
        
        # Add gcloud commands to PATH if not already there
        if [ -f "$GCLOUD_SDK_PATH/path.bash.inc" ]; then
            . "$GCLOUD_SDK_PATH/path.bash.inc"
        elif [ -f "$GCLOUD_SDK_PATH/path.zsh.inc" ]; then
            . "$GCLOUD_SDK_PATH/path.zsh.inc"
        fi
    fi
    
    # Common gcloud aliases
    alias gc='gcloud'
    alias gcconfig='gcloud config'
    alias gcauth='gcloud auth'
    alias gccompute='gcloud compute'
    alias gcstorage='gcloud storage'
    alias gccontainer='gcloud container'
    alias gciam='gcloud iam'
    alias gcprojects='gcloud projects list'
    alias gcconfigs='gcloud config configurations list'
    alias gcsetproject='gcloud config set project'
fi
