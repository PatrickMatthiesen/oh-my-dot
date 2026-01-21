# Homebrew PATH Setup for Linux
# Sets up Homebrew's bin directory in PATH for package management

# Check if Homebrew is installed in the standard Linux location
if [ -d "/home/linuxbrew/.linuxbrew" ]; then
    # Add Homebrew to PATH if not already present
    if ! echo "$PATH" | grep -q "/home/linuxbrew/.linuxbrew/bin"; then
        export PATH="/home/linuxbrew/.linuxbrew/bin:$PATH"
    fi
    
    # Set up Homebrew environment
    eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
fi
