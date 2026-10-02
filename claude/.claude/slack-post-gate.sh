#!/usr/bin/env bash
# PreToolUse gate: every Slack write needs confirmation, with the message shown. A post or an
# edit to a channel on the unattended allowlist goes through; deletes, scheduled messages,
# uploads, reactions and pins always ask. No allowlist means every write asks.
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
write_methods='chat\.(postMessage|update|delete|scheduleMessage|deleteScheduledMessage|postEphemeral|meMessage)'
write_methods="$write_methods|files\.(upload|uploadV2|getUploadURLExternal|completeUploadExternal|delete)"
write_methods="$write_methods|reactions\.(add|remove)|pins\.(add|remove)"
method=$(grep -oE "\b($write_methods)\b" <<<"$cmd" | head -1)
[ -n "$method" ] || {
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
# `slack api chat.postMessage ... text=<literal>` or `text="$(cat <file>)"`: shlex reads the quoting, a file is read.
if [ -z "$text" ] && command -v python3 >/dev/null 2>&1; then
    text=$(
        python3 - "$cmd" <<'PY' 2>/dev/null
import pathlib, re, shlex, sys
try:
    tokens = shlex.split(sys.argv[1])
except ValueError:
    sys.exit(0)
for tok in tokens:
    if tok.startswith("text="):
        value = tok[len("text="):]
        m = re.fullmatch(r"\$\((?:cat\s+|<\s*)(\S+)\)", value)
        if m:
            path = pathlib.Path(m.group(1)).expanduser()
            value = path.read_text() if path.is_file() else f"(text is read from {path}, which does not exist)"
        sys.stdout.write(value)
        break
PY
    )
fi
[ -n "$channel" ] || channel=$(grep -oE '\bC[A-Z0-9]{8,}\b' <<<"$cmd" | head -1)

case "$method" in
chat.postMessage | chat.update)
    for c in $allowed; do
        [ -n "$channel" ] && [ "$channel" = "$c" ] && {
            echo '{}'
            exit 0
        }
    done
    why="which is not on the unattended allowlist"
    ;;
*) why="which always asks" ;;
esac

if [ -z "$channel" ]; then
    reason="Slack ${method} whose target channel could not be read from the command."
    reason="$reason Confirm the channel and message before sending."
else
    preview=$text
    [ -n "$preview" ] || preview="(message text not found in the command; inspect it before approving)"
    reason="Slack ${method} to channel ${channel}, ${why}.

--- message ---
${preview}
--- end ---"
fi
jq -n --arg r "$reason" \
    '{hookSpecificOutput:{hookEventName:"PreToolUse",permissionDecision:"ask",permissionDecisionReason:$r}}'
