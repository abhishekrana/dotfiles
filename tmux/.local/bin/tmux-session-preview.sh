#!/usr/bin/env bash
# fzf preview for tmux-session-picker. Shows cwd, git repo+branch,
# aggregated Claude state, and per-pane breakdown.

set -u

# tmux sanitizes tabs in -F output to "_" outside a UTF-8 locale.
export LC_ALL=C.UTF-8

# Glyphs, colors and the state ranking come from the shared language the picker
# and the sidebar use - this pane used to draw its own emoji circles instead.
# Resolved next to this script, not via ~/.local/bin: that path only exists once
# the package is stowed.
# shellcheck source=tmux/.local/bin/tmux-agent-state.sh
. "$(dirname "$(realpath "${BASH_SOURCE[0]}")")/tmux-agent-state.sh"
session=${1:-}
[ -z "$session" ] && exit 0

path=$(tmux display-message -p -t "$session" '#{pane_current_path}' 2>/dev/null)
home_short=${path/#$HOME/\~}
now=$(date +%s)

# Per-pane Claude state from the @agent_* pane options the agentbar
# hook stamps - the same source the sidebar and picker read. @agent_since is
# the unix time of the last state change; a pane counts as a live agent only
# with @agent_present=1 and a claude/node foreground command. Only active
# states are kept, so idle/registered panes show no state line.
pane_fmt=$'#{pane_id}\t#{pane_current_command}\t#{@agent_present}'
pane_fmt+=$'\t#{@agent_state}\t#{@agent_since}'
win_fmt=$'#{window_index}\t#{window_name}\t#{pane_id}\t#{pane_active}\t#{window_active}'

declare -A PANE_STATE PANE_TS
while IFS=$'\t' read -r pid cmd present state since; do
    [ "$present" = 1 ] || continue
    case $cmd in claude | node) ;; *) continue ;; esac
    case $state in working | permission | question | done) ;; *) continue ;; esac
    PANE_STATE[$pid]=$state
    PANE_TS[$pid]=${since:-0}
done < <(tmux list-panes -a -F "$pane_fmt")

fmt_ago() {
    local delta=$((now - $1))
    if [ $delta -lt 60 ]; then
        echo "${delta}s ago"
    elif [ $delta -lt 3600 ]; then
        echo "$((delta / 60))m ago"
    elif [ $delta -lt 86400 ]; then
        echo "$((delta / 3600))h ago"
    else
        echo "$((delta / 86400))d ago"
    fi
}

# ---- Aggregate state across all panes in this session ----------------------
agg_state=
agg_ts=0
panes_in_session=$(tmux list-panes -s -t "$session" -F '#{pane_id}' 2>/dev/null)
for p in $panes_in_session; do
    st=${PANE_STATE[$p]:-}
    [ -z "$st" ] && continue
    rank=$(agent_state_rank "$st")
    best=$(agent_state_rank "$agg_state")
    if [ "$rank" -gt "$best" ]; then
        agg_state=$st
        agg_ts=${PANE_TS[$p]}
    elif [ "$rank" = "$best" ] && [ "${PANE_TS[$p]}" -gt "$agg_ts" ]; then
        agg_ts=${PANE_TS[$p]}
    fi
done

# ---- Header lines ----------------------------------------------------------
echo "dir:     $home_short"

if [ -n "$path" ] && [ -d "$path" ]; then
    # --git-common-dir is stable across worktrees; --show-toplevel would name
    # each worktree as a different repo.
    common=$(git -C "$path" rev-parse --git-common-dir 2>/dev/null)
    if [ -n "$common" ]; then
        [ "${common#/}" = "$common" ] && common="$path/$common"
        common=${common%/.git}
        repo=$(basename "$(readlink -f "$common" 2>/dev/null || echo "$common")")
        branch=$(git -C "$path" symbolic-ref --short HEAD 2>/dev/null ||
            git -C "$path" rev-parse --short HEAD 2>/dev/null)
        echo "repo:    $repo"
        echo "branch:  $branch"
    fi
fi

if [ -n "$agg_state" ]; then
    echo "state:   $(agent_icon "$agg_state") $agg_state ($(fmt_ago "$agg_ts"))"
fi

# ---- Windows + per-pane state ----------------------------------------------
echo
echo "windows:"
# Use tab (#{\t} not supported; tmux passes literal $'\t' through -F if quoted).
# Filter to this session via -t. Fields: window_index, window_name, pane_id,
# pane_active, window_active.
tmux list-panes -s -t "$session" -F "$win_fmt" 2>/dev/null |
    while IFS=$'\t' read -r widx wname pid pactive wactive; do
        marker=' '
        [ "$wactive" = "1" ] && [ "$pactive" = "1" ] && marker='*'
        st=${PANE_STATE[$pid]:-}
        icon=$(agent_icon "$st")
        if [ -n "$st" ]; then
            printf '  %s:%-8s %s  %s %s (%s)\n' "$widx" "$wname" "$marker" "$icon" "$st" "$(fmt_ago "${PANE_TS[$pid]}")"
        else
            printf '  %s:%-8s %s\n' "$widx" "$wname" "$marker"
        fi
    done
