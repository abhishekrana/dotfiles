#!/bin/bash
# tmux-gitlab.sh -- GitLab issue / MR / pipeline segments for the tmux status bar.
#
# Subcommands:
#   render <path>      Print the status-right segment for the repo at <path>.
#                      Fast: reads a cache and kicks a background refresh when
#                      stale. Emits clickable #[range=user|gl-*] regions.
#   refresh <path>     Query glab and rewrite the cache. Runs detached from
#                      render; one refresh per repo+branch at a time (flock).
#   open <seg> <path>  Handle a click on gl-issue|gl-mr|gl-ci: open the URL in a
#                      local browser, or copy it to the clipboard when the tmux
#                      server is reached over SSH (OSC 52 via set-clipboard).
#
# Cache lives in $XDG_CACHE_HOME/tmux-gitlab, keyed by repo toplevel + branch.
# Only local git commands run on every status redraw; network calls (glab) are
# throttled to TTL and never block the status bar.

# Seconds a cache entry stays fresh. A newly created MR or ticket is invisible
# until the entry expires, and the bar then paints it at the next redraw
# (status-interval, 5s) - so this is most of the "why is my MR not showing yet".
# Lower costs more `glab` calls, which take ~2.6s each but run detached.
TTL=${TMUX_GITLAB_TTL:-15}
CACHE_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/tmux-gitlab"

# Colours come from the theme switcher, never hardcoded - this renders styled
# text itself, so it reads the palette rather than a tmux option. The fallbacks
# are the palette's solarized-light values, for a box where `theme` never ran.
_theme_accent="#268bd2" _theme_changes="#6c71c4" _theme_done="#859900"
_theme_blocked="#dc322f" _theme_asking="#b58900" _theme_emphasis="#586e75" _theme_fg="#657b83"
_theme_colors="${XDG_CONFIG_HOME:-$HOME/.config}/theme/colors.sh"
# shellcheck source=/dev/null
[ -r "$_theme_colors" ] && . "$_theme_colors"

C_ISSUE="#[fg=$_theme_accent]"
C_MR="#[fg=$_theme_changes]"
C_RESET='#[default]'

ci_color() { # pipeline status -> fg style
    case "$1" in
        success) printf '#[fg=%s]' "$_theme_done" ;;
        running | preparing) printf '#[fg=%s]' "$_theme_accent" ;;
        failed) printf '#[fg=%s]' "$_theme_blocked" ;;
        pending | created | waiting_for_resource | scheduled | manual) printf '#[fg=%s]' "$_theme_asking" ;;
        canceled | skipped) printf '#[fg=%s]' "$_theme_emphasis" ;;
        *) printf '#[fg=%s]' "$_theme_fg" ;;
    esac
}

# One fixed-width glyph per state, not a word: the colour already says which
# state it is, and a word that changes length ("passed" -> "running") would shift
# the clock left and right as pipelines flip. Glyphs are single-width and NOT
# emoji-presentation (⏸ renders double in some terminals; ● ○ do not).
ci_glyph() {
    case "$1" in
        success) printf '✓' ;;
        failed) printf '✗' ;;
        running | preparing | pending | created | waiting_for_resource | scheduled | manual) printf '●' ;;
        canceled | skipped) printf '○' ;;
        *) printf '·' ;;
    esac
}

# ⇢ for open, not ⇄: a merge request points one way, at main, and has not landed
# yet - which pairs with ✓ for arrived. Merged and closed print CI's own ✓ and ✗
# (U+2713, U+2717) rather than lookalikes - ✔ (U+2714) renders visibly larger
# beside them - so the footer holds one tick and one cross at one weight.
# Position separates the two indicators: the MR glyph sits before the word CI.
#
# GitLab has four MR states - opened, merged, closed, locked - and `draft` is a
# separate boolean alongside them, not a fifth state. Conflicts are orthogonal
# too: an open MR that will not merge keeps the open glyph and turns red, rather
# than spending a sixth symbol (and ⚠ is emoji-presentation, so it would render
# double in some terminals and shift the clock).
mr_glyph() { # <state> <draft> -> one single-width glyph
    case "$1" in
        merged) printf '✓' ;;
        closed) printf '✗' ;;
        locked) printf '⊘' ;;
        opened) [ "$2" = true ] && printf '✎' || printf '⇢' ;;
        *) printf '·' ;;
    esac
}

