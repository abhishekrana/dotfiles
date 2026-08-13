#!/bin/bash
# tmux-diff-pane.sh -- summon / switch the hunk diff pane from the status bar.
#
# The centred "◧ diff" chip opens a display-menu (see tmux.conf MouseUp1Status)
# whose items call:
#   tmux-diff-pane.sh <work|staged|main|last> [dir]  ensure the diff pane, in that mode
#   tmux-diff-pane.sh follow                     re-point at the agent's workdir
#   tmux-diff-pane.sh pick                       fzf popup to choose a worktree
#   tmux-diff-pane.sh autofollow-toggle          flip this window's auto-follow
#   tmux-diff-pane.sh autofollow                 (internal) the agentbar workdir hook
#   tmux-diff-pane.sh layout                     toggle split<->stack
#   tmux-diff-pane.sh close                      kill the diff pane
#   tmux-diff-pane.sh --run <mode> <layout>      (internal) the command run INSIDE the pane
#
# The pane follows the AGENT, not the pane's cwd: @agent_workdir (stamped by the
# agentbar hook on every Edit/Write) is the target, so a session whose Claude
# edits a sibling worktree gets that worktree's diff instead of its own clean
# one. @diff_target records what is on screen; when no agent is touching that
# worktree the chip turns amber, and a click on it follows. Auto-follow (off by
# default, per window) does the same unprompted, via the hook. The pane's own
# rail names the worktree it shows - that comes from its cwd, not from here.
#
# State is WINDOW-scoped: @diff_pane / @diff_mode / @diff_layout are set on the
# active window, so every window owns an independent diff pane (and the chip label
# reflects the current window's). We only ever touch the pane whose id is that
# window's @diff_pane, so a logs/server pane you opened yourself is never clobbered
# -- if that id is dead/unset we split a fresh one. respawn-pane -k collapses
# "missing / a shell / wrong mode" into one path. On quit (q) hunk falls back to an
# interactive shell in the repo, matching the old "run gdw in a pane" habit. The
# source pane / repo are resolved like dictate: the focused pane, unless that IS
# the diff pane (then the agent pane, else any sibling in the window).

set -u

# Absolute: the pane's command is re-exec'd from the target repo, where a
# relative $0 would not resolve (tmux respawns with -c <repo>).
SELF=$(realpath "$0")
AGENT_CMD="${DICTATE_TMUX_CMD:-claude}" # which pane runs the agent (mirrors dictate)
PANE_SIZE="${DIFF_PANE_SIZE:-50%}"      # width of the diff split (matches tmux-realign.sh's even split)
TAB=$(printf '\t')

hunk_bin() {
    # Tests point this at a stub so they never spawn the real viewer (or its daemon).
    [ -n "${DIFF_HUNK_BIN:-}" ] && {
        printf '%s' "$DIFF_HUNK_BIN"
        return
    }
    command -v hunk 2>/dev/null && return
    [ -x "$HOME/.local/bin/hunk" ] && printf '%s' "$HOME/.local/bin/hunk"
}

trace() { "$HOME/.local/bin/dotfiles-trace" log tmux diff "$@" 2>/dev/null || true; }

# hunk args for a mode -> global MARGS array.
set_mode_args() {
    case "$1" in
        work) MARGS=(diff --watch) ;;            # working tree, auto-reload (gdw)
        staged) MARGS=(diff --staged) ;;         # staged only
        main) MARGS=(diff origin/main...HEAD) ;; # branch vs main (gdm)
        last) MARGS=(show) ;;                    # last commit (gds)
        *) return 1 ;;
    esac
}

pane_alive() {
    [ -n "${1:-}" ] || return 1
    tmux list-panes -a -F '#{pane_id}' 2>/dev/null | grep -qxF "$1"
}

# Session shown by the most-recently-active attached client (mirrors dictate).
active_session() {
    tmux list-clients -F "#{client_activity}${TAB}#{client_session}" 2>/dev/null |
        sort -rn | head -1 | cut -f2-
}

