#!/bin/bash
# tmux-diff-pane.sh -- summon / switch the hunk diff pane from the status bar.
#
# The centred "◧ diff" chip opens a display-menu (see tmux.conf MouseUp1Status)
# whose items call:
#   tmux-diff-pane.sh <work|staged|main|last>   ensure the diff pane, in that mode
#   tmux-diff-pane.sh layout                     toggle split<->stack
#   tmux-diff-pane.sh close                      kill the diff pane
#   tmux-diff-pane.sh --run <mode> <layout>      (internal) the command run INSIDE the pane
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

SELF="$0"
AGENT_CMD="${DICTATE_TMUX_CMD:-claude}"     # which pane runs the agent (mirrors dictate)
PANE_SIZE="${DIFF_PANE_SIZE:-45%}"          # width of the diff split
TAB=$(printf '\t')

hunk_bin() {
  command -v hunk 2>/dev/null && return
  [ -x "$HOME/.local/bin/hunk" ] && printf '%s' "$HOME/.local/bin/hunk"
}

trace() { "$HOME/.local/bin/dotfiles-trace" log tmux diff "$@" 2>/dev/null || true; }

# hunk args for a mode -> global MARGS array.
set_mode_args() {
  case "$1" in
    work)   MARGS=(diff --watch) ;;              # working tree, auto-reload (gdw)
    staged) MARGS=(diff --staged) ;;             # staged only
    main)   MARGS=(diff origin/main...HEAD) ;;   # branch vs main (gdm)
    last)   MARGS=(show) ;;                      # last commit (gds)
    *)      return 1 ;;
  esac
}

pane_alive() {
  [ -n "${1:-}" ] || return 1
  tmux list-panes -a -F '#{pane_id}' 2>/dev/null | grep -qxF "$1"
}

# Session shown by the most-recently-active attached client (mirrors dictate).
active_session() {
  tmux list-clients -F "#{client_activity}${TAB}#{client_session}" 2>/dev/null \
    | sort -rn | head -1 | cut -f2-
}

# Resolve the target context: SESS, WIN (active window id), ACTIVE (its focused pane).
resolve_context() {
  SESS=$(active_session)
  [ -n "$SESS" ] || SESS=$(tmux display-message -p '#{session_name}' 2>/dev/null)
  WIN=$(tmux display-message -p -t "$SESS" '#{window_id}' 2>/dev/null)
  ACTIVE=$(tmux display-message -p -t "$SESS" '#{pane_id}' 2>/dev/null)
}

# Window-scoped user options (state lives on the active window, WIN).
opt_get()   { tmux show -wqv -t "$WIN" "$1" 2>/dev/null; }
opt_set()   { tmux set -w -t "$WIN" "$1" "$2"; }
opt_unset() { tmux set -uw -t "$WIN" "$1"; }

# Resolve SRC_PANE (repo source) and REPO within WIN. Arg: the diff pane id (may be empty).
resolve_repo() {
  local diff="$1" src=''
  if [ -n "$diff" ] && [ "$ACTIVE" = "$diff" ]; then
    # Focused on the diff pane itself: prefer the agent pane, else any sibling.
    src=$(tmux list-panes -t "$WIN" -F '#{pane_id} #{pane_current_command}' 2>/dev/null \
          | awk -v d="$diff" -v c="$AGENT_CMD" '$1!=d && $2==c {print $1; exit}')
    [ -n "$src" ] || src=$(tmux list-panes -t "$WIN" -F '#{pane_id}' 2>/dev/null \
                           | grep -vxF "$diff" | head -1)
  fi
  [ -n "$src" ] || src="$ACTIVE"
  SRC_PANE="$src"
  REPO=$(tmux display-message -p -t "$src" '#{pane_current_path}' 2>/dev/null)
}

cmd_ensure() {
  local mode="$1" diff hunk layout
  set_mode_args "$mode" || { tmux display-message "diff: unknown mode '$mode'"; return 0; }

  resolve_context
  [ -n "${WIN:-}" ] || { tmux display-message "diff: no active window"; return 0; }

  diff=$(opt_get @diff_pane)
  pane_alive "$diff" || diff=''

  resolve_repo "$diff"
  [ -n "${REPO:-}" ] || { tmux display-message "diff: no source pane"; return 0; }

  if ! git -C "$REPO" rev-parse --git-dir >/dev/null 2>&1; then
    tmux display-message "diff: $(basename "$REPO") is not a git repo"
    trace mode="$mode" ok=0 why=notrepo; return 0
  fi
  hunk=$(hunk_bin)
  [ -n "$hunk" ] || { tmux display-message "diff: hunk not on PATH"; trace mode="$mode" ok=0 why=nohunk; return 0; }

  layout=$(opt_get @diff_layout); [ -n "$layout" ] || layout='-'   # '-' -> hunk's config default

  # hunk probes the terminal on startup (mode reports, OSC 52). tmux routes those
  # replies to the ACTIVE pane, so launching hunk in a background pane leaks the
  # replies as junk input into the agent pane. Launch/relaunch it FOCUSED instead.
  if [ -n "$diff" ]; then
    tmux select-pane -t "$diff"
    tmux respawn-pane -k -c "$REPO" -t "$diff" "$SELF" --run "$mode" "$layout"
    trace mode="$mode" action=respawn pane="$diff"
  else
    # Unzoom the source window first, or the split gets swallowed.
    [ "$(tmux display-message -p -t "$SRC_PANE" '#{window_zoomed_flag}' 2>/dev/null)" = "1" ] \
      && tmux resize-pane -Z -t "$SRC_PANE"
    # No -d: the new pane becomes active, so hunk's startup queries route to it.
    diff=$(tmux split-window -h -l "$PANE_SIZE" -c "$REPO" -t "$SRC_PANE" \
            -P -F '#{pane_id}' "$SELF" --run "$mode" "$layout")
    opt_set @diff_pane "$diff"
    trace mode="$mode" action=create pane="$diff"
  fi
  opt_set @diff_mode "$mode"
  tmux refresh-client -S
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
  cur=$(opt_get @diff_layout); [ -n "$cur" ] || cur=split   # config default
  case "$cur" in stack) new=split ;; *) new=stack ;; esac
  opt_set @diff_layout "$new"
  mode=$(opt_get @diff_mode)
  diff=$(opt_get @diff_pane)
  if [ -n "$mode" ] && pane_alive "$diff"; then
    cmd_ensure "$mode"                              # respawn picks up the new @diff_layout
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
  opt_unset @diff_mode                              # keep @diff_layout as the window's preference
  tmux refresh-client -S
  trace action=close
}

case "${1:-}" in
  --run)                  shift; cmd_run "${1:-work}" "${2:--}" ;;
  work|staged|main|last)  cmd_ensure "$1" ;;
  layout)                 cmd_flip_layout ;;
  close)                  cmd_close ;;
  *)                      tmux display-message "diff: usage: work|staged|main|last|layout|close" 2>/dev/null ;;
esac
