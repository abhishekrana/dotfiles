#!/usr/bin/env bash
# fzf-based tmux session picker. Shows branch + Claude state inline, previews
# windows with per-pane state breakdown. Opens with the current session
# pre-selected; Enter switches to the highlighted session.
#
# Keys (in addition to fzf defaults):
#   j / k         move cursor down / up
#   g / G         jump to first / last
#   K / J         move highlighted session up / down (persisted)
#   r             rename highlighted session
#   D             kill highlighted session (with confirm)
#   c             create new session (prompts for name + starting dir)
#   ?             show this help overlay
#   Alt-;         dismiss picker
#
# Order is persisted in $XDG_STATE_HOME/tmux/session-order, one name per line.
# Sessions not in that file are appended alphabetically. The file auto-prunes
# dead sessions whenever any move or create rewrites it.
#
# Internal subcommands (invoked by fzf reload/execute callbacks):
#   --list                emit fzf line data (NAME<TAB>display)
#   --move-up   NAME      shift NAME up in the order file
#   --move-down NAME      shift NAME down
#   --rename    NAME      interactive rename of NAME
#   --kill      NAME      interactive kill of NAME
#   --new                 interactive new-session prompt
#   --help                show help overlay

set -euo pipefail

order_file="${XDG_STATE_HOME:-$HOME/.local/state}/tmux/session-order"
mkdir -p "$(dirname "$order_file")"
touch "$order_file"

# ---- Helpers ---------------------------------------------------------------

flash_error() {
  printf '\n✗ %s\n' "$1" >&2
  sleep 1.2
}

session_exists() {
  tmux list-sessions -F '#{session_name}' 2>/dev/null | grep -Fxq -- "$1"
}

# Emit live session names in display order: order-file entries first (skipping
# dead ones and dedup'd), then any remaining live sessions alphabetically.
ordered_session_names() {
  local -A live seen
  local s
  while IFS= read -r s; do live[$s]=1; done < <(tmux list-sessions -F '#{session_name}')
  while IFS= read -r s; do
    [ -z "$s" ] && continue
    if [ -n "${live[$s]:-}" ] && [ -z "${seen[$s]:-}" ]; then
      printf '%s\n' "$s"
      seen[$s]=1
    fi
  done < "$order_file"
  while IFS= read -r s; do
    [ -z "${seen[$s]:-}" ] && printf '%s\n' "$s"
  done < <(tmux list-sessions -F '#{session_name}' | sort)
  return 0
}

# ---- Order-file mutations (all atomic via mv) ------------------------------

move_in_order() {
  local target=$1 dir=$2 tmp
  tmp=$(mktemp)
  ordered_session_names | awk -v t="$target" -v d="$dir" '
    { a[NR] = $0 }
    END {
      idx = 0
      for (i = 1; i <= NR; i++) if (a[i] == t) { idx = i; break }
      j = idx + d
      if (idx >= 1 && j >= 1 && j <= NR) { x = a[idx]; a[idx] = a[j]; a[j] = x }
      for (i = 1; i <= NR; i++) print a[i]
    }
  ' > "$tmp"
  mv "$tmp" "$order_file"
}

rename_in_order() {
  local old=$1 new=$2 tmp
  tmp=$(mktemp)
  awk -v o="$old" -v n="$new" '{ print ($0 == o) ? n : $0 }' "$order_file" > "$tmp"
  mv "$tmp" "$order_file"
}

remove_in_order() {
  local target=$1 tmp
  tmp=$(mktemp)
  awk -v t="$target" '$0 != t' "$order_file" > "$tmp"
  mv "$tmp" "$order_file"
}

# Force NAME to be the last entry. Also rewrites the file from
# ordered_session_names, so previously-unfiled sessions get materialized
# alphabetically (predictable positions afterwards).
pin_last_in_order() {
  local target=$1 tmp
  tmp=$(mktemp)
  ordered_session_names | awk -v t="$target" '$0 != t' > "$tmp"
  printf '%s\n' "$target" >> "$tmp"
  mv "$tmp" "$order_file"
}

