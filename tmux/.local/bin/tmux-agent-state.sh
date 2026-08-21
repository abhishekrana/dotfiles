#!/usr/bin/env bash
# The agent-state language for shell surfaces - sourced, not run.
#
# One definition for the session picker AND its preview, because two copies is
# exactly how they drifted: the preview grew emoji circles while the picker and
# the sidebar drew glyphs. Glyphs and priority mirror the sidebar's `stateIcon`
# (apps/agentbar/internal/ui/render.go) - change them together.
#
# Colors come from the theme switcher's generated file, so the popup follows a
# `theme` switch like every other surface; per design/palette.toml nothing
# downstream hardcodes. The fallbacks below are that palette's solarized-light
# values, for a machine where `theme` has not run yet.

# shellcheck disable=SC2034  # consumers read these
_state_blocked="#dc322f"
_state_asking="#b58900"
_state_working="#2aa198"
_state_done="#859900"
_state_muted="#93a1a1"
_popup_fzf_color="light,fg:#657b83,bg:#fdf6e3,bg+:#268bd2,fg+:#fdf6e3,gutter:-1,pointer:-1,hl:#268bd2"
_popup_fzf_color+=",hl+:#fdf6e3,border:#93a1a1,info:#93a1a1,prompt:#586e75"

_theme_state_file="${XDG_CONFIG_HOME:-$HOME/.config}/theme/agent-state.sh"
# shellcheck source=/dev/null
[ -r "$_theme_state_file" ] && . "$_theme_state_file"

# hex_sgr <#rrggbb> - truecolor foreground escape for a palette hex.
hex_sgr() {
    local hex=${1#\#}
    printf '\033[38;2;%d;%d;%dm' "0x${hex:0:2}" "0x${hex:2:2}" "0x${hex:4:2}"
}

# agent_icon <state> - the one-cell glyph for a state, colored. Single-width on
# purpose: emoji are double-width and shift every column after them.
agent_icon() {
    case $1 in
        permission) printf '%s◔\033[0m' "$(hex_sgr "$_state_blocked")" ;;
        question) printf '%s?\033[0m' "$(hex_sgr "$_state_asking")" ;;
        working) printf '%s⠹\033[0m' "$(hex_sgr "$_state_working")" ;;
        done) printf '%s✓\033[0m' "$(hex_sgr "$_state_done")" ;;
        *) printf ' ' ;;
    esac
}

# agent_muted <text> - muted-grey run, for band dividers and other chrome.
agent_muted() {
    printf '%s%s\033[0m' "$(hex_sgr "$_state_muted")" "$1"
}

# agent_state_rank <state> - worst-state-wins priority for a per-session rollup:
# permission > asking > working > done > none.
agent_state_rank() {
    case $1 in
        permission) printf 4 ;;
        question) printf 3 ;;
        working) printf 2 ;;
        done) printf 1 ;;
        *) printf 0 ;;
    esac
}

# agent_title <pane-title> <host> - Claude's own title for a session, as it
# publishes it in its pane title. Claude marks it with a leading glyph and the
# set is open-ended (✳, ◐, ◑, and it changes between versions), so strip by
# shape: one non-ASCII character then a space. Both the pre-prompt placeholder
# and the hostname tmux seeds the title with mean it has none yet. Mirrors
# agentTitle in apps/agentbar/internal/tmux/snapshot.go - change them together.
agent_title() {
    local t=$1
    if [ "${t:1:1}" = " " ] && [[ ${t:0:1} == [!\ -~] ]]; then
        t=${t:2}
    fi
    case $t in "Claude Code" | "$2") t= ;; esac
    printf '%s' "$t"
}
