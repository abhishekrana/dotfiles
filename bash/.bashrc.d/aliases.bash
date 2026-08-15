# Shell
set -o vi
# vi-insert keymap defaults Ctrl+L to self-insert; restore clear-screen
bind -m vi-insert '"\C-l": clear-screen'
export VISUAL=nvim
export EDITOR=nvim
# No TERM here: tmux sets it inside panes, the terminal sets it outside. Forcing
# it hid Ghostty from tmux, so 'xterm-ghostty:*' terminal-features never matched.
export BROWSER="firefox"

# Git - gd/gds/gdw/gdm open hunk's interactive TUI; git's own pager is delta
alias gd='hunk diff'          # review working-tree changes (incl. untracked)
alias gds='hunk show'         # review the latest commit
alias gdw='hunk diff --watch' # working-tree review, auto-reload on change
alias gf='git fetch'
# alias gl='git pull'
alias gl='git log'
alias gp='git push'
alias gs='git status'
# alias gst='git status'
# gb: local branches by recency, then the current one. A function, not an alias,
# so the awk program needs no escaping and the lines stay readable.
gb() {
    git branch --sort=committerdate --format="%(refname:short) %(committerdate:relative)" |
        tail -20 |
        awk -F" " '{name=$1; $1=""; printf "%-50s (%s)\n", name, substr($0,2)}'
    echo ""
    echo "* $(git branch --show-current)"
}

# gbr: remote branches (last 500 by recency) with tip author; run gf first to refresh
gbr() {
    git branch -r --sort=committerdate \
        --format="%(refname:short)%09%(authorname)%09%(committerdate:relative)" |
        grep -v "/HEAD" |
        tail -500 |
        awk -F"\t" '{ printf "%-60s %-20s (%s)\n", $1, $2, $3 }'
}
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
    [ -n "$dir" ] || return
    cd "$dir" || return
}

# gwtm: put this worktree's branch (named after the worktree dir, created off
# origin/main if missing) on the latest origin/main, with the tool its state allows:
#   ff      nothing of our own
#   rebase  our own commits, branch not on origin - also clears `git pull` merges,
#           which are what make a branch un-fast-forwardable for good
#   merge   our own commits, branch on origin - never rewrite pushed history
# --autostash, so a dirty tree is fine. On conflict: abort, restore, say what to run.
gwtm() {
    local root name ahead
    root=$(git rev-parse --show-toplevel 2>/dev/null) || {
        echo "not in a git repo" >&2
        return 1
    }
    name=$(basename "$root")
    git fetch origin || return
    if git show-ref --verify --quiet "refs/heads/$name"; then
        git switch "$name" || return
    else
        git switch -c "$name" origin/main || return
    fi
    # The common case. stderr is git's generic divergence hint - dropped for ours.
    git merge --ff-only --autostash origin/main 2>/dev/null && return
    ahead=$(git rev-list --count origin/main..HEAD)
    # One op, one failure path: ${op[0]} names the command, its --abort and the advice.
    local -a op
    if git show-ref --verify --quiet "refs/remotes/origin/$name"; then
        echo "gwtm: $name is on origin ($ahead of its own) - merging, not rewriting" >&2
        op=(merge --no-edit --autostash origin/main)
    else
        echo "gwtm: $name has $ahead commits of its own - rebasing onto origin/main" >&2
        op=(rebase --autostash origin/main)
    fi
    git "${op[@]}" && return
    git "${op[0]}" --abort 2>/dev/null
    echo "gwtm: conflicts with origin/main - $name left as it was." >&2
    echo "gwtm: resolve by hand with: git ${op[0]} origin/main" >&2
    return 1
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
        # Already inside tmux: create-if-missing, then switch (never nest).
        tmux new-session -d -s "$name" 2>/dev/null
        tmux switch-client -t "$name"
    elif tmux ls >/dev/null 2>&1; then
        # Server already running: attach, creating the session if needed.
        tmux new-session -A -s "$name"
    else
        # Cold start (post-reboot): let tmux-continuum auto-restore populate the
        # server first. Boot a throwaway 2-pane session so tmux-resurrect's
        # "restore from scratch" (which fires only when the whole server has exactly
        # one pane) does NOT absorb our launch pane into a restored session and
        # scramble its layout. Then drop the scratch and attach to the target.
        tmux new-session -d -s _resurrect_boot -c "$HOME"
        tmux split-window -t _resurrect_boot -c "$HOME"
        for _ in $(seq 1 60); do
            tmux has-session -t "$name" 2>/dev/null && break
            sleep 0.25
        done
        # Restore succeeded if any real session now exists; if so, drop the scratch.
        if tmux ls -F '#{session_name}' 2>/dev/null | grep -qvx _resurrect_boot; then
            tmux kill-session -t _resurrect_boot 2>/dev/null
        fi
        tmux attach -t "$name" 2>/dev/null ||
            tmux attach 2>/dev/null ||
            tmux new-session -A -s "$name"
    fi
}

# Quick reference
alias cheat='bat ~/dotfiles/CHEATSHEET.md'
