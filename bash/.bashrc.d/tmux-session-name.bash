_tmux_rename_session_to_repo_branch() {
  [ -n "$TMUX" ] || return 0
  local sid
  sid=$(tmux display-message -p '#{session_id}' 2>/dev/null) || return 0
  "$HOME/.local/bin/tmux-rename-session.sh" "$sid" "$PWD" >/dev/null 2>&1 || true
}

case "$PROMPT_COMMAND" in
  *_tmux_rename_session_to_repo_branch*) ;;
  *) PROMPT_COMMAND="_tmux_rename_session_to_repo_branch${PROMPT_COMMAND:+;$PROMPT_COMMAND}" ;;
esac
