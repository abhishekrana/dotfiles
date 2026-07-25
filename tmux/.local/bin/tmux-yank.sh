#!/bin/bash
# Copy stdin to both Wayland clipboard and primary selection
buf=$(cat)
printf '%s' "$buf" | wl-copy
printf '%s' "$buf" | wl-copy --primary
bytes=$(printf %s "$buf" | wc -c | tr -d '[:space:]')
"$HOME/.local/bin/dotfiles-trace" log yank copy bytes="$bytes" 2>/dev/null || true
