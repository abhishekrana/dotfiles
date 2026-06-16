#!/usr/bin/env bash
input=$(cat)
model=$(echo "$input" | jq -r '.model.display_name // empty')
effort=$(echo "$input" | jq -r '.effort.level // empty')
used=$(echo "$input" | jq -r '.context_window.used_percentage // empty')
rl_pct=$(echo "$input" | jq -r '.rate_limits.five_hour.used_percentage // empty')
rl_reset=$(echo "$input" | jq -r '.rate_limits.five_hour.resets_at // empty')

# Compute time until rate limit reset
rl_time=""
if [ -n "$rl_reset" ] && [ "$rl_reset" != "null" ]; then
    secs_left=$(( rl_reset - $(date +%s) ))
    if [ "$secs_left" -gt 0 ]; then
        hrs=$(( secs_left / 3600 ))
        mins=$(( (secs_left % 3600) / 60 ))
        [ "$hrs" -gt 0 ] && rl_time="${hrs}h${mins}m" || rl_time="${mins}m"
    fi
fi

parts=()
[ -n "$model" ] && parts+=("$model")
[ -n "$effort" ] && parts+=("$effort")
[ -n "$used" ] && parts+=("ctx:${used}%")
if [ -n "$rl_pct" ]; then
    rl_str="limit:${rl_pct}%"
    [ -n "$rl_time" ] && rl_str+=" (resets ${rl_time})"
    parts+=("$rl_str")
fi

output="$(IFS='|'; echo "${parts[*]}" | sed 's/|/ · /g')"
printf "%s" "$output"

# Write to per-pane temp file so tmux status bar can display the correct session
pane="${TMUX_PANE:-default}"
[ -n "$output" ] && echo "$output" > "/tmp/claude-status-${pane}"
