#!/usr/bin/env bash
# Claude Code status line: where you are, and how you are doing.
#
#     repo ⎇ feature · ⚠ other ⎇ main
#     ● dictate · Opus 5 1M · ctx 24% · 5h 23% ↻2h14 · 7d 41%
#
# The ⚠ place is the worktree Claude last wrote in, recorded by this package's
# statusline-workdir.sh: an Edit by absolute path moves neither the cwd nor row
# one. Rate limits arrive on stdin; a window absent from it shows nothing.
#
# It re-runs once a second (refreshInterval in settings.json) in every open
# session, so the whole script is one process: helpers write a named global
# rather than print into a `$( )` subshell, and jq reading the payload is the
# only command it runs.
set -u

here=${BASH_SOURCE[0]%/*}
[ "$here" = "${BASH_SOURCE[0]}" ] && here=.
# shellcheck source=statusline-git.bash
. "$here/statusline-git.bash"

# US, not tab: bash folds runs of IFS whitespace, so an absent field would shift
# every field after it. jq reads stdin directly - the payload is never held.
IFS=$'\x1f' read -r model used dir sid five five_at seven seven_at < <(jq -r '[
    .model.display_name // "",
    (.context_window.used_percentage // ""),
    (.workspace.current_dir // .cwd // ""),
    (.session_id // ""),
    (.rate_limits.five_hour.used_percentage // ""),
    (.rate_limits.five_hour.resets_at // ""),
    (.rate_limits.seven_day.used_percentage // ""),
    (.rate_limits.seven_day.resets_at // "")
] | map(tostring) | join("")')

# The terminal's own 16, so every flavor of the palette follows. 96 is base1,
# the muted tone; 90 is base03, which is near-black in a light flavor.
warn=$'\033[33m'
hot=$'\033[31m'
muted=$'\033[96m'
reset=$'\033[0m'

# Parts joined with the row separator, into $join_out.
join_row() {
    local part
    join_out=
    for part in "$@"; do
        [ -n "$part" ] || continue
        join_out+="${join_out:+ · }$part"
    done
}

# <name> ⎇ <branch> into $place_out, the pane rail's words; the bare directory
# when not a checkout.
place() {
    local name
    if ! git_place "$1"; then
        place_out=${1##*/}
        return
    fi
    # shellcheck disable=SC2154  # set by git_place
    name=${place_root##*/}
    place_out=$name
    [ -n "$place_branch" ] && place_out="$name ⎇ $place_branch"
}

# Time left until epoch $1 into $until_out, as 3d, 1h05 or 38m; empty once the
# window has passed.
until_reset() {
    local left=$(($1 - EPOCHSECONDS))
    until_out=
    ((left > 0)) || return 0
    if ((left >= 86400)); then
        printf -v until_out '%dd' $((left / 86400))
    elif ((left >= 3600)); then
        printf -v until_out '%dh%02d' $((left / 3600)) $(((left % 3600) / 60))
    else
        printf -v until_out '%dm' $((left / 60))
    fi
}

# <label> <pct>% into $window_out, amber past 80 and red past 95. The countdown
# shows once the window is hot, or always with a fourth argument - the 5h window
# turns over inside a session, where days until the weekly reset change no decision.
window() {
    local label=$1 pct=$2 at=$3 always=${4-} n color='' tail=''
    window_out=
    [ -n "$pct" ] || return 0
    printf -v n '%.0f' "$pct"
    if ((n >= 95)); then
        color=$hot
    elif ((n >= 80)); then
        color=$warn
    fi
    if [ -n "$at" ] && { [ -n "$color" ] || [ -n "$always" ]; }; then
        until_reset "$at"
        [ -n "$until_out" ] && tail=" ↻$until_out"
    fi
    printf -v window_out '%s%s %s%%%s%s' "$color" "$label" "$n" "$tail" "${color:+$reset}"
}

# Whether the microphone is live, into $dictate_out: the herdr plugin's own state
# file, read and never written, so polling cannot disturb a recording. A pid with
# no process is a recorder that died, not a recording.
#
# Colour is the whole signal - grey idle, red recording, amber transcribing - and
# the label never changes, so the meters beside it never shift. Same rule as the
# tmux footer chip.
DICTATE_STATE=${XDG_STATE_HOME:-$HOME/.local/state}/herdr/plugins/abhishekrana.dictate/recording.json
dictate() {
    local state='' pid='' color=$muted
    [ -r "$DICTATE_STATE" ] && state=$(<"$DICTATE_STATE")
    [[ $state =~ \"pid\":([0-9]+) ]] && pid=${BASH_REMATCH[1]}
    if [ -n "$pid" ] && [ -d "/proc/$pid" ]; then
        color=$hot
        [[ $state == *'"phase":"transcribing"'* ]] && color=$warn
    fi
    dictate_out="${color}● dictate${reset}"
}

# Where you are, and the worktree Claude last wrote in when it is a different
# checkout - not the cwd, which an Edit by absolute path never moves.
row=()
if [ -n "$dir" ]; then
    place "$dir"
    row+=("$place_out")
fi

agent_dir=
state=${XDG_STATE_HOME:-$HOME/.local/state}/dotfiles/claude-workdir/$sid
[ -n "$sid" ] && [ -r "$state" ] && agent_dir=$(<"$state")

session_root=
git_place "$dir" && session_root=$place_root
agent_root=
# Roots, not paths: a subdirectory of the session's own checkout is not a move.
[ -n "$agent_dir" ] && git_place "$agent_dir" && agent_root=$place_root
if [ -n "$agent_root" ] && [ "$agent_root" != "$session_root" ]; then
    place "$agent_root"
    row+=("${warn}⚠ $place_out${reset}")
fi

# How you are doing. The model keeps its size but not the word around it.
dictate
meters=("$dictate_out")
if [ -n "$model" ]; then
    [[ $model =~ ^(.*)\ \(([0-9]+[kM])\ context\)(.*)$ ]] &&
        model="${BASH_REMATCH[1]} ${BASH_REMATCH[2]}${BASH_REMATCH[3]}"
    meters+=("$model")
fi
if [ -n "$used" ]; then
    printf -v ctx '%.0f' "$used"
    meters+=("ctx $ctx%")
fi
if [ -n "$five" ]; then
    window 5h "$five" "$five_at" always
    meters+=("$window_out")
fi
if [ -n "$seven" ]; then
    window 7d "$seven" "$seven_at"
    meters+=("$window_out")
fi

join_row "${row[@]}"
first=$join_out
join_row "${meters[@]}"
second=$join_out

# A row with nothing in it is skipped rather than printed blank.
rows=()
[ -n "$first" ] && rows+=("$first")
[ -n "$second" ] && rows+=("$second")
printf '%s' "${rows[0]-}"
for ((i = 1; i < ${#rows[@]}; i++)); do printf '\n%s' "${rows[i]}"; done
