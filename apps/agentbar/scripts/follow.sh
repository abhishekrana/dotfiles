#!/usr/bin/env bash
# session-window-changed hook handler: keep the sidebar pane in the
# session's active window.
set -euo pipefail

session=${1:?session name}
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

[ "$(tmux show-option -t "$session" -qv @sidebar_on)" = "1" ] || exit 0
pane=$(tmux show-option -t "$session" -qv @sidebar_pane)
[ -n "$pane" ] || exit 0

# Re-entrancy guard: our own join-pane below fires hooks too.
[ "$(tmux show-option -t "$session" -qv @sidebar_moving)" = "1" ] && exit 0

# Self-heal: the sidebar process is gone (killed pane, tmux-resurrect
# corpse). Drop the state and hooks; prefix+e opens a fresh one.
# (grep a variable, not a pipe: grep -q quitting early would SIGPIPE
# tmux and trip pipefail)
panes=$(tmux list-panes -s -t "$session" -F '#{pane_id} #{pane_current_command}' 2>/dev/null) || panes=""
if ! grep -q "^$pane agentbar$" <<<"$panes"; then
    tmux set-option -t "$session" -uq @sidebar_pane
    tmux set-option -t "$session" -uq @sidebar_on
    tmux set-hook -u -t "$session" session-window-changed 2>/dev/null || true
    exit 0
fi

curwin=$(tmux display-message -t "$session" -p '#{window_id}')
sidewin=$(tmux display-message -t "$pane" -p '#{window_id}')
[ "$curwin" = "$sidewin" ] && exit 0

# A window can decline the sidebar. A window opened to hold one transient thing -
# a diff, a pager - dies with the command it was opened for, and following into it
# leaves the sidebar as the last pane standing: the window survives, holding
# nothing but a full-width sidebar, and the thing you closed never gives the
# screen back. Declining keeps the sidebar where it was, so that window empties
# and tmux closes it.
[ "$(tmux show-option -t "$curwin" -wqv @agentbar-skip)" = "1" ] && exit 0

width=$(tmux show-option -gqv @agentbar-width)
width=${width:-30}

# -d: move without stealing focus or the window's automatic-rename.
# Clear the guard on every exit path: a set -e abort mid-move (the window or
# pane vanishing is the race this guards) strands it at 1, and every later hook
# then bails at the re-entrancy check above.
tmux set-option -t "$session" -q @sidebar_moving 1
trap 'tmux set-option -t "$session" -uq @sidebar_moving 2>/dev/null || true' EXIT
join() { tmux join-pane -dhbf -l "$width" -s "$pane" -t "$curwin" 2>/dev/null || true; }
insert_keeping_widths "$curwin" "$width" join
"$HOME/.local/bin/dotfiles-trace" log sidebar follow session="$session" pane="$pane" 2>/dev/null || true
