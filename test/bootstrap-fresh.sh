#!/usr/bin/env bash
# End-to-end fresh-machine test: runs bootstrap.sh in a clean Ubuntu 24.04
# container, then checks every pinned binary, every stow package, the post-stow
# steps (bashrc patch, vaults, agentbar, nvim plugins), a login shell with
# ~/.local/bin on PATH, that dictate runs, and that the whisper step was reached -
# then re-runs bootstrap for idempotency. The container pre-installs only sudo and
# ca-certificates; anything bootstrap should install is left missing on purpose.
# ~5 min on first run.
#
# Usage: test/bootstrap-fresh.sh [--gpu]
#   --gpu  pass the host render node in and require the Vulkan whisper.cpp build.
#          CI runners have no GPU, so the default run requires the skip instead.
# Requires: docker.

set -euo pipefail

DOTFILES_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DEVICE=()
WANT_GPU=
RENDER_GID=

case "${1:-}" in
    --gpu)
        [ -e /dev/dri/renderD128 ] || {
            echo "--gpu needs a host render node at /dev/dri/renderD128" >&2
            exit 2
        }
        DEVICE=(--device /dev/dri)
        WANT_GPU=1
        RENDER_GID=$(stat -c %g /dev/dri/renderD128)
        ;;
    -h | --help)
        sed -n '/^# Usage:/,/^# Requires:/p' "$0"
        exit 0
        ;;
    "") ;;
    *)
        echo "unknown option: $1" >&2
        exit 2
        ;;
esac

command -v docker >/dev/null || {
    echo "docker not found"
    exit 1
}

docker run --rm "${DEVICE[@]}" -e WANT_GPU="$WANT_GPU" -e RENDER_GID="$RENDER_GID" \
    -v "$DOTFILES_DIR:/dotfiles:ro" ubuntu:24.04 bash -c '
set -e
# Only what a real Ubuntu ships and this base image does not. Anything bootstrap
# is supposed to install - git, curl, the rest - is deliberately absent so the
# run proves it installs them itself.
apt-get update -qq && apt-get install -y -qq sudo ca-certificates >/dev/null
useradd -m -s /bin/bash test
# The render node is mode 660 root:render and `su -` drops supplementary groups,
# so the group must exist here and list the user.
if [ -n "${RENDER_GID:-}" ]; then
  getent group "$RENDER_GID" >/dev/null || groupadd -g "$RENDER_GID" hostrender
  usermod -aG "$(getent group "$RENDER_GID" | cut -d: -f1)" test
fi
echo "test ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers
cp -r /dotfiles /home/test/dotfiles
chown -R test:test /home/test/dotfiles

