#!/usr/bin/env bash
# tmux-rail.sh <dir> [pane-width] [zones] -- one zone of a pane's border rail:
#
#     <worktree>  ⎇ <branch>
#
# The worktree name, never truncated: with a dozen checkouts of one repo side by
# side it is the identity you are looking for. The branch is what gives, capped
# from the right so a ticket number at the head survives.
#
# The cap is the pane's own width divided by the number of zones this rail draws
# (2 while an agent's worktree sits on the right, 1 otherwise), less the rail's
# furniture - a fixed cap cut a 30-character branch on a 100-column pane that
# had rule to spare. Called with no width it falls back to 22.
#
# Called from pane-border-format via #(), once per zone per pane. tmux reruns a
# #() job at most once a second and keys its cache on the command string - which
# contains the directory - so a pane that has not moved costs nothing, and a
# fresh directory costs one `git branch --show-current` (~2ms). The memo below
# collapses even that for the repeated case: several panes in one worktree, and
# the left and right zones of the same rail, share one lookup.
#
# Prints nothing at all when the directory is not a checkout, so a rail on a
# plain directory is just rule.
set -u

# --branch: the branch alone, uncapped - the footer shows the full name the rails
# have to truncate.
if [ "${1:-}" = --branch ]; then
    shift
    only_branch=1
else
    only_branch=0
fi

dir=${1:-}
width=${2:-0}
zones=${3:-1}
case $width in '' | *[!0-9]*) width=0 ;; esac
case $zones in 1 | 2) ;; *) zones=1 ;; esac
# The rail's own furniture: 4 columns per zone (a space each side, the "→ " on
# the right one), and 2 of rule that stay between them at every width.
reserve=$((4 * zones + 2))
[ -n "$dir" ] && [ -d "$dir" ] || exit 0

memo=${XDG_RUNTIME_DIR:-${TMPDIR:-/tmp}}/dotfiles-rail-$(id -u)
mkdir -p "$memo" 2>/dev/null || true
key=$memo/$(printf '%s-%s-%s-%s' "$dir" "$width" "$zones" "$only_branch" | tr -c 'A-Za-z0-9' '_')

# 3 seconds: long enough that a redraw storm reads the file, short enough that a
# branch switch shows up while you are still looking at the pane.
if [ -f "$key" ]; then
    now=$(date +%s)
    at=$(stat -c %Y "$key" 2>/dev/null || echo 0)
    if [ $((now - at)) -lt 3 ]; then
        cat "$key"
        exit 0
    fi
fi

root=$(git -C "$dir" rev-parse --show-toplevel 2>/dev/null) || exit 0
[ -n "$root" ] || exit 0
branch=$(git -C "$dir" branch --show-current 2>/dev/null)
[ -n "$branch" ] || branch=$(git -C "$dir" rev-parse --short HEAD 2>/dev/null)

if [ "$only_branch" = 1 ]; then
    label=$branch
else
    label=$(basename "$root")
    if [ -n "$branch" ]; then
        # What the zone has left once the name and the " ⎇ " are in. A stub of a
        # branch still says which one it is; nothing at all does not.
        if [ "$width" -gt 0 ]; then
            cap=$(((width - reserve) / zones - ${#label} - 4))
            [ "$cap" -lt 6 ] && cap=6
        else
            cap=22
        fi
        [ ${#branch} -gt "$cap" ] && branch="${branch:0:$((cap - 1))}…"
        label="$label  ⎇ $branch"
    fi
fi

printf '%s' "$label" | tee "$key" 2>/dev/null || printf '%s' "$label"
