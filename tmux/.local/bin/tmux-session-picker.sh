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
#   --pin       NAME      toggle NAME's pin, print fzf actions to follow it
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
  p           pin / unpin session
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
do_pin() {
    local name=$1 pos
    { [ -z "$name" ] || [ "$name" = "$BAND_MARK" ]; } && return 0
    [ -x "$agentbar_bin" ] || return 0
    "$agentbar_bin" pin "$name" >/dev/null 2>&1 || return 0
    pos=$(build_lines | awk -F'\t' -v n="$name" '$1 == n { print NR; exit }')
    printf 'reload-sync(%s --list)+pos(%s)' "$self" "${pos:-1}"
    "$HOME/.local/bin/dotfiles-trace" log picker pin name="$name" 2>/dev/null || true
}

# ---- Display lines ---------------------------------------------------------

# Glyphs + colours match the sidebar's five-state language (Solarized Light):
# ◔ blocked=red, ? asking=amber, ⠹ working=cyan, ✓ done=green. Needs fzf --ansi.
icon_for() {
    case $1 in
        permission) printf '\033[38;2;220;50;47m◔\033[0m' ;;
        question) printf '\033[38;2;181;137;0m?\033[0m' ;;
        working) printf '\033[38;2;42;161;152m⠹\033[0m' ;;
        done) printf '\033[38;2;133;153;0m✓\033[0m' ;;
        *) printf ' ' ;;
    esac
}

# A band divider, muted like the sidebar's: label then a trailing rule. An
# empty label is the bare rule the bar draws between pinned and active.
band_row() {
    local label=$1 rule
    rule=$(printf '─%.0s' $(seq $((34 - ${#label}))))
    printf '%s\t\033[38;2;147;161;161m%s%s\033[0m\n' "$BAND_MARK" "$label" "$rule"
}

# Priority for the per-session rollup: permission(4) > asking(3) > working(2) >
# done(1) > blank(0). State comes from the @agent_* pane options the agentbar
# hook stamps on each Claude pane - the same source the sidebar reads, so picker
# and sidebar always agree. Pane options die with the pane, so no liveness
# bookkeeping.
state_rank() {
    case $1 in
        permission) printf 4 ;;
        question) printf 3 ;;
        working) printf 2 ;;
        done) printf 1 ;;
        *) printf 0 ;;
    esac
}

# Render the whole list as "<name>TAB<display>" - fzf hides field 1 via
# --with-nth=2 but binds use it for {1} substitution. Band headers head each
# group, and only when they actually separate two non-empty bands (a one-band
# list shows none), exactly as the sidebar does it.
build_lines() {
    local -A STATE_BY_SESSION
    local sess cmd present state prev current ordered band name path branch icon mark
    local -A count

    ordered=$(ordered_sessions)
    [ -z "$ordered" ] && return 0
    current=$(tmux display-message -p '#S' 2>/dev/null || true)

    # One pass over the server's panes: a pane is a live agent when the hook has
    # stamped @agent_present=1 and its foreground command is still claude/node
    # (guards a pane whose Claude died but left its options behind) - the same
    # filter the sidebar uses.
    while IFS=$'\t' read -r sess cmd present state; do
        [ "$present" = 1 ] || continue
        case $cmd in claude | node) ;; *) continue ;; esac
        prev=${STATE_BY_SESSION[$sess]:-}
        if [ "$(state_rank "$state")" -gt "$(state_rank "$prev")" ]; then
            STATE_BY_SESSION[$sess]=$state
        fi
    done < <(tmux list-panes -a -F $'#{session_name}\t#{pane_current_command}\t#{@agent_present}\t#{@agent_state}')

    while IFS=$'\t' read -r band name; do
        [ -n "$name" ] && count[$band]=$((${count[$band]:-0} + 1))
    done <<<"$ordered"

    prev=
    while IFS=$'\t' read -r band name; do
        [ -z "$name" ] && continue
        if [ "$band" != "$prev" ]; then
            case $band in
                pinned) [ $((${count[active]:-0} + ${count[dormant]:-0})) -gt 0 ] &&
                    band_row "pinned ·${count[pinned]} " ;;
                active) [ "${count[pinned]:-0}" -gt 0 ] && band_row "" ;;
                dormant) [ $((${count[pinned]:-0} + ${count[active]:-0})) -gt 0 ] &&
                    band_row "dormant ·${count[dormant]} " ;;
            esac
            prev=$band
        fi
        path=$(tmux display-message -p -t "$name" '#{pane_current_path}' 2>/dev/null || true)
        branch=
        if [ -n "$path" ] && [ -d "$path" ]; then
            branch=$(git -C "$path" symbolic-ref --short HEAD 2>/dev/null ||
                git -C "$path" rev-parse --short HEAD 2>/dev/null ||
                true)
        fi
        icon=$(icon_for "${STATE_BY_SESSION[$name]:-}")
        [ "$name" = "$current" ] && mark='▸' || mark=' '
        printf '%s\t%s %s %-18s  %s\n' "$name" "$mark" "$icon" "$name" "$branch"
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
    --pin)
        do_pin "${2:-}"
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
current_pos=$(printf '%s\n' "$lines" | awk -F'\t' -v c="$current" '$1 == c { print NR; exit }')
: "${current_pos:=1}"

fzf_colors='bg+:#268bd2,fg+:#fdf6e3,gutter:-1,pointer:-1,hl:#268bd2'
fzf_colors+=',hl+:#fdf6e3,border:#93a1a1,info:#93a1a1,prompt:#586e75'

# Band headers are rows fzf has no way to mark unselectable, so every motion
# chases itself off one: `down` then, if we landed on a header, one more. The
# transform runs after the move, so {1} is the row we arrived at. Parenthesized
# form, always: `transform:` swallows the rest of the --bind string, silently
# eating every binding listed after it.
skip_down="transform([ {1} = $BAND_MARK ] && echo down || true)"
# Going up, the top row is a header with nothing above it: another `up` would
# park the cursor on it, so bounce back down instead (FZF_POS is 1-based).
skip_up="transform([ {1} = $BAND_MARK ] && { [ \"\$FZF_POS\" -gt 1 ] && echo up || echo down; })"

# --sync + start:pos ensures the initial cursor position fires exactly once,
# at startup. Using load:pos here would re-fire on every reload, snapping
# the cursor back to the original session after rename/pin/kill/new.
target=$(
    printf '%s\n' "$lines" |
        fzf --ansi --sync --reverse --no-input --highlight-line \
            --header='? for help' --header-first \
            --delimiter=$'\t' --with-nth=2 \
            --preview "$HOME/.local/bin/tmux-session-preview.sh {1}" \
            --preview-window=down:50% \
            --pointer=' ' \
            --color="$fzf_colors" \
            --bind "start:pos($current_pos)" \
            --bind "j:down+$skip_down,k:up+$skip_up" \
            --bind "g:first+$skip_down,G:last+$skip_up" \
            --bind 'alt-;:abort' \
            --bind "enter:transform([ {1} = $BAND_MARK ] && echo ignore || echo accept)" \
            --bind "?:execute($self --help)" \
            --bind "p:transform($self --pin {1})" \
            --bind "r:execute($self --rename {1})+reload($self --list)" \
            --bind "D:execute($self --kill {1})+reload($self --list)" \
            --bind "c:execute($self --new)+reload($self --list)+last" |
        cut -f1
) || exit 0

if [ -n "$target" ] && [ "$target" != "$BAND_MARK" ]; then
    tmux switch-client -t "$target"
    "$HOME/.local/bin/dotfiles-trace" log picker switch target="$target" 2>/dev/null || true
fi
