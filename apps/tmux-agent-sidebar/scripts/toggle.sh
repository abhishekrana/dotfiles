#!/usr/bin/env bash
# Global sidebar toggle (bound to prefix+e).
#
# If any session has a live sidebar: close them all everywhere.
# Otherwise: open one in every session, and install a global
# session-created hook so sessions born later get one too.
# State is derived from live panes, never from a stored flag, so it
# can't go stale.
set -euo pipefail

PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

close_in() {
    local session=$1 panes id
    # Kill every sidebar pane, tracked or not (resurrect orphans).
    panes=$(tmux list-panes -s -t "$session" -F '#{pane_id} #{pane_current_command}' 2>/dev/null) || panes=""
    for id in $(awk '$2 == "tmux-agent-sidebar" {print $1}' <<<"$panes"); do
        tmux kill-pane -t "$id"
    done
    tmux set-option -t "$session" -uq @sidebar_pane
    tmux set-option -t "$session" -uq @sidebar_on
    tmux set-option -t "$session" -uq @sidebar_moving
    tmux set-hook -u -t "$session" session-window-changed 2>/dev/null || true
}

any_alive() {
    local commands
    commands=$(tmux list-panes -a -F '#{pane_current_command}' 2>/dev/null) || return 1
    grep -qx "tmux-agent-sidebar" <<<"$commands"
}

if any_alive; then
    tmux set-hook -gu session-created 2>/dev/null || true
    tmux set-hook -gu client-session-changed 2>/dev/null || true
    while IFS= read -r session; do
        close_in "$session"
    done < <(tmux list-sessions -F '#{session_name}')
else
    while IFS= read -r session; do
        "$PLUGIN_DIR/scripts/open.sh" "$session"
    done < <(tmux list-sessions -F '#{session_name}')
    tmux set-hook -g session-created \
        "run-shell '$PLUGIN_DIR/scripts/open.sh #{hook_session_name}'"
fi
