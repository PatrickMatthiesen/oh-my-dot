# Directory Shortcuts - Quick navigation to common directories
# Customize these shortcuts to your frequently used directories

# Home directory shortcuts
alias ~='cd ~'
alias home='cd ~'

# Common development directories
alias dev='cd ~/dev'
alias proj='cd ~/projects'
alias repos='cd ~/repos'
alias docs='cd ~/Documents'
alias dl='cd ~/Downloads'
alias desk='cd ~/Desktop'

# Quick back navigation
alias .1='cd ..'
alias .2='cd ../..'
alias .3='cd ../../..'
alias .4='cd ../../../..'
alias .5='cd ../../../../..'

# Directory stack shortcuts
alias d='dirs -v'
alias pd='pushd'
alias po='popd'

# Show current directory
alias pwd='pwd'
alias here='pwd'

# Open current directory in file manager (if available)
if command -v xdg-open >/dev/null 2>&1; then
    alias open='xdg-open'
elif command -v open >/dev/null 2>&1; then
    # macOS: 'open' already exists; keep the built-in command and do nothing here
    :
fi

# Quick directory opening
alias o='open .'
