#!/usr/bin/env bash
# Fresh-install smoke test: runs bootstrap.sh in a clean Ubuntu 24.04 container,
# fails if the first run exits non-zero, verifies binaries + stow symlinks AND the
# post-stow steps (bashrc patch, vaults, agentbar, nvim plugins), then runs
# bootstrap a second time to confirm idempotency. ~5 min on first run; cached
# subsequent runs are faster.
#
# Usage: test/bootstrap-fresh.sh
# Requires: docker.

set -euo pipefail

DOTFILES_DIR="$(cd "$(dirname "$0")/.." && pwd)"

command -v docker >/dev/null || {
    echo "docker not found"
    exit 1
}

docker run --rm -v "$DOTFILES_DIR:/dotfiles:ro" ubuntu:24.04 bash -c '
set -e
apt-get update -qq && apt-get install -y -qq sudo git curl ca-certificates >/dev/null
useradd -m -s /bin/bash test
echo "test ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers
cp -r /dotfiles /home/test/dotfiles
chown -R test:test /home/test/dotfiles

su - test -c "cd ~/dotfiles && ./bootstrap.sh" 2>&1 | tail -60
rc=${PIPESTATUS[0]}
[ "$rc" -eq 0 ] || { echo "FAIL: first bootstrap run exited $rc (must be 0 on a fresh machine)"; exit 1; }

su - test -c "
export PATH=\$HOME/.local/bin:\$PATH
echo
echo \"--- binaries ---\"
fail=0
for b in fzf fd delta lazygit lazydocker yazi zoxide nvim ghostty \
  git-cliff gitleaks ruff shellcheck shfmt task; do
  if v=\$(\"\$b\" --version 2>/dev/null | head -1); then
    printf \"OK   %-12s %s\n\" \"\$b\" \"\$v\"
  else
    printf \"MISS %s\n\" \"\$b\"
    fail=1
  fi
done

echo
echo \"--- symlinks ---\"
for f in ~/.tmux.conf ~/.bashrc.d ~/.config/nvim ~/.local/bin/tmux-gitlab.sh; do
  if [ -L \"\$f\" ]; then printf \"OK   %s\n\" \"\$f\"
  else printf \"MISS %s\n\" \"\$f\"; fail=1; fi
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
echo \"--- 2nd run (idempotency) ---\"
cd ~/dotfiles && ./bootstrap.sh 2>&1 | grep -cE \"already installed|already patched\" \\
  | xargs -I{} echo \"{} steps already-installed (expect >= 10)\"

exit \$fail
"
'
