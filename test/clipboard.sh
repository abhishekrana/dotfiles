#!/usr/bin/env bash
# Guards the copy path: selecting text must reach both the clipboard and the
# primary selection, from every trigger tmux offers.
#
# Two layers. The contract checks assert every trigger routes through
# copy-command, which is the whole point of the single-path design and the thing
# a plugin or a stray bind-key silently breaks. The delivery checks push real
# bytes through a real tmux server with clip stubbed, so a path that resolves
# correctly but drops the text still fails.
#
# Mouse events cannot be synthesized, but the mouse bindings are only wrappers
# around `send-keys -X`, so driving those covers the same code.
#
# Runs on private sockets (tmux -S), never the live server.
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLIP="$REPO/clip/.local/bin/clip"
CONF="$REPO/tmux/.tmux.conf"
pass=0 fail=0

ok() {
    pass=$((pass + 1))
    printf '  \033[32m✓\033[0m %s\n' "$1"
}
no() {
    fail=$((fail + 1))
    printf '  \033[31m✗\033[0m %s\n' "$1"
    [ $# -gt 1 ] && printf '      %s\n' "$2"
}

# eq <name> <want> <got>
eq() { [ "$2" = "$3" ] && ok "$1" || no "$1" "want [$2] got [$3]"; }

# --- tmux server helpers -----------------------------------------------------
SOCK=""
start_tmux() { # start_tmux <config>
    SOCK=$(mktemp -u /tmp/clipt.XXXX)
    tmux -S "$SOCK" -f "$1" new-session -d -s t -x 80 -y 24 2>/dev/null
}
stop_tmux() {
    [ -n "$SOCK" ] && tmux -S "$SOCK" kill-server 2>/dev/null
    rm -f "$SOCK"
    SOCK=""
}
trap stop_tmux EXIT

# key <table> <key> - the command a key is bound to, "" if unbound
key() {
    tmux -S "$SOCK" list-keys -T "$2" 2>/dev/null |
        sed -n "s|^bind-key *-T $2 *$(printf '%s' "$3" | sed 's/[].[^$*\\/]/\\&/g') *||p" | head -1
}

echo
echo "clipboard: binding contract (real .tmux.conf)"
start_tmux "$CONF"

eq "copy-command is the yank script" '$HOME/.local/bin/tmux-yank.sh' \
    "$(tmux -S "$SOCK" show-options -gv copy-command 2>/dev/null)"

# Every trigger must end in copy-pipe-and-cancel with no command of its own, so
# it uses copy-command. A trigger carrying its own command bypasses the path.
for spec in "copy-mode-vi:MouseDragEnd1Pane" "copy-mode:MouseDragEnd1Pane" \
    "copy-mode-vi:DoubleClick1Pane" "copy-mode-vi:TripleClick1Pane" \
    "copy-mode-vi:Enter" "copy-mode-vi:y" "copy-mode:y"; do
    table=${spec%%:*} k=${spec#*:}
    got=$(key "$SOCK" "$table" "$k")
    case "$got" in
        "") no "$table $k is bound" "unbound - this trigger copies nothing" ;;
        *copy-pipe-and-cancel) ok "$table $k -> copy-command" ;;
        *copy-pipe-and-cancel\ *)
            no "$table $k -> copy-command" "carries its own command, bypassing copy-command: $got"
            ;;
        *) no "$table $k -> copy-command" "unexpected: $got" ;;
    esac
done
stop_tmux

echo
echo "clipboard: delivery through the real path"
# A stub clip records what each selection received, so this needs no display.
STUB=$(mktemp -d)
mkdir -p "$STUB/bin"
cat >"$STUB/bin/clip" <<'EOF'
#!/usr/bin/env bash
if [ "${1:-}" = "--primary" ]; then cat >"$CLIP_TEST_OUT/primary"; else cat >"$CLIP_TEST_OUT/clipboard"; fi
EOF
chmod +x "$STUB/bin/clip"

# tmux-yank.sh calls clip by absolute path, so point HOME at the stub tree and
# mirror the layout it expects.
mkdir -p "$STUB/.local/bin"
cp "$STUB/bin/clip" "$STUB/.local/bin/clip"
cp "$REPO/tmux/.local/bin/tmux-yank.sh" "$STUB/.local/bin/tmux-yank.sh"
: >"$STUB/.local/bin/dotfiles-trace"
chmod +x "$STUB/.local/bin/dotfiles-trace"

