#!/usr/bin/env bash
# fzf-based tmux session picker. Shows branch + Claude state inline, previews
# windows with per-pane state breakdown. Opens with the current session
# pre-selected; Enter switches to the highlighted session.
#
# Keys (in addition to fzf defaults):
#   j / k         move cursor down / up (band headers are skipped)
#   g / G         jump to first / last
#   p             pin / unpin highlighted session (floats it to the top band)
#   r             rename highlighted session
#   D             kill highlighted session (with confirm)
#   c             create new session (prompts for name + starting dir)
#   ?             show this help overlay
#   Alt-;         dismiss picker
#
# Order comes from `agentbar order`, so this popup, the sidebar and the Alt-h /
# Alt-l session keys all walk one list: the pinned / active / dormant bands,
# alphabetical inside each. Nothing here keeps an order of its own - `p` is the
# only thing that moves a session, and it writes the same pin set the sidebar's
# own `p` key does. Without the binary it degrades to flat alphabetical.
#
# Internal subcommands (invoked by fzf reload/execute callbacks):
#   --list                emit fzf line data (NAME<TAB>display)
#   --band      NAME BAND put NAME in BAND, print fzf actions to follow it
#   --rename    NAME      interactive rename of NAME
#   --kill      NAME      interactive kill of NAME
#   --new                 interactive new-session prompt
#   --help                show help overlay

set -euo pipefail

# tmux sanitizes tabs in -F output to "_" outside a UTF-8 locale.
export LC_ALL=C.UTF-8

agentbar_bin=${AGENTBAR_BIN:-$HOME/dotfiles/apps/agentbar/bin/agentbar}
self=$(realpath "$0")

# Field 1 of a band header row. Real rows carry a session name there, so every
# action can tell a header apart and no-op on it.
BAND_MARK='__band__'

# Glyphs, colors and the state ranking are shared with the preview (and mirror
# the sidebar's), so the three can't drift apart. Resolved next to this script,
# not via ~/.local/bin: that path only exists once the package is stowed.
# shellcheck source=tmux/.local/bin/tmux-agent-state.sh
. "$(dirname "$(realpath "${BASH_SOURCE[0]}")")/tmux-agent-state.sh"

# ---- Helpers ---------------------------------------------------------------

flash_error() {
    printf '\n✗ %s\n' "$1" >&2
    sleep 1.2
}

session_exists() {
    tmux list-sessions -F '#{session_name}' 2>/dev/null | grep -Fxq -- "$1"
}

# Emit "band<TAB>name" per session in sidebar order. Falls back to flat
# alphabetical (one band, so no headers) when the binary isn't built yet.
ordered_sessions() {
    if [ -x "$agentbar_bin" ] && out=$("$agentbar_bin" order 2>/dev/null) && [ -n "$out" ]; then
        printf '%s\n' "$out"
        return 0
    fi
    tmux list-sessions -F 'active	#{session_name}' 2>/dev/null | sort -t'	' -k2 || true
}

# ---- Interactive actions ---------------------------------------------------

do_help() {
    clear
    cat <<'EOF'
tmux session picker

  j / k       cursor down / up
  g / G       jump to first / last
  p           pin session (floats to the top band)
  a           put session in the active band
  d           put session in the dormant band
  r           rename session
  D           kill session (with confirm)
  c           new session (name + start dir)
  ?           this help
  Enter       switch to selected session
  Alt-;       dismiss

  Order mirrors the agent bar: pinned, active, dormant.

EOF
    printf 'press any key to dismiss...'
    read -rsn1 _
}

do_kill() {
    local name=$1 ans err
    { [ -z "$name" ] || [ "$name" = "$BAND_MARK" ]; } && return 0
    clear
    printf 'kill session "%s"? [y/N] ' "$name"
    read -r ans || return 0
    case $ans in y | Y | yes) ;; *) return 0 ;; esac
    if ! err=$(tmux kill-session -t "$name" 2>&1); then
        flash_error "$err"
        return 0
    fi
    "$HOME/.local/bin/dotfiles-trace" log picker kill name="$name" 2>/dev/null || true
}

