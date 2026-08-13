#!/usr/bin/env bash
# Guards where the diff pane points. The regression is silent and was the whole
# reason this exists: a Claude session started in one worktree edits files in
# another, the pane's cwd never follows, and a diff pane resolved from that cwd
# shows a clean tree - a convincing "no changes" for work that is right there.
#
# So: the agentbar hook must stamp @agent_workdir from the edited file (pane,
# window and session scope), flag @agent_elsewhere only when it really is, keep a
# recent list for agents that span several worktrees, and tmux-diff-pane.sh must
# prefer all that over #{pane_current_path}. Also covers tmux-rail.sh, the text a
# pane border carries.
#
# Everything runs on a private socket via a PATH shim (the scripts and the binary
# call bare `tmux`), with a stub hunk so no real viewer or daemon is launched.
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIFF="$REPO/tmux/.local/bin/tmux-diff-pane.sh"
PICKER="$REPO/tmux/.local/bin/tmux-worktree-picker.sh"
RAIL="$REPO/tmux/.local/bin/tmux-rail.sh"
BIN="$REPO/apps/agentbar/bin/agentbar"
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
has() { case "$2" in *"$3"*) ok "$1" ;; *) no "$1" "[$2] lacks [$3]" ;; esac }

if [ ! -x "$BIN" ]; then
    printf 'diff-target: %s not built, run `task agentbar:build`\n' "$BIN" >&2
    exit 1
fi

TMP=$(mktemp -d)
export PATH="$TMP/shim:$PATH"
export XDG_STATE_HOME="$TMP/state"
export DOTFILES_TRACE=0
export LC_ALL=C.UTF-8
export DIFF_HUNK_BIN="$TMP/shim/hunk-stub"
trap 'command tmux -S "$TMP/sock" kill-server 2>/dev/null; rm -rf "$TMP"' EXIT

mkdir -p "$TMP/shim" "$TMP/state"
cat >"$TMP/shim/tmux" <<EOF
#!/bin/sh
exec $(command -v tmux) -S $TMP/sock -f /dev/null "\$@"
EOF
# The viewer the diff pane would run: sits there so the pane stays alive.
cat >"$TMP/shim/hunk-stub" <<EOF
#!/bin/sh
echo "hunk-stub \$*" > "$TMP/hunk-args"
exec sleep 300
EOF
chmod +x "$TMP/shim/tmux" "$TMP/shim/hunk-stub"
# A real ELF named claude: the scripts key the agent pane on pane_current_command.
cp "$(command -v sleep)" "$TMP/shim/claude"

t() { command tmux -S "$TMP/sock" -f /dev/null "$@"; }

# --- fixture: one bare repo, three linked worktrees -------------------------
g() { git -c user.email=t@t -c user.name=t -c init.defaultBranch=main "$@"; }
WT="$TMP/wt"
mkdir -p "$WT" "$TMP/seed/sub"
g init -q "$TMP/seed"
echo one >"$TMP/seed/sub/f.txt"
g -C "$TMP/seed" add -A
g -C "$TMP/seed" commit -qm seed
g clone -q --bare "$TMP/seed" "$WT/.bare"
g -C "$WT/.bare" worktree add -q "$WT/home" main
g -C "$WT/.bare" worktree add -q -b feature "$WT/other" main
g -C "$WT/.bare" worktree add -q -b third "$WT/third" main
g -C "$WT/.bare" worktree add -q -b spare "$WT/spare" main # the agent never touches this one
echo two >>"$WT/other/sub/f.txt"                           # the agent's edit lands here

printf '\ntmux-rail.sh (a pane rail zone)\n'
export XDG_RUNTIME_DIR="$TMP/run" # memo files stay in the tempdir
mkdir -p "$XDG_RUNTIME_DIR"
eq "worktree and branch" "home  ⎇ main" "$(bash "$RAIL" "$WT/home")"
eq "from a subdirectory too" "home  ⎇ main" "$(bash "$RAIL" "$WT/home/sub")"
eq "a sibling worktree's branch" "other  ⎇ feature" "$(bash "$RAIL" "$WT/other")"
# The worktree name is the identity and is never cut; the branch is what gives,
# and only when the pane is too narrow to hold it.
eq "wide pane, whole branch" "other  ⎇ feature" "$(bash "$RAIL" "$WT/other" 100 2)"
eq "narrow pane, branch capped" "other  ⎇ featu…" "$(bash "$RAIL" "$WT/other" 30 2)"
eq "one zone has room two do not" "other  ⎇ feature" "$(bash "$RAIL" "$WT/other" 30 1)"
eq "nothing outside a checkout" "" "$(bash "$RAIL" "$TMP")"
eq "nothing for a missing dir" "" "$(bash "$RAIL" "$TMP/nope")"