mr_color() { # <state> <conflicts> -> fg style
    case "$1" in
        merged) printf '#[fg=%s]' "$_theme_done" ;;
        closed) printf '#[fg=%s]' "$_theme_fg" ;;
        locked) printf '#[fg=%s]' "$_theme_emphasis" ;;
        *) [ "$2" = true ] && printf '#[fg=%s]' "$_theme_blocked" || printf '%s' "$C_MR" ;; # red if it will not merge
    esac
}

# Cache key for a repo+branch pair. Parameter expansion, not md5sum: this runs on
# every status redraw and a fork costs more than the string work. Tail-truncated
# so a deep worktree plus a long branch cannot exceed NAME_MAX.
cache_key() {
    local s="$1::$2"
    s=${s//[^A-Za-z0-9]/_}
    # Only truncate when it is actually too long: bash returns EMPTY for
    # ${s: -n} when n exceeds the length, it does not clamp to the whole string.
    [ ${#s} -gt 180 ] && s=${s: -180}
    printf '%s' "$s"
}
cache_file() { printf '%s/%s' "$CACHE_DIR" "$(cache_key "$1" "$2")"; }

# Echo "<toplevel>\t<branch>" for a GitLab checkout, or return 1 (not a repo,
# detached HEAD, or a non-GitLab remote -- stay silent in all three cases).
git_info() {
    local path="$1" out top branch remote memo
    # One rev-parse for both facts: three git forks per redraw was most of this
    # script's cost, and the toplevel and branch come out of the same call.
    out=$(git -C "$path" rev-parse --show-toplevel --abbrev-ref HEAD 2>/dev/null) || return 1
    top=${out%%$'\n'*}
    branch=${out##*$'\n'}
    { [ -n "$top" ] && [ -n "$branch" ] && [ "$branch" != HEAD ]; } || return 1
    # Whether origin is GitLab is a property of the checkout, not of this redraw,
    # so it is answered once per worktree and then read back with a builtin.
    memo="$CACHE_DIR/remote-$(cache_key "$top" '')"
    if [ -f "$memo" ]; then
        read -r remote <"$memo"
    else
        remote=$(git -C "$path" remote get-url origin 2>/dev/null) || return 1
        mkdir -p "$CACHE_DIR" 2>/dev/null && printf '%s\n' "$remote" >"$memo"
    fi
    case "$remote" in *gitlab*) ;; *) return 1 ;; esac
    printf '%s\t%s\n' "$top" "$branch"
}

# Read a single key's value out of a cache file (no sourcing -- cache is data).
cache_get() {
    local file="$1" want="$2" k v
    while IFS='=' read -r k v; do [ "$k" = "$want" ] && {
        printf '%s' "$v"
        return
    }; done <"$file"
}

cmd_render() {
    local path="$1" info top branch cache
    [ -n "$path" ] || return 0
    info=$(git_info "$path") || return 0
    top=${info%%$'\t'*}
    branch=${info#*$'\t'}
    cache=$(cache_file "$top" "$branch")

    # Age comes from the `updated` key the refresh writes and $EPOCHSECONDS, so
    # neither `date` nor `stat` is forked here. flock in cmd_refresh is what stops
    # a slow refresh from being started again by the next redraw.
    local issue='' mr='' ci='' updated=0 k v out=''
    local mr_state='' mr_draft='' mr_conflicts=''
    if [ -f "$cache" ]; then
        while IFS='=' read -r k v; do
            case "$k" in
                issue_iid) issue=$v ;;
                mr_iid) mr=$v ;;
                mr_state) mr_state=$v ;;
                mr_draft) mr_draft=$v ;;
                mr_conflicts) mr_conflicts=$v ;;
                ci_status) ci=$v ;;
                updated) updated=$v ;;
            esac
        done <"$cache" # one pass for every key, not one pass per key
    fi
    if [ $((EPOCHSECONDS - updated)) -ge "$TTL" ]; then
        setsid -f "$0" refresh "$path" >/dev/null 2>&1 || ("$0" refresh "$path" >/dev/null 2>&1 &)
    fi
    [ -f "$cache" ] || return 0
    # No "issue"/"MR" words: # and ! are GitLab's own notation for them, and the
    # footer is short of columns. CI keeps its word as the click target, with the
    # state in a glyph beside it.
    [ -n "$issue" ] && out+=" ${C_ISSUE}#[range=user|gl-issue]#${issue}#[norange]"
    if [ -n "$mr" ]; then
        out+=" $(mr_color "$mr_state" "$mr_conflicts")#[range=user|gl-mr]"
        out+="!${mr} $(mr_glyph "$mr_state" "$mr_draft")#[norange]"
    fi
    [ -n "$ci" ] && out+=" $(ci_color "$ci")#[range=user|gl-ci]CI $(ci_glyph "$ci")#[norange]"
    [ -n "$out" ] && printf '%s%s' "$out" "$C_RESET"
}

