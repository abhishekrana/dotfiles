#!/usr/bin/env bash
# Claude Code status line: model, context, place, and the agent's worktree when
# it differs from the session's.
#
#     Opus | ctx: 12% used | repo ⎇ main | ⚠ other ⎇ feature
#
# @agent_workdir is the pane option the agentbar hook stamps on every write; an
# Edit by absolute path moves neither the cwd nor this row. No tmux, no warning.
set -u

here=${BASH_SOURCE[0]%/*}
[ "$here" = "${BASH_SOURCE[0]}" ] && here=.
# shellcheck source=statusline-git.bash
. "$here/statusline-git.bash"

input=$(cat)
# US, not tab: bash folds runs of IFS whitespace, so an absent field would shift
# every field after it.
IFS=$'\x1f' read -r model used dir <<<"$(jq -r '[
    .model.display_name // "",
    (.context_window.used_percentage // ""),
    (.workspace.current_dir // .cwd // "")
] | map(tostring) | join("\u001f")' <<<"$input")"

warn=$'\033[33m'
reset=$'\033[0m'

# <name> ⎇ <branch>, the pane rail's words; the bare directory when not a checkout.
place() {
    local name
    git_place "$1" || {
        printf '%s' "${1##*/}"
        return
    }
    # shellcheck disable=SC2154  # set by git_place
    name=${place_root##*/}
    [ -n "$place_branch" ] || {
        printf '%s' "$name"
        return
    }
    printf '%s ⎇ %s' "$name" "$place_branch"
}

parts=()
[ -n "$model" ] && parts+=("$model")
[ -n "$used" ] && parts+=("ctx: ${used}% used")
[ -n "$dir" ] && parts+=("$(place "$dir")")

if [ -n "${TMUX_PANE:-}" ] && command -v tmux >/dev/null 2>&1; then
    agent_dir=$(tmux show-options -pqv -t "$TMUX_PANE" @agent_workdir 2>/dev/null)
    session_root=
    git_place "$dir" && session_root=$place_root
    agent_root=
    # Roots, not paths: a subdirectory of the session's own checkout is not a move.
    [ -n "$agent_dir" ] && git_place "$agent_dir" && agent_root=$place_root
    if [ -n "$agent_root" ] && [ "$agent_root" != "$session_root" ]; then
        parts+=("${warn}⚠ $(place "$agent_root")${reset}")
    fi
fi

printf '%s' "$(
    IFS='|'
    echo "${parts[*]}" | sed 's/|/ | /g'
)"