# Resolve the target context: SESS, WIN (active window id), ACTIVE (its focused pane).
# A caller that already knows its window (auto-follow, which must act on the
# agent's window rather than whatever the user is looking at) presets both.
resolve_context() {
    [ -n "${WIN:-}" ] && [ -n "${ACTIVE:-}" ] && return 0
    SESS=$(active_session)
    [ -n "$SESS" ] || SESS=$(tmux display-message -p '#{session_name}' 2>/dev/null)
    WIN=$(tmux display-message -p -t "$SESS" '#{window_id}' 2>/dev/null)
    ACTIVE=$(tmux display-message -p -t "$SESS" '#{pane_id}' 2>/dev/null)
}

# Window-scoped user options (state lives on the active window, WIN).
opt_get() { tmux show -wqv -t "$WIN" "$1" 2>/dev/null; }
opt_set() { tmux set -w -t "$WIN" "$1" "$2"; }
opt_unset() { tmux set -uw -t "$WIN" "$1"; }

# Resolve SRC_PANE (repo source) and REPO within WIN. Arg: the diff pane id (may be empty).
#
# REPO is where the agent is *writing*, not where its pane sits: @agent_workdir
# (stamped by the agentbar hook on every Edit/Write) wins over
# #{pane_current_path}, which never follows the Bash tool's `cd` and so keeps
# pointing at the worktree the session was started in - the reason a diff pane
# could sit on a clean tree while the work happened next door. The format read
# resolves pane -> window on its own, so a bash or diff source pane still gets
# the window's latest agent workdir.
resolve_repo() {
    local diff="$1" src=''
    if [ -n "$diff" ] && [ "$ACTIVE" = "$diff" ]; then
        # Focused on the diff pane itself: prefer the agent pane, else any sibling.
        src=$(tmux list-panes -t "$WIN" -F '#{pane_id} #{pane_current_command}' 2>/dev/null |
            awk -v d="$diff" -v c="$AGENT_CMD" '$1!=d && $2==c {print $1; exit}')
        [ -n "$src" ] || src=$(tmux list-panes -t "$WIN" -F '#{pane_id}' 2>/dev/null |
            grep -vxF "$diff" | head -1)
    fi
    [ -n "$src" ] || src="$ACTIVE"
    SRC_PANE="$src"
    REPO=$(agent_workdir "$src")
    [ -d "${REPO:-}" ] || REPO=$(tmux display-message -p -t "$src" '#{pane_current_path}' 2>/dev/null)
}

# Where the agent in (or around) PANE is writing. Empty when no hook has stamped
# one yet - a session that has only read files, or one older than the hook.
agent_workdir() {
    tmux display-message -p -t "$1" '#{@agent_workdir}' 2>/dev/null
}

