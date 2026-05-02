# Shell
set -o vi
# vi-insert keymap defaults Ctrl+L to self-insert; restore clear-screen
bind -m vi-insert '"\C-l": clear-screen'
export VISUAL=nvim
export EDITOR=nvim
export TERM="tmux-256color"
export BROWSER="firefox"

# Git
alias gd='git diff'
alias gdl='git diff HEAD~1 HEAD'
alias gf='git fetch'
# alias gl='git pull'
alias gl='git log'
alias glog='git log --oneline --decorate --graph'
alias gp='git push'
alias gs='git status'
# alias gst='git status'
alias gb='git branch --sort=committerdate --format="%(refname:short) %(committerdate:relative)" | tail -20 | awk -F" " "{name=\$1; \$1=\"\"; printf \"%-50s (%s)\\n\", name, substr(\$0,2)}" && echo "" && echo "* $(git branch --show-current)"'
alias gdd='nvim -c "DiffviewOpen"'
alias gddm='nvim -c "DiffviewOpen main"'
alias gmr='nvim -c "DiffviewOpen origin/main...HEAD"'
alias gw='while clear; do git diff --stat --color && echo "---" && git diff --color | head -60; sleep 2; done'

# Docker
alias docker-stop-all='docker stop $(docker ps -a -q)'
alias docker-rm-all='docker rm $(docker ps -a -q)'

# Files
alias ll='ls -lrth'

# Editor
alias vim='nvim'
alias vimr='NVIM_RESTORE=1 nvim'

# Quick reference
alias cheat='bat ~/dotfiles/CHEATSHEET.md'
