# Homebrew PATH Setup for Linux (Fish Shell)
# Sets up Homebrew's bin directory in PATH for package management

# Check if Homebrew is installed in the standard Linux location
if test -d "/home/linuxbrew/.linuxbrew"
    # Add Homebrew to PATH if not already present
    if not string match -q "*/home/linuxbrew/.linuxbrew/bin*" $PATH
        set -gx PATH "/home/linuxbrew/.linuxbrew/bin" $PATH
    end
    
    # Set up Homebrew environment
    eval (/home/linuxbrew/.linuxbrew/bin/brew shellenv)
end
