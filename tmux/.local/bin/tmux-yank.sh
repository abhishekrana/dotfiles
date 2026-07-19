#!/bin/bash
# Copy stdin to both Wayland clipboard and primary selection
buf=$(cat)
printf '%s' "$buf" | wl-copy
printf '%s' "$buf" | wl-copy --primary
"$HOME/.local/bin/dotfiles-trace" log yank copy bytes="$(printf %s "$buf" | wc -c | tr -d '[:space:]')" 2>/dev/null || true
