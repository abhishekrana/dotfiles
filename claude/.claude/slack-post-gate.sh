#!/usr/bin/env bash
# PreToolUse gate: a Slack post needs confirmation, with the message shown, unless its
# channel is listed in the unattended allowlist. No allowlist means every post asks.
#
# Allowlist: $SLACK_UNATTENDED_CHANNELS_FILE, else ~/.config/claude/slack-unattended-channels.
# One channel id per line; '#' comments and blank lines ignored. Keep it out of version
# control - channel ids describe a private workspace.
set -uo pipefail

payload=$(cat 2>/dev/null) || {
    echo '{}'
    exit 0
}
command -v jq >/dev/null 2>&1 || {
    echo '{}'
    exit 0
}
[ "$(jq -r '.tool_name // empty' <<<"$payload")" = "Bash" ] || {
    echo '{}'
    exit 0
}
cmd=$(jq -r '.tool_input.command // empty' <<<"$payload")
grep -q 'chat\.postMessage' <<<"$cmd" || {
    echo '{}'
    exit 0
}

allow_file=${SLACK_UNATTENDED_CHANNELS_FILE:-$HOME/.config/claude/slack-unattended-channels}
allowed=""
[ -f "$allow_file" ] &&
    allowed=$(sed -e 's/#.*//' -e 's/[[:space:]]//g' "$allow_file" 2>/dev/null | grep -v '^$' || true)

channel=""
text=""
body_file=$(grep -oE '(--data-binary|--data|-d)[[:space:]]+@[^[:space:]]+' <<<"$cmd" | head -1 | sed -E 's/.*@//')
if [ -n "$body_file" ] && [ -f "$body_file" ]; then
    channel=$(jq -r '.channel // empty' "$body_file" 2>/dev/null)
    text=$(jq -r '.text // empty' "$body_file" 2>/dev/null)
fi
[ -n "$channel" ] || channel=$(grep -oE '\bC[A-Z0-9]{8,}\b' <<<"$cmd" | head -1)

for c in $allowed; do
    [ -n "$channel" ] && [ "$channel" = "$c" ] && {
        echo '{}'
        exit 0
    }
done

if [ -z "$channel" ]; then
    reason="Slack post whose target channel could not be read from the command."
    reason="$reason Confirm the channel and message before sending."
else
    preview=$(printf '%s' "$text" | head -c 1200)
    [ -n "$preview" ] || preview="(message text not found in the command; inspect it before approving)"
    reason="Slack post to channel ${channel}, which is not on the unattended allowlist.

--- message ---
${preview}
--- end ---"
fi
jq -n --arg r "$reason" \
    '{hookSpecificOutput:{hookEventName:"PreToolUse",permissionDecision:"ask",permissionDecisionReason:$r}}'
