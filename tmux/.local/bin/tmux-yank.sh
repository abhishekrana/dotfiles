#!/bin/bash
# Copy stdin to the system clipboard (and Linux primary selection when available).
buf=$(cat)
if command -v pbcopy >/dev/null 2>&1; then
    printf '%s' "$buf" | pbcopy
elif command -v wl-copy >/dev/null 2>&1; then
    printf '%s' "$buf" | wl-copy
    printf '%s' "$buf" | wl-copy --primary
elif command -v xclip >/dev/null 2>&1; then
    printf '%s' "$buf" | xclip -selection clipboard
    printf '%s' "$buf" | xclip -selection primary
fi
