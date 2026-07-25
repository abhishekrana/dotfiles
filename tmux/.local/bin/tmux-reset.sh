#!/usr/bin/env bash
# tmux-reset.sh [session] -- reset the UI to its defaults (prefix+R).
#
# Reloads the config, restarts that session's sidebar so rebuilt code takes
# effect, then puts every window back to its default geometry: sidebar leftmost
# at @agentbar-width, remaining columns even. Also clears status chips left
# stuck by a script that died. Nothing is killed, so no work is lost.
#
# The reload runs with autostart off and only one sidebar process is restarted:
# a bulk sidebar respawn (prefix+e, or a plain reload) storms this tmux+Ghostty
# client. See CLAUDE.md "Debugging".

set -uo pipefail

AGENTBAR="$HOME/dotfiles/apps/agentbar"
# Tests point this at a private socket, which also skips the live-server steps.
SOCKET="${RESET_SOCKET:-}"

tmux() {
    if [ -n "$SOCKET" ]; then
        command tmux -S "$SOCKET" "$@"
    else
        command tmux "$@"
    fi
}

trace() {
    local evt=$1
    shift
    "$HOME/.local/bin/dotfiles-trace" log tmux "$evt" "$@" 2>/dev/null || true
}

# window_layout is tmux's own complete description of a window, so recording it
# either side of the reset is what makes a bad one diagnosable afterwards. Name
# last: a session name can contain a space, a layout string cannot.
layouts() {
    tmux list-windows -a -F '#{window_id} #{window_layout} #{session_name}:#{window_index}' 2>/dev/null
}

# session_name last: it is the only field that can contain a space, so `read`
# and awk can split the rest on whitespace (a tab-separated -F is mangled
# outside a UTF-8 locale).
snapshot() {
    local fmt='#{window_id} #{window_width} #{window_height} #{pane_id} #{pane_width}'
    fmt="$fmt #{pane_left} #{pane_height} #{pane_current_command} #{session_name}"
    tmux list-panes -a -F "$fmt" 2>/dev/null
}

session=${1:-$(tmux display-message -p '#{session_name}' 2>/dev/null)}

# Pick up config and code changes. autostart off keeps the plugin from opening a
# sidebar in every session; restart.sh refreshes this one.
if [ -z "$SOCKET" ]; then
    tmux set-option -g @agentbar-autostart off
    tmux source-file "$HOME/.tmux.conf" 2>/dev/null || true
    tmux set-option -gu @agentbar-autostart
    [ -n "$session" ] && bash "$AGENTBAR/scripts/restart.sh" "$session" 2>/dev/null
fi

# After the reload, so a changed @agentbar-width applies on the same press.
width=$(tmux show-option -gqv @agentbar-width)
width=${width:-30}

declare -A before
while read -r win layout _; do before[$win]=$layout; done < <(layouts)

# A sidebar in the wrong window or off the left edge has to be moved, not resized;
# width is left to the pass below. list-panes groups by session, so comparing with
# the previous line skips orphan sidebars.
moved=0
last=""
while read -r win _ wh pane _ left ph cmd sess; do
    [ "$cmd" = agentbar ] || continue
    [ "$sess" = "$last" ] && continue
    last="$sess"
    active=$(tmux display-message -p -t "$sess" '#{window_id}' 2>/dev/null) || continue
    [ "$win" = "$active" ] && [ "$left" = 0 ] && [ "$ph" = "$wh" ] && continue
    tmux join-pane -dhbf -l "$width" -s "$pane" -t "$active" 2>/dev/null && moved=$((moved + 1))
done < <(snapshot)

# Even out every cell of every window, at any split depth - tmux does the tree
# arithmetic, we just ask for it once per pane.
evened=0
while read -r _ _ _ pane _; do
    tmux select-layout -E -t "$pane" 2>/dev/null && evened=$((evened + 1))
done < <(snapshot)

# The sidebar is the one pane that is not equal-width, so its column is set here
# and the other columns share what is left - which is why resized>0 alongside
# changed=0 is normal: this undoes the equal share -E just gave the sidebar. A column is a distinct pane_left (a
# stacked column moves as one), and the rightmost is skipped so it absorbs the
# rounding remainder. -hbf makes the sidebar full height, which is what proves the
# top level is columns rather than rows.
plan() {
    sort -k1,1 -k6,6n | awk -v sw="$width" '
    {
        w = $1; left = $6
        if (!(w in ncol)) { order[++total] = w; ww[w] = $2; wh[w] = $3 }
        if (!((w, left) in seen)) {
            seen[w, left] = 1
            i = ++ncol[w]
            id[w, i] = $4; wd[w, i] = $5; ht[w, i] = $7; cmd[w, i] = $8
        }
    }
    END {
        for (k = 1; k <= total; k++) {
            w = order[k]
            if (ncol[w] < 2) continue
            if (cmd[w, 1] != "agentbar" || ht[w, 1] != wh[w]) continue # no sidebar
            rest = ncol[w] - 1
            avail = ww[w] - (ncol[w] - 1) - sw # each border costs a column
            if (avail < rest) continue
            base = int(avail / rest); rem = avail % rest
            # skip a column already at its target: no tmux call, no redraw
            if (wd[w, 1] != sw) print id[w, 1], sw
            for (i = 2; i < ncol[w]; i++) {
                t = base + (rem-- > 0 ? 1 : 0)
                if (wd[w, i] != t) print id[w, i], t
            }
        }
    }'
}

resized=0
while read -r pane target; do
    tmux resize-pane -t "$pane" -x "$target" 2>/dev/null && resized=$((resized + 1))
done < <(snapshot | plan)

for opt in @dictate @submit_flash @push_flash; do
    tmux set-option -gu "$opt" 2>/dev/null || true
done
while IFS= read -r sess; do
    tmux set-option -t "$sess" -uq @sidebar_moving 2>/dev/null || true
done < <(tmux list-sessions -F '#{session_name}' 2>/dev/null)

tmux refresh-client -S 2>/dev/null || true

# One record per window the reset actually changed, carrying both layouts. In a
# steady state that is nothing, which is what keeps the shared log an edge log.
changed=0
while read -r win layout name; do
    [ "${before[$win]:-}" = "$layout" ] && continue
    changed=$((changed + 1))
    trace layout win="$win" name="$name" before="${before[$win]:-}" after="$layout"
done < <(layouts)

tmux display-message "UI reset: $changed windows changed" 2>/dev/null || true
trace reset session="$session" evened="$evened" resized="$resized" moved="$moved" changed="$changed"

# No resurrect save here: what gets saved stays a deliberate act (prefix + C-s,
# or the autosave), so a reset never overwrites a layout you arranged by hand.
