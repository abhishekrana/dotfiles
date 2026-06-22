# Shell
set -o vi
# vi-insert keymap defaults Ctrl+L to self-insert; restore clear-screen
bind -m vi-insert '"\C-l": clear-screen'
export VISUAL=nvim
export EDITOR=nvim
export TERM="tmux-256color"
export BROWSER="firefox"

# Git — gd/gds/gdw/gdm open hunk's interactive TUI; git's own pager is delta
alias gd='hunk diff'          # review working-tree changes (incl. untracked)
alias gds='hunk show'         # review the latest commit
alias gdw='hunk diff --watch' # working-tree review, auto-reload on change
alias gf='git fetch'
# alias gl='git pull'
alias gl='git log'
alias gp='git push'
alias gs='git status'
# alias gst='git status'
alias gb='git branch --sort=committerdate --format="%(refname:short) %(committerdate:relative)" | tail -20 | awk -F" " "{name=\$1; \$1=\"\"; printf \"%-50s (%s)\\n\", name, substr(\$0,2)}" && echo "" && echo "* $(git branch --show-current)"'
# gbr: remote branches (last 500 by recency) with tip author; run gf first to refresh
alias gbr='git branch -r --sort=committerdate --format="%(refname:short)%09%(authorname)%09%(committerdate:relative)" | grep -v "/HEAD" | tail -500 | awk -F"\t" "{ printf \"%-60s %-20s (%s)\\n\", \$1, \$2, \$3 }"'
alias gdm='hunk diff origin/main...HEAD'
alias gw='while clear; do git diff --stat --color && echo "---" && git diff --color | head -60; sleep 2; done'

# Worktree family (mirrors oh-my-zsh git plugin naming: gwt/gwta/gwtls/gwtmv/gwtrm)
alias gwt='git worktree'
alias gwtls='git worktree list'
alias gwtmv='git worktree move'
alias gwtrm='git worktree remove'

# gwta: add a worktree for a branch, resolving it wherever it exists.
# Arg order mirrors `git worktree add <dir> <branch>`.
# Remote (latest) > local-only > new branch off base (default origin/main).
# Usage: gwta <dir> <branch> [base]
gwta() {
  local dir="$1" branch="$2" base="${3:-origin/main}"
  if [ -z "$dir" ] || [ -z "$branch" ]; then
    echo "usage: gwta <dir> <branch> [base]" >&2
    return 1
  fi
  git fetch origin "$branch" 2>/dev/null
  if git show-ref --verify --quiet "refs/remotes/origin/$branch"; then
    # exists on remote -> (re)point local branch at remote tip
    git worktree add "$dir" -B "$branch" "origin/$branch"
  elif git show-ref --verify --quiet "refs/heads/$branch"; then
    # local-only branch -> check out as-is
    git worktree add "$dir" "$branch"
  else
    # nowhere -> create new branch off base
    git fetch origin 2>/dev/null
    echo "branch '$branch' not found; creating it off '$base'" >&2
    git worktree add "$dir" -b "$branch" "$base"
  fi
}

# gwts: fuzzy-switch between worktrees of the current repo (cd into the pick).
gwts() {
  local line dir
  line=$(git worktree list | grep -v ' (bare)$' | fzf --prompt='worktree> ') || return
  dir="${line%% *}"
  [ -n "$dir" ] && cd "$dir"
}

# Docker
alias docker-stop-all='docker stop $(docker ps -a -q)'
alias docker-rm-all='docker rm $(docker ps -a -q)'

# Files
alias ll='ls -lrth'
# --pager less -RFX: -X keeps paged output in tmux scrollback (no alt-screen),
# so mouse-drag selection still copies; -F prints inline when it fits one screen.
alias bat='bat --style=plain --pager="less -RFX"'

# Editor
alias vim='nvim'
alias vimr='NVIM_RESTORE=1 nvim'

# Tmux
# Attach to session (default: current dir name), creating if missing.
# Inside tmux, switch the client instead of nesting sessions.
ta() {
  local name="${1:-$(basename "$PWD")}"
  if [ -n "$TMUX" ]; then
    tmux new-session -d -s "$name" 2>/dev/null
    tmux switch-client -t "$name"
  else
    tmux new-session -A -s "$name"
  fi
}

# Quick reference
alias cheat='bat ~/dotfiles/CHEATSHEET.md'
