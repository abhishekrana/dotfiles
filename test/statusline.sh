#!/usr/bin/env bash
# Guards the Claude status line: what it says about place, and when it warns.
#
# The row's whole job is to name two things - where the session sits, and the
# worktree Claude is writing in when that differs. Both have failed silently
# before: an absent field shifted every field after it, a subdirectory read as a
# move, a task with no `name` rendered as its branch alone, and the warning
# could not fire at all where $TMUX_PANE was unset.
#
# Everything runs with tmux off PATH, which is the point: nothing in this row
# may depend on a multiplexer.
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
pass=0 fail=0

command -v jq >/dev/null 2>&1 || {
    echo "jq missing; skipping" >&2
    exit 0
}

ok() {
    pass=$((pass + 1))
    printf '  \033[32m✓\033[0m %s\n' "$1"
}
no() {
    fail=$((fail + 1))
    printf '  \033[31m✗\033[0m %s\n' "$1"
    printf '      %s\n' "$2"
}
eq() { [ "$2" = "$3" ] && ok "$1" || no "$1" "want [$2] got [$3]"; }

TMP=$(mktemp -d)
STATE=$TMP/state/dotfiles/claude-workdir
SID=test-session
trap 'rm -rf "$TMP"' EXIT

# A repo with a linked worktree, so the .git-as-a-file path is exercised too.
MAIN=$TMP/main
git init -q -b main "$MAIN"
git -C "$MAIN" config user.email t@t
git -C "$MAIN" config user.name t
: >"$MAIN/f"
git -C "$MAIN" add f
git -C "$MAIN" commit -qm init
WT=$TMP/side
git -C "$MAIN" worktree add -q -b side "$WT"

# tmux off PATH: the row must not reach for one.
row() {
    local dir=${1:-$MAIN} used=${2-12}
    printf '{"model":{"display_name":"Opus"},"session_id":"%s","workspace":{"current_dir":"%s"}%s}' \
        "$SID" "$dir" "${used:+,\"context_window\":{\"used_percentage\":$used}}" |
        env XDG_STATE_HOME="$TMP/state" PATH=/usr/bin:/bin \
            bash "$REPO/claude/.claude/statusline-command.sh" | sed 's/\x1b\[[0-9;]*m//g'
}

# One PostToolUse event, as Claude Code sends it.
wrote() {
    printf '{"session_id":"%s","tool_name":"%s","tool_input":{"%s":"%s"}}' \
        "$SID" "${2:-Write}" "${3:-file_path}" "$1" |
        env XDG_STATE_HOME="$TMP/state" bash "$REPO/claude/.claude/statusline-workdir.sh"
}

rows() {
    printf '{"tasks":[%s]}' "$1" |
        env XDG_STATE_HOME="$TMP/state" bash "$REPO/claude/.claude/subagent-statusline.sh" |
        jq -r '.content' | sed 's/\x1b\[[0-9;]*m//g'
}

echo "status line: place"

eq "names the checkout and its branch" "Opus | ctx: 12% used | main ⎇ main" "$(row)"
eq "a linked worktree reads its own branch" "Opus | ctx: 12% used | side ⎇ side" "$(row "$WT")"
eq "a directory in no repo is just its name" "Opus | ctx: 12% used | state" "$(row "$TMP/state")"
# An absent percentage used to collapse into the next field and carry the path
# into the context segment.
eq "an absent field shifts nothing" "Opus | main ⎇ main" "$(row "$MAIN" "")"

echo "status line: the second place"

eq "silent before anything is written" "Opus | ctx: 12% used | main ⎇ main" "$(row)"

wrote "$WT/f"
eq "names the worktree Claude wrote in" \
    "Opus | ctx: 12% used | main ⎇ main | ⚠ side ⎇ side" "$(row)"

mkdir -p "$MAIN/sub"
: >"$MAIN/sub/f"
wrote "$MAIN/sub/f"
# Roots are compared, never paths.
eq "a subdirectory is not a move" "Opus | ctx: 12% used | main ⎇ main" "$(row)"

wrote "$WT/f"
eq "coming home clears the warning" "Opus | ctx: 12% used | main ⎇ main" "$(
    wrote "$MAIN/f"
    row
)"

wrote "$WT/f"
eq "the session's own view never warns" "Opus | ctx: 12% used | side ⎇ side" "$(row "$WT")"

wrote "$WT/f"
git -C "$MAIN" worktree remove --force "$WT"
eq "a worktree deleted underneath goes quiet" "Opus | ctx: 12% used | main ⎇ main" "$(row)"
git -C "$MAIN" worktree add -q "$WT" side

echo "status line: what the hook records"

recorded() { cat "$STATE/$SID" 2>/dev/null || echo "(none)"; }

wrote "$MAIN/f"
wrote "$WT/f" Bash
eq "a command is not a write" "$MAIN" "$(recorded)"
wrote "relative.md"
eq "a relative path is not resolvable" "$MAIN" "$(recorded)"
wrote "$TMP/state/loose.md"
eq "a file in no repo records nothing" "$MAIN" "$(recorded)"
wrote "$WT/n.ipynb" NotebookEdit notebook_path
eq "a notebook is a write" "$WT" "$(recorded)"

echo "subagent rows"

eq "names the agent and its worktree" "refactorer ⎇ side  running  8%" \
    "$(rows "{\"id\":\"t\",\"name\":\"refactorer\",\"status\":\"running\",\"cwd\":\"$WT\",\"tokenCount\":16000,\"contextWindowSize\":200000}")"
# An inline Agent call leaves .name empty and names itself in label.
eq "falls back to the label" "Idle sleeper ⎇ main  running  1%" \
    "$(rows "{\"id\":\"t\",\"label\":\"Idle sleeper\",\"type\":\"general-purpose\",\"status\":\"running\",\"cwd\":\"$MAIN\",\"tokenCount\":2000,\"contextWindowSize\":200000}")"
eq "a task with no cwd keeps its name" "solo  running" \
    "$(rows '{"id":"t","name":"solo","status":"running"}')"
eq "no context window, no percentage" "solo ⎇ main  done" \
    "$(rows "{\"id\":\"t\",\"name\":\"solo\",\"status\":\"done\",\"cwd\":\"$MAIN\",\"tokenCount\":50}")"

echo
printf '%d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
