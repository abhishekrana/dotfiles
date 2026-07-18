#!/usr/bin/env bash
# Open the sidebar in one session (no-op if it already has a live one).
# Used by toggle.sh for every session and by the session-created hook
# for sessions born while the sidebar is globally on.
set -euo pipefail

PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="$PLUGIN_DIR/bin/tmux-agent-sidebar"
. "$PLUGIN_DIR/scripts/common.sh"

session=${1:?usage: open.sh <session>}

width=$(tmux show-option -gqv @agent-sidebar-width)
width=${width:-30}
theme=$(tmux show-option -gqv @agent-sidebar-theme)
theme=${theme:-solarized-light}

panes=$(tmux list-panes -s -t "$session" -F '#{pane_id} #{pane_current_command}' 2>/dev/null) || exit 0
# Adopt any pane already running the sidebar (ours or a resurrect orphan).
alive=$(awk '$2 == "tmux-agent-sidebar" {print $1; exit}' <<<"$panes")
if [ -n "$alive" ]; then
    tmux set-option -t "$session" -q @sidebar_pane "$alive"
    tmux set-option -t "$session" -q @sidebar_on 1
    tmux set-hook -t "$session" session-window-changed \
        "run-shell '$PLUGIN_DIR/scripts/follow.sh #{session_name}'"
    exit 0
fi

# -d: don't steal focus or the window's automatic-rename.
active=$(tmux display-message -p -t "$session" '#{pane_id}')
curwin=$(tmux display-message -p -t "$session" '#{window_id}')
split() { new=$(tmux split-window -dhbf -l "$width" -t "$active" -P -F '#{pane_id}' "$BIN run --theme $theme"); }
insert_keeping_widths "$curwin" "$width" split
tmux set-option -t "$session" -q @sidebar_pane "$new"
tmux set-option -t "$session" -q @sidebar_on 1

focus=$(tmux show-option -gqv @agent-sidebar-focus)
if [ "$focus" = "on" ]; then
    tmux select-pane -t "$new"
fi

tmux set-hook -t "$session" session-window-changed \
    "run-shell '$PLUGIN_DIR/scripts/follow.sh #{session_name}'"
