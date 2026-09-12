#!/usr/bin/env bash
# PostToolUse hook: record the worktree Claude is writing in, for the status line.
#
# Claude Code tells the status line where the session's cwd is, never which file
# Claude just wrote - and the two differ exactly when it matters, since an Edit
# by absolute path moves no cwd. The edited file's repo root is written to
#
#     ${XDG_STATE_HOME:-~/.local/state}/dotfiles/claude-workdir/<session id>
#
# which statusline-command.sh reads. One file per session, dropped after a week.
# Nothing here touches tmux: the status line works wherever Claude runs.
set -u

here=${BASH_SOURCE[0]%/*}
[ "$here" = "${BASH_SOURCE[0]}" ] && here=.
# shellcheck source=statusline-git.bash
. "$here/statusline-git.bash"

IFS=$'\x1f' read -r sid tool path npath <<<"$(jq -r '[
    (.session_id // ""),
    (.tool_name // ""),
    (.tool_input.file_path // ""),
    (.tool_input.notebook_path // "")
] | map(tostring) | join("")')"

# The tools that change a file on disk. A cwd is not a write.
case $tool in
    Edit | Write | MultiEdit | NotebookEdit) ;;
    *) exit 0 ;;
esac

[ -n "$path" ] || path=$npath
# A relative path is not resolvable from here, and the id has to name a file.
case $path in /*) ;; *) exit 0 ;; esac
case $sid in '' | *[!A-Za-z0-9_-]*) exit 0 ;; esac

dir=$path
[ -d "$dir" ] || dir=${dir%/*}
git_place "$dir" || exit 0

state=${XDG_STATE_HOME:-$HOME/.local/state}/dotfiles/claude-workdir
mkdir -p "$state" 2>/dev/null || exit 0
printf '%s' "$place_root" >"$state/$sid" 2>/dev/null

find "$state" -maxdepth 1 -type f -mtime +7 -delete 2>/dev/null
exit 0
