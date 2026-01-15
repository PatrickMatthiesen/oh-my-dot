# Node Version Manager (NVM) Integration
# Loads NVM for Node.js version management

# NVM installation directory
export NVM_DIR="$HOME/.nvm"

# Load NVM if it exists
if [ -s "$NVM_DIR/nvm.sh" ]; then
    # Source nvm script
    . "$NVM_DIR/nvm.sh"
    
    # Load bash completion if available
    if [ -s "$NVM_DIR/bash_completion" ]; then
        . "$NVM_DIR/bash_completion"
    fi
    
    # Auto-switch Node version based on .nvmrc if present
    # For zsh users: uncomment the following block to enable auto-switching when entering a directory.
    # The code is wrapped in a zsh version check so it is safe to keep in POSIX shells like bash.
    # if [ -n "$ZSH_VERSION" ]; then
    #   autoload -U add-zsh-hook
    #   load-nvmrc() {
    #     if [[ -f .nvmrc && -r .nvmrc ]]; then
    #       nvm use
    #     elif [[ $(nvm version) != $(nvm version default)  ]]; then
    #       echo "Reverting to nvm default version"
    #       nvm use default
    #     fi
    #   }
    #   add-zsh-hook chpwd load-nvmrc
    #   load-nvmrc
    # fi
fi

# Helpful NVM aliases
alias nvmls='nvm ls'
alias nvmlsr='nvm ls-remote'
alias nvmi='nvm install'
alias nvmu='nvm use'
alias nvmdefault='nvm alias default'