# ---- Interactive actions ---------------------------------------------------

do_help() {
  clear
  cat <<'EOF'
tmux session picker

  j / k       cursor down / up
  g / G       jump to first / last
  K / J       move session up / down (persisted)
  r           rename session
  D           kill session (with confirm)
  c           new session (name + start dir)
  ?           this help
  Enter       switch to selected session
  Alt-;       dismiss

EOF
  printf 'press any key to dismiss...'
  read -rsn1 _
}

do_kill() {
  local name=$1 ans err
  [ -z "$name" ] && return 0
  clear
  printf 'kill session "%s"? [y/N] ' "$name"
  read -r ans || return 0
  case $ans in y|Y|yes) ;; *) return 0 ;; esac
  if ! err=$(tmux kill-session -t "$name" 2>&1); then
    flash_error "$err"
    return 0
  fi
  remove_in_order "$name"
  "$HOME/.local/bin/dotfiles-trace" log picker kill name="$name" 2>/dev/null || true
}

do_new() {
  local name dir default_dir err
  clear
  read -re -p "new session name: " name || return 0
  [ -z "$name" ] && return 0
  if session_exists "$name"; then
    flash_error "session \"$name\" already exists"
    return 0
  fi
  default_dir=$(tmux display-message -p '#{pane_current_path}' 2>/dev/null || true)
  { [ -z "$default_dir" ] || [ ! -d "$default_dir" ]; } && default_dir=$HOME
  read -re -i "$default_dir" -p "start dir: " dir || return 0
  dir=${dir/#~/$HOME}
  if [ ! -d "$dir" ]; then
    flash_error "no such directory: $dir"
    return 0
  fi
  if ! err=$(tmux new-session -d -s "$name" -c "$dir" 2>&1); then
    flash_error "$err"
    return 0
  fi
  pin_last_in_order "$name"
  "$HOME/.local/bin/dotfiles-trace" log picker new name="$name" dir="$dir" 2>/dev/null || true
}

do_rename() {
  local old=$1 new err
  [ -z "$old" ] && return 0
  clear
  read -re -i "$old" -p "rename to: " new || return 0
  [ -z "$new" ] && return 0
  [ "$new" = "$old" ] && return 0
  if session_exists "$new"; then
    flash_error "session \"$new\" already exists"
    return 0
  fi
  if ! err=$(tmux rename-session -t "$old" -- "$new" 2>&1); then
    flash_error "rename failed: $err"
    return 0
  fi
  rename_in_order "$old" "$new"
  "$HOME/.local/bin/dotfiles-trace" log picker rename from="$old" to="$new" 2>/dev/null || true
}

# ---- Subcommand dispatch ---------------------------------------------------

list_only=0
case "${1:-}" in
  --list)       list_only=1 ;;
  --help)       do_help;                 exit 0 ;;
  --kill)       do_kill "${2:-}";        exit 0 ;;
  --move-down)  move_in_order "${2:-}"  1; exit 0 ;;
  --move-up)    move_in_order "${2:-}" -1; exit 0 ;;
  --new)        do_new;                  exit 0 ;;
  --rename)     do_rename "${2:-}";      exit 0 ;;
esac

# ---- Main flow: build display lines ----------------------------------------

current=$(tmux display-message -p '#S')
sessions=$(ordered_session_names)
[ -z "$sessions" ] && exit 0

# Aggregated Claude state per tmux session, worst state wins. State comes from
# the @agent_* pane options the agentbar hook stamps on each Claude
# pane - the same source the sidebar itself reads, so picker and sidebar always
# agree. Pane options die with the pane, so no staleness/liveness bookkeeping.
# Priority: permission(4) > asking(3) > working(2) > done(1) > blank(0).
declare -A STATE_BY_SESSION
state_rank() {
  case $1 in
    permission) printf 4 ;;
    question)   printf 3 ;;
    working)    printf 2 ;;
    done)       printf 1 ;;
    *)          printf 0 ;;
  esac
}

