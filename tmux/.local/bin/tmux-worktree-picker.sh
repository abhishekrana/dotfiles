#!/usr/bin/env bash
# fzf popup that points the diff pane at another worktree. Opened by the "◧ diff"
# menu ("Pick worktree…"), and the escape hatch for the case the agent-workdir
# signal cannot cover: several agents writing in several worktrees at once, or
# simply wanting to look somewhere else.
#
# Bands mirror the session picker's language - a "<name> ·<count>" header with
# rows beneath, headers skipped by the cursor:
#   agent      worktrees the agents in this window are writing in (@agent_workdir)
#   worktrees  every other worktree of the source repo
# ◧ marks where the diff pane points right now, and the cursor opens on it.
#
# Keys: j/k/g/G move (headers are stepped over), / filters when the family is
# large, Enter points the diff pane at the highlighted worktree.
#
# Rows are built from ONE `git worktree list --porcelain` call - branches come
# out of it, so the popup opens instantly on a repo with dozens of worktrees.
# Dirty state is per-row in the preview, where it is worth reading and costs
# nothing until you land on the row.
#
# Internal subcommands (fzf callbacks):
#   --list <repo>        emit "dir<TAB>display" rows
#   --preview <dir>      the preview pane for one row
set -uo pipefail

# tmux sanitizes tabs in -F output to "_" outside a UTF-8 locale.
export LC_ALL=C.UTF-8

BAND_MARK='__band__'
self=$(realpath "$0")
# Resolved next to this script, not via ~/.local/bin: that path only exists once
# the package is stowed.
DIFF_PANE="$(dirname "$self")/tmux-diff-pane.sh"

# --- context ----------------------------------------------------------------

# Window the diff pane belongs to, resolved like tmux-diff-pane.sh does: the
# session of the most-recently-active client.
resolve_win() {
    local sess
    sess=$(tmux list-clients -F "#{client_activity}	#{client_session}" 2>/dev/null |
        sort -rn | head -1 | cut -f2-)
    # No client at all (a detached server, or a scripted run): the first session
    # is the only sensible answer, and beats failing.
    [ -n "$sess" ] || sess=$(tmux list-sessions -F '#{session_name}' 2>/dev/null | head -1)
    tmux display-message -p -t "$sess" '#{window_id}' 2>/dev/null
}

# Every worktree the agents in this window have been writing in, most recent
# first, deduped. @agent_workdirs is the pane's recent list (pipe-wrapped, newest
# first) - one Claude often spans three worktrees in a turn, and all three are
# worth offering.
agent_dirs() {
    tmux list-panes -t "$1" -F '#{@agent_workdirs}' 2>/dev/null |
        tr '|' '\n' | awk 'NF && !seen[$0]++'
}

# The repo whose worktrees to offer: the diff pane's current target, else any
# agent's workdir, else the active pane's cwd.
source_repo() {
    local win="$1" d
    d=$(tmux show -wqv -t "$win" @diff_target)
    [ -d "${d:-}" ] || d=$(agent_dirs "$win" | head -1)
    [ -d "${d:-}" ] || d=$(tmux display-message -p -t "$win" '#{pane_current_path}' 2>/dev/null)
    printf '%s' "$d"
}

# --- rows -------------------------------------------------------------------

