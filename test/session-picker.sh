#!/usr/bin/env bash
# Guards the session picker's list: it must render the agent bar's own order and
# bands, never an order of its own. The regression is silent - a picker that
# falls back to alphabetical still looks like a session list, and the Alt-h /
# Alt-l keys walking a different order than the popup shows is exactly the
# jarring jump the bands removed.
#
# The picker and the binary both call bare `tmux`, so a PATH shim points them at
# a private socket - never the live server. A copy of `sleep` named `claude`
# reports as an agent through pane_current_command, which is all the snapshot
# keys on; XDG_STATE_HOME is redirected so the pin mirror stays in the tempdir.
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PICKER="$REPO/tmux/.local/bin/tmux-session-picker.sh"
BIN="$REPO/apps/agentbar/bin/agentbar"
BAND_MARK='__band__'
pass=0 fail=0

ok() {
    pass=$((pass + 1))
    printf '  \033[32m✓\033[0m %s\n' "$1"
}
no() {
    fail=$((fail + 1))
    printf '  \033[31m✗\033[0m %s\n' "$1"
    [ $# -gt 1 ] && printf '      %s\n' "$2"
}

# eq <name> <want> <got>
eq() { [ "$2" = "$3" ] && ok "$1" || no "$1" "want [$2] got [$3]"; }

if [ ! -x "$BIN" ]; then
    printf 'picker: %s not built, run `task agentbar:build`\n' "$BIN" >&2
    exit 1
fi

TMP=$(mktemp -d)
export PATH="$TMP/shim:$PATH"
export XDG_STATE_HOME="$TMP/state"
export AGENTBAR_BIN="$BIN"
export DOTFILES_TRACE=0
export LC_ALL=C.UTF-8

mkdir -p "$TMP/shim" "$TMP/state"
cat >"$TMP/shim/tmux" <<EOF
#!/bin/sh
exec $(command -v tmux) -S $TMP/sock -f /dev/null "\$@"
EOF
chmod +x "$TMP/shim/tmux"
cp /usr/bin/sleep "$TMP/shim/claude"

cleanup() {
    tmux kill-server 2>/dev/null
    rm -rf "$TMP"
}
trap cleanup EXIT

# session <name> [agent] - a detached session, optionally holding an agent pane
session() {
    tmux new-session -d -s "$1" -x 200 -y 40
    [ "${2:-}" = agent ] || return 0
    local pane
    pane=$(tmux split-window -d -t "$1" -P -F '#{pane_id}' "claude 60")
    printf '{"hook_event_name":"SessionStart","session_id":"t"}' |
        TMUX_PANE="$pane" "$BIN" hook
}

# names / bands - field 1 / the band column of the picker's list, one per line
names() { "$PICKER" --list | cut -f1; }
bands() { "$PICKER" --list | cut -f2 | sed 's/\x1b\[[0-9;]*m//g'; }

printf '\npicker: order and bands mirror the agent bar\n'

tmux kill-server 2>/dev/null
session api agent
session blog agent
session dotfiles agent
session payments # no agent: dormant
"$BIN" pin blog >/dev/null
"$BIN" pin dotfiles >/dev/null

# Alphabetically this list would be api, blog, dotfiles, payments.
eq "sessions follow the bands, not the alphabet" \
    "$BAND_MARK blog dotfiles $BAND_MARK api $BAND_MARK payments" \
    "$(names | tr '\n' ' ' | sed 's/ $//')"

eq "picker order matches agentbar order exactly" \
    "$("$BIN" order | cut -f2 | tr '\n' ' ')" \
    "$(names | grep -Fxv "$BAND_MARK" | tr '\n' ' ')"

# All three named: an unlabelled middle band left you counting rows to see
# where "the rest" ended.
eq "every band is labelled and counted" \
    "pinned ·2|active ·1|dormant ·1" \
    "$(bands | grep -oE '(pinned|active|dormant) ·[0-9]+' | paste -sd'|')"

printf '\npicker: headers only when they divide something\n'

tmux kill-server 2>/dev/null
session solo agent
eq "a single band shows no header rows" "solo" "$(names | tr '\n' ' ' | sed 's/ $//')"

printf '\npicker: p pins through the binary and follows the row\n'

tmux kill-server 2>/dev/null
session api agent
session payments
# payments is dormant and last; pinning floats it to row 2, under the header.
eq "pin prints the fzf actions that redraw and follow" \
    "reload-sync+pos(2)" \
    "$("$PICKER" --pin payments | sed -E 's/reload-sync\([^)]*\)/reload-sync/')"
eq "the pin reached the shared set" "payments" "$("$BIN" order | head -1 | cut -f2)"
eq "pin is a toggle" "" "$("$PICKER" --pin payments >/dev/null; "$BIN" order | grep -c '^pinned' | sed 's/^0$//')"
eq "a band header row is never pinned" "" "$("$PICKER" --pin "$BAND_MARK")"

printf '\npicker: no binary means no crash\n'

eq "falls back to a flat alphabetical list" "api payments" \
    "$(AGENTBAR_BIN=/nonexistent "$PICKER" --list | cut -f1 | tr '\n' ' ' | sed 's/ $//')"

printf '\npicker: %d passed, %d failed\n\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