cmd_refresh() {
    local path="$1" info top branch cache
    [ -n "$path" ] || return 0
    info=$(git_info "$path") || return 0
    top=${info%%$'\t'*}
    branch=${info#*$'\t'}
    cache=$(cache_file "$top" "$branch")
    mkdir -p "$CACHE_DIR"

    # One refresh per repo+branch at a time.
    exec 9>"${cache}.lock"
    flock -n 9 || return 0
    cd "$path" || return 0

    # Merge request for this branch.
    # --all, because glab lists only OPEN merge requests by default: without it the
    # segment does not change when yours merges, it disappears, and "merged" is
    # indistinguishable from "never had one". Newest first, so a branch reused
    # across several MRs shows the current one rather than an old closed one.
    local mr_json mr_iid='' mr_url='' mr_state='' mr_draft='' mr_conflicts=''
    mr_json=$(glab mr list --source-branch="$branch" --all -F json 2>/dev/null)
    if [ -n "$mr_json" ]; then
        local mr_one
        mr_one=$(printf '%s' "$mr_json" | jq -c 'sort_by(.iid) | reverse | .[0] // empty' 2>/dev/null)
        if [ -n "$mr_one" ]; then
            mr_iid=$(printf '%s' "$mr_one" | jq -r '.iid // empty' 2>/dev/null)
            mr_url=$(printf '%s' "$mr_one" | jq -r '.web_url // empty' 2>/dev/null)
            mr_state=$(printf '%s' "$mr_one" | jq -r '.state // empty' 2>/dev/null)
            mr_draft=$(printf '%s' "$mr_one" | jq -r '.draft // false' 2>/dev/null)
            mr_conflicts=$(printf '%s' "$mr_one" | jq -r '.has_conflicts // false' 2>/dev/null)
        fi
    fi

    # Issue: branch-name prefix (e.g. 42-fix-login), else the MR's first closing issue.
    local issue_iid='' issue_url='' prefix
    prefix=$(printf '%s' "$branch" | grep -oE '^[0-9]+')
    if [ -n "$prefix" ]; then
        local ij
        ij=$(glab issue view "$prefix" -F json 2>/dev/null)
        if [ -n "$ij" ]; then
            issue_iid=$(printf '%s' "$ij" | jq -r '.iid // empty' 2>/dev/null)
            issue_url=$(printf '%s' "$ij" | jq -r '.web_url // empty' 2>/dev/null)
        fi
    fi
    if [ -z "$issue_iid" ] && [ -n "$mr_iid" ]; then
        local cj
        cj=$(glab api "projects/:fullpath/merge_requests/$mr_iid/closes_issues" 2>/dev/null)
        if [ -n "$cj" ]; then
            issue_iid=$(printf '%s' "$cj" | jq -r '.[0].iid // empty' 2>/dev/null)
            issue_url=$(printf '%s' "$cj" | jq -r '.[0].web_url // empty' 2>/dev/null)
        fi
    fi

    # Pipeline for this branch.
    local pj ci_status='' ci_url=''
    pj=$(glab ci get -b "$branch" -F json 2>/dev/null)
    if [ -n "$pj" ]; then
        ci_status=$(printf '%s' "$pj" | jq -r '.status // empty' 2>/dev/null)
        ci_url=$(printf '%s' "$pj" | jq -r '.web_url // empty' 2>/dev/null)
    fi

    # Write atomically so render never reads a half-written file.
    local tmp="${cache}.tmp.$$"
    {
        printf 'issue_iid=%s\n' "$issue_iid"
        printf 'issue_url=%s\n' "$issue_url"
        printf 'mr_iid=%s\n' "$mr_iid"
        printf 'mr_url=%s\n' "$mr_url"
        printf 'mr_state=%s\n' "$mr_state"
        printf 'mr_draft=%s\n' "$mr_draft"
        printf 'mr_conflicts=%s\n' "$mr_conflicts"
        printf 'ci_status=%s\n' "$ci_status"
        printf 'ci_url=%s\n' "$ci_url"
        printf 'updated=%s\n' "$EPOCHSECONDS"
    } >"$tmp" && mv -f "$tmp" "$cache"
}

cmd_open() {
    local seg="$1" path="$2" info top branch cache field url
    [ -n "$seg" ] && [ -n "$path" ] || return 0
    info=$(git_info "$path") || return 0
    top=${info%%$'\t'*}
    branch=${info#*$'\t'}
    cache=$(cache_file "$top" "$branch")
    [ -f "$cache" ] || return 0

    case "$seg" in
        gl-issue) field=issue_url ;;
        gl-mr) field=mr_url ;;
        gl-ci) field=ci_url ;;
        *) return 0 ;;
    esac
    url=$(cache_get "$cache" "$field")
    [ -n "$url" ] || {
        tmux display-message "GitLab: no ${seg#gl-} link"
        return 0
    }

    open_url "$url"
}

