#!/usr/bin/env bash
# Restart one session's sidebar in place, so a rebuilt binary takes effect.
# respawn-pane keeps the pane id and the layout, so @sidebar_* stay valid; a
# session without a sidebar gets one. Per-session on purpose: restarting every
# sidebar at once storms this tmux+Ghostty client (see the dotfiles CLAUDE.md).
set -euo pipefail

PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="$PLUGIN_DIR/bin/agentbar"

session=${1:?usage: restart.sh <session>}
[ -x "$BIN" ] || exit 0

pane=$(tmux list-panes -s -t "$session" -F '#{pane_id} #{pane_current_command}' 2>/dev/null |
    awk '$2 == "agentbar" {print $1; exit}') || pane=""
if [ -z "$pane" ]; then
    exec "$PLUGIN_DIR/scripts/open.sh" "$session"
fi

theme=$(tmux show-option -gqv @agentbar-theme)
theme=${theme:-solarized-light}
tmux respawn-pane -k -t "$pane" "$BIN run --theme $theme"
"$HOME/.local/bin/dotfiles-trace" log sidebar restart session="$session" pane="$pane" 2>/dev/null || true
