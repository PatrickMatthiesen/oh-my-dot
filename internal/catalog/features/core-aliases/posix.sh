# Core Aliases - Essential command shortcuts
# Cross-platform aliases for common commands

# Navigation shortcuts
alias ..='cd ..'
alias ...='cd ../..'
alias ....='cd ../../..'

# ls aliases (with color support if available)
if ls --color=auto / >/dev/null 2>&1; then
    # GNU ls (Linux)
    alias ls='ls --color=auto'
    alias ll='ls -lh --color=auto'
    alias la='ls -lah --color=auto'
else
    # BSD ls (macOS) or fallback
    alias ls='ls -G'
    alias ll='ls -lhG'
    alias la='ls -lahG'
fi

# Git shortcuts
alias g='git'
alias gs='git status'
alias ga='git add'
alias gc='git commit'
alias gca='git commit -a'
alias gcm='git commit -m'
alias gp='git push'
alias gpl='git pull'
alias gd='git diff'
alias gco='git checkout'
alias gb='git branch'
alias gl='git log --oneline --graph --decorate'

# Safety aliases
alias rm='rm -i'
alias cp='cp -i'
alias mv='mv -i'

# Directory listing
alias lsd='ls -d */'

# Grep with color
alias grep='grep --color=auto'

# Show PATH in readable format
alias path='echo $PATH | tr ":" "\n"'

# Create parent directories as needed
alias mkdir='mkdir -pv'

# Reload shell configuration
alias reload='exec $SHELL'
