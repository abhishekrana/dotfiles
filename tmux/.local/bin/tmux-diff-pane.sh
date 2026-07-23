#!/bin/bash
# tmux-diff-pane.sh -- summon / switch the hunk diff pane from the status bar.
#
# The centred "◧ diff" chip opens a display-menu (see tmux.conf MouseUp1Status)
# whose items call:
#   tmux-diff-pane.sh <work|staged|main|last>   ensure the diff pane, in that mode
#   tmux-diff-pane.sh focus                      focus the diff pane
#   tmux-diff-pane.sh close                      kill the diff pane
#   tmux-diff-pane.sh --run <mode>               (internal) the command run INSIDE the pane
#
# One dedicated pane is tracked in @diff_pane; @diff_mode drives the chip label.
# We only ever touch the pane whose id is @diff_pane, so a logs/server pane you
# opened yourself is never clobbered -- if that id is dead/unset we split a fresh
# one. respawn-pane -k collapses "missing / a shell / wrong mode" into one path.
# On quit (q) hunk falls back to an interactive shell in the repo, matching the
# old "run gdw in a pane" habit. Source pane / repo are resolved the same way
# dictate picks its target: focused pane, unless that IS the diff pane.

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

# Resolve SRC_PANE (repo source) and REPO. Arg: the diff pane id (may be empty).
resolve_repo() {
  local diff="$1" sess active src=''
  sess=$(active_session)
  [ -n "$sess" ] || sess=$(tmux display-message -p '#{session_name}' 2>/dev/null)
  active=$(tmux display-message -p -t "$sess" '#{pane_id}' 2>/dev/null)
  if [ -n "$diff" ] && [ "$active" = "$diff" ]; then
    # Focused on the diff pane itself: prefer the agent pane, else any other pane.
    src=$(tmux list-panes -s -t "$sess" -F '#{pane_id} #{pane_current_command}' 2>/dev/null \
          | awk -v d="$diff" -v c="$AGENT_CMD" '$1!=d && $2==c {print $1; exit}')
    [ -n "$src" ] || src=$(tmux list-panes -s -t "$sess" -F '#{pane_id}' 2>/dev/null \
                           | grep -vxF "$diff" | head -1)
  fi
  [ -n "$src" ] || src="$active"
  SRC_PANE="$src"
  REPO=$(tmux display-message -p -t "$src" '#{pane_current_path}' 2>/dev/null)
}

cmd_ensure() {
  local mode="$1" diff hunk
  set_mode_args "$mode" || { tmux display-message "diff: unknown mode '$mode'"; return 0; }

  diff=$(tmux show -gv @diff_pane 2>/dev/null)
  pane_alive "$diff" || diff=''

  resolve_repo "$diff"
  [ -n "${REPO:-}" ] || { tmux display-message "diff: no source pane"; return 0; }

  if ! git -C "$REPO" rev-parse --git-dir >/dev/null 2>&1; then
    tmux display-message "diff: $(basename "$REPO") is not a git repo"
    trace mode="$mode" ok=0 why=notrepo; return 0
  fi
  hunk=$(hunk_bin)
  [ -n "$hunk" ] || { tmux display-message "diff: hunk not on PATH"; trace mode="$mode" ok=0 why=nohunk; return 0; }

  # hunk probes the terminal on startup (mode reports, OSC 52). tmux routes those
  # replies to the ACTIVE pane, so launching hunk in a background pane leaks the
  # replies as junk input into the agent pane. Launch/relaunch it FOCUSED instead.
  if [ -n "$diff" ]; then
    tmux select-pane -t "$diff"
    tmux respawn-pane -k -c "$REPO" -t "$diff" "$SELF" --run "$mode"
    trace mode="$mode" action=respawn pane="$diff"
  else
    # Unzoom the source window first, or the split gets swallowed.
    [ "$(tmux display-message -p -t "$SRC_PANE" '#{window_zoomed_flag}' 2>/dev/null)" = "1" ] \
      && tmux resize-pane -Z -t "$SRC_PANE"
    # No -d: the new pane becomes active, so hunk's startup queries route to it.
    diff=$(tmux split-window -h -l "$PANE_SIZE" -c "$REPO" -t "$SRC_PANE" \
            -P -F '#{pane_id}' "$SELF" --run "$mode")
    tmux set -g @diff_pane "$diff"
    trace mode="$mode" action=create pane="$diff"
  fi
  tmux set -g @diff_mode "$mode"
  tmux refresh-client -S
}

# Runs INSIDE the diff pane: show the diff, then fall back to a shell on quit.
# Mirrors the hunk() wrapper in theme.bash: carry the current flavor via --theme
# on the `diff` subcommand (env.sh may be newer than tmux's inherited THEME).
cmd_run() {
  local mode="${1:-work}" hunk
  export PATH="$HOME/.local/bin:$PATH"
  [ -f "$HOME/.config/theme/env.sh" ] && . "$HOME/.config/theme/env.sh"
  set_mode_args "$mode" || exec bash -i
  hunk=$(hunk_bin)
  if [ -n "$hunk" ]; then
    if [ -n "${THEME:-}" ] && [ "${MARGS[0]}" = diff ]; then
      "$hunk" "${MARGS[@]}" --theme "$THEME"
    else
      "$hunk" "${MARGS[@]}"
    fi
  fi
  exec bash -i
}

cmd_focus() {
  local diff; diff=$(tmux show -gv @diff_pane 2>/dev/null)
  if pane_alive "$diff"; then
    tmux select-pane -t "$diff"
  else
    tmux display-message "diff: no diff pane yet (pick a mode first)"
  fi
}

cmd_close() {
  local diff; diff=$(tmux show -gv @diff_pane 2>/dev/null)
  pane_alive "$diff" && tmux kill-pane -t "$diff"
  tmux set -gu @diff_pane
  tmux set -gu @diff_mode
  tmux refresh-client -S
  trace action=close
}

case "${1:-}" in
  --run)                  shift; cmd_run "${1:-work}" ;;
  work|staged|main|last)  cmd_ensure "$1" ;;
  focus)                  cmd_focus ;;
  close)                  cmd_close ;;
  *)                      tmux display-message "diff: usage: work|staged|main|last|focus|close" 2>/dev/null ;;
esac
