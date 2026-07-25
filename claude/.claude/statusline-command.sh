#!/usr/bin/env bash
input=$(cat)
model=$(echo "$input" | jq -r '.model.display_name // empty')
used=$(echo "$input" | jq -r '.context_window.used_percentage // empty')

parts=()
[ -n "$model" ] && parts+=("$model")
[ -n "$used" ] && parts+=("ctx: ${used}% used")

printf "%s" "$(
    IFS='|'
    echo "${parts[*]}" | sed 's/|/ | /g'
)"
