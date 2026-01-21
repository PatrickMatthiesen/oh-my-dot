# Homebrew PATH Setup for Linux (Fish Shell)
# Sets up Homebrew's bin directory in PATH for package management

# Homebrew paths
set -l homebrew_prefix "/home/linuxbrew/.linuxbrew"
set -l homebrew_bin "$homebrew_prefix/bin"

# Check if Homebrew is installed in the standard Linux location
if test -d $homebrew_prefix
    # Add Homebrew to PATH if not already present
    if not contains $homebrew_bin $PATH
        set -gx PATH $homebrew_bin $PATH
    end
    
    # Set up Homebrew environment
    eval ($homebrew_bin/brew shellenv)
end
