#!/usr/bin/env bash
# Turn the sidebar on everywhere: open one in each session (adopting any that
# already have one) and install the global session-created hook so sessions
# born later get one too. Idempotent - safe to re-run (open.sh no-ops a
# session that already has a live sidebar).
set -euo pipefail

PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

while IFS= read -r session; do
    "$PLUGIN_DIR/scripts/open.sh" "$session"
done < <(tmux list-sessions -F '#{session_name}' 2>/dev/null)

tmux set-hook -g session-created \
    "run-shell '$PLUGIN_DIR/scripts/open.sh #{hook_session_name}'"
