#!/usr/bin/env bash
# One agent-panel row per subagent: name, worktree, state, context share.
#
#     refactorer ⎇ worktree-x  running  8%
#
# A subagent with `isolation: worktree` runs in a checkout the session never
# enters, so its own cwd is the only place that names it. One JSON line per row;
# a task left unnamed keeps its default rendering.
set -u

here=${BASH_SOURCE[0]%/*}
[ "$here" = "${BASH_SOURCE[0]}" ] && here=.
# shellcheck source=statusline-git.bash
. "$here/statusline-git.bash"

dim=$'\033[2m'
reset=$'\033[0m'

# US, not tab: bash folds runs of IFS whitespace, so a task with no cwd would
# shift every field after it.
while IFS=$'\x1f' read -r id name status cwd tokens window; do
    [ -n "$id" ] || continue

    branch=
    if git_place "$cwd" && [ -n "$place_branch" ]; then
        branch=" ⎇ ${place_branch}"
    fi

    pct=
    case $window in
        '' | 0 | *[!0-9]*) ;;
        *) pct="  $((tokens * 100 / window))%" ;;
    esac

    jq -cn --arg id "$id" \
        --arg content "${name}${branch}  ${dim}${status}${reset}${pct}" \
        '{id: $id, content: $content}'
done < <(jq -r '.tasks[]? | [
    .id // "",
    .name // "",
    .status // "",
    .cwd // "",
    .tokenCount // 0,
    .contextWindowSize // 0
] | map(tostring) | join("\u001f")')
