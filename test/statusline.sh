#!/usr/bin/env bash
# Guards the Claude status line: what it says about place, and when it warns.
#
# The row's whole job is to name two things - where the session sits, and the
# worktree Claude is writing in when that differs. Both have failed silently
# before: an absent field shifted every field after it, a subdirectory read as a
# move, a task with no `name` rendered as its branch alone, and the warning
# could not fire at all where $TMUX_PANE was unset.
#
# Everything runs with tmux off PATH, which is the point: nothing in this row
# may depend on a multiplexer.
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
pass=0 fail=0

command -v jq >/dev/null 2>&1 || {
    echo "jq missing; skipping" >&2
    exit 0
}

ok() {
    pass=$((pass + 1))
    printf '  \033[32m✓\033[0m %s\n' "$1"
}
no() {
    fail=$((fail + 1))
    printf '  \033[31m✗\033[0m %s\n' "$1"
    printf '      %s\n' "$2"
}
eq() { [ "$2" = "$3" ] && ok "$1" || no "$1" "want [$2] got [$3]"; }

TMP=$(mktemp -d)
STATE=$TMP/state/dotfiles/claude-workdir
SID=test-session
trap 'rm -rf "$TMP"' EXIT

# A repo with a linked worktree, so the .git-as-a-file path is exercised too.
MAIN=$TMP/main
git init -q -b main "$MAIN"
git -C "$MAIN" config user.email t@t
git -C "$MAIN" config user.name t
: >"$MAIN/f"
git -C "$MAIN" add f
git -C "$MAIN" commit -qm init
WT=$TMP/side
git -C "$MAIN" worktree add -q -b side "$WT"

# tmux off PATH: the row must not reach for one. The payload is built by jq, not
# printf, so an absent field cannot leave the JSON malformed.
row() {
    local dir=${1:-$MAIN} used=${2-12} limits=${3-}
    jq -nc --arg sid "$SID" --arg dir "$dir" --arg used "$used" --argjson lim "${limits:-null}" \
        '{model: {display_name: "Opus"}, session_id: $sid, workspace: {current_dir: $dir}}
         + (if $used == "" then {} else {context_window: {used_percentage: ($used | tonumber)}} end)
         + (if $lim == null then {} else {rate_limits: $lim} end)' |
        env XDG_STATE_HOME="$TMP/state" PATH=/usr/bin:/bin \
            bash "$REPO/claude/.claude/statusline-command.sh"
}

# A row without its colours.
plain() { sed 's/\x1b\[[0-9;]*m//g'; }

# What the herdr dictate plugin leaves behind while a dictation is live. No
# argument clears it, as the plugin does when the transcript lands.
DICTATE=$TMP/state/herdr/plugins/abhishekrana.dictate
CHIP="● dictate   "
recording() {
    mkdir -p "$DICTATE"
    [ -n "${1-}" ] || {
        rm -f "$DICTATE/recording.json"
        return
    }
    printf '{"pid":%s,"pane":"w1:p3","submit":true%s}' \
        "$1" "${2:+,\"phase\":\"$2\"}" >"$DICTATE/recording.json"
}

# Row one is the dictation chip and where you are; row two is the model and the meters.
place_row() { row "$@" | sed -n 1p | plain; }
meter_row() { row "$@" | sed -n 2p | plain; }

# A rate_limits object whose windows reset $3 and $4 seconds from now.
limits() {
    jq -nc --argjson five "${1:-null}" --argjson seven "${2:-null}" \
        --argjson fa "$(($(date +%s) + ${3:-$((3 * 86400 + 3600))}))" \
        --argjson sa "$(($(date +%s) + ${4:-$((5 * 86400 + 3600))}))" \
        '(if $five == null then {} else {five_hour: {used_percentage: $five, resets_at: $fa}} end)
         + (if $seven == null then {} else {seven_day: {used_percentage: $seven, resets_at: $sa}} end)'
}

# One PostToolUse event, as Claude Code sends it.
wrote() {
    printf '{"session_id":"%s","tool_name":"%s","tool_input":{"%s":"%s"}}' \
        "$SID" "${2:-Write}" "${3:-file_path}" "$1" |
        env XDG_STATE_HOME="$TMP/state" bash "$REPO/claude/.claude/statusline-workdir.sh"
}

