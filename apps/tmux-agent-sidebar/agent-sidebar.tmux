#!/usr/bin/env bash
# TPM entry point for tmux-agent-sidebar.
#
# Options (set in ~/.tmux.conf before the TPM run line):
#   @agent-sidebar-key     toggle key after prefix       (default: e)
#   @agent-sidebar-width   sidebar width in columns      (default: 30)
#   @agent-sidebar-theme   solarized-light|solarized-dark|catppuccin-latte|catppuccin-mocha
#                          (default: solarized-light)
#   @agent-sidebar-focus   'on' to focus the sidebar when opening
set -euo pipefail

CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="$CURRENT_DIR/bin/tmux-agent-sidebar"

# Build when the binary is missing or older than the source (TPM update).
stale() {
    [ ! -x "$BIN" ] ||
        [ -n "$(find "$CURRENT_DIR/cmd" "$CURRENT_DIR/internal" "$CURRENT_DIR/go.mod" \
            -newer "$BIN" -print -quit 2>/dev/null)" ]
}
if stale && command -v go >/dev/null 2>&1; then
    (cd "$CURRENT_DIR" && go build -o bin/tmux-agent-sidebar ./cmd/tmux-agent-sidebar) ||
        tmux display-message "tmux-agent-sidebar: go build failed"
fi
if [ ! -x "$BIN" ]; then
    tmux display-message "tmux-agent-sidebar: missing binary (install Go and reload)"
    exit 0
fi

key=$(tmux show-option -gqv @agent-sidebar-key)
key=${key:-e}
# Global toggle: opens/closes the sidebar in every session at once.
tmux bind-key "$key" run-shell "$CURRENT_DIR/scripts/toggle.sh"

# tmux-resurrect: stamp the sidebar's restore command into each save
# (see scripts/resurrect-save.sh). Don't clobber a user-set hook.
if [ -z "$(tmux show-option -gqv @resurrect-hook-post-save-layout)" ]; then
    tmux set-option -g @resurrect-hook-post-save-layout "$CURRENT_DIR/scripts/resurrect-save.sh"
fi

# Replace the #{agent_sidebar_status} placeholder in the status line
# with the live segment (standard TPM interpolation pattern).
placeholder="\#{agent_sidebar_status}"
segment="#($BIN status)"
for side in status-left status-right; do
    value=$(tmux show-option -gqv "$side")
    case "$value" in
    *"$placeholder"*)
        tmux set-option -g "$side" "${value//$placeholder/$segment}"
        ;;
    esac
done
