# Shared by open.sh / follow.sh.

# Run an insert command (split/join of the sidebar), then re-apply pane
# widths left-to-right so the whole cost lands on the leftmost pane.
# tmux takes the inserted width proportionally from all columns but
# returns it to the leftmost only — every move drained the others.
# Locals are prefixed: bash scoping leaks them into the called command.
insert_keeping_widths() {
    local ikw_window=$1 ikw_inserted=$2; shift 2
    local ikw_before ikw_min ikw_left ikw_id ikw_width ikw_shrunk=""
    ikw_before=$(tmux list-panes -t "$ikw_window" -F '#{pane_left} #{pane_id} #{pane_width}' | sort -n)
    "$@" || return 0
    ikw_min=$(head -1 <<<"$ikw_before" | cut -d' ' -f1)
    while read -r ikw_left ikw_id ikw_width; do
        if [ "$ikw_left" = "$ikw_min" ]; then
            [ -n "$ikw_shrunk" ] && continue # same column, one resize is enough
            ikw_shrunk=1
            ikw_width=$((ikw_width - ikw_inserted - 1)) # sidebar + border
        fi
        tmux resize-pane -t "$ikw_id" -x "$ikw_width" 2>/dev/null || true
    done <<<"$ikw_before"
}