# open_url <url> - the one place a GitLab link leaves this machine, for the chips and for
# workdesk's o. Over ssh the browser is at the other end, so the link goes to the local
# clipboard through OSC 52 instead.
open_url() {
    local url="$1"
    [ -n "$url" ] || return 1
    if [ "$(tmux show -gv @is_ssh 2>/dev/null)" = "1" ]; then
        tmux set-buffer -w -- "$url" # OSC 52 -> local clipboard
        tmux display-message "Copied: $url"
        return 0
    fi
    # Checked, not assumed: xdg-open is backgrounded, so its own failure is invisible
    # here and the caller would read the whole thing as opened.
    if ! command -v xdg-open >/dev/null 2>&1; then
        tmux display-message "No xdg-open for: $url"
        return 1
    fi
    xdg-open "$url" >/dev/null 2>&1 &
    tmux display-message "Opening: $url"
}

case "${1:-render}" in
    refresh)
        shift
        cmd_refresh "$@"
        ;;
    open)
        shift
        cmd_open "$@"
        ;;
    open-url)
        shift
        open_url "$1"
        ;;
    render)
        shift
        cmd_render "$@"
        ;;
    /*) cmd_render "$@" ;; # tolerate a bare path
    *)
        printf 'tmux-gitlab.sh: no such command: %s\n' "$1" >&2
        exit 2
        ;;
esac