cmd_list() {
    local repo="$1" win="$2" target
    target=$(tmux show -wqv -t "$win" @diff_target)

    # dir<TAB>branch for every non-bare worktree, from a single git call.
    local -a wt=()
    mapfile -t wt < <(git -C "$repo" worktree list --porcelain 2>/dev/null |
        awk '/^worktree /{d=$2; b=""; bare=0}
             /^bare/{bare=1}
             /^branch /{sub("refs/heads/", "", $2); b=$2}
             /^$/{if (d != "" && !bare) print d "\t" (b == "" ? "detached" : b); d=""}
             END{if (d != "" && !bare) print d "\t" (b == "" ? "detached" : b)}')

    branch_of() {
        local e
        for e in "${wt[@]}"; do
            [ "${e%%	*}" = "$1" ] && {
                printf '%s' "${e#*	}"
                return
            }
        done
        git -C "$1" branch --show-current 2>/dev/null
    }

    band() { printf '%s\t%s ·%s\n' "$BAND_MARK" "$1" "$2"; }
    row() { # row <dir> <branch>
        local mark='  '
        [ "$1" = "$target" ] && mark='◧ '
        printf '%s\t%s%-14s %s\n' "$1" "$mark" "$(basename "$1")" "$2"
    }

    local -a agents=()
    mapfile -t agents < <(agent_dirs "$win")
    if [ ${#agents[@]} -gt 0 ]; then
        band agent ${#agents[@]}
        local d
        for d in "${agents[@]}"; do row "$d" "$(branch_of "$d")"; done
    fi

    # A set, not a joined string: "${agents[*]}" joins on a space, so a
    # tab-delimited membership test silently missed every dir but the first - and
    # the recent list routinely holds three.
    local -A shown=()
    local d e
    for d in "${agents[@]}"; do shown[$d]=1; done
    local -a others=()
    for e in "${wt[@]}"; do
        d=${e%%	*}
        [ -n "${shown[$d]:-}" ] && continue
        others+=("$e")
    done
    if [ ${#others[@]} -gt 0 ]; then
        band worktrees ${#others[@]}
        # -V so wt-2 sorts before wt-10.
        printf '%s\n' "${others[@]}" | sort -V | while IFS=$'\t' read -r d b; do row "$d" "$b"; done
    fi
}

# --- preview ----------------------------------------------------------------

cmd_preview() {
    local dir="$1"
    [ "$dir" = "$BAND_MARK" ] && return 0
    [ -d "$dir" ] || return 0
    local n
    n=$(git -C "$dir" status --porcelain 2>/dev/null | grep -c .)
    if [ "$n" -eq 0 ]; then
        printf 'clean\n\n'
    else
        printf '%s changed\n' "$n"
        git -C "$dir" status --short 2>/dev/null | head -6
        printf '\n'
    fi
    git -C "$dir" log --oneline --no-decorate -3 2>/dev/null
}

# --- entry ------------------------------------------------------------------

case "${1:-}" in
    --list)
        cmd_list "$2" "$3"
        exit 0
        ;;
    --preview)
        cmd_preview "${2:-}"
        exit 0
        ;;
esac

win=$(resolve_win)
[ -n "$win" ] || {
    printf 'no window\n' >&2
    exit 1
}
repo=$(source_repo "$win")
git -C "${repo:-/}" rev-parse --git-dir >/dev/null 2>&1 || {
    printf '\n  not a git repo: %s\n' "${repo:-?}" >&2
    sleep 1.2
    exit 1
}

# Same popup language as the session picker: no input line, j/k/g/G motion that
# steps over band headers, shared colours. --with-nth=2 hides the path column,
# which {1} still carries to the preview and to enter's guard.
# shellcheck source=tmux/.local/bin/tmux-agent-state.sh
. "$(dirname "$self")/tmux-agent-state.sh"

lines=$("$self" --list "$repo" "$win")
# Open on the worktree the pane is showing, else the first row that is not a band
# header (row 1 always is one).
start=$(printf '%s\n' "$lines" |
    awk -F'\t' -v t="$(tmux show -wqv -t "$win" @diff_target)" '$1 == t {print NR; f = 1; exit}
        END {if (!f) print 2}')

# Parenthesized transform form, always: `transform:` swallows the rest of the
# --bind string and silently eats every binding after it.
skip_down="transform([ {1} = $BAND_MARK ] && echo down || true)"
skip_up="transform([ {1} = $BAND_MARK ] && { [ \"\$FZF_POS\" -gt 1 ] && echo up || echo down; })"

# --height/--border are set explicitly to override FZF_DEFAULT_OPTS, which the
# popup inherits from the server's environment: its --height=80% left a band of
# dead space inside the popup, and its --border drew a second frame inside tmux's
# own titled one. The preview is a share of the height, so both halves grow with
# the popup instead of the list staying four rows tall on a big screen.
pick=$(printf '%s\n' "$lines" |
    fzf --ansi --sync --reverse --no-input --highlight-line --no-multi \
        --height=100% --border=none \
        --header='/ to filter' --header-first \
        --delimiter=$'\t' --with-nth=2 --pointer=' ' \
        --color="$_popup_fzf_color" \
        --preview "$self --preview {1}" \
        --preview-window='down,45%,border-top' \
        --bind "start:pos($start)" \
        --bind "j:down+$skip_down+$skip_down,k:up+$skip_up+$skip_up" \
        --bind "g:first+$skip_down+$skip_down,G:last+$skip_up+$skip_up" \
        --bind '/:toggle-input' \
        --bind "enter:transform([ {1} = $BAND_MARK ] && echo ignore || echo accept)" |
    cut -f1)

[ -n "${pick:-}" ] || exit 0
[ "$pick" = "$BAND_MARK" ] && exit 0

mode=$(tmux show -wqv -t "$win" @diff_mode)
[ -n "$mode" ] || mode=work
"$DIFF_PANE" "$mode" "$pick"
