#!/usr/bin/env bash
# Claude Code status line: where you are, how you are doing, and what you are on.
#
#     repo ⎇ feature · ⚠ other ⎇ main
#     ● dictate · Opus 5 1M · ctx 24% · 5h 23% ↻2h14 · 7d 41%
#     CI ✓ · https://<host>/<project>/-/issues/<iid>
#
# The ⚠ place is the worktree Claude last wrote in, recorded by this package's
# statusline-workdir.sh: an Edit by absolute path moves neither the cwd nor row
# one. Rate limits arrive on stdin; a window absent from it shows nothing.
#
# Row three needs an issue or a pipeline. Its URL is bare because only text the
# terminal linkifies is clickable here - OSC 8 dies in the multiplexer - and glab
# returns it, so no host or project is written down.
#
# It re-runs once a second (refreshInterval in settings.json) in every open
# session, so the whole script is one process: helpers write a named global
# rather than print into a `$( )` subshell, and jq reading the payload is the
# only command it runs. Anything costing more sits behind a TTL and refreshes
# detached.
set -u

# How long each answer stays fresh, one dial per thing that moves at its own
# rate. Every `glab` call costs seconds, and every branch on screen pays these on
# repeat, so raising a TTL is the way to spend fewer calls.
GL_CI_TTL=${CLAUDE_GITLAB_CI_TTL:-60}         # the pipeline, while you work
GL_LINK_TTL=${CLAUDE_GITLAB_LINK_TTL:-600}    # the issue, fixed once resolved
GL_REPO_TTL=${CLAUDE_GITLAB_REPO_TTL:-86400}  # whether the checkout is GitLab
GL_QUIET_TTL=${CLAUDE_GITLAB_QUIET_TTL:-3600} # a checkout that is not
GL_CACHE="${XDG_CACHE_HOME:-$HOME/.cache}/claude-statusline"

# One cache file per checkout and branch, into $key_out.
gl_key() {
    local s=$1::$2
    s=${s//[^A-Za-z0-9]/_}
    # bash returns EMPTY for ${s: -n} when n exceeds the length, so clamp only
    # when it is actually too long.
    [ ${#s} -gt 180 ] && s=${s: -180}
    key_out=$s
}

# Every key of a cache file, in one pass. Sets gl_<key> for the caller.
gl_read() {
    local file=$1 k v
    gl_url='' gl_url_at=0 gl_ci='' gl_ci_at=0 gl_repo='' gl_repo_at=0
    [ -f "$file" ] || return 0
    while IFS='=' read -r k v; do
        case $k in
            url) gl_url=$v ;;
            url_at) gl_url_at=$v ;;
            ci) gl_ci=$v ;;
            ci_at) gl_ci_at=$v ;;
            repo) gl_repo=$v ;;
            repo_at) gl_repo_at=$v ;;
        esac
    done <"$file"
}

# Whether this checkout is GitLab at all, asked once per GL_REPO_TTL. A checkout
# that is not gets GL_QUIET_TTL before being asked again: a remote can be added.
gl_repo_ok() {
    local file=$1 ttl
    gl_read "$file"
    ttl=$GL_REPO_TTL
    [ "$gl_repo" = 0 ] && ttl=$GL_QUIET_TTL
    if [ -z "$gl_repo" ] || ((EPOCHSECONDS - gl_repo_at >= ttl)); then
        gl_repo=0
        glab repo view -F json >/dev/null 2>&1 && gl_repo=1
        printf 'repo=%s\nrepo_at=%s\n' "$gl_repo" "$EPOCHSECONDS" >"$file.tmp" && mv -f "$file.tmp" "$file"
    fi
    [ "$gl_repo" = 1 ]
}

