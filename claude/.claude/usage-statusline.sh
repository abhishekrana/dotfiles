#!/usr/bin/env bash
#
# Claude Code status line: the model, how full the context window is, and how much of the 5-hour and weekly usage
# limits is spent, with the time until each resets.
#
#     Opus 5.5 1M    context ▰▰▱▱▱▱▱▱  24%    5h ▰▰▱▱▱▱▱▱  23% ↻2h13    week ... ↻3d
#
# Install: save as ~/.claude/usage-statusline.sh and add to ~/.claude/settings.json:
#
#     "statusLine": { "type": "command", "command": "bash ~/.claude/usage-statusline.sh", "refreshInterval": 30 }
#
# Requires jq. Runs on bash 3.2 and later, so the stock bash on macOS works.
#
# Claude Code sends a JSON payload on stdin and shows whatever this prints. The usage limits (rate_limits) are only in
# that payload for Pro and Max subscribers, and only after the first reply in a session; until then only the model and
# context meter show. Payload reference: https://code.claude.com/docs/en/statusline

# No `set -e`: a (( )) test that is false returns 1, which would end the script on an ordinary comparison.
set -uo pipefail

readonly BAR_WIDTH=8
readonly WARN_AT=80     # percent at which a meter turns yellow
readonly CRITICAL_AT=95 # percent at which a meter turns red

# The terminal's own palette, so the line follows whatever colour theme it has.
readonly DIM=$'\033[2m'
readonly YELLOW=$'\033[33m'
readonly RED=$'\033[31m'
readonly RESET=$'\033[0m'

#######################################
# Prints the time from now until an epoch timestamp, as 3d, 2h05 or 38m. Prints nothing once it has passed.
# Arguments:
#   Unix epoch seconds.
#######################################
time_until() {
    local seconds_left=$(($1 - $(date +%s)))

    if ((seconds_left <= 0)); then
        return
    elif ((seconds_left >= 86400)); then
        printf '%dd' $((seconds_left / 86400))
    elif ((seconds_left >= 3600)); then
        printf '%dh%02d' $((seconds_left / 3600)) $((seconds_left % 3600 / 60))
    else
        printf '%dm' $((seconds_left / 60))
    fi
}

#######################################
# Prints one meter: a dim label, a bar, the percentage and, when given, the time until it resets. The bar and the
# number turn yellow at WARN_AT and red at CRITICAL_AT.
# Arguments:
#   Label, e.g. "5h".
#   Percentage used, a whole number; may exceed 100.
#   Reset time in Unix epoch seconds, or "" for none.
#######################################
meter() {
    local label=$1 percent=$2 resets_at=$3
    local color='' bar='' filled i countdown

    if ((percent >= CRITICAL_AT)); then
        color=$RED
    elif ((percent >= WARN_AT)); then
        color=$YELLOW
    fi

    # Cells to fill, rounded to the nearest and capped at a full bar.
    filled=$(((percent * BAR_WIDTH + 50) / 100))
    ((filled > BAR_WIDTH)) && filled=$BAR_WIDTH

    for ((i = 0; i < BAR_WIDTH; i++)); do
        if ((i < filled)); then
            bar+="${color}▰${RESET}"
        else
            bar+="${DIM}▱${RESET}"
        fi
    done

    # %3d pads the number, so the meters after it do not shift as it grows.
    printf '%s%s%s %s %s%3d%%%s' "${DIM}" "${label}" "${RESET}" "${bar}" "${color}" "${percent}" "${RESET}"

    if [[ -n "${resets_at}" ]]; then
        countdown=$(time_until "${resets_at}")
        [[ -n "${countdown}" ]] && printf ' %s↻%s%s' "${DIM}" "${countdown}" "${RESET}"
    fi
}

main() {
    local model context five_hour five_hour_resets week week_resets
    local parts=() line='' part

    if ! command -v jq >/dev/null 2>&1; then
        printf 'status line needs jq'
        return
    fi

    # One jq call pulls every field out of the payload, joined by the ASCII unit separator (\x1f). A tab would not
    # do: read folds runs of whitespace separators into one, so an empty field would shift the ones after it.
    # Percentages are rounded here, not in bash, where printf would reject "23.5" in a comma-decimal locale.
    IFS=$'\x1f' read -r model context five_hour five_hour_resets week week_resets < <(
        jq -r '
            def percent: if . == null then "" else round end;
            [
                .model.display_name // "",
                (.context_window.used_percentage | percent),
                (.rate_limits.five_hour.used_percentage | percent),
                .rate_limits.five_hour.resets_at // "",
                (.rate_limits.seven_day.used_percentage | percent),
                .rate_limits.seven_day.resets_at // ""
            ] | map(tostring) | join("\u001f")'
    )

    # "Opus 5.5 (1M context)" -> "Opus 5.5 1M"
    if [[ "${model}" =~ ^(.*)\ \(([0-9]+[kM])\ context\)$ ]]; then
        model="${BASH_REMATCH[1]} ${BASH_REMATCH[2]}"
    fi

    # A field missing from the payload leaves its meter out.
    [[ -n "${model}" ]] && parts+=("${model}")
    [[ -n "${context}" ]] && parts+=("$(meter context "${context}" "")")
    [[ -n "${five_hour}" ]] && parts+=("$(meter 5h "${five_hour}" "${five_hour_resets}")")
    [[ -n "${week}" ]] && parts+=("$(meter week "${week}" "${week_resets}")")

    # The ${parts[@]+...} form keeps bash 3.2 from calling an empty array unbound under set -u.
    for part in ${parts[@]+"${parts[@]}"}; do
        line+="${line:+    }${part}"
    done
    printf '%s' "${line}"
}

main "$@"
