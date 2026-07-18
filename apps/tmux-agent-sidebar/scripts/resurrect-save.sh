#!/usr/bin/env bash
# tmux-resurrect post-save hook. Two jobs, both rewriting the just-written
# state file in place:
#
#   1. Sidebar panes: resurrect saves the pane shell's child command, which
#      for a sidebar pane is empty or the blocked `tmux wait-for` helper, so
#      stamp the real `run` command for the whitelist to relaunch.
#   2. Claude panes: append `--resume <session-id>` so a restore resumes the
#      conversation instead of launching a blank claude. The id was stamped
#      on the pane by the hook (@agent_session_id); state lines key by
#      session/window/pane index, so look the id up by those.
#
# The restore side needs "~claude" in @resurrect-processes to relaunch it.
set -euo pipefail

PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
state_file=${1:?usage: resurrect-save.sh <state-file>}

theme=$(tmux show-option -gqv @agent-sidebar-theme)
theme=${theme:-solarized-light}
cmd="$PLUGIN_DIR/bin/tmux-agent-sidebar run --theme $theme"

sed -i -E "s|\ttmux-agent-sidebar\t:.*$|\ttmux-agent-sidebar\t:$cmd|" "$state_file"

# Live map: session/window/pane index -> Claude session id. Tab-delimited to
# match the state file; built via a variable so no literal tab lives in source.
tab=$(printf '\t')
ids=$(tmux list-panes -a \
    -F "#{session_name}${tab}#{window_index}${tab}#{pane_index}${tab}#{@agent_session_id}" 2>/dev/null) || ids=""
[ -n "$ids" ] || exit 0

# awk (not sed): each claude pane needs a different id, keyed by coordinates.
# FS=tab so empty fields survive; only $11 (the :full_command) is touched.
awk -F'\t' -v OFS='\t' -v ids="$ids" '
    BEGIN {
        n = split(ids, rows, "\n")
        for (i = 1; i <= n; i++) {
            split(rows[i], f, "\t")
            if (f[4] != "") id[f[1], f[2], f[3]] = f[4]
        }
    }
    # A stamped @agent_session_id marks a claude pane (only its hook sets it),
    # so match by id, not command name -- claude may show up as node.
    $1 == "pane" && (($2, $3, $6) in id) {
        cmd = substr($11, 2)                       # drop the leading ":"
        gsub(/ (--resume|-r)[= ][^ ]*/, "", cmd)   # strip any prior resume (idempotent)
        if (cmd == "") cmd = "claude"              # claude was the pane root: no saved command
        $11 = ":" cmd " --resume " id[$2, $3, $6]
    }
    { print }
' "$state_file" >"$state_file.tas" && mv "$state_file.tas" "$state_file"
