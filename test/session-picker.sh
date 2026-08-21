#!/usr/bin/env bash
# Guards the session picker's list: it must render the agent bar's own order and
# bands, never an order of its own. The regression is silent - a picker that
# falls back to alphabetical still looks like a session list, and the Alt-h /
# Alt-l keys walking a different order than the popup shows is exactly the
# jarring jump the bands removed.
#
# The picker and the binary both call bare `tmux`, so a PATH shim points them at
# a private socket - never the live server. A copy of `sleep` named `claude`
# reports as an agent through pane_current_command, which is all the snapshot
# keys on; XDG_STATE_HOME is redirected so the pin mirror stays in the tempdir.
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PICKER="$REPO/tmux/.local/bin/tmux-session-picker.sh"
PREVIEW="$REPO/tmux/.local/bin/tmux-session-preview.sh"
BIN="$REPO/apps/agentbar/bin/agentbar"
BAND_MARK='__band__'
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

# eq <name> <want> <got>
eq() { [ "$2" = "$3" ] && ok "$1" || no "$1" "want [$2] got [$3]"; }

if [ ! -x "$BIN" ]; then
    printf 'picker: %s not built, run `task agentbar:build`\n' "$BIN" >&2
    exit 1
fi

TMP=$(mktemp -d)
export PATH="$TMP/shim:$PATH"
export XDG_STATE_HOME="$TMP/state"
export AGENTBAR_BIN="$BIN"
export DOTFILES_TRACE=0
export LC_ALL=C.UTF-8

mkdir -p "$TMP/shim" "$TMP/state"
cat >"$TMP/shim/tmux" <<EOF
#!/bin/sh
exec $(command -v tmux) -S $TMP/sock -f /dev/null "\$@"
EOF
chmod +x "$TMP/shim/tmux"
cp /usr/bin/sleep "$TMP/shim/claude"

cleanup() {
    tmux kill-server 2>/dev/null
    rm -rf "$TMP"
}
trap cleanup EXIT

# session <name> [agent] - a detached session, optionally holding an agent pane
session() {
    tmux new-session -d -s "$1" -x 200 -y 40
    [ "${2:-}" = agent ] || return 0
    local pane
    pane=$(tmux split-window -d -t "$1" -P -F '#{pane_id}' "claude 60")
    printf '{"hook_event_name":"SessionStart","session_id":"t"}' |
        TMUX_PANE="$pane" "$BIN" hook
}

# names / bands - field 1 / the band column of the picker's list, one per line
names() { "$PICKER" --list | cut -f1; }

# rows - the list's structure: session names, plus hdr/gap for the two kinds of
# inert row (a band header and the blank spacer above it).
rows() {
    "$PICKER" --list | awk -F'\t' -v m="$BAND_MARK" \
        '{ print ($1 == m ? ($2 == "" ? "gap" : "hdr") : $1) }'
}
bands() { "$PICKER" --list | cut -f2 | sed 's/\x1b\[[0-9;]*m//g'; }

printf '\npicker: order and bands mirror the agent bar\n'

tmux kill-server 2>/dev/null
session api agent
session blog agent
session dotfiles agent
session payments # no agent: dormant
"$BIN" pin blog >/dev/null
"$BIN" pin dotfiles >/dev/null

# Alphabetically this list would be api, blog, dotfiles, payments. Each band but
# the first also opens with a blank spacer, so the groups read apart - the same
# spacing the sidebar gives them.
eq "bands, gaps and sessions in sidebar order" \
    "hdr blog dotfiles gap hdr api gap hdr payments" \
    "$(rows | tr '\n' ' ' | sed 's/ $//')"

eq "picker order matches agentbar order exactly" \
    "$("$BIN" order | cut -f2 | tr '\n' ' ')" \
    "$(names | grep -Fxv "$BAND_MARK" | tr '\n' ' ')"

# All three named: an unlabelled middle band left you counting rows to see
# where "the rest" ended.
eq "every band is labelled and counted" \
    "pinned ·2|active ·1|dormant ·1" \
    "$(bands | grep -oE '(pinned|active|dormant) ·[0-9]+' | paste -sd'|')"

printf '\npicker: headers only when they divide something\n'

tmux kill-server 2>/dev/null
session solo agent
eq "a single band shows no header or gap rows" "solo" "$(rows | tr '\n' ' ' | sed 's/ $//')"

printf '\npicker: p pins through the binary and follows the row\n'

tmux kill-server 2>/dev/null
session api agent
session payments
# payments is dormant and last; pinning floats it to row 2, under the header.
eq "pin prints the fzf actions that redraw and follow" \
    "reload-sync+pos(2)" \
    "$("$PICKER" --pin payments | sed -E 's/reload-sync\([^)]*\)/reload-sync/')"
