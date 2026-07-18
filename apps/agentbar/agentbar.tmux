#!/usr/bin/env bash
# TPM entry point for agentbar.
#
# Options (set in ~/.tmux.conf before the TPM run line):
#   @agentbar-key     toggle key after prefix       (default: e)
#   @agentbar-width   sidebar width in columns      (default: 30)
#   @agentbar-theme   solarized-light|solarized-dark|catppuccin-latte|catppuccin-mocha
#                          (default: solarized-light)
#   @agentbar-focus   'on' to focus the sidebar when opening
set -euo pipefail

CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="$CURRENT_DIR/bin/agentbar"

# Build when the binary is missing or older than the source (TPM update).
stale() {
    [ ! -x "$BIN" ] ||
        [ -n "$(find "$CURRENT_DIR/cmd" "$CURRENT_DIR/internal" "$CURRENT_DIR/go.mod" \
            -newer "$BIN" -print -quit 2>/dev/null)" ]
}
if stale && command -v go >/dev/null 2>&1; then
    (cd "$CURRENT_DIR" && go build -o bin/agentbar ./cmd/agentbar) ||
        tmux display-message "agentbar: go build failed"
fi
if [ ! -x "$BIN" ]; then
    tmux display-message "agentbar: missing binary (install Go and reload)"
    exit 0
fi

key=$(tmux show-option -gqv @agentbar-key)
key=${key:-e}
# Global toggle: opens/closes the sidebar in every session at once.
tmux bind-key "$key" run-shell "$CURRENT_DIR/scripts/toggle.sh"

# tmux-resurrect: stamp the sidebar's restore command into each save
# (see scripts/resurrect-save.sh). Don't clobber a user-set hook.
if [ -z "$(tmux show-option -gqv @resurrect-hook-post-save-layout)" ]; then
    tmux set-option -g @resurrect-hook-post-save-layout "$CURRENT_DIR/scripts/resurrect-save.sh"
fi

# Replace the #{agentbar_status} placeholder in the status line
# with the live segment (standard TPM interpolation pattern).
placeholder="\#{agentbar_status}"
segment="#($BIN status)"
for side in status-left status-right; do
    value=$(tmux show-option -gqv "$side")
    case "$value" in
    *"$placeholder"*)
        tmux set-option -g "$side" "${value//$placeholder/$segment}"
        ;;
    esac
done
