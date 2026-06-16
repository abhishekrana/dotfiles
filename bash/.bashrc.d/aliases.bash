# Shell
set -o vi
# vi-insert keymap defaults Ctrl+L to self-insert; restore clear-screen.
# `bind` is bash-only; zsh has `bindkey` and a separate vicmd/viins logic, so
# skip when sourced under zsh (the macOS default shell).
if [ -n "${BASH_VERSION:-}" ]; then
    bind -m vi-insert '"\C-l": clear-screen'
    bind -m vi-command -x '"\C-l": printf "\033[2J\033[H"'
fi
export VISUAL=nvim
export EDITOR=nvim
export TERM="tmux-256color"
# `open` (macOS) opens URLs in the user's default browser; firefox elsewhere.
if [ "$(uname)" = "Darwin" ]; then
    export BROWSER="open"
else
    export BROWSER="firefox"
fi

# Git
alias gsm='git switch main'
alias gsw='git switch'
alias gpm='git pull origin main'
alias ga='git add .'
alias gd='git diff'
alias gdl='git diff HEAD~1 HEAD'
alias gm='git commit -m'
alias gf='git fetch --prune'
# alias gl='git pull'
alias gl='git log'
alias glog='git log --oneline --decorate --graph'
alias gp='git push'
alias gpub='git push origin $(git branch --show-current)'
alias gplb='git pull origin $(git branch --show-current)'
alias gs='git status'
# alias gst='git status'
alias gb='git branch --sort=committerdate --format="%(refname:short) %(committerdate:relative)" | tail -20 | awk -F" " "{name=\$1; \$1=\"\"; printf \"%-50s (%s)\\n\", name, substr(\$0,2)}" && echo "" && echo "* $(git branch --show-current)"'
alias gdd='nvim -c "DiffviewOpen"'
alias gddm='nvim -c "DiffviewOpen main"'
alias gmr='nvim -c "DiffviewOpen origin/main...HEAD"'
alias gmrd='git diff origin/main...HEAD'
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
export DOCKER_BUILDKIT=1
export COMPOSE_DOCKER_CLI_BUILD=1
alias docker-stop-all='docker stop $(docker ps -a -q)'
alias docker-rm-all='docker rm $(docker ps -a -q)'

# Files
alias ll='ls -lrth'
alias bat='bat --paging=never --style=plain'

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
alias cheat='bat ~/git/dotfiles/CHEATSHEET.md 2>/dev/null || bat ~/dotfiles/CHEATSHEET.md'

# Fuzzy grep: interactive ripgrep across all files from cwd
alias rgg='rg --line-number "" | fzf --delimiter : --preview "bat --color=always {1} --highlight-line {2}"'