# --- the hook stamps where the agent writes ---------------------------------
printf '\nagentbar hook -> @agent_workdir\n'
t new-session -d -s s -x 200 -y 30 -c "$WT/home" "exec $TMP/shim/claude 300"
win=$(t display-message -p -t s '#{window_id}')
pane=$(t list-panes -t s -F '#{pane_id}' | head -1)
hook() { # hook <file>
    printf '{"hook_event_name":"PostToolUse","tool_name":"Edit","session_id":"t",
             "tool_input":{"file_path":"%s"}}\n' "$1" |
        TMUX_PANE="$pane" "$BIN" hook
}
hook "$WT/other/sub/f.txt"
eq "pane option stamped" "$WT/other" "$(t show -pqv -t "$pane" @agent_workdir)"
eq "window option stamped" "$WT/other" "$(t show -wqv -t "$win" @agent_workdir)"
hook "$WT/home/sub/f.txt"
eq "moves with the agent" "$WT/home" "$(t show -pqv -t "$pane" @agent_workdir)"

printf '{"hook_event_name":"PostToolUse","tool_name":"Read","session_id":"t",
         "tool_input":{"file_path":"%s"}}\n' "$WT/other/sub/f.txt" |
    TMUX_PANE="$pane" "$BIN" hook
eq "a Read does not move it" "$WT/home" "$(t show -pqv -t "$pane" @agent_workdir)"

# Nor does a cwd. Claude resets its shell cwd to the session's own directory
# after every Bash call, so this fires seconds after an edit in a sibling
# worktree - and used to drag the rail and the diff pane home with it.
hook "$WT/other/sub/f.txt"
printf '{"hook_event_name":"CwdChanged","session_id":"t","cwd":"%s"}\n' "$WT/home" |
    TMUX_PANE="$pane" "$BIN" hook
eq "a cwd reset does not move it" "$WT/other" "$(t show -pqv -t "$pane" @agent_workdir)"
hook "$WT/home/sub/f.txt" # back where the rest of the suite expects it

# The rail's right zone appears only when the agent writes outside its own pane's
# worktree - a shell pane must never inherit that claim.
hook "$WT/other/sub/f.txt"
eq "elsewhere flagged" "1" "$(t show -pqv -t "$pane" @agent_elsewhere)"
hook "$WT/home/sub/f.txt"
eq "and cleared when writing at home" "" "$(t show -pqv -t "$pane" @agent_elsewhere)"
eq "never set window-wide" "" "$(t show -wqv -t "$win" @agent_elsewhere)"

# Recent worktrees, newest first: one agent often spans several in a turn.
hook "$WT/other/sub/f.txt"
hook "$WT/third/sub/f.txt"
eq "recent list, newest first" "|$WT/third|$WT/other|$WT/home|" \
    "$(t show -pqv -t "$pane" @agent_workdirs)"
eq "mirrored on the window" "|$WT/third|$WT/other|$WT/home|" \
    "$(t show -wqv -t "$win" @agent_workdirs)"
eq "and on the session, for other windows" "$WT/third" "$(t show -qv -t "$pane" @agent_workdir)"

# --- the diff pane follows the agent, not the pane's cwd --------------------
printf '\ntmux-diff-pane.sh target\n'
hook "$WT/other/sub/f.txt" # agent writes in `other`; the pane still sits in `home`
bash "$DIFF" work
eq "targets the agent's worktree" "$WT/other" "$(t show -wqv -t "$win" @diff_target)"
eq "not the pane's cwd" "$WT/home" "$(t display-message -p -t "$pane" '#{pane_current_path}')"
diffpane=$(t show -wqv -t "$win" @diff_pane)
eq "diff pane is alive" "1" "$(t list-panes -t s -F '#{pane_id}' | grep -c "^$diffpane$")"

