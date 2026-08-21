#!/usr/bin/env bash
# Guards the ⛭ settings dialogue's interaction, which the theme gate does not
# reach: it checks the files a switch writes, not the dialogue that triggers one.
#
# The regression that prompted this was silent in the worst way - the theme
# applied correctly while the cursor jumped to another row, so the click looked
# like it had gone somewhere else. fzf's reload resets the cursor, and a plain
# (async) reload lands a position fix against the list still on screen.
#
# `theme` is stubbed: it records the flavor it was asked for and updates
# theme/current, which is all the dialogue reads. So the assertions are exact and
# nothing is re-skinned, respawned or signalled. tmux runs on a private socket.
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIALOGUE="$REPO/tmux/.local/bin/tmux-settings.sh"
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
eq() { [ "$2" = "$3" ] && ok "$1" || no "$1" "want [$2] got [$3]"; }

TMP=$(mktemp -d)
TMUX_BIN=$(command -v tmux)
export PATH="$TMP/shim:$PATH"
export XDG_CONFIG_HOME="$TMP/config"
export DOTFILES_TRACE=0
export LC_ALL=C.UTF-8
mkdir -p "$TMP/shim" "$XDG_CONFIG_HOME/theme"

# Every bare `tmux` lands on the private socket, never the live server.
cat >"$TMP/shim/tmux" <<EOF
#!/bin/sh
exec $TMUX_BIN -S $TMP/sock -f /dev/null "\$@"
EOF
# The stub stands in for the real switcher: record the ask, move `current`.
cat >"$TMP/shim/theme" <<EOF
#!/bin/sh
printf '%s\n' "\$1" >>$TMP/applied
printf '%s' "\$1" >$XDG_CONFIG_HOME/theme/current
EOF
chmod +x "$TMP/shim/tmux" "$TMP/shim/theme"

cleanup() {
    tmux kill-server 2>/dev/null
    rm -rf "$TMP"
}
trap cleanup EXIT

# ---- helpers ---------------------------------------------------------------
# start <flavor> - a fresh dialogue with <flavor> active.
start() {
    tmux kill-server 2>/dev/null
    : >"$TMP/applied"
    rm -f "${XDG_RUNTIME_DIR:-${TMPDIR:-/tmp}}/dotfiles-settings-last-$UID"
    printf '%s' "$1" >"$XDG_CONFIG_HOME/theme/current"
    # THEME_BIN points the dialogue at the recorder instead of the real switcher.
    tmux new-session -d -s ui -x 80 -y 12 \
        "PATH=$TMP/shim:\$PATH THEME_BIN=$TMP/shim/theme $DIALOGUE"
    sleep 2
}

pane() { tmux list-panes -a -F '#{pane_id}' 2>/dev/null | head -1; }
snap() { tmux capture-pane -p -t "$(pane)" 2>/dev/null; }

# row <glyph> - which Theme row (1-based in that group) carries the glyph, or 0.
row() { snap | sed -n '2,5p' | grep -n -- "$1" | head -1 | cut -d: -f1; }

# lrow <glyph> - the same for the Labels group, which follows the four flavors.
lrow() { snap | sed -n '6,7p' | grep -n -- "$1" | head -1 | cut -d: -f1; }

keys() {
    tmux send-keys -t "$(pane)" "$@"
    sleep 1
}

# click <item> - a left click on that list item (item 1 is pane row 2).
click() {
    local r=$(($1 + 1)) p
    p=$(pane)
    tmux send-keys -t "$p" -l "$(printf '\033[<0;30;%dM' "$r")"
    tmux send-keys -t "$p" -l "$(printf '\033[<0;30;%dm' "$r")"
    sleep 3
}

applied() { tr -d '\n' <"$TMP/applied"; }

# ---- helpers per group -----------------------------------------------------
# Rows are contiguous and grouped by area, so each group is a span of pane lines
# (list item N is pane line N+1). A per-group lookup keeps every assertion local:
# there are now three ● marks on screen, one per group.
in_span() { snap | sed -n "$2,${3}p" | grep -n -- "$1" | head -1 | cut -d: -f1; }
hrow() { in_span "$1" 2 3; } # agentbar · Headline
nrow() { in_span "$1" 4 5; } # agentbar · Notify
trow() { in_span "$1" 6 9; } # theme · Theme

# ---- the list ---------------------------------------------------------------
printf '\nlist\n'
start solarized-light
out=$(snap)
for want in agentbar theme Headline Notify Theme; do
    grep -qF "$want" <<<"$out" && ok "names $want" || no "missing $want"
done
for want in Branch Title Off On "Solarized Light" "Catppuccin Mocha"; do
    grep -qF "$want" <<<"$out" && ok "shows $want" || no "missing $want"
done
eq "the headline defaults to the title" 1 "$(hrow '●')"
eq "notify defaults to off" 1 "$(nrow '●')"
eq "the active flavor is marked" 1 "$(trow '●')"
eq "the cursor starts on the first row" 1 "$(hrow '▸')"

# ---- keyboard ---------------------------------------------------------------
printf '\nkeyboard\n'
keys j
eq "j moves down" 2 "$(hrow '▸')"
keys Enter
eq "Enter applies the headline" branch "$(tmux show-option -gqv @agentbar-headline)"
eq "the marker follows the choice" 2 "$(hrow '●')"
eq "and the cursor stays in its group" 2 "$(hrow '▸')"
eq "the theme was left alone" "" "$(applied)"

keys j
keys j
eq "j walks into the next setting" 2 "$(nrow '▸')"
keys Enter
eq "Enter applies notify" on "$(tmux show-option -gqv @agent_notify)"
eq "and the cursor stays there" 2 "$(nrow '▸')"
eq "the headline was left alone" branch "$(tmux show-option -gqv @agentbar-headline)"

keys k
keys Enter
eq "and back off again" off "$(tmux show-option -gqv @agent_notify)"

# ---- mouse ------------------------------------------------------------------
printf '\nmouse\n'
start solarized-light
click 8
eq "a click applies that flavor" "catppuccin-mocha" "$(applied)"
eq "the marker moved to it" 4 "$(trow '●')"
eq "the cursor stayed on it" 4 "$(trow '▸')"

click 2
eq "a click in another area applies there" branch "$(tmux show-option -gqv @agentbar-headline)"
eq "cursor on the clicked row" 2 "$(hrow '▸')"
eq "without re-applying the theme" "catppuccin-mocha" "$(applied)"

# ---- closing ----------------------------------------------------------------
printf '\nclose\n'
start solarized-light
keys q
sleep 1
if tmux has-session -t ui 2>/dev/null; then
    no "q closes the dialogue" "session still alive"
else
    ok "q closes the dialogue"
fi

printf '\n%d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
