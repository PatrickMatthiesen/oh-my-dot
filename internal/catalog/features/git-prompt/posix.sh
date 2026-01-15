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
        # Choose appropriate non-printing markers for color codes based on shell
        local red_start green_start color_end
        if [ -n "$BASH_VERSION" ]; then
            # Bash uses \001 and \002 to mark non-printing characters in PS1
            red_start=$'\001\033[0;31m\002'
            green_start=$'\001\033[0;32m\002'
            color_end=$'\001\033[0m\002'
        elif [ -n "$ZSH_VERSION" ]; then
            # Zsh uses %{ %} to wrap non-printing sequences in the prompt
            red_start=$'%{\033[0;31m%}'
            green_start=$'%{\033[0;32m%}'
            color_end=$'%{\033[0m%}'
        else
            # Fallback: plain ANSI escapes without non-printing markers
            red_start=$'\033[0;31m'
            green_start=$'\033[0;32m'
            color_end=$'\033[0m'
        fi

        local dirty=$(parse_git_dirty)
        if [ -n "$dirty" ]; then
            printf " (%s%s%s%s)" "$red_start" "$branch" "$dirty" "$color_end"
        else
            printf " (%s%s%s)" "$green_start" "$branch" "$color_end"
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
    # Preserve existing PROMPT where possible and only add git_prompt once
    if [[ "$PROMPT" != *'$(git_prompt)'* ]]; then
        if [ -z "$PROMPT" ]; then
            # Default zsh prompt style with git_prompt
            export PROMPT='%n@%h:%~$(git_prompt)%# '
        else
            # Append git_prompt to the existing prompt
            export PROMPT="${PROMPT}\$(git_prompt)"
        fi
    fi
fi
