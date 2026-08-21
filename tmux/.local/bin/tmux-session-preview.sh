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

now=$(date +%s)

# Per-pane Claude state from the @agent_* pane options the agentbar
# hook stamps - the same source the sidebar and picker read. @agent_since is
# the unix time of the last state change; a pane counts as a live agent only
# with @agent_present=1 and a claude/node foreground command. Only active
# states are kept, so idle/registered panes show no state line.
# Optional fields carry a "-" placeholder: tab is IFS whitespace, so bash
# collapses a run of empty ones and every field after them shifts left.
pane_fmt=$'#{session_name}\t#{pane_id}\t#{pane_current_command}\t#{?@agent_present,1,0}'
pane_fmt+=$'\t#{?@agent_state,#{@agent_state},-}\t#{?@agent_since,#{@agent_since},-}'
pane_fmt+=$'\t#{?pane_title,#{pane_title},-}\t#{host}'
pane_fmt+=$'\t#{?@agent_workdir,#{@agent_workdir},-}\t#{pane_current_path}'
pane_fmt+=$'\t#{window_index}\t#{pane_index}'
# One line per window, from tmux's own window list. The flags are spelled out
# rather than taken from #{window_flags}: that one carries activity and silence
# too, which the status line deliberately ignores (monitor-activity is on, so #
# would be on every row), and it escapes # as ## for re-parsing. Current, zoomed
# and bell are what the status line shows, so they are what this shows.
# The flags go last because they are the field that can be empty: tab is IFS
# whitespace, so an empty one in the middle collapses and shifts the rest left.
win_fmt=$'#{window_index}\t#{window_name}\t#{window_panes}'
win_fmt+=$'\t#{?window_active,*,}#{?window_zoomed_flag,Z,}#{?window_bell_flag,!,}'

declare -A PANE_STATE PANE_TS PANE_AGENT PANE_TITLE PANE_PANE DIR_VOTES
vote_dir='' agent_dir='' vote_best=0
while IFS=$'\t' read -r sess pid cmd present state since title host wd ppath widx pidx; do
    [ "$state" = - ] && state=
    [ "$since" = - ] && since=
    [ "$title" = - ] && title=
    [ "$wd" = - ] && wd=
    # The directory that names this session: where most of its panes sit, not
    # its active pane - that is usually the sidebar, whose cwd is only wherever
    # that process started, and every session then reported the same repo. The
    # picker's rows resolve it the same way; the sidebar is excluded outright.
    if [ "$sess" = "$session" ] && [ "$cmd" != agentbar ] && [ -n "$ppath" ]; then
        DIR_VOTES[$ppath]=$((${DIR_VOTES[$ppath]:-0} + 1))
        if [ "${DIR_VOTES[$ppath]}" -gt "$vote_best" ]; then
            vote_best=${DIR_VOTES[$ppath]}
            vote_dir=$ppath
        fi
    fi
    [ "$present" = 1 ] || continue
    case $cmd in claude | node) ;; *) continue ;; esac
    # An agent's worktree wins: it is where the work is, and a pane's cwd never
    # follows it.
    [ "$sess" = "$session" ] && [ -n "$wd" ] && [ -z "$agent_dir" ] && agent_dir=$wd
    # Every registered agent, idle included, for the agents block below.
    PANE_AGENT[$pid]=${state:-idle}
    PANE_TITLE[$pid]=$(agent_title "$title" "$host")
    # tmux's own notation for a pane: window.pane, as `-t` takes it.
    PANE_PANE[$pid]="$widx.$pidx"
    case $state in working | permission | question | done) ;; *) continue ;; esac
    PANE_STATE[$pid]=$state
    PANE_TS[$pid]=${since:-0}
done < <(tmux list-panes -a -F "$pane_fmt")

path=${agent_dir:-$vote_dir}
home_short=${path/#$HOME/\~}

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
        # A worktree of a bare clone puts the common dir in .bare/.git, whose
        # basename names neither the repo nor anything useful - its parent does.
        case $repo in .bare | .git | .*) repo=$(basename "$(dirname "$common")") ;; esac
        branch=$(git -C "$path" symbolic-ref --short HEAD 2>/dev/null ||
            git -C "$path" rev-parse --short HEAD 2>/dev/null)
        echo "repo:    $repo"
        echo "branch:  $branch"
    fi
fi

if [ -n "$agg_state" ]; then
    echo "state:   $(agent_icon "$agg_state") $agg_state ($(fmt_ago "$agg_ts"))"
fi

# ---- Agents: what each Claude here is on -----------------------------------
# The row above shows one title, the most urgent agent's, with "+N" for the
# rest; this is the N. Idle agents included, since the row's count counts them.
for p in $panes_in_session; do
    [ -n "${PANE_AGENT[$p]:-}" ] || continue
    if [ -z "${agents_shown:-}" ]; then
        echo
        echo "agents:"
        agents_shown=1
    fi
    ttl=${PANE_TITLE[$p]:-}
    [ -n "$ttl" ] || ttl=$(agent_muted "not titled yet")
    tail="${PANE_AGENT[$p]}"
    [ -n "${PANE_TS[$p]:-}" ] && tail="$tail · $(fmt_ago "${PANE_TS[$p]}")"
    printf '  %s %s  %s\n' "$(agent_icon "${PANE_AGENT[$p]}")" "$ttl" \
        "$(agent_muted "$tail · pane ${PANE_PANE[$p]}")"
done

# ---- Windows ---------------------------------------------------------------
# One line per window, not per pane: this block used to iterate panes under a
# "windows:" heading, so a window with three panes appeared three times. Panes
# are the splits inside a window; what each agent is doing is the block above.
echo
echo "windows:"
tmux list-windows -t "$session" -F "$win_fmt" 2>/dev/null |
    while IFS=$'\t' read -r widx wname wpanes wflags; do
        [ "$wpanes" = 1 ] && unit=pane || unit=panes
        printf '  %s:%-10s %-2s %s\n' "$widx" "$wname" "$wflags" \
            "$(agent_muted "$wpanes $unit")"
    done