# Glyphs + colours match the sidebar's five-state language (Solarized Light):
# ◔ blocked=red, ? asking=amber, ⠹ working=cyan, ✓ done=green. Needs fzf --ansi.
icon_for() {
  case $1 in
    permission) printf '\033[38;2;220;50;47m◔\033[0m' ;;
    question)   printf '\033[38;2;181;137;0m?\033[0m' ;;
    working)    printf '\033[38;2;42;161;152m⠹\033[0m' ;;
    done)       printf '\033[38;2;133;153;0m✓\033[0m' ;;
    *)          printf ' ' ;;
  esac
}

# One pass over the server's panes: a pane is a live agent when the hook has
# stamped @agent_present=1 and its foreground command is still claude/node
# (guards a pane whose Claude died but left its options behind) - the same
# filter the sidebar uses.
while IFS=$'\t' read -r sess cmd present state; do
  [ "$present" = 1 ] || continue
  case $cmd in claude|node) ;; *) continue ;; esac
  prev=${STATE_BY_SESSION[$sess]:-}
  if [ "$(state_rank "$state")" -gt "$(state_rank "$prev")" ]; then
    STATE_BY_SESSION[$sess]=$state
  fi
done < <(tmux list-panes -a -F $'#{session_name}\t#{pane_current_command}\t#{@agent_present}\t#{@agent_state}')

# Render lines as "<name>TAB<display>" - fzf hides field 1 via --with-nth=2
# but we use it for {1} placeholder substitution in binds.
lines=$(
  while IFS= read -r name; do
    [ -z "$name" ] && continue
    path=$(tmux display-message -p -t "$name" '#{pane_current_path}' 2>/dev/null || true)
    branch=
    if [ -n "$path" ] && [ -d "$path" ]; then
      branch=$(git -C "$path" symbolic-ref --short HEAD 2>/dev/null \
               || git -C "$path" rev-parse --short HEAD 2>/dev/null \
               || true)
    fi
    icon=$(icon_for "${STATE_BY_SESSION[$name]:-}")
    [ "$name" = "$current" ] && mark='▸' || mark=' '
    printf '%s\t%s %s %-18s  %s\n' "$name" "$mark" "$icon" "$name" "$branch"
  done <<< "$sessions"
)

if [ "$list_only" = 1 ]; then
  printf '%s\n' "$lines"
  exit 0
fi

# ---- fzf invocation --------------------------------------------------------

current_pos=$(printf '%s\n' "$lines" | awk -F'\t' -v c="$current" '$1 == c { print NR; exit }')
: "${current_pos:=1}"

self=$(realpath "$0")

# --sync + start:pos ensures the initial cursor position fires exactly once,
# at startup. Using load:pos here would re-fire on every reload, snapping
# the cursor back to the original session after rename/move/kill/new.
target=$(
  printf '%s\n' "$lines" \
    | fzf --ansi --sync --reverse --no-input --highlight-line \
          --header='? for help' --header-first \
          --delimiter=$'\t' --with-nth=2 \
          --preview "$HOME/.local/bin/tmux-session-preview.sh {1}" \
          --preview-window=down:50% \
          --pointer=' ' \
          --color='bg+:#268bd2,fg+:#fdf6e3,gutter:-1,pointer:-1,hl:#268bd2,hl+:#fdf6e3,border:#93a1a1,info:#93a1a1,prompt:#586e75' \
          --bind "start:pos($current_pos)" \
          --bind 'j:down,k:up,g:first,G:last,alt-;:abort' \
          --bind "?:execute($self --help)" \
          --bind "r:execute($self --rename {1})+reload($self --list)" \
          --bind "D:execute($self --kill {1})+reload($self --list)" \
          --bind "c:execute($self --new)+reload($self --list)+last" \
          --bind "K:execute-silent($self --move-up {1})+reload($self --list)+up" \
          --bind "J:execute-silent($self --move-down {1})+reload($self --list)+down" \
    | cut -f1
) || exit 0

if [ -n "$target" ]; then
  tmux switch-client -t "$target"
  "$HOME/.local/bin/dotfiles-trace" log picker switch target="$target" 2>/dev/null || true
fi