# Built out here so the nested su -c below needs no extra quote escaping.
sed -n "s/^ *local packages=(\(.*\))$/\1/p" /home/test/dotfiles/bootstrap.sh > /tmp/pkgs
cat > /tmp/probe-shell.sh <<"EOS"
set -e
for f in "$HOME"/.bashrc.d/*.bash; do . "$f"; done
command -v uv >/dev/null
EOS
grep -E "^(TMUX_VERSION|GO_VERSION)=" /home/test/dotfiles/install.sh | sed "s/\"//g" > /tmp/pins
chmod 644 /tmp/pkgs /tmp/pins; chmod 755 /tmp/probe-shell.sh

rc=0
su - test -c "cd ~/dotfiles && ./bootstrap.sh" >/tmp/bootstrap.log 2>&1 || rc=$?
tail -60 /tmp/bootstrap.log
[ "$rc" -eq 0 ] || { echo "FAIL: first bootstrap run exited $rc (must be 0 on a fresh machine)"; exit 1; }

# Assert the whisper step ran. Checking only that whisper-server is absent would
# also pass if all_tools never called it.
if [ -n "$WANT_GPU" ]; then
  grep -q "whisper-server installed" /tmp/bootstrap.log \
    || { echo "FAIL: --gpu run did not build whisper.cpp"; exit 1; }
  echo "OK   whisper.cpp built against Vulkan"
else
  grep -q "no DRM render node" /tmp/bootstrap.log \
    || { echo "FAIL: the whisper step never ran (dropped from all_tools?)"; exit 1; }
  echo "OK   whisper step ran and skipped (no GPU in the container)"
fi

su - test -c "
export PATH=\$HOME/.local/bin:\$PATH
echo
echo \"--- binaries ---\"
fail=0
for b in fzf fd delta lazygit lazydocker yazi zoxide nvim ghostty \
  git-cliff gitleaks ruff shellcheck shfmt task uv parec pactl git stow node; do
  if v=\$(\"\$b\" --version 2>/dev/null | head -1); then
    printf \"OK   %-12s %s\n\" \"\$b\" \"\$v\"
  else
    printf \"MISS %s\n\" \"\$b\"
    fail=1
  fi
done

echo
echo \"--- pinned builds (Ubuntu ships older tmux; the pin is load-bearing) ---\"
. /tmp/pins
if tmux -V | grep -q \"\$TMUX_VERSION\"; then printf \"OK   %-12s %s\n\" tmux \"\$(tmux -V)\"
else printf \"MISS %s\n\" \"tmux is not the pinned \$TMUX_VERSION (got: \$(tmux -V))\"; fail=1; fi
if go version | grep -q \"go\$GO_VERSION\"; then printf \"OK   %-12s %s\n\" go \"\$(go version)\"
else printf \"MISS %s\n\" \"go is not the pinned \$GO_VERSION\"; fail=1; fi

echo
echo \"--- stow packages (every package bootstrap.sh stows) ---\"
for pkg in \$(cat /tmp/pkgs); do
  rel=\$(cd ~/dotfiles/\$pkg && git ls-files \
    | grep -vE \"^(CLAUDE|README)[.]md\$|^[.]stow-local-ignore\$\" | sort | head -1)
  t=\$HOME/\$rel
  if [ -n \"\$rel\" ] && [ -e \"\$t\" ] \
     && [ \"\$(readlink -f \"\$t\")\" = \"\$(readlink -f ~/dotfiles/\$pkg/\$rel)\" ]; then
    printf \"OK   %-8s %s\n\" \"\$pkg\" \"\$rel\"
  else
    printf \"MISS %-8s %s\n\" \"\$pkg\" \"\$rel\"; fail=1
  fi
done

echo
echo \"--- post-stow steps (skipped entirely if bootstrap aborted early) ---\"
grep -q \"Load dotfiles shell customizations\" ~/.bashrc \\
  && printf \"OK   %s\n\" \".bashrc patched\" \\
  || { printf \"MISS %s\n\" \".bashrc patch\"; fail=1; }
for d in ~/vaults/personal ~/vaults/work ~/.local/share/nvim/lazy; do
  if [ -d \"\$d\" ]; then printf \"OK   %s\n\" \"\$d\"; else printf \"MISS %s\n\" \"\$d\"; fail=1; fi
done
if [ -x ~/dotfiles/apps/agentbar/bin/agentbar ]; then printf \"OK   %s\n\" \"agentbar built\"
  else printf \"MISS %s\n\" \"agentbar\"; fail=1; fi

echo
echo \"--- login shell ---\"
if bash /tmp/probe-shell.sh >/dev/null 2>&1; then
  printf \"OK   %s\n\" \".bashrc.d loads clean and puts ~/.local/bin on PATH\"
else
  printf \"MISS %s\n\" \".bashrc.d errored, or ~/.local/bin is not on PATH\"; fail=1
fi

echo
echo \"--- dictate end to end ---\"
if timeout 420 dictate --help >/dev/null 2>&1; then
  printf \"OK   %s\n\" \"dictate runs - uv resolved its PEP 723 deps\"
else
  printf \"MISS %s\n\" \"dictate did not run\"; fail=1
fi

echo
echo \"--- 2nd run (idempotency) ---\"
cd ~/dotfiles && ./bootstrap.sh 2>&1 | grep -cE \"already installed|already patched\" \\
  | xargs -I{} echo \"{} steps already-installed (expect >= 10)\"

exit \$fail
"
'
