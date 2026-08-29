#!/usr/bin/env bash
# Alt+n and the ▤ workdesk chip: raise the workdesk float, or close the one already up.
#
# The float is a pane (tmux 3.7 `new-pane`), not a popup, and that is what makes the
# chip a toggle: a popup is an overlay and swallows every click outside its own box, so
# the second click on the chip never reached tmux at all. Probed on both - a status
# click lands with a float open and is dropped with a popup open.
#
# One window, one float: the search is this window's panes, so a second window opens
# its own.
set -uo pipefail

TRACE="$HOME/.local/bin/dotfiles-trace"

# The float is ours if it is floating and its command is the workdesk binary - true for
# the mockup's fixture override too, which runs the same binary on a fake mirror.
pane=$(tmux list-panes -F '#{pane_id} #{pane_floating_flag} #{pane_start_command}' 2>/dev/null |
    awk '!found && $2 == 1 && index($0, "workdesk") { print $1; found = 1 }')

if [ -n "$pane" ]; then
    tmux kill-pane -t "$pane" 2>/dev/null
    "$TRACE" log tmux workdesk action=close pane="$pane" rc=$?
    exit 0
fi

open=$(tmux show-option -gqv @workdesk_open)
if [ -z "$open" ]; then
    "$TRACE" log tmux workdesk action=open rc=1 err=no_command
    exit 0
fi
# eval: the option is a tmux command line carrying its own quoting, the same string the
# binding used to hand to sh. It is set by this repo's config, never by input.
eval "tmux $open" >/dev/null 2>&1
"$TRACE" log tmux workdesk action=open rc=$?