# cmd_ensure <mode> [dir] [bg]
#   dir  target worktree; default is where the agent is writing (resolve_repo)
#   bg   "bg" respawns without focusing, for auto-follow
cmd_ensure() {
    local mode="$1" want="${2:-}" bg="${3:-}" diff hunk layout
    set_mode_args "$mode" || {
        tmux display-message "diff: unknown mode '$mode'"
        return 0
    }

    resolve_context
    [ -n "${WIN:-}" ] || {
        tmux display-message "diff: no active window"
        return 0
    }

    diff=$(opt_get @diff_pane)
    pane_alive "$diff" || diff=''

    resolve_repo "$diff"
    [ -n "$want" ] && REPO="$want" # explicit target: follow / picker
    [ -n "${REPO:-}" ] || {
        tmux display-message "diff: no source pane"
        return 0
    }

    if ! git -C "$REPO" rev-parse --git-dir >/dev/null 2>&1; then
        tmux display-message "diff: $(basename "$REPO") is not a git repo"
        trace mode="$mode" ok=0 why=notrepo
        return 0
    fi
    hunk=$(hunk_bin)
    [ -n "$hunk" ] || {
        tmux display-message "diff: hunk not on PATH"
        trace mode="$mode" ok=0 why=nohunk
        return 0
    }

    layout=$(opt_get @diff_layout)
    [ -n "$layout" ] || layout='-' # '-' -> hunk's config default

    # hunk probes the terminal on startup (mode reports, OSC 52). tmux routes those
    # replies to the ACTIVE pane, so launching hunk in a background pane leaks the
    # replies as junk input into the agent pane. Launch/relaunch it FOCUSED instead
    # -- except for auto-follow (bg), which must not yank the cursor out of the
    # agent pane mid-typing; that path accepts the risk it was opted into.
    if [ -n "$diff" ]; then
        [ "$bg" = bg ] || tmux select-pane -t "$diff"
        tmux respawn-pane -k -c "$REPO" -t "$diff" "$SELF" --run "$mode" "$layout"
        trace mode="$mode" action=respawn pane="$diff" target="$REPO" bg="${bg:-0}"
    else
        # Unzoom the source window first, or the split gets swallowed.
        [ "$(tmux display-message -p -t "$SRC_PANE" '#{window_zoomed_flag}' 2>/dev/null)" = "1" ] &&
            tmux resize-pane -Z -t "$SRC_PANE"
        # No -d: the new pane becomes active, so hunk's startup queries route to it.
        diff=$(tmux split-window -h -l "$PANE_SIZE" -c "$REPO" -t "$SRC_PANE" \
            -P -F '#{pane_id}' "$SELF" --run "$mode" "$layout")
        opt_set @diff_pane "$diff"
        trace mode="$mode" action=create pane="$diff" target="$REPO"
    fi
    opt_set @diff_mode "$mode"
    # What the pane is showing: the target is what "is the agent working
    # somewhere this pane is not showing?" compares against, and the pane's own
    # cwd is the target too, so its rail names it without any extra state.
    opt_set @diff_target "$REPO"
    tmux refresh-client -S 2>/dev/null
}

# Re-point the diff pane at the agent's current workdir (the amber chip's click,
# and the menu's "Follow agent"). Keeps the current mode.
cmd_follow() {
    local bg="${1:-}" mode dir cur
    resolve_context
    [ -n "${WIN:-}" ] || return 0
    dir=$(agent_workdir "$(opt_get @diff_pane)")
    [ -d "${dir:-}" ] || dir=$(agent_workdir "$ACTIVE")
    [ -d "${dir:-}" ] || {
        tmux display-message "diff: no agent workdir stamped yet"
        return 0
    }
    cur=$(opt_get @diff_target)
    [ "$dir" = "$cur" ] && {
        tmux display-message "diff: already on $(basename "$dir")"
        return 0
    }
    mode=$(opt_get @diff_mode)
    [ -n "$mode" ] || mode=work
    trace action=follow from="$cur" to="$dir" bg="${bg:-0}"
    cmd_ensure "$mode" "$dir" "$bg"
}

# Invoked by the agentbar hook (@agentbar-workdir-cmd) whenever an agent's
# workdir changes: a no-op unless this window opted into auto-follow and has a
# live diff pane. Never focuses, and never touches a window that didn't ask.
cmd_autofollow() {
    local pane="${AGENTBAR_PANE:-}" win diff
    [ -n "$pane" ] || return 0
    win=$(tmux display-message -p -t "$pane" '#{window_id}' 2>/dev/null) || return 0
    [ -n "$win" ] || return 0
    [ "$(tmux show -wqv -t "$win" @diff_follow)" = on ] || return 0
    diff=$(tmux show -wqv -t "$win" @diff_pane)
    pane_alive "$diff" || return 0
    WIN="$win" ACTIVE="$pane" cmd_follow bg
}

# Flip this window's auto-follow preference.
cmd_follow_toggle() {
    local cur
    resolve_context
    [ -n "${WIN:-}" ] || return 0
    cur=$(opt_get @diff_follow)
    case "$cur" in
        on) opt_set @diff_follow off ;;
        *) opt_set @diff_follow on ;;
    esac
    tmux display-message "diff: auto-follow $(opt_get @diff_follow)"
    trace action=autofollow state="$(opt_get @diff_follow)"
}

