#!/bin/bash
# Copy stdin to both the clipboard and the primary selection.
# Logs the yank edge: bytes tmux handed us plus each clip outcome, so a flaky
# mouse-drag copy can be told apart from an empty selection (bytes=0) or a
# backend that failed (rc). clip logs the backend detail on its own line.
buf=$(cat)
CLIP="$HOME/.local/bin/clip"
rc=0
rc_primary=0
printf '%s' "$buf" | "$CLIP" || rc=$?
printf '%s' "$buf" | "$CLIP" --primary || rc_primary=$?
bytes=$(printf %s "$buf" | wc -c | tr -d '[:space:]')
"$HOME/.local/bin/dotfiles-trace" log yank copy \
    bytes="$bytes" rc="$rc" rc_primary="$rc_primary" 2>/dev/null || true
