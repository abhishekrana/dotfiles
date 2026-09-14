#!/usr/bin/env bash
# Claude Code status line: where you are, how you are doing, and what you are on.
#
#     repo ⎇ feature · ⚠ other ⎇ main
#     Opus 5 1M · ctx 24% · 5h 23% ↻2h14 · 7d 41%
#     CI ✓ · https://<host>/<project>/-/issues/<iid>
#
# The ⚠ place is the worktree Claude last wrote in, recorded by this package's
# statusline-workdir.sh: an Edit by absolute path moves neither the cwd nor row
# one. Rate limits arrive on stdin; a window absent from it shows nothing.
#
# Row three needs an issue or a pipeline. Its URL is bare because only text the
# terminal linkifies is clickable here - OSC 8 dies in the multiplexer - and glab
# returns it, so no host or project is written down.
set -u

# How long each answer stays fresh, one dial per thing that moves at its own
# rate. Every `glab` call costs seconds, and every branch on screen pays these on
# repeat, so raising a TTL is the way to spend fewer calls.
GL_CI_TTL=${CLAUDE_GITLAB_CI_TTL:-60}         # the pipeline, while you work
GL_LINK_TTL=${CLAUDE_GITLAB_LINK_TTL:-600}    # the issue, fixed once resolved
GL_REPO_TTL=${CLAUDE_GITLAB_REPO_TTL:-86400}  # whether the checkout is GitLab
GL_QUIET_TTL=${CLAUDE_GITLAB_QUIET_TTL:-3600} # a checkout that is not
GL_CACHE="${XDG_CACHE_HOME:-$HOME/.cache}/claude-statusline"

# One cache file per checkout and branch.
gl_key() {
    local s=$1::$2
    s=${s//[^A-Za-z0-9]/_}
    # bash returns EMPTY for ${s: -n} when n exceeds the length, so clamp only
    # when it is actually too long.
    [ ${#s} -gt 180 ] && s=${s: -180}
    printf '%s' "$s"
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

# The terminal's own 16, so every flavor of the palette follows. 96 is base1,
# the muted tone; 90 is base03, which is near-black in a light flavor.
warn=$'\033[33m'
hot=$'\033[31m'
good=$'\033[32m'
live=$'\033[34m'
muted=$'\033[96m'
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

# The pipeline as one coloured glyph, or nothing when there is no pipeline.
gl_ci() {
    case $1 in
        success) printf '%sCI ✓%s' "$good" "$reset" ;;
        failed) printf '%sCI ✗%s' "$hot" "$reset" ;;
        running | preparing) printf '%sCI ●%s' "$live" "$reset" ;;
        pending | created | scheduled | manual | waiting_for_resource) printf '%sCI ○%s' "$warn" "$reset" ;;
        canceled | skipped) printf 'CI ◌' ;;
        *) ;;
    esac
}

# The issue this branch is named for, and the state of its pipeline.
gl_row() {
    local root=$1 branch=$2 file repo_file url ci url_at ci_at parts=() out i
    { [ -n "$root" ] && [ -n "$branch" ]; } || return 0
    file=$GL_CACHE/$(gl_key "$root" "$branch")
    repo_file=$GL_CACHE/repo-$(gl_key "$root" '')
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

    [ -n "$ci" ] && parts+=("$(gl_ci "$ci")")
    [ -n "$url" ] && parts+=("$muted$url$reset")
    ((${#parts[@]})) || return 0
    out=${parts[0]}
    for ((i = 1; i < ${#parts[@]}; i++)); do out+=" · ${parts[i]}"; done
    printf '%s' "$out"
}

# The link follows the worktree Claude writes in.
link=
git_place "${agent_root:-$dir}" && link=$(gl_row "$place_root" "$place_branch")

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
[ -n "$link" ] && printf '\n%s' "$link"
