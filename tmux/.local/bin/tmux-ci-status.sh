#!/bin/bash
# Shows GitLab CI pipeline status for the current git branch in tmux status bar
cd "$(tmux display-message -p '#{pane_current_path}' 2>/dev/null)" 2>/dev/null || exit 0

# Check if inside a git repo
git rev-parse --is-inside-work-tree &>/dev/null || exit 0

pipeline_state=$(glab ci status 2>/dev/null | sed 's/\x1b\[[0-9;]*[a-zA-Z]//g' | grep 'Pipeline state:' | awk '{print $NF}')

case "$pipeline_state" in
  running)  echo "CI:running" ;;
  success)  echo "CI:passed" ;;
  failed)   echo "CI:FAILED" ;;
  canceled) echo "CI:canceled" ;;
  pending)  echo "CI:pending" ;;
  created)  echo "CI:created" ;;
  *)        echo "" ;;
esac
