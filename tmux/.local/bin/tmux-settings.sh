#!/usr/bin/env bash
# Settings dialogue for the footer's ⛭ chip: setting names down the left, their
# values down the right, every value on its own row. Click a value or press Enter
# on it to apply - there is no submenu and nothing to open, because every value is
# already on screen.
#
# fzf has one cursor and its preview pane cannot be focused, so a dialogue with a
# cursor on each side would need a TUI of its own. Showing every value instead
# makes two cursors unnecessary.
#
# Adding a setting: a group() call in list_rows and an arm in do_apply.
#
# Subcommands (invoked by fzf callbacks):
#   --list           the rows
#   --apply ACTION   apply one row
set -uo pipefail

SELF=$(realpath "${BASH_SOURCE[0]}")
STATE="${XDG_CONFIG_HOME:-$HOME/.config}/theme"
# Absolute, because a tmux popup inherits the server's minimal PATH. Overridable
# so the interaction test can stand a recorder in for the real switcher.
THEME_BIN="${THEME_BIN:-$HOME/.local/bin/theme}"
FLAVORS="solarized-light solarized-dark catppuccin-latte catppuccin-mocha"
LABELS="branch name"
# The cursor must land back on the row just applied, so the dialogue never jumps
# between groups. fzf's transform bind runs after the reload, when {1} is already
# the first row again - so --apply leaves its group name here for --pos to read.
LAST="${XDG_RUNTIME_DIR:-${TMPDIR:-/tmp}}/dotfiles-settings-last-$UID"

# Popup palette, the same one the session and worktree pickers use: a solid
# accent selection bar, louder than the shell fzf's.
# shellcheck source=tmux/.local/bin/tmux-agent-state.sh
. "$(dirname "$(realpath "${BASH_SOURCE[0]}")")/tmux-agent-state.sh"

current_flavor() { cat "$STATE/current" 2>/dev/null || echo solarized-light; }
# What heads a sidebar row: Claude's own name for the session, or the git branch.
current_labels() {
    case "$(tmux show-option -gqv @agent_labels 2>/dev/null)" in
        name) echo name ;;
        *) echo branch ;;
    esac
}
title() { echo "$1" | tr '-' ' ' | sed 's/\b\(.\)/\u\1/g'; }

# group <label> <current> <action-prefix> <value>...
# The label prints once, on the group's first row, so it reads as a left column.
group() {
    local label=$1 cur=$2 prefix=$3 v mark
    shift 3
    for v in "$@"; do
        [ "$v" = "$cur" ] && mark='\033[32m●\033[0m' || mark=' '
        printf '%s%s\t  \033[1m%-11s\033[0m \033[2m│\033[0m %b %s\n' \
            "$prefix" "$v" "$label" "$mark" "$(title "$v")"
        label=''
    done
}

# ---- rows: "<action>TAB<display>", fzf shows field 2 and binds read field 1 --
# shellcheck disable=SC2086  # FLAVORS is a word list, one value per row
list_rows() {
    group "Theme" "$(current_flavor)" "theme:" $FLAVORS
    group "Labels" "$(current_labels)" "labels:" $LABELS
}

# fzf's reload resets the cursor to the first row, so after applying we put it back
# on the row that is now active. Printed as one fzf action, for a transform bind.
do_pos() {
    local cur i=1 v group values
    group=$(cat "$LAST" 2>/dev/null || echo theme)
    case $group in
        labels)
            # Rows are contiguous, so the Labels group starts after the flavors.
            # shellcheck disable=SC2086  # word list, deliberately unquoted
            set -- $FLAVORS
            i=$(($# + 1))
            cur=$(current_labels)
            values=$LABELS
            ;;
        *)
            cur=$(current_flavor)
            values=$FLAVORS
            ;;
    esac
    # shellcheck disable=SC2086  # word list, deliberately unquoted
    for v in $values; do
        [ "$v" = "$cur" ] && break
        i=$((i + 1))
    done
    printf 'pos(%d)' "$i"
}

do_apply() {
    printf '%s' "${1%%:*}" >"$LAST" 2>/dev/null || true
    case "${1:-}" in
        theme:*) "$THEME_BIN" "${1#theme:}" >/dev/null 2>&1 ;;
        # Global and re-read every poll, so every sidebar re-labels itself on its
        # next tick - nothing restarts and no row moves.
        labels:*) tmux set-option -g @agent_labels "${1#labels:}" 2>/dev/null || true ;;
    esac
}

case "${1:-}" in
    --list)
        list_rows
        exit 0
        ;;
    --apply)
        do_apply "${2:-}"
        exit 0
        ;;
    --pos)
        do_pos
        exit 0
        ;;
esac

# ---- fzf -------------------------------------------------------------------
# A bind string must stay on one line: a newline in it makes fzf read the tail as
# an action name.
step="execute-silent($SELF --apply {1})+reload-sync($SELF --list)+transform($SELF --pos)"

# Mouse: a click is a pick, same as Enter. double-click has to be bound too - its
# default is accept, which would close the dialogue on the second click.
#
# --height/--border explicit: FZF_DEFAULT_OPTS comes from the server and would
# draw a second frame inside tmux's own and leave a dead band below.
"$SELF" --list |
    fzf --ansi --sync --no-input --highlight-line --reverse \
        --height=100% --border=none --no-preview \
        --delimiter=$'\t' --with-nth=2 \
        --pointer='▸' \
        --header='click or ↵ to apply · q close' --header-first \
        --color="${_popup_fzf_color:-}" \
        --bind "enter:$step" \
        --bind "left-click:$step" \
        --bind "double-click:$step" \
        --bind 'j:down,k:up' \
        --bind 'q:abort,esc:abort' \
        >/dev/null || true
