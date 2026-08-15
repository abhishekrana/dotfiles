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

TTL=30 # seconds a cache entry stays fresh
CACHE_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/tmux-gitlab"

# Solarized Light styles (match .tmux.conf).
C_ISSUE='#[fg=#268bd2]' # blue
C_MR='#[fg=#6c71c4]'    # violet
C_RESET='#[default]'

ci_color() { # pipeline status -> fg style
    case "$1" in
        success) printf '#[fg=#859900]' ;;                                                       # green
        running | preparing) printf '#[fg=#268bd2]' ;;                                           # blue
        failed) printf '#[fg=#dc322f]' ;;                                                        # red
        pending | created | waiting_for_resource | scheduled | manual) printf '#[fg=#b58900]' ;; # yellow
        canceled | skipped) printf '#[fg=#586e75]' ;;                                            # base01
        *) printf '#[fg=#657b83]' ;;                                                             # base00
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

# Cache file for a repo+branch pair.
cache_file() { printf '%s/%s' "$CACHE_DIR" "$(printf '%s' "$1::$2" | md5sum | cut -c1-16)"; }

# Echo "<toplevel>\t<branch>" for a GitLab checkout, or return 1 (not a repo,
# detached HEAD, or a non-GitLab remote -- stay silent in all three cases).
git_info() {
    local path="$1" top branch remote
    top=$(git -C "$path" rev-parse --show-toplevel 2>/dev/null) || return 1
    branch=$(git -C "$path" symbolic-ref --quiet --short HEAD 2>/dev/null) || return 1
    remote=$(git -C "$path" remote get-url origin 2>/dev/null) || return 1
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

    # Refresh in the background when missing or older than TTL. Touch first so the
    # next redraw (~5s) doesn't spawn another refresh while this one is running.
    local now mtime
    now=$(date +%s)
    mtime=$([ -f "$cache" ] && stat -c %Y "$cache" 2>/dev/null || echo 0)
    if [ $((now - mtime)) -ge "$TTL" ]; then
        [ -f "$cache" ] && touch "$cache"
        setsid -f "$0" refresh "$path" >/dev/null 2>&1 || ("$0" refresh "$path" >/dev/null 2>&1 &)
    fi

    [ -f "$cache" ] || return 0
    local issue mr ci out=''
    issue=$(cache_get "$cache" issue_iid)
    mr=$(cache_get "$cache" mr_iid)
    ci=$(cache_get "$cache" ci_status)
    # No "issue"/"MR" words: # and ! are GitLab's own notation for them, and the
    # footer is short of columns. CI keeps its word as the click target, with the
    # state in a glyph beside it.
    [ -n "$issue" ] && out+=" ${C_ISSUE}#[range=user|gl-issue]#${issue}#[norange]"
    [ -n "$mr" ] && out+=" ${C_MR}#[range=user|gl-mr]!${mr}#[norange]"
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
    local mr_json mr_iid='' mr_url=''
    mr_json=$(glab mr list --source-branch="$branch" -F json 2>/dev/null)
    if [ -n "$mr_json" ]; then
        mr_iid=$(printf '%s' "$mr_json" | jq -r '.[0].iid // empty' 2>/dev/null)
        mr_url=$(printf '%s' "$mr_json" | jq -r '.[0].web_url // empty' 2>/dev/null)
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
        printf 'ci_status=%s\n' "$ci_status"
        printf 'ci_url=%s\n' "$ci_url"
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

    if [ "$(tmux show -gv @is_ssh 2>/dev/null)" = "1" ]; then
        tmux set-buffer -w -- "$url" # OSC 52 -> local clipboard
        tmux display-message "Copied: $url"
    else
        xdg-open "$url" >/dev/null 2>&1 &
        tmux display-message "Opening: $url"
    fi
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
    render)
        shift
        cmd_render "$@"
        ;;
    *) cmd_render "$@" ;; # tolerate a bare path
esac
