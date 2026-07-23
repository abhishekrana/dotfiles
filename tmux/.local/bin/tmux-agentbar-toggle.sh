#!/bin/bash
# tmux-agentbar-toggle.sh -- toggle the agent sidebar for the CURRENT session only.
#
# The "☰ agents" status chip calls this. Unlike prefix+e (agentbar's own global
# toggle.sh, which opens/closes a sidebar in EVERY session at once -- 13 spinners
# redrawing together can thrash the client), this touches just the session you're
# looking at: one split, one agentbar process. Open path reuses agentbar's own
# open.sh; close mirrors toggle.sh's close_in for a single session.

set -u

PLUGIN_DIR="$HOME/dotfiles/apps/agentbar"
TAB=$(printf '\t')

# Session shown by the most-recently-active attached client (mirrors dictate/diff).
active_session() {
  tmux list-clients -F "#{client_activity}${TAB}#{client_session}" 2>/dev/null \
    | sort -rn | head -1 | cut -f2-
}

sess=$(active_session)
[ -n "$sess" ] || sess=$(tmux display-message -p '#{session_name}' 2>/dev/null)
[ -n "$sess" ] || exit 0

# Any live sidebar pane in this session? (tracked or a resurrect orphan.)
alive=$(tmux list-panes -s -t "$sess" -F '#{pane_id} #{pane_current_command}' 2>/dev/null \
        | awk '$2 == "agentbar" {print $1}')

if [ -n "$alive" ]; then
  for id in $alive; do tmux kill-pane -t "$id" 2>/dev/null; done
  tmux set-option -t "$sess" -uq @sidebar_pane
  tmux set-option -t "$sess" -uq @sidebar_on
  tmux set-option -t "$sess" -uq @sidebar_moving
  tmux set-hook -u -t "$sess" session-window-changed 2>/dev/null || true
  "$HOME/.local/bin/dotfiles-trace" log tmux agentbar mode=close-one session="$sess" 2>/dev/null || true
else
  "$PLUGIN_DIR/scripts/open.sh" "$sess"
  "$HOME/.local/bin/dotfiles-trace" log tmux agentbar mode=open-one session="$sess" 2>/dev/null || true
fi
