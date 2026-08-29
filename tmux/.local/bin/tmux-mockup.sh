#!/usr/bin/env bash
# tmux-mockup.sh -- the whole UI with fake data, on a private tmux server.
#
#   task mockup            build it and print how to attach
#   task mockup -- --print capture it as text (no attach needed)
#   task mockup -- --stop  tear it down
#
# What is real here: the shipped tmux/.tmux.conf (status line, chips, pane
# border labels, bindings), the shipped tmux-diff-pane.sh / tmux-cwd.sh /
# tmux-worktree-picker.sh, the real hunk, and the real `agentbar hook` - the
# drift key feeds it a genuine PostToolUse event, so @agent_workdir is stamped
# by production code. What is fake: the sidebar (`agentbar mockup`, the fixture
# that already exists for previewing it) and two throwaway git repos named after
# worktrees. Your live server is never touched: everything runs on socket
# "mockup", and the config is loaded with its plugin lines stripped (TPM,
# resurrect/continuum, the agentbar plugin - a fresh server must not restore a
# saved session or storm sidebars). One thing does leak: the real hunk registers
# each mock pane with its session daemon, so every run leaves an entry in
# `hunk session list` pointing at a dead pid. Cosmetic, and cleared whenever the
# daemon restarts; using a stub viewer instead would cost the mock the real diff.
#
# Inside the mock (prefix is C-a, so the outer tmux keeps C-b):
#   M-d      the agent moves to the next worktree (fires a real hook event)
#   M-w      the "Pick worktree…" popup
#   M-n      the workdesk float on a fixture mirror (1-4 and tab switch views)
#   click ≡ workdesk  the same float, from the toolbar
#   click ◧ changes  violet = menu, amber = follow the agent
#   C-a d    detach
set -uo pipefail

SOCK=${MOCKUP_SOCKET:-mockup}
SELF=$(realpath "$0")
REPO=${MOCKUP_REPO:-$(cd "$(dirname "$SELF")/../../.." && pwd)}
TMP=${TMPDIR:-/tmp}/dotfiles-mockup-$(id -u)
BIN="$REPO/apps/agentbar/bin/agentbar"
export LC_ALL=C.UTF-8
export DOTFILES_TRACE=0 # a mock must not write to the live trace log

M() { command tmux -L "$SOCK" -f /dev/null "$@"; }

usage() {
    # Everything from line 2 to the first non-comment line, so editing the header
    # block can never silently truncate what --help prints.
    awk 'NR > 1 { if (!/^#/) exit; sub(/^# ?/, ""); print }' "$SELF"
}