rows() {
    printf '{"tasks":[%s]}' "$1" |
        env XDG_STATE_HOME="$TMP/state" bash "$REPO/claude/.claude/subagent-statusline.sh" |
        jq -r '.content' | sed 's/\x1b\[[0-9;]*m//g'
}

echo "status line: place"

eq "names the checkout and its branch" "${CHIP}main ⎇ main" "$(place_row)"
eq "a linked worktree reads its own branch" "${CHIP}side ⎇ side" "$(place_row "$WT")"
eq "a directory in no repo is just its name" "${CHIP}state" "$(place_row "$TMP/state")"
# An absent percentage used to collapse into the next field and carry the path
# into the context segment.
eq "an absent field shifts nothing" "Opus" "$(meter_row "$MAIN" "")"

echo "status line: the second place"

eq "silent before anything is written" "${CHIP}main ⎇ main" "$(place_row)"

wrote "$WT/f"
eq "names the worktree Claude wrote in" "${CHIP}main ⎇ main   ⚠ side ⎇ side" "$(place_row)"

mkdir -p "$MAIN/sub"
: >"$MAIN/sub/f"
wrote "$MAIN/sub/f"
# Roots are compared, never paths.
eq "a subdirectory is not a move" "${CHIP}main ⎇ main" "$(place_row)"

wrote "$WT/f"
eq "coming home clears the warning" "${CHIP}main ⎇ main" "$(
    wrote "$MAIN/f"
    place_row
)"

wrote "$WT/f"
eq "the session's own view never warns" "${CHIP}side ⎇ side" "$(place_row "$WT")"

wrote "$WT/f"
git -C "$MAIN" worktree remove --force "$WT"
eq "a worktree deleted underneath goes quiet" "${CHIP}main ⎇ main" "$(place_row)"
git -C "$MAIN" worktree add -q "$WT" side

echo "status line: meters"

# Each meter is a word, an 8-cell bar and a padded number, four spaces apart.
CTX12="Opus    context ▰▱▱▱▱▱▱▱  12%"
eq "context reads as a meter" "$CTX12" "$(meter_row)"
eq "an absent window shows nothing" "$CTX12" "$(meter_row "$MAIN" 12 "$(limits)")"
FIVE23="5h ▰▰▱▱▱▱▱▱  23% ↻3d" WEEK41="week ▰▰▰▱▱▱▱▱  41% ↻5d"
eq "both windows read their fill" "$CTX12    $FIVE23    $WEEK41" \
    "$(meter_row "$MAIN" 12 "$(limits 23 41)")"
# The week counts down too, hot or not: how many days are left is what the meter is for.
eq "a quiet week still counts down" "$CTX12    $WEEK41" \
    "$(meter_row "$MAIN" 12 "$(limits null 41)")"
eq "past 80 the reset joins the meter" "$CTX12    week ▰▰▰▰▰▰▰▱  84% ↻3d" \
    "$(meter_row "$MAIN" 12 "$(limits null 84 3600 $((3 * 86400 + 3600)))")"
eq "past 95 reads the same way" "$CTX12    5h ▰▰▰▰▰▰▰▰  96% ↻3d" \
    "$(meter_row "$MAIN" 12 "$(limits 96 null)")"
# The 5h window turns over inside a session, so its countdown never waits for a threshold.
eq "the five-hour window always counts down" "$CTX12    5h ▰▰▱▱▱▱▱▱  23% ↻3d" \
    "$(meter_row "$MAIN" 12 "$(limits 23 null)")"
eq "a window past its reset counts nothing" "$CTX12    5h ▰▰▱▱▱▱▱▱  23%" \
    "$(meter_row "$MAIN" 12 "$(limits 23 null -60)")"
five=$(meter_row "$MAIN" 5)
eq "a meter keeps its width as its number moves" "${#CTX12}" "${#five}"
hot=$(row "$MAIN" 96 | sed -n 2p)
case $hot in *$'\e[31m▰▰▰▰▰▰▰▰\e[0m'*$'\e[31m 96%\e[0m'*) ok "a hot meter turns bar and number red" ;;
*) no "a hot meter turns bar and number red" "$(printf '%q' "$hot")" ;;
esac
calm=$(row "$MAIN" 12 | sed -n 2p)
case $calm in *$'\e[31m'* | *$'\e[33m'*) no "a calm meter stays in the text colour" "$(printf '%q' "$calm")" ;;
*) ok "a calm meter stays in the text colour" ;;
esac
wide=$(row "$MAIN" 96 "$(limits 96 97 60 60)" | sed -n 2p | plain | LC_ALL=C.UTF-8 wc -m)
[ "$wide" -le 98 ] && ok "the meter row fits a 99-column pane" ||
    no "the meter row fits a 99-column pane" "$wide columns"

