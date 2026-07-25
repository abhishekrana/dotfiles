#!/usr/bin/env bash
# Open the sidebar in one session (no-op if it already has a live one).
# Used by toggle.sh for every session and by the session-created hook
# for sessions born while the sidebar is globally on.
set -euo pipefail

PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="$PLUGIN_DIR/bin/agentbar"
. "$PLUGIN_DIR/scripts/common.sh"

session=${1:?usage: open.sh <session>}

width=$(tmux show-option -gqv @agentbar-width)
width=${width:-30}
theme=$(tmux show-option -gqv @agentbar-theme)
theme=${theme:-solarized-light}

panes=$(tmux list-panes -s -t "$session" -F '#{pane_id} #{pane_current_command}' 2>/dev/null) || exit 0
# Adopt any pane already running the sidebar (ours or a resurrect orphan).
alive=$(awk '$2 == "agentbar" {print $1; exit}' <<<"$panes")
if [ -n "$alive" ]; then
    tmux set-option -t "$session" -q @sidebar_pane "$alive"
    tmux set-option -t "$session" -q @sidebar_on 1
    tmux set-hook -t "$session" session-window-changed \
        "run-shell '$PLUGIN_DIR/scripts/follow.sh #{session_name}'"
    "$HOME/.local/bin/dotfiles-trace" log sidebar open session="$session" action=adopt pane="$alive" 2>/dev/null || true
    exit 0
fi

# -d: don't steal focus or the window's automatic-rename.
active=$(tmux display-message -p -t "$session" '#{pane_id}')
curwin=$(tmux display-message -p -t "$session" '#{window_id}')
new=''
split() { new=$(tmux split-window -dhbf -l "$width" -t "$active" -P -F '#{pane_id}' "$BIN run --theme $theme"); }
insert_keeping_widths "$curwin" "$width" split
# A failed split (no room for another pane) leaves $new empty. Leave the
# session unmarked: @sidebar_on with an empty @sidebar_pane wedges follow.sh,
# which bails on the empty pane before its self-heal can clear the flags.
# Exit 0 so on.sh keeps opening the remaining sessions.
if [ -z "$new" ]; then
    "$HOME/.local/bin/dotfiles-trace" log sidebar open session="$session" action=fail why=nosplit 2>/dev/null || true
    exit 0
fi
tmux set-option -t "$session" -q @sidebar_pane "$new"
tmux set-option -t "$session" -q @sidebar_on 1
"$HOME/.local/bin/dotfiles-trace" log sidebar open session="$session" action=spawn pane="$new" 2>/dev/null || true

focus=$(tmux show-option -gqv @agentbar-focus)
if [ "$focus" = "on" ]; then
    tmux select-pane -t "$new"
fi

tmux set-hook -t "$session" session-window-changed \
    "run-shell '$PLUGIN_DIR/scripts/follow.sh #{session_name}'"
