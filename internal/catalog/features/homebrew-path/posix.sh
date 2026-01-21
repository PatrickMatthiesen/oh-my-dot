# Homebrew PATH Setup for Linux
# Sets up Homebrew's bin directory in PATH for package management

# Homebrew paths
HOMEBREW_PREFIX="/home/linuxbrew/.linuxbrew"
HOMEBREW_BIN="$HOMEBREW_PREFIX/bin"

# Check if Homebrew is installed in the standard Linux location
if [ -d "$HOMEBREW_PREFIX" ]; then
    # Add Homebrew to PATH if not already present
    # Use case statement for exact directory matching
    case ":$PATH:" in
        *":$HOMEBREW_BIN:"*) ;;
        *) export PATH="$HOMEBREW_BIN:$PATH" ;;
    esac
    
    # Set up Homebrew environment
    eval "$("$HOMEBREW_BIN/brew" shellenv)"
fi