# Write down what GitLab has for this branch, refreshing only what has expired.
# Detached by its caller, one at a time per checkout and branch.
gl_refresh() {
    local root=$1 branch=$2 file=$3 repo_file=$4 url ci url_at ci_at iid json
    command -v glab >/dev/null || return 0
    mkdir -p "$GL_CACHE" || return 0
    exec 9>"$file.lock"
    flock -n 9 || return 0
    cd "$root" || return 0
    gl_repo_ok "$repo_file" || return 0

    gl_read "$file"
    url=$gl_url url_at=$gl_url_at ci=$gl_ci ci_at=$gl_ci_at

    if ((EPOCHSECONDS - url_at >= GL_LINK_TTL)); then
        # The issue the branch is named for, else the one its merge request closes.
        url=''
        iid=$(printf '%s' "$branch" | grep -oE '^[0-9]+')
        [ -n "$iid" ] && url=$(glab issue view "$iid" -F json 2>/dev/null | jq -r '.web_url // empty' 2>/dev/null)
        if [ -z "$url" ]; then
            # --all, because glab lists only open merge requests by default.
            json=$(glab mr list --source-branch="$branch" --all -F json 2>/dev/null)
            iid=$(printf '%s' "$json" | jq -r 'sort_by(.iid) | reverse | .[0].iid // empty' 2>/dev/null)
            [ -n "$iid" ] && url=$(glab api "projects/:fullpath/merge_requests/$iid/closes_issues" 2>/dev/null |
                jq -r '.[0].web_url // empty' 2>/dev/null)
        fi
        url_at=$EPOCHSECONDS
    fi
    if ((EPOCHSECONDS - ci_at >= GL_CI_TTL)); then
        ci=$(glab ci get -b "$branch" -F json 2>/dev/null | jq -r '.status // empty' 2>/dev/null)
        ci_at=$EPOCHSECONDS
    fi

    printf 'url=%s\nurl_at=%s\nci=%s\nci_at=%s\n' "$url" "$url_at" "$ci" "$ci_at" >"$file.tmp" &&
        mv -f "$file.tmp" "$file"
}

here=${BASH_SOURCE[0]%/*}
[ "$here" = "${BASH_SOURCE[0]}" ] && here=.
# shellcheck source=statusline-git.bash
. "$here/statusline-git.bash"

# The detached refresh re-enters this script, and waits on no payload.
if [ "${1-}" = "--refresh" ]; then
    gl_refresh "$2" "$3" "$4" "$5"
    exit 0
fi

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
good=$'\033[32m'
live=$'\033[34m'
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

# The pipeline as one coloured glyph into $ci_out, or nothing when there is none.
gl_glyph() {
    case $1 in
        success) ci_out="${good}CI ✓${reset}" ;;
        failed) ci_out="${hot}CI ✗${reset}" ;;
        running | preparing) ci_out="${live}CI ●${reset}" ;;
        pending | created | scheduled | manual | waiting_for_resource) ci_out="${warn}CI ○${reset}" ;;
        canceled | skipped) ci_out='CI ◌' ;;
        *) ci_out= ;;
    esac
}

# The issue this branch is named for, and the state of its pipeline, into $link_out.
gl_row() {
    local root=$1 branch=$2 file repo_file url ci url_at ci_at parts=()
    link_out=
    { [ -n "$root" ] && [ -n "$branch" ]; } || return 0
    gl_key "$root" "$branch"
    file=$GL_CACHE/$key_out
    gl_key "$root" ''
    repo_file=$GL_CACHE/repo-$key_out
    gl_read "$file"
    url=$gl_url ci=$gl_ci url_at=$gl_url_at ci_at=$gl_ci_at

    # Either answer expiring brings the other along: one refresh, asking only for
    # what it needs.
    gl_read "$repo_file"
    if { [ "$gl_repo" != 0 ] || ((EPOCHSECONDS - gl_repo_at >= GL_QUIET_TTL)); } &&
        { ((EPOCHSECONDS - ci_at >= GL_CI_TTL)) || ((EPOCHSECONDS - url_at >= GL_LINK_TTL)); } &&
        command -v glab >/dev/null; then
        setsid -f bash "$0" --refresh "$root" "$branch" "$file" "$repo_file" >/dev/null 2>&1 ||
            (bash "$0" --refresh "$root" "$branch" "$file" "$repo_file" >/dev/null 2>&1 &)
    fi

    if [ -n "$ci" ]; then
        gl_glyph "$ci"
        parts+=("$ci_out")
    fi
    [ -n "$url" ] && parts+=("$muted$url$reset")
    join_row "${parts[@]}"
    link_out=$join_out
}

# The link follows the worktree Claude writes in.
link=
git_place "${agent_root:-$dir}" && gl_row "$place_root" "$place_branch" && link=$link_out

join_row "${row[@]}"
first=$join_out
join_row "${meters[@]}"
second=$join_out

# A row with nothing in it is skipped rather than printed blank.
rows=()
[ -n "$first" ] && rows+=("$first")
[ -n "$second" ] && rows+=("$second")
[ -n "$link" ] && rows+=("$link")
printf '%s' "${rows[0]-}"
for ((i = 1; i < ${#rows[@]}; i++)); do printf '\n%s' "${rows[i]}"; done
