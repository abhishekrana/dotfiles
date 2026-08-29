#!/usr/bin/env bash
# Guards the ≡ workdesk toggle: one command opens the float and the same command closes
# it. That is the whole point of the float - a popup is an overlay and eats the second
# click on the chip, so the chip could only ever open.
#
# The float here runs a stub named `workdesk`, which is what the toggle matches on. tmux
# runs on a private socket and nothing touches the live server.
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TOGGLE="$REPO/tmux/.local/bin/tmux-workdesk.sh"
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
eq() { [ "$2" = "$3" ] && ok "$1" || no "$1" "want [$2] got [$3]"; }
# near <name> <want> <got> - equal to within a cell, which is all a border rounds to
near() {
    local d=$(($2 - $3))
    [ "$d" -lt 0 ] && d=$((-d))
    [ "$d" -le 1 ] && ok "$1" || no "$1" "want [$2] got [$3], more than a cell apart"
}

TMP=$(mktemp -d)
TMUX_BIN=$(command -v tmux)
SOCK="workdesk-test-$$"
export PATH="$TMP/shim:$PATH"
export DOTFILES_TRACE=0
export LC_ALL=C.UTF-8
mkdir -p "$TMP/shim"

# Every bare `tmux` the toggle runs lands on the private socket.
cat >"$TMP/shim/tmux" <<EOF
#!/bin/sh
exec "$TMUX_BIN" -L "$SOCK" "\$@"
EOF
chmod +x "$TMP/shim/tmux"

# The float's command. Named workdesk because that name is what the toggle looks for.
cat >"$TMP/shim/workdesk" <<'EOF'
#!/bin/sh
sleep 300
EOF
chmod +x "$TMP/shim/workdesk"

t() { "$TMUX_BIN" -L "$SOCK" "$@"; }
cleanup() {
    t kill-server 2>/dev/null
    rm -rf "$TMP"
}
trap cleanup EXIT

floats() { t list-panes -t "$1" -F '#{pane_floating_flag}' 2>/dev/null | grep -c '^1$'; }

t -f /dev/null new-session -d -s w -x 120 -y 40 'sleep 300'
t set -g @workdesk_open "new-pane -x 90% -y 85% -X 5% -Y 7% '$TMP/shim/workdesk'"
sleep 0.3

echo
echo "workdesk: the chip's command opens the float and closes it"
"$TOGGLE"
sleep 0.5
eq "one float after the first run" "1" "$(floats w:0)"
"$TOGGLE"
sleep 0.5
eq "none after the second" "0" "$(floats w:0)"
eq "the window it floated over is untouched" "1" "$(t list-panes -t w:0 -F '#{pane_id}' | wc -l)"

echo
echo "workdesk: one float per window"
"$TOGGLE"
sleep 0.5
t new-window -d -t w: 'sleep 300'
sleep 0.3
# The toggle reads the current window, which the new -d window did not become.
eq "the first window still has its float" "1" "$(floats w:0)"
eq "the new window has none" "0" "$(floats w:1)"

echo
echo "workdesk: the float lands in the same place every time"
# the case above left one up
[ "$(floats w:0)" = 1 ] && {
    "$TOGGLE"
    sleep 0.3
}
"$TOGGLE"
sleep 0.5
read -r left top pw ph < <(t list-panes -t w:0 -f '#{pane_floating_flag}' \
    -F '#{pane_left} #{pane_top} #{pane_width} #{pane_height}')
ww=$(t display-message -p -t w:0 '#{window_width}')
wh=$(t display-message -p -t w:0 '#{window_height}')
near "centred across the window" "$(((ww - pw) / 2))" "$left"
near "centred down the window" "$(((wh - ph) / 2))" "$top"
"$TOGGLE"
sleep 0.3
# With no -X/-Y tmux cascades: 4 columns right and 2 rows down on every open, so this
# is the assertion that the offsets are still there.
for _ in 1 2 3 4; do
    "$TOGGLE"
    sleep 0.4
    "$TOGGLE"
    sleep 0.3
done
"$TOGGLE"
sleep 0.5
eq "the sixth open is where the first was" "$left $top" \
    "$(t list-panes -t w:0 -f '#{pane_floating_flag}' -F '#{pane_left} #{pane_top}')"
"$TOGGLE"

echo
printf 'workdesk: %d passed, %d failed\n\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