# --- internal: fire a real hook event, alternating the worktree --------------
# Bound to M-d. The mock's whole point is that this is production code: the Go
# hook stamps @agent_workdir, and the chip/border/menu react to it.
cmd_hook() {
    local pane=$1 cur next
    shift
    local -a wts=("$@")
    cur=$(M display-message -p -t "$pane" '#{@agent_workdir}')
    next=${wts[0]}
    local i
    for i in "${!wts[@]}"; do
        [ "${wts[$i]}" = "$cur" ] || continue
        next=${wts[$(((i + 1) % ${#wts[@]}))]}
        break
    done
    printf '{"hook_event_name":"PostToolUse","tool_name":"Edit","session_id":"mock",
             "cwd":"%s","tool_input":{"file_path":"%s/mock.txt"}}\n' "$next" "$next" |
        PATH="$TMP/shim:$PATH" TMUX_PANE="$pane" "$BIN" hook
    M refresh-client -S 2>/dev/null
    M display-message "mock: agent now writing in $(basename "$next")"
}

case "${1:-}" in
    --help | -h)
        usage
        exit 0
        ;;
    --stop)
        M kill-server 2>/dev/null && echo "mockup stopped" || echo "no mockup server running"
        exit 0
        ;;
    --hook)
        shift
        cmd_hook "$@"
        exit 0
        ;;
esac

[ -x "$BIN" ] || {
    echo "mockup: $BIN not built - run \`task agentbar:build\`" >&2
    exit 1
}
command -v hunk >/dev/null || [ -x "$HOME/.local/bin/hunk" ] || {
    echo "mockup: hunk not on PATH - run \`./install.sh install_hunk\`" >&2
    exit 1
}

# --- fixtures ---------------------------------------------------------------
# One bare repo with several linked worktrees, the layout this whole feature is
# about: the pane sits in one checkout while the agent writes in another, and the
# picker has a family to list.
gitm() { git -c user.email=mock@mock -c user.name=mock -c init.defaultBranch=main "$@"; }

rm -rf "$TMP"
mkdir -p "$TMP/shim" "$TMP/wt"
FILE=cloud/dashboard_ui/StationsView.vue
BARE=$TMP/wt/.bare
seed=$TMP/seed
mkdir -p "$(dirname "$seed/$FILE")"
gitm init -q "$seed"
printf 'const a = 1\nconst b = 2\nconst c = 3\n' >"$seed/$FILE"
gitm -C "$seed" add -A
gitm -C "$seed" commit -qm "seed"
gitm clone -q --bare "$seed" "$BARE"

wt_add() { # wt_add <name> <branch> [dirty]
    local d=$TMP/wt/$1
    if [ "$2" = main ]; then
        gitm -C "$BARE" worktree add -q "$d" main
    else
        gitm -C "$BARE" worktree add -q -b "$2" "$d" main
    fi
    [ "${3:-}" = dirty ] &&
        printf 'const a = 1\nconst b = 42        // deployed_at\nconst c = 3\nconst d = 4\n' >"$d/$FILE"
    return 0
}
HOME_WT="$TMP/wt/proj-main" # where the session (and so its pane) sits: clean
WORK="$TMP/wt/proj-b"       # where the agent is actually writing: dirty
wt_add proj-main main
wt_add proj-b 2871-queue-drain-and-retry-count dirty
wt_add proj-c 1503-cache-key-rewrite dirty
wt_add proj-d 4180-basis-config-tests

# Every script under test calls bare `tmux`; the shim points them at the private
# socket, exactly as test/session-picker.sh does.
cat >"$TMP/shim/tmux" <<EOF
#!/bin/sh
exec $(command -v tmux) -L $SOCK -f /dev/null "\$@"
EOF
chmod +x "$TMP/shim/tmux"

# A real ELF named claude, so the pane reads as an agent pane (pane_current_command).
cp "$(command -v sleep)" "$TMP/shim/claude"

cat >"$TMP/agent.txt" <<TXT

  ● Bash(cd $WORK && bazel run //tools:format)
    ⎿  Formatted YAML in 0m0.348s
       M cloud/dashboard_ui/StationsView.vue
       Shell cwd was reset to $HOME_WT

  ● Wrote cloud/dashboard_ui/StationsView.vue

    ────────────────────────────────────
      MOCKUP.  This pane's cwd is proj-main; the agent writes in proj-b.
        M-d   move the agent to the next worktree (real hook event)
        M-w   Pick worktree… popup
        M-n   workdesk: inbox, issues, merge requests, agents
              (fixture data; 1-4 and tab switch views, P promotes to a pane;
               rows, tabs and the wheel take the mouse)
        click ◧ changes   violet = menu · amber = follow the agent
        click ≡ workdesk  opens it, and closes it again
        C-a d detach, then: task mockup -- --stop
    ────────────────────────────────────

  ›
TXT

# The shipped config, minus what a throwaway server must not run: TPM and its
# plugins (continuum would restore a saved session into the mock), and the
# agentbar plugin (it would spawn real sidebars). Everything that draws the UI
# stays, so the mock cannot drift from what ships.
grep -v -e "^run '~/.tmux/plugins/tpm/tpm'" \
    -e "^run-shell '~/dotfiles/apps/agentbar/agentbar.tmux'" \
    -e '^set -g @plugin' \
    -e '^set -g @continuum' \
    -e '^set -g @resurrect' \
    -e '^run-shell .tmux set -g @resurrect-save-script-path' \
    "$REPO/tmux/.tmux.conf" >"$TMP/tmux.conf"

# Point this package's own scripts at the checkout, not ~/.local/bin: a script
# added since the last `stow` has no symlink there yet, and tmux swallows a
# missing #() command as empty output - so the mock would quietly show less than
# the change does. Installed tools ($HOME/.local/bin and friends) are
# left alone.
repoify() { # repoify <file> - point this package's scripts at the checkout
    local f=$1 n
    for n in "$REPO"/tmux/.local/bin/*; do
        n=$(basename "$n")
        # Both spellings: .tmux.conf carries the literal "$HOME/..." while the
        # theme switcher's generated overlay has it already expanded.
        sed -i "s|\$HOME/\.local/bin/$n|$REPO/tmux/.local/bin/$n|g" "$f"
        sed -i "s|$HOME/\.local/bin/$n|$REPO/tmux/.local/bin/$n|g" "$f"
    done
}
repoify "$TMP/tmux.conf"

# The theme switcher's generated overlay is sourced last and so has the final say
# on every colour and chip it emits. Generate a fresh one from the current
# theme/.local/bin/theme into a temp XDG dir and source that instead of the live
# one: the mock then shows what the change actually ships, and a chip added to
# .tmux.conf but not to the generator shows up here as the stale version it is.
flavor=$(cat "${XDG_CONFIG_HOME:-$HOME/.config}/theme/current" 2>/dev/null)
[ -n "$flavor" ] || flavor=solarized-light
# The switcher also pushes to a running server and signals ghostty. Both are
# stubbed out here: it must only WRITE the overlay, never reach the live server
# or the terminal (it calls bare `tmux`, which is the live one).
mkdir -p "$TMP/noop"
for stub in tmux pkill; do
    printf '#!/bin/sh\nexit 0\n' >"$TMP/noop/$stub"
    chmod +x "$TMP/noop/$stub"
done
PATH="$TMP/noop:$PATH" XDG_CONFIG_HOME="$TMP/config" \
    "$REPO/theme/.local/bin/theme" "$flavor" >/dev/null 2>&1
sed -i "s|~/.config/theme/tmux.conf|$TMP/config/theme/tmux.conf|g" "$TMP/tmux.conf"
# The overlay is sourced last, so its own #() paths decide what renders.
repoify "$TMP/config/theme/tmux.conf"

# --- build the window -------------------------------------------------------
# A fresh server can race the previous one's teardown: kill, wait for the socket
# to actually go, then create. Without the wait the new session occasionally
# lands on a dying server and the whole mock comes up empty.
M kill-server 2>/dev/null
for _ in 1 2 3 4 5 6 7 8 9 10; do
    M has-session -t mock 2>/dev/null || break
    sleep 0.2
done
M new-session -d -s mock -x 220 -y 42 -c "$HOME_WT" \
    "cat $TMP/agent.txt; exec $TMP/shim/claude 100000"
M has-session -t mock 2>/dev/null || {
    echo "mockup: could not start a server on socket $SOCK" >&2
    exit 1
}
M source-file "$TMP/tmux.conf"
M set -g prefix C-a
M set -g window-size latest # size to whatever client attaches, so widths are honest
M set -g status-interval 2

M split-window -hb -l 30 -t mock "exec $BIN mockup"
agent=$(M list-panes -t mock -F '#{pane_id} #{pane_current_command}' | awk '$2 == "claude" {print $1; f=1}
    END {if (!f) print ""}')
[ -n "$agent" ] || agent=$(M list-panes -t mock -F '#{pane_id}' | sed -n 2p)
M select-pane -t "$agent"

# Stamp the agent's workdir through the real hook, then open the diff pane with
# the real script: it resolves the target from @agent_workdir, so the pane lands
# on proj-b even though the pane's cwd is proj-main.
"$SELF" --hook "$agent" "$WORK" "$TMP/wt/proj-c" "$HOME_WT" >/dev/null 2>&1
PATH="$TMP/shim:$PATH" "$REPO/tmux/.local/bin/tmux-diff-pane.sh" work
M select-pane -t "$agent"

# Fake the bits that need a live GitLab remote / dictation, so the footer reads
# like the real one instead of showing gaps.
M set -g @dictate_seg "#[fg=#93a1a1 bg=#eee8d5] ● dictate "

M bind -n M-d run-shell -b "$SELF --hook $agent $WORK $TMP/wt/proj-c $HOME_WT"
M split-window -v -l 8 -t "$agent" -c "$REPO" 'bash --norc -i'
M select-pane -t "$agent"

M bind -n M-w run-shell -b "PATH=$TMP/shim:\$PATH $REPO/tmux/.local/bin/tmux-diff-pane.sh pick"

# The workdesk float, on a fixture mirror of invented merge requests, agents and todos,
# so the mock never renders a real queue into a screenshot. The shipped binary writes and
# renders it - `workdesk fixture` is the same code path a real sync uses, only the data
# is fake.
#
# One binding, not four: 1-4 and tab switch views inside the float, and M-m is a real
# production binding (select-pane) that a mock key would shadow.
#
# Override the option, never the key: M-n and the ≡ workdesk chip both run
# tmux-workdesk.sh, which opens whatever @workdesk_open says, so one override points both
# at the fixture. Rebinding M-n alone would leave a click on the chip opening the real
# queue - the one thing this mock must not do.
#
# WORKDESK_DRY makes the three write keys (a, e, M) print the command they would run and
# stop there, so every key in the popup is safe to press here.
WORKDESK="$REPO/apps/agentbar/bin/workdesk"
if [ -x "$WORKDESK" ] && "$WORKDESK" fixture "$TMP/workdesk" >/dev/null 2>&1; then
    M set -g @workdesk_open \
        "new-pane -x 90% -y 80% -X 5% -Y 10% \
         'WORKDESK_MIRROR=$TMP/workdesk WORKDESK_AGENTS=$TMP/workdesk/agents.tsv \
          WORKDESK_DRY=1 $WORKDESK open inbox'"
else
    # Must be overridden even on failure: the sourced conf points it at the real
    # mirror, and a mock that opens the live queue is the one outcome to rule out.
    M set -g @workdesk_open "display-message 'mockup: no workdesk fixture'"
    printf 'mockup: workdesk fixture failed; M-n and the workdesk chip will say so\n' >&2
fi

# --- output -----------------------------------------------------------------
if [ "${1:-}" = "--print" ]; then
    # A client is needed for borders and the status line to render; attach one on
    # a second private server and capture its screen.
    V() { command tmux -L "$SOCK-view" -f /dev/null "$@"; }
    V kill-server 2>/dev/null
    V new-session -d -s v -x 220 -y 42 "TERM=xterm-256color command tmux -L $SOCK -f /dev/null attach -t mock"
    V set -g window-size manual
    V set -g status off
    V resize-window -t v -x 220 -y 42
    sleep 3
    V capture-pane -p -t v "${@:2}"
    V kill-server 2>/dev/null
    exit 0
fi

# Self-check: a mock that half-built is worse than none - it reads as the change
# being broken. Say what is actually there.
panes=$(M list-panes -t mock -F '#{pane_id}' | grep -c .)
target=$(M show -wqv -t mock @diff_target)
agent_wt=$(M show -wqv -t mock @agent_workdir)
if [ "$panes" -lt 4 ] || [ -z "$target" ]; then
    printf 'mockup: incomplete - %s panes, diff target [%s]\n' "$panes" "${target:-unset}" >&2
    exit 1
fi

cat <<TXT
mockup ready: $panes panes, diff pane on $(basename "$target"), agent in $(basename "$agent_wt").
Attach with:

    tmux -L $SOCK attach -t mock

  M-d  agent writes in the other worktree   M-w  pick worktree
  M-n  workdesk float: inbox, issues, merge requests, agents
       (1-4 and tab switch views; a/e/M are dry-run here; rows, tabs and the wheel take the mouse)
  click ◧ changes (violet = menu, amber = follow)   click ≡ workdesk (opens and closes)
  C-a d  detach   stop: task mockup -- --stop
TXT
