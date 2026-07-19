#!/usr/bin/env bash
# TPM entry point for agentbar.
#
# Options (set in ~/.tmux.conf before the TPM run line):
#   @agentbar-key     toggle key after prefix       (default: e)
#   @agentbar-width   sidebar width in columns      (default: 30)
#   @agentbar-theme   solarized-light|solarized-dark|catppuccin-latte|catppuccin-mocha
#                          (default: solarized-light)
#   @agentbar-focus   'on' to focus the sidebar when opening
#   @agentbar-autostart 'off' to skip opening the sidebar at server start
#                          (default: on - every session gets one, so do new ones)
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

# Autostart: turn the sidebar on at server start so every session has one and
# every session born later gets one via the session-created hook. Toggle it
# off with prefix+key like normal; set @agentbar-autostart 'off' to opt out.
autostart=$(tmux show-option -gqv @agentbar-autostart)
autostart=${autostart:-on}
if [ "$autostart" = "on" ]; then
    # tmux-continuum recreates saved sessions ~1s after start (a backgrounded
    # restore). Opening a sidebar into a session mid-restore fights the layout
    # restore and races the whitelisted sidebar pane, so suspend auto-open for
    # the duration and re-run it after (on.sh adopts the restored sidebars).
    # Don't clobber a user-set hook.
    if [ -z "$(tmux show-option -gqv @resurrect-hook-pre-restore-all)" ]; then
        tmux set-option -g @resurrect-hook-pre-restore-all "tmux set-hook -gu session-created"
    fi
    if [ -z "$(tmux show-option -gqv @resurrect-hook-post-restore-all)" ]; then
        tmux set-option -g @resurrect-hook-post-restore-all "$CURRENT_DIR/scripts/on.sh"
    fi
    "$CURRENT_DIR/scripts/on.sh"
fi
