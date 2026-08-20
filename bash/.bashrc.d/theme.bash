# Terminal theme flavor, written by the `theme` switcher (~/.local/bin/theme).
# Loads after fzf.bash (files sourced in name order), so BAT_THEME here overrides its default.
# shellcheck source=/dev/null  # written by the `theme` switcher, absent until first run
[ -f ~/.config/theme/env.sh ] && . ~/.config/theme/env.sh

# hunk has no theme env/config-file override, only a `--theme` flag that must follow the
# subcommand. Wrap `hunk diff …` (the review path) to carry the current flavor; hunk falls
# back gracefully on a flavor it doesn't ship (e.g. solarized-dark). Other subcommands and
# path-invoked hunk (bootstrap) are untouched.
# Read the flavor per call, not from $THEME: a shell captures that once at
# startup, so every shell older than the last switch launched hunk on the old
# theme - and looked like the switch had not worked.
hunk() {
    local flavor
    flavor=$(cat ~/.config/theme/current 2>/dev/null)
    if [ -n "$flavor" ] && [ "${1:-}" = diff ]; then
        command hunk "$@" --theme "$flavor"
    else
        command hunk "$@"
    fi
}
