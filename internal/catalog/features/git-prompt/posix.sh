# Git Prompt - Show git branch and status in prompt
# Displays current git branch and repository status in your shell prompt

# Parse git branch name
parse_git_branch() {
    git branch 2>/dev/null | grep '^*' | colrm 1 2
}

# Parse git status to show dirty/clean state
parse_git_dirty() {
    local status=$(git status --porcelain 2>/dev/null)
    if [ -n "$status" ]; then
        echo "*"
    fi
}

# Colorful git prompt function
git_prompt() {
    local branch=$(parse_git_branch)
    if [ -n "$branch" ]; then
        local dirty=$(parse_git_dirty)
        if [ -n "$dirty" ]; then
            echo " (\033[0;31m$branch$dirty\033[0m)"
        else
            echo " (\033[0;32m$branch\033[0m)"
        fi
    fi
}

# Add git info to PS1
# Customize this to match your preferred prompt style
if [ -n "$BASH_VERSION" ]; then
    # Bash prompt
    export PS1='\u@\h:\w$(git_prompt)\$ '
elif [ -n "$ZSH_VERSION" ]; then
    # Zsh prompt - enable parameter expansion in prompts
    setopt PROMPT_SUBST
    export PROMPT='%n@%h:%~$(git_prompt)%# '
fi