echo "status line: dictation"

# Colour carries the phase, so the colour is what gets asserted. The label is
# checked separately, and must never change: the place sits beside it.
GREY=96 RED=31 AMBER=33
chip() { row | sed -n 1p | sed -n 's/^\x1b\[\([0-9;]*\)m● dictate.*/\1/p'; }

eq "nothing recording, the chip is grey" "$GREY" "$(chip)"
recording $$
eq "a live recorder turns it red" "$RED" "$(chip)"
recording $$ transcribing
eq "the dictation outlives the microphone" "$AMBER" "$(chip)"
# The plugin omits the field while recording, so absent must read as recording.
recording $$ recording
eq "an explicit phase reads as the absent one" "$RED" "$(chip)"
# A recorder that died must not leave the chip red until the next press.
sleep 0 &
dead=$!
wait $dead
recording $dead
eq "a state file with no process is no recording" "$GREY" "$(chip)"
recording
eq "a cleared file rests again" "$GREY" "$(chip)"

# A dictation on another machine delivering here leaves a phase and an expiry, not a pid.
remote() {
    mkdir -p "$DICTATE"
    printf '{"phase":"%s","until":%d}\n' "$1" "$(($(date +%s) + $2))" >"$DICTATE/remote.json"
}
remote recording 10
eq "a remote dictation turns it red" "$RED" "$(chip)"
remote transcribing 10
eq "a remote dictation outlives the microphone too" "$AMBER" "$(chip)"
remote recording -1
eq "an expired remote dictation is none" "$GREY" "$(chip)"
remote recording 10
sleep 0 &
dead=$!
wait $dead
recording $dead
eq "a dead local recorder does not hide a remote dictation" "$RED" "$(chip)"
recording
rm -f "$DICTATE/remote.json"
eq "a cleared remote file rests again" "$GREY" "$(chip)"

# Nothing beside the chip may move as the phase changes.
resting=$(place_row)
recording $$
eq "recording shifts no text" "$resting" "$(place_row)"
recording $$ transcribing
eq "transcribing shifts no text" "$resting" "$(place_row)"
recording
eq "idle shifts no text" "$resting" "$(place_row)"

echo "status line: what the hook records"

recorded() { cat "$STATE/$SID" 2>/dev/null || echo "(none)"; }

wrote "$MAIN/f"
wrote "$WT/f" Bash
eq "a command is not a write" "$MAIN" "$(recorded)"
wrote "relative.md"
eq "a relative path is not resolvable" "$MAIN" "$(recorded)"
wrote "$TMP/state/loose.md"
eq "a file in no repo records nothing" "$MAIN" "$(recorded)"
wrote "$WT/n.ipynb" NotebookEdit notebook_path
eq "a notebook is a write" "$WT" "$(recorded)"

echo "subagent rows"

task="{\"id\":\"t\",\"name\":\"refactorer\",\"status\":\"running\",\"cwd\":\"$WT\","
task="$task\"tokenCount\":16000,\"contextWindowSize\":200000}"
eq "names the agent and its worktree" "refactorer ⎇ side  running  8%" "$(rows "$task")"
# An inline Agent call leaves .name empty and names itself in label.
task="{\"id\":\"t\",\"label\":\"Idle sleeper\",\"type\":\"general-purpose\",\"status\":\"running\","
task="$task\"cwd\":\"$MAIN\",\"tokenCount\":2000,\"contextWindowSize\":200000}"
eq "falls back to the label" "Idle sleeper ⎇ main  running  1%" "$(rows "$task")"
eq "a task with no cwd keeps its name" "solo  running" \
    "$(rows '{"id":"t","name":"solo","status":"running"}')"
eq "no context window, no percentage" "solo ⎇ main  done" \
    "$(rows "{\"id\":\"t\",\"name\":\"solo\",\"status\":\"done\",\"cwd\":\"$MAIN\",\"tokenCount\":50}")"

echo
printf '%d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
