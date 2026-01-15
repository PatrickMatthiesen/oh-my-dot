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
            printf " (\001\033[0;31m\002%s%s\001\033[0m\002)" "$branch" "$dirty"
        else
            printf " (\001\033[0;32m\002%s\001\033[0m\002)" "$branch"
        fi
    fi
}

# Add git info to PS1
# Customize this to match your preferred prompt style
if [ -n "$BASH_VERSION" ]; then
    # Bash prompt - preserve existing PS1 and add git_prompt
    # Only modify PS1 if it doesn't already contain git_prompt
    if [[ "$PS1" != *'$(git_prompt)'* ]]; then
        # Insert git_prompt before the final $ or # prompt character
        if [[ "$PS1" =~ (.*)(\\[$\#])([[:space:]]*)$ ]]; then
            export PS1="${BASH_REMATCH[1]}\$(git_prompt)${BASH_REMATCH[2]}${BASH_REMATCH[3]}"
        else
            # Fallback: just append to the end
            export PS1="${PS1}\$(git_prompt)"
        fi
    fi
elif [ -n "$ZSH_VERSION" ]; then
    # Zsh prompt - enable parameter expansion in prompts
    setopt PROMPT_SUBST
    export PROMPT='%n@%h:%~$(git_prompt)%# '
fi