want="clipboard-delivery-probe-$$"
conf=$(mktemp /tmp/clipc.XXXX.conf)
printf 'set -g copy-command "%s/.local/bin/tmux-yank.sh"\nset -g mode-keys vi\n' "$STUB" >"$conf"

SOCK=$(mktemp -u /tmp/clipt.XXXX)
CLIP_TEST_OUT="$STUB" tmux -S "$SOCK" -f "$conf" \
    new-session -d -s t -x 80 -y 24 2>/dev/null
tmux -S "$SOCK" set-environment -g CLIP_TEST_OUT "$STUB"
tmux -S "$SOCK" set-environment -g HOME "$STUB"
tmux -S "$SOCK" set-environment -g PATH "$STUB/bin:$PATH"

# Put a known line in the pane, then select and copy it the way a mouse drag or
# a double-click does: begin-selection, extend, copy-pipe-and-cancel.
tmux -S "$SOCK" send-keys -t t "printf '%s\\n' $want" Enter
sleep 0.6
tmux -S "$SOCK" copy-mode -t t
tmux -S "$SOCK" send-keys -X -t t history-top
tmux -S "$SOCK" send-keys -X -t t search-forward "$want"
tmux -S "$SOCK" send-keys -X -t t begin-selection
tmux -S "$SOCK" send-keys -X -t t end-of-line
tmux -S "$SOCK" send-keys -X -t t copy-pipe-and-cancel
sleep 0.8

for sel in clipboard primary; do
    if [ ! -f "$STUB/$sel" ]; then
        no "$sel received the selection" "clip was never invoked for $sel"
    else
        got=$(tr -d '\n' <"$STUB/$sel")
        case "$got" in
            *"$want"*) ok "$sel received the selection" ;;
            *) no "$sel received the selection" "want to contain [$want] got [$got]" ;;
        esac
    fi
done
stop_tmux
rm -rf "$STUB" "$conf"

echo
echo "clip: backends and fidelity"
# A hermetic PATH holding only the tools clip needs. Without this, /usr/bin's
# real wl-copy wins the backend search and the xclip and no-backend branches
# never run.
tmpd=$(mktemp -d)
mkdir -p "$tmpd/bin"
for t in mktemp cat wc cksum cut tr rm bash; do
    ln -sf "$(command -v "$t")" "$tmpd/bin/$t"
done

cat >"$tmpd/bin/xclip" <<EOF
#!/usr/bin/env bash
# record the selection name and the bytes, verbatim
echo "\$2" >"$tmpd/xclip.sel"
cat >"$tmpd/xclip.out"
EOF
chmod +x "$tmpd/bin/xclip"

# Byte fidelity: \$(cat) would strip trailing newlines, so clip must not use it.
printf 'line\n\n' | env -i PATH="$tmpd/bin" HOME="$tmpd" DOTFILES_TRACE=0 \
    /bin/bash "$CLIP" >/dev/null 2>&1
eq "byte-exact through xclip (trailing newlines kept)" \
    "$(printf 'line\n\n' | od -An -c | tr -s ' ')" \
    "$(od -An -c <"$tmpd/xclip.out" 2>/dev/null | tr -s ' ')"
eq "default selection is the clipboard" "clipboard" "$(cat "$tmpd/xclip.sel" 2>/dev/null)"

printf 'x' | env -i PATH="$tmpd/bin" HOME="$tmpd" DOTFILES_TRACE=0 \
    /bin/bash "$CLIP" --primary >/dev/null 2>&1
eq "--primary targets the primary selection" "primary" "$(cat "$tmpd/xclip.sel" 2>/dev/null)"

# No backend at all: only the coreutils remain on PATH.
rm -f "$tmpd/bin/xclip"
out=$(printf 'x' | env -i PATH="$tmpd/bin" HOME="$tmpd" DOTFILES_TRACE=0 /bin/bash "$CLIP" 2>&1)
rc=$?
[ "$rc" -ne 0 ] && ok "no backend exits non-zero" || no "no backend exits non-zero" "rc=$rc"
case "$out" in
    *"no clipboard backend"*) ok "no backend explains itself" ;;
    *) no "no backend explains itself" "got [$out]" ;;
esac
rm -rf "$tmpd"

echo
if [ "$fail" -gt 0 ]; then
    printf '\033[31m%d failed\033[0m, %d passed\n\n' "$fail" "$pass"
    exit 1
fi
printf '\033[32mall %d passed\033[0m\n\n' "$pass"
