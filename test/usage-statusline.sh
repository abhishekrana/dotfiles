#!/usr/bin/env bash
# Guards the shareable status line: the model, the context meter and the two usage windows.
#
# It is installed on machines this repo never sees, so it may rely on nothing but bash 3.2 and jq.
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT=$REPO/claude/.claude/usage-statusline.sh
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

NOW=$(date +%s)

# The rendered row for payload $1.
raw() { printf '%s' "$1" | bash "$SCRIPT"; }
# The same row without its colours.
row() { raw "$1" | sed $'s/\033\\[[0-9;]*m//g'; }

# A payload with model $1, context $2, and 5h / week windows at $3 / $5 resetting $4 / $6 seconds from now. An empty
# argument leaves that field out.
payload() {
    jq -nc --arg m "$1" --arg ctx "$2" --arg f "$3" --arg fa "$4" --arg w "$5" --arg wa "$6" --argjson now "$NOW" '
        def at($s): if $s == "" then {} else {resets_at: ($now + ($s | tonumber))} end;
        def win($p; $s): if $p == "" then {} else {used_percentage: ($p | tonumber)} + at($s) end;
        (if $m == "" then {} else {model: {display_name: $m}} end)
        + (if $ctx == "" then {} else {context_window: {used_percentage: ($ctx | tonumber)}} end)
        + ({five_hour: win($f; $fa), seven_day: win($w; $wa)}
            | with_entries(select(.value != {})) | if . == {} then {} else {rate_limits: .} end)'
}

echo "usage status line: meters"
# The expected row is built in pieces: `task width` counts bytes where there is no locale, and ▰ is three.
full="Opus 5.5 1M    context ▰▰▱▱▱▱▱▱  24%"
full+="    5h ▰▰▱▱▱▱▱▱  23% ↻2h13"
full+="    week ▰▰▰▱▱▱▱▱  41% ↻3d"
eq "every meter, four spaces apart" "$full" "$(row "$(payload "Opus 5.5 (1M context)" 24 23 8000 41 300000)")"
eq "a model with no context size is kept whole" "Sonnet 5" "$(row "$(payload "Sonnet 5" "" "" "" "" "")")"
eq "no rate_limits, no windows" "Opus    context ▱▱▱▱▱▱▱▱   5%" "$(row "$(payload Opus 5 "" "" "" "")")"
eq "one window alone" "Opus    week ▰▰▰▰▱▱▱▱  50% ↻1d" "$(row "$(payload Opus "" "" "" 50 90000)")"
eq "a fraction rounds" "Opus    context ▰▰▰▰▰▰▰▱  82%" "$(row "$(payload Opus 81.6 "" "" "" "")")"
eq "over 100 fills the bar and no more" "Opus    context ▰▰▰▰▰▰▰▰ 104%" \
    "$(row "$(payload Opus 104 "" "" "" "")")"
eq "an empty payload prints nothing" "" "$(row '{}')"

echo "usage status line: resets"
eq "under an hour reads in minutes" "Opus    5h ▰▰▰▰▰▰▰▱  90% ↻25m" \
    "$(row "$(payload Opus "" 90 1530 "" "")")"
eq "a reset already passed drops the countdown" "Opus    5h ▱▱▱▱▱▱▱▱   0%" \
    "$(row "$(payload Opus "" 0 -10 "" "")")"
eq "a window with no reset time has no countdown" "Opus    5h ▰▰▱▱▱▱▱▱  30%" \
    "$(row "$(payload Opus "" 30 "" "" "")")"

echo "usage status line: colour"
# The colour a context meter at $1 percent is drawn in.
colour_at() {
    case $(raw "$(payload Opus "$1" "" "" "" "")") in
        *$'\033[31m'*) echo red ;;
        *$'\033[33m'*) echo yellow ;;
        *) echo plain ;;
    esac
}
eq "79 is plain" plain "$(colour_at 79)"
eq "80 is yellow" yellow "$(colour_at 80)"
eq "94 is still yellow" yellow "$(colour_at 94)"
eq "95 is red" red "$(colour_at 95)"

echo "usage status line: portability"
# A comma-decimal locale makes bash's printf reject "81.6"; the rounding has to happen before bash sees it.
if locale -a 2>/dev/null | grep -qi '^de_DE\.utf-\?8$'; then
    eq "a comma-decimal locale reads the percentage" "Opus    context ▰▰▰▰▰▰▰▱  82%" \
        "$(printf '%s' "$(payload Opus 81.6 "" "" "" "")" | LC_ALL=de_DE.UTF-8 bash "$SCRIPT" |
            sed $'s/\033\\[[0-9;]*m//g')"
fi
eq "without jq it says so" "status line needs jq" "$(printf '{}' | env PATH=/nonexistent "$BASH" "$SCRIPT")"
# bash 4+ features that macOS's bash 3.2 lacks; a hit here is a broken footer on a Mac.
hits=$(grep -nE 'EPOCHSECONDS|EPOCHREALTIME|declare -A|local -A|mapfile|readarray|\$\{[a-z_]+(\^|,)|;;&|\|&' \
    "$SCRIPT")
eq "no bash 4+ features" "" "$hits"

echo
printf '%d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
