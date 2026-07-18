# Terminal theme flavor, written by the `theme` switcher (~/.local/bin/theme).
# Loads after fzf.bash (files sourced in name order), so BAT_THEME here overrides its default.
[ -f ~/.config/theme/env.sh ] && . ~/.config/theme/env.sh

# hunk has no theme env/config-file override, only a `--theme` flag that must follow the
# subcommand. Wrap `hunk diff …` (the review path) to carry the current flavor; hunk falls
# back gracefully on a flavor it doesn't ship (e.g. solarized-dark). Other subcommands and
# path-invoked hunk (bootstrap) are untouched.
hunk() {
	if [ -n "${THEME:-}" ] && [ "${1:-}" = diff ]; then
		command hunk "$@" --theme "$THEME"
	else
		command hunk "$@"
	fi
}
