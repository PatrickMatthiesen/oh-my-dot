# Node Version Manager (NVM) Integration for Fish
# Loads NVM for Node.js version management

# NVM installation directory
set -gx NVM_DIR "$HOME/.nvm"

# Load NVM if it exists
if test -s "$NVM_DIR/nvm.sh"
    # Fish doesn't use nvm.sh directly, we need bass or a fish-nvm plugin
    # For basic functionality, we'll create wrapper functions
    
    # If you're using fisher with fish-nvm plugin, it will be loaded automatically
    # Otherwise, you can install it with: fisher install jorgebucaran/nvm.fish
    
    # Basic wrapper using bass (if available)
    if type -q bass
        bass source "$NVM_DIR/nvm.sh"
    end
end

# Helpful NVM aliases
alias nvmls='nvm ls'
alias nvmlsr='nvm ls-remote'
alias nvmi='nvm install'
alias nvmu='nvm use'
alias nvmdefault='nvm alias default'

# Note: For full NVM support in Fish, consider installing:
# fisher install jorgebucaran/nvm.fish
# or
# fisher install edc/bass (for sourcing bash scripts in Fish)