eq "the pin reached the shared set" "payments" "$("$BIN" order | head -1 | cut -f2)"
eq "pin is a toggle" "" "$(
    "$PICKER" --pin payments >/dev/null
    "$BIN" order | grep -c '^pinned' | sed 's/^0$//'
)"
eq "a band header row is never pinned" "" "$("$PICKER" --pin "$BAND_MARK")"

printf '\npicker: list and preview share one state language\n'

tmux kill-server 2>/dev/null
session api agent
# SessionStart leaves it idle, which renders blank; a prompt makes it working.
pane=$(tmux list-panes -s -t api -F '#{pane_id} #{pane_current_command}' | awk '$2 == "claude" {print $1}')
printf '{"hook_event_name":"UserPromptSubmit","session_id":"t"}' | TMUX_PANE="$pane" "$BIN" hook

# The preview drew its own emoji circles while the list and the sidebar drew
# glyphs; both now come from tmux-agent-state.sh, so they cannot disagree again.
list_icon=$("$PICKER" --list | grep -F api | grep -oE $'\033\[38;2;[0-9;]+m.' | head -1)
prev_icon=$("$PREVIEW" api | grep -F 'state:' | grep -oE $'\033\[38;2;[0-9;]+m.' | head -1)
eq "the list row and the preview render the same glyph" "$list_icon" "$prev_icon"
eq "and it is the working glyph, not an emoji" "working ⠹" \
    "working $(printf '%s' "$list_icon" | sed 's/\x1b\[[0-9;]*m//g')"

# Colors come from the theme switcher's file, per design/palette.toml: nothing
# downstream hardcodes, so the popup follows a `theme` switch.
mkdir -p "$TMP/config/theme"
printf '_state_working="#ff00ff"\n' >"$TMP/config/theme/agent-state.sh"
themed=$(XDG_CONFIG_HOME="$TMP/config" "$PICKER" --list | grep -F api |
    grep -oE $'\033\[38;2;255;0;255m' | head -1)
eq "the glyph color follows the active flavor" $'\033[38;2;255;0;255m' "$themed"

printf '\npicker: the branch names the session, not whatever pane is focused\n'

# The regression: the row's branch came from `display-message -t <session>`, i.e.
# the session's ACTIVE pane. That is usually the sidebar - whose cwd is only
# wherever that process started - or the diff pane, sitting in whatever worktree
# it was pointed at. Every session then reported the same unrelated branch.
tmux kill-server 2>/dev/null
g() { git -c user.email=t@t -c user.name=t -c init.defaultBranch=main "$@"; }
repo() { # repo <dir> <branch>
    mkdir -p "$1"
    g init -q "$1"
    : >"$1/f"
    g -C "$1" add -A
    g -C "$1" commit -qm init
    g -C "$1" checkout -q -b "$2"
}
repo "$TMP/work" work-branch
repo "$TMP/decoy" decoy-branch
repo "$TMP/edited" edited-branch
cp /usr/bin/sleep "$TMP/shim/agentbar" # a pane that reports as the sidebar

# The branch column, which is the token after the ⎇ glyph. Not the last word of
# the row any more: that is the title column now.
branch_of() {
    "$PICKER" --list | sed 's/\x1b\[[0-9;]*m//g' |
        awk -F'\t' -v n="$1" '$1 == n { print $2 }' |
        sed 's/.*⎇ *//; s/ .*//'
}

tmux new-session -d -s solo -x 200 -y 40 -c "$TMP/work"
tmux split-window -t solo -c "$TMP/decoy" "agentbar 60" # focused, and elsewhere
eq "tmux calls the sidebar's cwd the session's path" "$TMP/decoy" \
    "$(tmux display-message -p -t solo '#{pane_current_path}')"
eq "the row shows the session's own branch" "work-branch" "$(branch_of solo)"

pane=$(tmux split-window -d -t solo -c "$TMP/work" -P -F '#{pane_id}' "claude 60")
printf '{"hook_event_name":"SessionStart","session_id":"t"}' | TMUX_PANE="$pane" "$BIN" hook
tmux set -p -t "$pane" @agent_workdir "$TMP/edited"
eq "an agent's worktree wins, as on the sidebar" "edited-branch" "$(branch_of solo)"

tmux new-session -d -s crowd -x 200 -y 40 -c "$TMP/decoy" # one stray pane
tmux split-window -d -t crowd -c "$TMP/work"
tmux split-window -d -t crowd -c "$TMP/work"
eq "one stray pane cannot outvote the rest" "work-branch" "$(branch_of crowd)"