do_new() {
    local name dir default_dir err
    clear
    read -re -p "new session name: " name || return 0
    [ -z "$name" ] && return 0
    if session_exists "$name"; then
        flash_error "session \"$name\" already exists"
        return 0
    fi
    default_dir=$(tmux display-message -p '#{pane_current_path}' 2>/dev/null || true)
    { [ -z "$default_dir" ] || [ ! -d "$default_dir" ]; } && default_dir=$HOME
    read -re -i "$default_dir" -p "start dir: " dir || return 0
    dir=${dir/#\~/$HOME}
    if [ ! -d "$dir" ]; then
        flash_error "no such directory: $dir"
        return 0
    fi
    if ! err=$(tmux new-session -d -s "$name" -c "$dir" 2>&1); then
        flash_error "$err"
        return 0
    fi
    "$HOME/.local/bin/dotfiles-trace" log picker new name="$name" dir="$dir" 2>/dev/null || true
}

do_rename() {
    local old=$1 new err
    { [ -z "$old" ] || [ "$old" = "$BAND_MARK" ]; } && return 0
    clear
    read -re -i "$old" -p "rename to: " new || return 0
    [ -z "$new" ] && return 0
    [ "$new" = "$old" ] && return 0
    if session_exists "$new"; then
        flash_error "session \"$new\" already exists"
        return 0
    fi
    if ! err=$(tmux rename-session -t "$old" -- "$new" 2>&1); then
        flash_error "rename failed: $err"
        return 0
    fi
    "$HOME/.local/bin/dotfiles-trace" log picker rename from="$old" to="$new" 2>/dev/null || true
}

# Toggle NAME's pin, then print the fzf actions that redraw the list and follow
# the row to its new band - a pin moves it, and leaving the cursor at the old
# index would aim the next keypress at a different session. Bound as a
# `transform` action, whose stdout fzf reads as an action list.
# do_band <session> <pinned|active|dormant> - the sidebar's p, a and d keys,
# through the same binary so both views drive one store. One key, one
# destination; pressing it again changes nothing.
#
# Repositions after the reload: fzf's reload resets the cursor, and the first
# match is taken without `exit` because exiting closes the pipe while
# build_lines is still writing - gawk (the CI runner's awk) then SIGPIPEs it,
# and 141 through pipefail would kill this script before the printf below, so
# the band would land but nothing would redraw.
do_band() {
    local name=$1 band=$2 pos
    { [ -z "$name" ] || [ "$name" = "$BAND_MARK" ]; } && return 0
    [ -x "$agentbar_bin" ] || return 0
    "$agentbar_bin" band "$name" "$band" >/dev/null 2>&1 || return 0
    pos=$(build_lines | awk -F'\t' -v n="$name" '$1 == n && !p { print NR; p = 1 }')
    printf 'reload-sync(%s --list)+pos(%s)' "$self" "${pos:-1}"
    "$HOME/.local/bin/dotfiles-trace" log picker band name="$name" band="$band" 2>/dev/null || true
}

# ---- Display lines ---------------------------------------------------------

# A band divider, muted like the sidebar's: label then a trailing rule.
band_row() {
    local label=$1 rule
    rule=$(printf '─%.0s' $(seq $((34 - ${#label}))))
    printf '%s\t%s\n' "$BAND_MARK" "$(agent_muted "$label$rule")"
}

# State comes from the @agent_* pane options the agentbar hook stamps on each
# Claude pane - the same source the sidebar reads, so picker and sidebar always
# agree. Pane options die with the pane, so no liveness bookkeeping.

# Render the whole list as "<name>TAB<display>" - fzf hides field 1 via
# --with-nth=2 but binds use it for {1} substitution. Band headers head each
# group, and only when they actually separate two non-empty bands (a one-band
# list shows none), exactly as the sidebar does it.
build_lines() {
    local -A STATE_BY_SESSION WORKDIR_BY_SESSION DIR_BY_SESSION DIR_VOTES PANES_IN
    local -A TITLE_BY_SESSION AGENTS_IN BRANCH_BY_SESSION
    local sess cmd present state wd path title host seen prev current ordered band name branch icon mark header
    local rank prev_rank col more bw=0 b
    local headed=
    local -A count

    ordered=$(ordered_sessions)
    [ -z "$ordered" ] && return 0
    current=$(tmux display-message -p '#S' 2>/dev/null || true)

    # One pass over the server's panes: a pane is a live agent when the hook has
    # stamped @agent_present=1 and its foreground command is still claude/node
    # (guards a pane whose Claude died but left its options behind) - the same
    # filter the sidebar uses.
    # Optional fields carry a "-" placeholder: tab is IFS whitespace, so bash
    # collapses a run of empty ones into a single separator and every field after
    # them shifts left - a pane with no @agent_* options would lose its path.
    while IFS=$'\t' read -r sess cmd present state wd title host path; do
        [ "$state" = - ] && state=
        [ "$wd" = - ] && wd=
        [ "$title" = - ] && title=
        # The directory that names a session: where most of its panes sit. NOT the
        # session's active pane - that is usually the sidebar, whose cwd is only
        # wherever that process started, or the diff pane, pointed at whatever
        # worktree you asked for; both named a branch the session was not on. The
        # sidebar is excluded outright, and one stray shell cannot outvote the rest.
        if [ "$cmd" != agentbar ] && [ -n "$path" ]; then
            seen=$((${PANES_IN[$sess$'\t'$path]:-0} + 1))
            PANES_IN[$sess$'\t'$path]=$seen
            if [ "$seen" -gt "${DIR_VOTES[$sess]:-0}" ]; then
                DIR_VOTES[$sess]=$seen
                DIR_BY_SESSION[$sess]=$path
            fi
        fi
        [ "$present" = 1 ] || continue
        case $cmd in claude | node) ;; *) continue ;; esac
        # The worktree the agent WRITES in, like the sidebar's own branch line.
        if [ -n "$wd" ] && [ -z "${WORKDIR_BY_SESSION[$sess]:-}" ]; then
            WORKDIR_BY_SESSION[$sess]=$wd
        fi
        # One agent speaks for the row: the most urgent one, which is already
        # the one the glyph describes - so glyph and title never disagree. The
        # first agent seen sets both, since idle outranks nothing.
        prev=${STATE_BY_SESSION[$sess]:-}
        rank=$(agent_state_rank "$state")
        prev_rank=$(agent_state_rank "$prev")
        if [ -z "${AGENTS_IN[$sess]:-}" ] || [ "$rank" -gt "$prev_rank" ]; then
            STATE_BY_SESSION[$sess]=$state
            TITLE_BY_SESSION[$sess]=$(agent_title "$title" "$host")
        fi
        AGENTS_IN[$sess]=$((${AGENTS_IN[$sess]:-0} + 1))
    done < <(tmux list-panes -a -F "$(
        printf '#{session_name}\t#{pane_current_command}\t#{?@agent_present,1,0}\t'
        printf '#{?@agent_state,#{@agent_state},-}\t#{?@agent_workdir,#{@agent_workdir},-}\t'
        printf '#{?pane_title,#{pane_title},-}\t#{host}\t#{pane_current_path}'
    )")

    while IFS=$'\t' read -r band name; do
        [ -n "$name" ] && count[$band]=$((${count[$band]:-0} + 1))
    done <<<"$ordered"

    # The branch column is sized to the longest name on screen, not to a guess:
    # the popup has ~187 columns, and branches here run past 30. One git call per
    # session, the same one the render loop used to make.
    while IFS=$'\t' read -r band name; do
        [ -z "$name" ] && continue
        path=${WORKDIR_BY_SESSION[$name]:-${DIR_BY_SESSION[$name]:-}}
        b=
        if [ -n "$path" ] && [ -d "$path" ]; then
            b=$(git -C "$path" symbolic-ref --short HEAD 2>/dev/null ||
                git -C "$path" rev-parse --short HEAD 2>/dev/null ||
                true)
        fi
        # Past 44 a branch is eating the title's room, so cap it there.
        [ ${#b} -gt 44 ] && b="${b:0:43}…"
        BRANCH_BY_SESSION[$name]=$b
        [ ${#b} -gt "$bw" ] && bw=${#b}
    done <<<"$ordered"

    prev=
    while IFS=$'\t' read -r band name; do
        [ -z "$name" ] && continue
        if [ "$band" != "$prev" ]; then
            # All three bands are named, on one rule: a header shows when it
            # actually divides this band from a non-empty neighbour, so a
            # single-band list stays clean. Matches the sidebar's sectionHeader.
            header=
            case $band in
                pinned) [ $((${count[active]:-0} + ${count[dormant]:-0})) -gt 0 ] &&
                    header="pinned ·${count[pinned]} " ;;
                active) [ $((${count[pinned]:-0} + ${count[dormant]:-0})) -gt 0 ] &&
                    header="active ·${count[active]} " ;;
                dormant) [ $((${count[pinned]:-0} + ${count[active]:-0})) -gt 0 ] &&
                    header="dormant ·${count[dormant]} " ;;
            esac
            if [ -n "$header" ]; then
                # A blank row above every divider but the first, so each band
                # reads as its own group - the spacing the sidebar's `pad` gives
                # them. Marked inert like the header, so nav and every action
                # skip it; that makes runs of two skippable rows, which the
                # j/k/g binds handle with two conditional steps.
                [ -n "$headed" ] && printf '%s\t\n' "$BAND_MARK"
                band_row "$header"
                headed=1
            fi
            prev=$band
        fi
        branch=${BRANCH_BY_SESSION[$name]:-}
        # Fixed columns, so cycling lands the eye in the same place on every row.
        # printf pads by bytes, so only pad the all-ASCII case: a capped branch
        # ends in an ellipsis and is already the full width.
        case $branch in
            '') col=$(printf "%-$((bw + 2))s" '') ;;
            *…) col="⎇ $branch" ;;
            *) col="⎇ $(printf "%-${bw}s" "$branch")" ;;
        esac
        # "+N" owns up to the agents the one title does not speak for.
        more=$((${AGENTS_IN[$name]:-0} - 1))
        [ "$more" -gt 0 ] && more=" $(agent_muted "+$more")" || more=
        icon=$(agent_icon "${STATE_BY_SESSION[$name]:-}")
        [ "$name" = "$current" ] && mark='▸' || mark=' '
        printf '%s\t%s %s %-18s  %s  %s%s\n' "$name" "$mark" "$icon" "$name" \
            "$(agent_muted "$col")" "${TITLE_BY_SESSION[$name]:-$(agent_muted —)}" "$more"
    done <<<"$ordered"
    return 0
}

# ---- Subcommand dispatch ---------------------------------------------------

case "${1:-}" in
    --list)
        build_lines
        exit 0
        ;;
    --help)
        do_help
        exit 0
        ;;
    --kill)
        do_kill "${2:-}"
        exit 0
        ;;
    --new)
        do_new
        exit 0
        ;;

    --band)
        do_band "${2:-}" "${3:-}"
        exit 0
        ;;
    --rename)
        do_rename "${2:-}"
        exit 0
        ;;
