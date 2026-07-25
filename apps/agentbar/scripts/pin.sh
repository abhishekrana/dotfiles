#!/usr/bin/env bash
# window-resized hook: put the sidebar back to @agentbar-width. tmux takes a
# shrink evenly from every pane in the row rather than in proportion, so the
# narrow sidebar is the first thing it loses. Width only - the pane beside it
# absorbs those columns, exactly as tmux would have. No-op when already right.
set -euo pipefail

window=${1:?usage: pin.sh <window>}

width=$(tmux show-option -gqv @agentbar-width)
width=${width:-30}

pane=$(tmux list-panes -t "$window" \
    -F '#{pane_id} #{pane_left} #{pane_width} #{pane_current_command}' 2>/dev/null |
    awk -v w="$width" '$4 == "agentbar" && $2 + 0 == 0 && $3 + 0 != w + 0 {print $1; exit}') || exit 0
[ -n "$pane" ] || exit 0

tmux resize-pane -t "$pane" -x "$width"
"$HOME/.local/bin/dotfiles-trace" log sidebar pin window="$window" pane="$pane" w="$width" 2>/dev/null || true
