#!/usr/bin/env bash
# Settings dialogue for the footer's ⛭ chip: categories on the left, the
# highlighted category's choices on the right. Enter opens a category, then Enter
# on a choice applies it; h goes back.
#
# Enter always runs the row's action and reloads the list, so the script decides
# what a row means and no fzf `transform` is involved. The level on screen lives
# in $SETTINGS_STATE, inherited by every fzf child.
#
# Adding a category: a row in list_root, an arm in list_level, one in do_choices.
#
# Subcommands (invoked by fzf callbacks):
#   --list             rows for the current level
#   --enter  ACTION    run one row's action
#   --choices ACTION   the right-hand pane
set -uo pipefail

SELF=$(realpath "${BASH_SOURCE[0]}")
STATE="${XDG_CONFIG_HOME:-$HOME/.config}/theme"
FLAVORS="solarized-light solarized-dark catppuccin-latte catppuccin-mocha"

# Themed fzf colors. _fzf_color is the name that file actually defines.
# shellcheck source=/dev/null
[ -f "$STATE/fzf.sh" ] && . "$STATE/fzf.sh"

current_flavor() { cat "$STATE/current" 2>/dev/null || echo solarized-light; }
level() { cat "${SETTINGS_STATE:-/dev/null}" 2>/dev/null; }
title() { echo "$1" | tr '-' ' ' | sed 's/\b\(.\)/\u\1/g'; }

# A choice line, marked when it is the one in effect.
choice() {
    if [ "$1" = "$2" ]; then
        printf '  \033[32m●\033[0m %s\n' "$(title "$1")"
    else
        printf '    %s\n' "$(title "$1")"
    fi
}

# ---- rows: "<action>TAB<display>", fzf shows field 2 and binds read field 1 --
list_root() {
    printf 'open:theme\t  %-12s %s  ›\n' "Theme" "$(title "$(current_flavor)")"
}

list_level() {
    local cur f
    case "$(level)" in
        theme)
            printf 'back\t  \033[2m‹ back\033[0m\n'
            cur=$(current_flavor)
            for f in $FLAVORS; do
                printf 'theme:%s\t%s\n' "$f" "$(choice "$f" "$cur")"
            done
            ;;
        *) list_root ;;
    esac
}

do_enter() {
    case "${1:-}" in
        open:*) printf '%s' "${1#open:}" >"${SETTINGS_STATE:?}" ;;
        back) : >"${SETTINGS_STATE:?}" ;;
        theme:*) "$HOME/.local/bin/theme" "${1#theme:}" >/dev/null 2>&1 ;;
    esac
}

# The right-hand pane: what the highlighted category holds.
do_choices() {
    local cur f
    case "${1:-}" in
        open:theme)
            cur=$(current_flavor)
            for f in $FLAVORS; do choice "$f" "$cur"; done
            ;;
    esac
}

case "${1:-}" in
    --list)
        list_level
        exit 0
        ;;
    --enter)
        do_enter "${2:-}"
        exit 0
        ;;
    --choices)
        do_choices "${2:-}"
        exit 0
        ;;
esac

# ---- fzf -------------------------------------------------------------------
SETTINGS_STATE=$(mktemp)
export SETTINGS_STATE
trap 'rm -f "$SETTINGS_STATE"' EXIT

# A bind string must stay on one line: a newline in it makes fzf read the tail as
# an action name.
step="execute-silent($SELF --enter {1})+reload($SELF --list)"

# --height/--border explicit: FZF_DEFAULT_OPTS comes from the server and would
# draw a second frame inside tmux's own and leave a dead band below.
# shellcheck disable=SC2086  # _fzf_color is a word list of --color flags
"$SELF" --list |
    fzf --ansi --sync --reverse --no-input --highlight-line \
        --height=100% --border=none \
        --delimiter=$'\t' --with-nth=2 \
        --pointer='▸' \
        --header='↵ open · h back · q close' --header-first \
        --preview "$SELF --choices {1}" \
        --preview-window='right:50%:border-left' \
        ${_fzf_color:-} \
        --bind "enter:$step" \
        --bind "h:execute-silent($SELF --enter back)+reload($SELF --list)" \
        --bind 'j:down,k:up' \
        --bind 'q:abort,esc:abort' \
        >/dev/null || true
