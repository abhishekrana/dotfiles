#!/usr/bin/env bash
# Claude Code status line, two rows: where you are, then how you are doing.
#
#     alpha-4 ⎇ feature · ⚠ work ⎇ main
#     Opus 5 1M · ctx 24% · 5h 23% ↻2h14 · 7d 41%
#
# The second place is recorded by statusline-workdir.sh, this package's own
# PostToolUse hook: an Edit by absolute path moves neither the cwd nor this row.
#
# The rate limits arrive on stdin, so they cost no process and no network. Each
# window is absent until the session's first API response and after its reset,
# and a window that is absent shows nothing.
set -u

here=${BASH_SOURCE[0]%/*}
[ "$here" = "${BASH_SOURCE[0]}" ] && here=.
# shellcheck source=statusline-git.bash
. "$here/statusline-git.bash"

input=$(cat)
# US, not tab: bash folds runs of IFS whitespace, so an absent field would shift
# every field after it.
IFS=$'\x1f' read -r model used dir sid five five_at seven seven_at <<<"$(jq -r '[
    .model.display_name // "",
    (.context_window.used_percentage // ""),
    (.workspace.current_dir // .cwd // ""),
    (.session_id // ""),
    (.rate_limits.five_hour.used_percentage // ""),
    (.rate_limits.five_hour.resets_at // ""),
    (.rate_limits.seven_day.used_percentage // ""),
    (.rate_limits.seven_day.resets_at // "")
] | map(tostring) | join("")' <<<"$input")"

warn=$'\033[33m'
hot=$'\033[31m'
reset=$'\033[0m'

# <name> ⎇ <branch>, the pane rail's words; the bare directory when not a checkout.
place() {
    local name
    git_place "$1" || {
        printf '%s' "${1##*/}"
        return
    }
    # shellcheck disable=SC2154  # set by git_place
    name=${place_root##*/}
    [ -n "$place_branch" ] || {
        printf '%s' "$name"
        return
    }
    printf '%s ⎇ %s' "$name" "$place_branch"
}

# Time left until epoch $1, as 3d, 1h05 or 38m; nothing once the window has passed.
until_reset() {
    local left=$(($1 - $(date +%s)))
    ((left <= 0)) && return
    if ((left >= 86400)); then
        printf '%dd' $((left / 86400))
    elif ((left >= 3600)); then
        printf '%dh%02d' $((left / 3600)) $(((left % 3600) / 60))
    else
        printf '%dm' $((left / 60))
    fi
}

# <label> <pct>%, amber past 80 and red past 95. The countdown shows once the window
# is hot, or always with a fourth argument - the 5h window turns over inside a session,
# where days until the weekly reset change no decision.
window() {
    local label=$1 pct=$2 at=$3 always=${4-} n color='' tail='' left
    [ -n "$pct" ] || return
    n=$(printf '%.0f' "$pct")
    if ((n >= 95)); then
        color=$hot
    elif ((n >= 80)); then
        color=$warn
    fi
    if [ -n "$at" ] && { [ -n "$color" ] || [ -n "$always" ]; }; then
        left=$(until_reset "$at")
        [ -n "$left" ] && tail=" ↻$left"
    fi
    printf '%s%s %s%%%s%s' "$color" "$label" "$n" "$tail" "${color:+$reset}"
}

# Where you are, and the worktree Claude last wrote in when it is a different
# checkout - not the cwd, which an Edit by absolute path never moves.
row=()
[ -n "$dir" ] && row+=("$(place "$dir")")

agent_dir=
state=${XDG_STATE_HOME:-$HOME/.local/state}/dotfiles/claude-workdir/$sid
[ -n "$sid" ] && [ -r "$state" ] && agent_dir=$(<"$state")

session_root=
git_place "$dir" && session_root=$place_root
agent_root=
# Roots, not paths: a subdirectory of the session's own checkout is not a move.
[ -n "$agent_dir" ] && git_place "$agent_dir" && agent_root=$place_root
if [ -n "$agent_root" ] && [ "$agent_root" != "$session_root" ]; then
    row+=("${warn}⚠ $(place "$agent_root")${reset}")
fi

# How you are doing. The model keeps its size but not the word around it.
meters=()
[ -n "$model" ] && meters+=("$(sed 's/ (\([0-9]\+[kM]\) context)/ \1/' <<<"$model")")
[ -n "$used" ] && meters+=("ctx $(printf '%.0f' "$used")%")
[ -n "$five" ] && meters+=("$(window 5h "$five" "$five_at" always)")
[ -n "$seven" ] && meters+=("$(window 7d "$seven" "$seven_at")")

first=$(
    IFS='|'
    echo "${row[*]}" | sed 's/|/ · /g'
)
second=$(
    IFS='|'
    echo "${meters[*]}" | sed 's/|/ · /g'
)

printf '%s' "$first"
[ -n "$second" ] && printf '\n%s' "$second"
