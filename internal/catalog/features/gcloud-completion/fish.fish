#!/usr/bin/env fish
# Google Cloud CLI (gcloud) Command Completion for Fish
# Enables gcloud shell completions for faster command-line usage

# Check if gcloud is installed
if command -v gcloud >/dev/null 2>&1
    # Try to find gcloud installation directory
    set -l GCLOUD_SDK_PATH ""
    
    # Common installation paths
    if test -d "$HOME/google-cloud-sdk"
        set GCLOUD_SDK_PATH "$HOME/google-cloud-sdk"
    else if test -d "/usr/local/Caskroom/google-cloud-sdk/latest/google-cloud-sdk"
        set GCLOUD_SDK_PATH "/usr/local/Caskroom/google-cloud-sdk/latest/google-cloud-sdk"
    else if test -d "/opt/google-cloud-sdk"
        set GCLOUD_SDK_PATH "/opt/google-cloud-sdk"
    else if test -d "/usr/lib/google-cloud-sdk"
        set GCLOUD_SDK_PATH "/usr/lib/google-cloud-sdk"
    end
    
    # Load completion if found
    if test -n "$GCLOUD_SDK_PATH"
        # Source path file
        if test -f "$GCLOUD_SDK_PATH/path.fish.inc"
            source "$GCLOUD_SDK_PATH/path.fish.inc"
        end
        
        # Fish completion is typically handled by the gcloud installation
    end
    
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
end