# Pick a worktree to point the diff pane at (fzf popup; the picker calls back in).
# The picker is resolved next to this script, not via ~/.local/bin: that path only
# exists once the package is stowed, and a missing popup command exits 127.
#
# Sized from the client, not fixed: a percentage alone is unreadable on a laptop
# and a fixed 74x22 is a stamp in the middle of a 250-column screen. Rows are
# ~50 columns, so width is capped once there is nothing left to show.
cmd_pick() {
    local cw ch w h
    cw=$(tmux display-message -p '#{client_width}' 2>/dev/null)
    ch=$(tmux display-message -p '#{client_height}' 2>/dev/null)
    [ -n "$cw" ] && [ -n "$ch" ] || {
        cw=100
        ch=30
    }
    w=$((cw * 55 / 100))
    [ "$w" -lt 74 ] && w=74
    [ "$w" -gt 110 ] && w=110
    [ "$w" -gt "$cw" ] && w=$cw
    h=$((ch * 70 / 100))
    [ "$h" -lt 20 ] && h=20
    [ "$h" -gt 44 ] && h=44
    [ "$h" -gt "$ch" ] && h=$ch
    # The popup's status is the picker's: fzf exits 130 when you press Escape and
    # the shell takes a SIGHUP (129) when the popup is dismissed. run-shell turns
    # any non-zero into an error banner in the pane, so dismissing the picker
    # would look like a crash.
    tmux display-popup -E -w "$w" -h "$h" -T " ◧ pick worktree " \
        "$(dirname "$SELF")/tmux-worktree-picker.sh" || true
}

# Runs INSIDE the diff pane: show the diff, then fall back to a shell on quit.
# Carries the current flavor (--theme, like theme.bash's hunk() wrapper) and the
# chosen layout (--mode split|stack; '-' means leave hunk's config default). env.sh
# may hold a newer THEME than tmux inherited; hunk >=0.17 takes both on diff/show.
cmd_run() {
    local mode="${1:-work}" layout="${2:--}" hunk
    export PATH="$HOME/.local/bin:$PATH"
    [ -f "$HOME/.config/theme/env.sh" ] && . "$HOME/.config/theme/env.sh"
    set_mode_args "$mode" || exec bash -i
    [ "$layout" != '-' ] && MARGS+=(--mode "$layout")
    [ -n "${THEME:-}" ] && MARGS+=(--theme "$THEME")
    hunk=$(hunk_bin)
    [ -n "$hunk" ] && "$hunk" "${MARGS[@]}"
    exec bash -i
}

# Flip this window's @diff_layout split<->stack and re-run its current mode with it.
cmd_flip_layout() {
    local cur new mode diff
    resolve_context
    [ -n "${WIN:-}" ] || return 0
    cur=$(opt_get @diff_layout)
    [ -n "$cur" ] || cur="split" # config default
    case "$cur" in stack) new="split" ;; *) new="stack" ;; esac
    opt_set @diff_layout "$new"
    mode=$(opt_get @diff_mode)
    diff=$(opt_get @diff_pane)
    if [ -n "$mode" ] && pane_alive "$diff"; then
        # Same target: a layout flip must not re-resolve and move the pane.
        cmd_ensure "$mode" "$(opt_get @diff_target)"
    else
        tmux display-message "diff: layout -> $new (applies when you open a diff)"
    fi
}

cmd_close() {
    local diff
    resolve_context
    [ -n "${WIN:-}" ] || return 0
    diff=$(opt_get @diff_pane)
    pane_alive "$diff" && tmux kill-pane -t "$diff"
    opt_unset @diff_pane
    opt_unset @diff_mode # keep @diff_layout and @diff_follow as window preferences
    opt_unset @diff_target
    tmux refresh-client -S 2>/dev/null
    trace action=close
}

case "${1:-}" in
    --run)
        shift
        cmd_run "${1:-work}" "${2:--}"
        ;;
    # A dir argument targets that worktree instead of resolving one (the picker).
    work | staged | main | last) cmd_ensure "$1" "${2:-}" ;;
    follow) cmd_follow ;;
    autofollow) cmd_autofollow ;;
    autofollow-toggle) cmd_follow_toggle ;;
    pick) cmd_pick ;;
    layout) cmd_flip_layout ;;
    close) cmd_close ;;
    *)
        tmux display-message \
            "diff: usage: work|staged|main|last [dir]|follow|pick|autofollow-toggle|layout|close" 2>/dev/null
        ;;
esac