esac

# ---- fzf invocation --------------------------------------------------------

lines=$(build_lines)
[ -z "$lines" ] && exit 0

current=$(tmux display-message -p '#S')
# No `exit` here either, for the reason do_band gives: a closed pipe under
# pipefail would take the popup down before fzf ever runs.
current_pos=$(printf '%s\n' "$lines" | awk -F'\t' -v c="$current" '$1 == c && !p { print NR; p = 1 }')
: "${current_pos:=1}"

# Band headers are rows fzf has no way to mark unselectable, so every motion
# chases itself off one: `down` then, if we landed on a header, one more. The
# transform runs after the move, so {1} is the row we arrived at. Parenthesized
# form, always: `transform:` swallows the rest of the --bind string, silently
# eating every binding listed after it.
skip_down="transform([ {1} = $BAND_MARK ] && echo down || true)"
# Going up, the top row is a header with nothing above it: another `up` would
# park the cursor on it, so bounce back down instead (FZF_POS is 1-based).
skip_up="transform([ {1} = $BAND_MARK ] && { [ \"\$FZF_POS\" -gt 1 ] && echo up || echo down; })"

# --height/--border are explicit to override FZF_DEFAULT_OPTS, which the popup
# inherits from the server's environment: its --height=80% inside an already 85%
# popup left a dead band at the bottom while the session list was still scrolling,
# and its --border drew a second frame inside tmux's own. The preview is a fixed
# 12 rows - enough for the header block and the windows - so every extra row the
# client has goes to the list.
#
# --sync + start:pos ensures the initial cursor position fires exactly once,
# at startup. Using load:pos here would re-fire on every reload, snapping
# the cursor back to the original session after rename/pin/kill/new.
target=$(
    printf '%s\n' "$lines" |
        fzf --ansi --sync --reverse --no-input --highlight-line \
            --height=100% --border=none \
            --header='? for help' --header-first \
            --delimiter=$'\t' --with-nth=2 \
            --preview "$HOME/.local/bin/tmux-session-preview.sh {1}" \
            --preview-window=down:12 \
            --pointer=' ' \
            --color="$_popup_fzf_color" \
            --bind "start:pos($current_pos)" \
            --bind "j:down+$skip_down+$skip_down,k:up+$skip_up+$skip_up" \
            --bind "g:first+$skip_down+$skip_down,G:last+$skip_up+$skip_up" \
            --bind 'alt-;:abort' \
            --bind "enter:transform([ {1} = $BAND_MARK ] && echo ignore || echo accept)" \
            --bind "?:execute($self --help)" \
            --bind "p:transform($self --band {1} pinned)" \
            --bind "a:transform($self --band {1} active)" \
            --bind "d:transform($self --band {1} dormant)" \
            --bind "r:execute($self --rename {1})+reload($self --list)" \
            --bind "D:execute($self --kill {1})+reload($self --list)" \
            --bind "c:execute($self --new)+reload($self --list)+last" |
        cut -f1
) || exit 0

if [ -n "$target" ] && [ "$target" != "$BAND_MARK" ]; then
    tmux switch-client -t "$target"
    "$HOME/.local/bin/dotfiles-trace" log picker switch target="$target" 2>/dev/null || true
fi