# The predicate the chip colours on and the border reads.
drift() { t display-message -p -t "$1" '#{E:@diff_drift}'; }
# The shipped predicate, verbatim (tmux/.tmux.conf): is the diff pane showing a
# worktree that appears nowhere in the agent's recent list?
touched='#{m:*|#{@diff_target}|*,#{@agent_workdirs}}'
t set -g @diff_drift "#{&&:#{@diff_target},#{&&:#{@agent_workdirs},#{!=:$touched,1}}}"
eq "showing a worktree the agent touched: quiet" "0" "$(drift "$pane")"
# Alternating between two worktrees must not flicker the chip: the one on screen
# is still one the agent is working in.
hook "$WT/third/sub/f.txt"
eq "still quiet while it alternates" "0" "$(drift "$pane")"
t set -w -t "$win" @diff_target "$WT/spare"
eq "showing an untouched worktree: amber" "1" "$(drift "$pane")"
t set -w -t "$win" @diff_target "$WT/other"

printf '\nfollow / autofollow\n'
bash "$DIFF" follow
eq "follow re-points" "$WT/third" "$(t show -wqv -t "$win" @diff_target)"
eq "follow clears drift" "0" "$(drift "$pane")"
eq "follow keeps one diff pane" "$diffpane" "$(t show -wqv -t "$win" @diff_pane)"

hook "$WT/other/sub/f.txt"
AGENTBAR_PANE="$pane" bash "$DIFF" autofollow
eq "autofollow off: stays put" "$WT/third" "$(t show -wqv -t "$win" @diff_target)"
t set -w -t "$win" @diff_follow on
AGENTBAR_PANE="$pane" bash "$DIFF" autofollow
eq "autofollow on: re-points" "$WT/other" "$(t show -wqv -t "$win" @diff_target)"

printf '\npicker rows\n'
rows=$(bash "$PICKER" --list "$WT/home" "$win")
has "agent band first" "$(printf '%s' "$rows" | head -1 | cut -f2)" "agent ·"
has "marks the current target" "$(printf '%s' "$rows" | grep "$WT/other" | cut -f2)" "◧"
has "lists the family" "$rows" "third"
eq "excludes the bare repo" "0" "$(printf '%s' "$rows" | cut -f1 | grep -c '\.bare')"
# The agent's worktree belongs in the agent band only, never twice.
paths=$(printf '%s\n' "$rows" | cut -f1 | grep -v '^__band__$')
eq "no worktree listed twice" "$(printf '%s\n' "$paths" | sort -u | grep -c .)" \
    "$(printf '%s\n' "$paths" | grep -c .)"
has "preview reports dirt" "$(bash "$PICKER" --preview "$WT/other")" "changed"
eq "preview ignores band rows" "" "$(bash "$PICKER" --preview __band__)"

# End to end with the chooser stubbed: picking a row must re-point the pane. This
# is the wiring that broke when either script named the other under ~/.local/bin -
# a path that only exists once the package is stowed, so the popup exited 127.
cat >"$TMP/shim/fzf" <<EOF
#!/bin/sh
grep -F "$WT/third" # stand in for the user landing on that row and hitting Enter
EOF
chmod +x "$TMP/shim/fzf"
# HOME points at an empty dir: nothing is stowed under it, so a script that
# names its sibling via ~/.local/bin fails here even on a machine where it is.
mkdir -p "$TMP/home"
HOME="$TMP/home" bash "$PICKER"
eq "picking a row re-points the pane" "$WT/third" "$(t show -wqv -t "$win" @diff_target)"
eq "and keeps the same diff pane" "$diffpane" "$(t show -wqv -t "$win" @diff_pane)"

printf '\nclose\n'
bash "$DIFF" close
eq "target cleared" "" "$(t show -wqv -t "$win" @diff_target)"
eq "auto-follow preference kept" "on" "$(t show -wqv -t "$win" @diff_follow)"

printf '\n%s passed, %s failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