# A pane with no @agent_* options must still report its path: tab is IFS
# whitespace, so a run of empty fields collapses and shifts the rest left.
tmux new-session -d -s plain -x 200 -y 40 -c "$TMP/work"
eq "a session with no agent still gets its branch" "work-branch" "$(branch_of plain)"

# The row shows the same thing the sidebar's agent line does: Claude's title for
# the session, falling back to the branch for one it has not titled yet.
# Substring, not a column: a title carries spaces, and an offset would count the
# current-row marker's bytes rather than its one cell.
in_row() { # in_row <session> <text>
    case "$("$PICKER" --list | sed 's/\x1b\[[0-9;]*m//g' |
        awk -F'\t' -v n="$1" '$1 == n { print $2 }')" in
        *"$2"*) return 0 ;;
        *) return 1 ;;
    esac
}
has() { in_row "$2" "$3" && ok "$1" || no "$1" "row [$2] has no [$3]"; }
hasnt() { in_row "$2" "$3" && no "$1" "row [$2] still shows [$3]" || ok "$1"; }

# ◐ rather than ✳ on purpose: the marker glyph varies per pane, so the row must
# strip whichever one Claude used.
tmux select-pane -t "$pane" -T '◐ Ship the parser'
has "the row shows Claude's title for the session" solo "Ship the parser"
has "beside its branch, in its own column" solo edited-branch
has "an untitled session still shows its branch" plain work-branch
hasnt "and an em dash where its title would be" plain "Ship the parser"

# ---- several agents in one session ----------------------------------------
# One agent speaks for the row: the most urgent, which is the one the glyph
# already describes. "+N" owns up to the rest, and the preview lists them.
printf '\npicker: several agents in one session\n'
pane2=$(tmux split-window -d -t solo -c "$TMP/work" -P -F '#{pane_id}' "claude 60")
printf '{"hook_event_name":"SessionStart","session_id":"t2"}' | TMUX_PANE="$pane2" "$BIN" hook
printf '{"hook_event_name":"PermissionRequest","tool_name":"Bash"}' | TMUX_PANE="$pane2" "$BIN" hook
tmux select-pane -t "$pane2" -T '◐ Approve the migration'

has "the row speaks for the agent that wants you" solo "Approve the migration"
hasnt "not the quieter one" solo "Ship the parser"
has "and owns up to the rest" solo "+1"

# ---- the preview names the session, not its active pane -------------------
# The regression: dir/repo/branch came from `display-message -t <session>`, the
# session's ACTIVE pane - usually the sidebar, whose cwd is only wherever that
# process started, so every session reported the same repo. The rows already
# resolved it by majority; the preview must agree.
printf '\npicker: the preview names the session, not its active pane\n'
prev() { "$PREVIEW" "$1" 2>/dev/null | sed 's/\x1b\[[0-9;]*m//g'; }
field() { prev "$1" | awk -v k="$2:" '$1 == k { print $2 }'; }

eq "an agent's worktree wins, as on the row" "edited-branch" "$(field solo branch)"
eq "a session with no agent takes its panes' branch" "work-branch" "$(field plain branch)"
eq "one stray pane cannot outvote the rest" "work-branch" "$(field crowd branch)"

prev_out=$(prev solo)
for want in "Approve the migration" "Ship the parser"; do
    case $prev_out in
        *"$want"*) ok "the preview lists [$want]" ;;
        *) no "the preview lists [$want]" ;;
    esac
done

# An agent lives in a pane, which tmux addresses as window.pane - so that is how
# the block places it, rather than naming the window it happens to sit in.
# Index-agnostic: this server runs with -f /dev/null, so base-index is 0 here
# and 1 under the real config.
case $prev_out in
    *"pane "[0-9]*.[0-9]*) ok "each agent is placed by window.pane" ;;
    *) no "each agent is placed by window.pane" "no 'pane N.N' in the block" ;;
esac

# The regression: this block iterated panes under a "windows:" heading, so solo's
# one window appeared once per pane. Panes are the splits inside a window.
eq "one line per window, not per pane" 1 \
    "$(prev solo | sed -n '/^windows:/,$p' | grep -c '^  ')"
case $prev_out in
    *"4 panes"*) ok "and it counts them" ;;
    *) no "and it counts them" "no '4 panes'" ;;
esac

printf '\npicker: no binary means no crash\n'

tmux kill-server 2>/dev/null
session api agent
session payments
eq "falls back to a flat alphabetical list" "api payments" \
    "$(AGENTBAR_BIN=/nonexistent "$PICKER" --list | cut -f1 | tr '\n' ' ' | sed 's/ $//')"

printf '\npicker: %d passed, %d failed\n\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
