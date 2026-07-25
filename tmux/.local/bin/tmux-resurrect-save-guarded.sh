#!/usr/bin/env bash
# Guard wrapper around tmux-resurrect's save.sh.
#
# WHY: tmux-resurrect parses its saved state with `IFS=$'\t' read`. Tab is an
# IFS-whitespace character, so bash COLLAPSES empty fields. A pane whose
# #{pane_title} is empty therefore emits two adjacent tabs that get collapsed,
# shifting every later column left by one. On restore the directory column is
# then read from the wrong field and `split-window -c <non-dir>` silently falls
# back to the launch/parent cwd -- panes come back in the wrong directory. The
# mis-read value is also written back as the pane title via `select-pane -T`,
# so corrupted ":/path"-style titles keep re-seeding the bug on every cycle.
#
# FIX: before every save, normalise any empty or ":"-prefixed pane title to a
# safe non-empty value (the short hostname, which is tmux's own default title).
# Wired in via `@resurrect-save-script-path`, so tmux-continuum's autosave and
# the systemd timer both go through it; also bound to the manual `prefix + C-s`.
#
# All arguments (e.g. "quiet") are forwarded verbatim to the real save.sh.
set -u

RESURRECT_SAVE="${HOME}/.tmux/plugins/tmux-resurrect/scripts/save.sh"

# Nothing running -> nothing to save (e.g. timer fired while detached from boot).
if ! tmux list-sessions >/dev/null 2>&1; then
    exit 0
fi

host="$(hostname -s 2>/dev/null || hostname 2>/dev/null || echo tmux)"

# Normalise empty / ":"-prefixed titles across every pane on the server.
while IFS=$'\t' read -r pane_id title; do
    case "$title" in
        "" | :*) tmux select-pane -t "$pane_id" -T "$host" 2>/dev/null ;;
    esac
done < <(tmux list-panes -a -F "#{pane_id}"$'\t'"#{pane_title}" 2>/dev/null)

# Trace manual saves only (prefix + C-s). Continuum autosave and the systemd
# timer pass "quiet" -- they're periodic, not user actions, so skip them.
case "$*" in
    *quiet*) : ;;
    *) "$HOME/.local/bin/dotfiles-trace" log resurrect save trigger=manual 2>/dev/null || true ;;
esac

exec "$RESURRECT_SAVE" "$@"
