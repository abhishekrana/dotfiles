#!/usr/bin/env bash
# Guards the theme switcher: for every flavor, each generated per-tool file must
# carry THAT flavor's colors from design/palette.toml.
#
# Every theme bug found by hand so far was one file left on the previous flavor -
# a stale popup ground, a stale env var - and each needed a screenshot to spot.
# This asserts all of them at once.
#
# Runs against a temp XDG_CONFIG_HOME, and with tmux and pkill hidden behind
# no-op stubs so `theme` only writes files: no live server is touched, no ghostty
# is signalled, no sidebar or diff pane is respawned.
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
THEME="$REPO/theme/.local/bin/theme"
PALETTE="$REPO/design/palette.toml"
FLAVORS="solarized-light solarized-dark catppuccin-latte catppuccin-mocha"
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

# has <name> <file> <needle>
has() {
    if [ ! -f "$2" ]; then
        no "$1" "no such file: $2"
    elif grep -qiF -- "$3" "$2"; then
        ok "$1"
    else
        no "$1" "expected $3 in $(basename "$2")"
    fi
}

# hasnt <name> <file> <needle> - a file must NOT carry another flavor's color
hasnt() {
    if [ ! -f "$2" ]; then
        no "$1" "no such file: $2"
    elif grep -qiF -- "$3" "$2"; then
        no "$1" "$(basename "$2") still carries $3"
    else
        ok "$1"
    fi
}

pget() { # pget <flavor> <key>
    awk -v w="[themes.$1]" -v k="$2" '
        /^\[/ { s = ($0 == w) }
        s && $1 == k { gsub(/"/, "", $3); print $3; exit }
    ' "$PALETTE"
}

# Stubs: `theme` guards its live steps on `command -v tmux`, so hiding tmux keeps
# this to file generation. pkill would reload the developer's own ghostty.
STUB=$(mktemp -d)
printf '#!/bin/sh\nexit 1\n' >"$STUB/pkill"
chmod +x "$STUB/pkill"
trap 'rm -rf "$STUB" "$XDG_CONFIG_HOME"' EXIT

for flavor in $FLAVORS; do
    XDG_CONFIG_HOME=$(mktemp -d)
    export XDG_CONFIG_HOME
    bg=$(pget "$flavor" bg)
    accent=$(pget "$flavor" accent)
    fg=$(pget "$flavor" fg)
    mode=$(pget "$flavor" mode)
    add=$(pget "$flavor" add)
    remove=$(pget "$flavor" remove)

    printf '\n%s\n' "$flavor  (bg $bg, accent $accent)"

    # A PATH without tmux, so no live server is touched.
    PATH="$STUB:/usr/bin:/bin" THEME_PALETTE="$PALETTE" \
        "$THEME" "$flavor" >/dev/null 2>&1

    S="$XDG_CONFIG_HOME/theme"
    eq_flavor=$(cat "$S/current" 2>/dev/null)
    [ "$eq_flavor" = "$flavor" ] && ok "current" || no "current" "got [$eq_flavor]"

    has "tmux frame carries the bg" "$S/tmux.conf" "$bg"
    has "tmux frame carries the accent" "$S/tmux.conf" "$accent"
    has "shell fzf carries the bg" "$S/fzf.sh" "$bg"
    has "shell fzf carries the mode" "$S/fzf.sh" "--color=$mode"
    has "shell fzf carries the fg" "$S/fzf.sh" "fg:$fg"
    # The popup ground is the one that silently kept the old flavor: a popup
    # inherits FZF_DEFAULT_OPTS from the tmux server, frozen at server start.
    has "popup palette carries its own bg" "$S/agent-state.sh" "bg:$bg"
    has "popup palette carries its own mode" "$S/agent-state.sh" "_popup_fzf_color=\"$mode,"
    # delta tints diffs and never paints a ground, so add/remove are its colors.
    has "delta carries the add tint" "$S/delta.gitconfig" "$add"
    has "delta carries the remove tint" "$S/delta.gitconfig" "$remove"
    has "nvim carries the background" "$S/nvim.lua" "$mode"
    has "env exports the flavor" "$S/env.sh" "THEME=\"$flavor\""

    # No generated file may still carry another flavor's ground.
    for other in $FLAVORS; do
        [ "$other" = "$flavor" ] && continue
        obg=$(pget "$other" bg)
        [ "$obg" = "$bg" ] && continue
        hasnt "no $other bg in the popup palette" "$S/agent-state.sh" "bg:$obg"
        hasnt "no $other bg in the tmux frame" "$S/tmux.conf" "$obg"
    done
done

printf '\n%d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
