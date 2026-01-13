# Git Prompt for Fish - Show git branch and status in prompt
# Displays current git branch and repository status in your shell prompt

# Fish has built-in git prompt support via fish_git_prompt
# We just need to configure it

# Enable git prompt
set -g __fish_git_prompt_showdirtystate 'yes'
set -g __fish_git_prompt_showstashstate 'yes'
set -g __fish_git_prompt_showuntrackedfiles 'yes'
set -g __fish_git_prompt_showupstream 'auto'

# Prompt colors
set -g __fish_git_prompt_color_branch yellow
set -g __fish_git_prompt_color_dirtystate red
set -g __fish_git_prompt_color_stagedstate green
set -g __fish_git_prompt_color_untrackedfiles cyan
set -g __fish_git_prompt_color_cleanstate green

# Status characters
set -g __fish_git_prompt_char_dirtystate '*'
set -g __fish_git_prompt_char_stagedstate '+'
set -g __fish_git_prompt_char_untrackedfiles '?'
set -g __fish_git_prompt_char_stashstate '$'
set -g __fish_git_prompt_char_upstream_ahead '↑'
set -g __fish_git_prompt_char_upstream_behind '↓'

# Define the fish_prompt function to include git information
function fish_prompt
    set -l last_status $status
    
    # User and host
    set_color cyan
    echo -n (whoami)
    set_color normal
    echo -n '@'
    set_color green
    echo -n (hostname -s)
    set_color normal
    echo -n ':'
    
    # Current directory
    set_color blue
    echo -n (prompt_pwd)
    set_color normal
    
    # Git information
    echo -n (fish_git_prompt)
    
    # Prompt character
    if test $last_status -eq 0
        set_color green
    else
        set_color red
    end
    echo -n ' > '
    set_color normal
end
