#!/usr/bin/env bash
# Guards the UI reset (prefix+R -> tmux-reset.sh): the sidebar must come back to
# its exact width at the left edge, the remaining columns to an even split, and
# stuck status chips must clear. The regressions are arithmetic (a miscounted
# border, the rounding remainder applied twice, a nested window flattened) and a
# two-column skew looks fine by eye.
#
# A copy of `sleep` named `agentbar` reports as the sidebar through
# pane_current_command, which is all the script keys on. Runs on private sockets
# (tmux -S), never the live server; RESET_SOCKET also skips the reload, the
# sidebar restart and the resurrect save.
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RESET="$REPO/tmux/.local/bin/tmux-reset.sh"
pass=0 fail=0

ok() {
    pass=$((pass + 1))
    printf '  \033[32m✓\033[0m %s\n' "$1"
}
no() {
    fail=$((fail + 1))
    printf '  \033[31m✗\033[0m %s\n' "$1"
    [ $# -gt 1 ] && printf '      %s\n' "$2"
}

# eq <name> <want> <got>
eq() { [ "$2" = "$3" ] && ok "$1" || no "$1" "want [$2] got [$3]"; }

TMP=$(mktemp -d)
SOCK="$TMP/sock"
FAKE="$TMP/agentbar"
cp /usr/bin/sleep "$FAKE"

t() { tmux -S "$SOCK" "$@"; }
cleanup() {
    t kill-server 2>/dev/null
    rm -rf "$TMP"
}
trap cleanup EXIT

start() { # start <session>
    t kill-server 2>/dev/null
    t -f /dev/null new-session -d -s "$1" -x 200 -y 40
    t set -g @agentbar-width 30
}

# sidebar <target> - a leftmost full-height pane that reports as agentbar
sidebar() {
    t split-window -d -hbf -l 30 -t "$1" "$FAKE 60"
    sleep 0.3
}

# DOTFILES_TRACE=0: a test run must not land in the live trace log.
reset_ui() { RESET_SOCKET="$SOCK" DOTFILES_TRACE=0 bash "$RESET" >/dev/null 2>&1; }

# pane <window> <n> - nth pane id from the left
pane() { t list-panes -t "$1" -F '#{pane_left} #{pane_id}' | sort -n | sed -n "$2p" | cut -d' ' -f2; }

# widths <window> - pane widths left to right
widths() {
    t list-panes -t "$1" -F '#{pane_left} #{pane_width}' | sort -n | cut -d' ' -f2 | tr '\n' ' ' | sed 's/ $//'
}

echo
echo "reset: even split with a sidebar"
start a
t split-window -h -t a
sidebar a
t resize-pane -t "$(pane a 2)" -x 140
skewed=$(widths a)
reset_ui
# 200 cols - 2 borders - 30 sidebar = 168, even over the two content panes
eq "sidebar 30, content 84|84 (was $skewed)" "30 84 84" "$(widths a)"

echo
echo "reset: even split with no sidebar"
start b
t split-window -h -t b
t split-window -h -t b
t resize-pane -t "$(pane b 1)" -x 20
reset_ui
# 200 cols - 2 borders = 198 over three panes
eq "content 66|66|66" "66 66 66" "$(widths b)"

echo
echo "reset: sidebar re-homed from another window"
start c
other=$(t new-window -d -P -F '#{window_id}' -t c:)
sidebar "$other"
active=$(t display-message -p -t c '#{window_id}')
reset_ui
side=$(t list-panes -a -F '#{pane_id} #{pane_current_command}' | awk '$2 == "agentbar" {print $1}')
eq "landed in the active window" "$active" "$(t display-message -p -t "$side" '#{window_id}')"
eq "left edge, 30 wide" "0 30" "$(t display-message -p -t "$side" '#{pane_left} #{pane_width}')"
eq "spans the window height" "$(t display-message -p -t "$side" '#{window_height}')" \
    "$(t display-message -p -t "$side" '#{pane_height}')"

echo
echo "reset: nested splits are evened, structure kept"
start d
sidebar d
t split-window -v -d -t "$(pane d 2)" # right column stacked, so the row is not flat
t split-window -v -d -t "$(pane d 2)"
t resize-pane -t "$(pane d 1)" -x 8
before=$(t list-panes -t d -F '#{pane_id}' | wc -l)
reset_ui
eq "no pane lost" "$before" "$(t list-panes -t d -F '#{pane_id}' | wc -l)"
eq "sidebar width still repaired" "30" \
    "$(t list-panes -t d -F '#{pane_current_command} #{pane_width}' | awk '$1 == "agentbar" {print $2}')"
eq "stack kept (three panes share a column)" "3" \
    "$(t list-panes -t d -F '#{pane_left}' | sort -n | uniq -c | awk '$1 == 3 {print $1}')"
# spread evenly means no two panes in the stack differ by more than a row
eq "stack heights evened" "ok" "$(t list-panes -t d -F '#{pane_left} #{pane_height}' |
    awk '$1 != 0 {if (!n++ || $2 < lo) lo = $2; if (n == 1 || $2 > hi) hi = $2}
         END {print (hi - lo <= 1) ? "ok" : "spread " lo "-" hi}')"

echo
echo "reset: layouts with no sidebar"
start e
reset_ui
eq "single pane untouched" "200" "$(t list-panes -t e -F '#{pane_width}')"
t split-window -v -t e
t resize-pane -t "$(t list-panes -t e -F '#{pane_top} #{pane_id}' | sort -n | sed -n 1p | cut -d' ' -f2)" -y 3
reset_ui
eq "vertical split evened" "ok" "$(t list-panes -t e -F '#{pane_height}' |
    awk '{if (!n++ || $1 < lo) lo = $1; if (n == 1 || $1 > hi) hi = $1}
         END {print (hi - lo <= 1) ? "ok" : "spread " lo "-" hi}')"

echo
echo "reset: stuck status chips clear"
start f
t set -g @dictate rec
t set -g @submit_flash 1
t set -g @push_flash 1
t set -t f -q @sidebar_moving 1
reset_ui
eq "@dictate cleared" "" "$(t show-option -gqv @dictate)"
eq "@submit_flash cleared" "" "$(t show-option -gqv @submit_flash)"
eq "@push_flash cleared" "" "$(t show-option -gqv @push_flash)"
eq "@sidebar_moving cleared" "" "$(t show-option -t f -qv @sidebar_moving)"

echo
printf 'reset: %d passed, %d failed\n\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
