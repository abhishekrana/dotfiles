#!/usr/bin/env bash
set -euo pipefail

DOTFILES_DIR="$(cd "$(dirname "$0")" && pwd)"
LOCAL_BIN="$HOME/.local/bin"

# Pinned versions (update these to upgrade)
DELTA_VERSION="0.19.2"
FD_VERSION="10.4.2"
FZF_VERSION="0.74.1"
GIT_CLIFF_VERSION="2.13.1"
GITLEAKS_VERSION="8.30.1"
GITMUX_VERSION="0.11.5"
GO_VERSION="1.26.5"
HUNK_VERSION="0.17.6"
LAZYDOCKER_VERSION="0.25.2"
LAZYGIT_VERSION="0.63.1"
NEOVIM_VERSION="0.12.4"
NERD_FONT_VERSION="3.4.0"
RUFF_VERSION="0.16.0"
SHELLCHECK_VERSION="0.11.0"
SHFMT_VERSION="3.13.1"
TASK_VERSION="3.52.0"
YAZI_VERSION="26.5.6"
ZOXIDE_VERSION="0.10.0"

log() { echo -e "\033[1;34m[dotfiles]\033[0m $*"; }
warn() { echo -e "\033[1;33m[dotfiles]\033[0m $*"; }
ok() { echo -e "\033[1;32m[dotfiles]\033[0m $*"; }

# =============================================================================
# APT packages
# =============================================================================

install_apt_packages() {
    # chafa: image previews for yazi
    # imagemagick: convert/identify, used by image.nvim to render images in nvim
    local pkgs=(bat build-essential chafa curl direnv fontconfig imagemagick jq ripgrep software-properties-common stow tmux tree unzip wget wl-clipboard)
    local to_install=()
    for pkg in "${pkgs[@]}"; do
        dpkg -s "$pkg" &>/dev/null || to_install+=("$pkg")
    done
    if [ ${#to_install[@]} -gt 0 ]; then
        log "Installing apt packages: ${to_install[*]}"
        sudo apt-get update -qq
        sudo apt-get install -y -qq "${to_install[@]}"
    else
        ok "APT packages already installed"
    fi

    # bat is installed as 'batcat' on Ubuntu, symlink it
    if command -v batcat &>/dev/null && [ ! -e "$LOCAL_BIN/bat" ]; then
        ln -s "$(command -v batcat)" "$LOCAL_BIN/bat"
    fi
}

# =============================================================================
# Node.js (via NodeSource)
# =============================================================================

install_nodejs() {
    if command -v node &>/dev/null && node --version | grep -q '^v24\.'; then
        ok "Node.js 24.x already installed"
        return
    fi
    log "Installing Node.js 24.x via NodeSource..."
    # Remove Ubuntu's outdated nodejs/npm if present
    sudo apt-get remove -y -qq nodejs npm 2>/dev/null || true
    curl -fsSL https://deb.nodesource.com/setup_24.x | sudo -E bash -
    sudo apt-get install -y -qq nodejs
    ok "Node.js $(node --version) / npm $(npm --version) installed"
}

# =============================================================================
# Binary tools -> ~/.local/bin
# =============================================================================

install_delta() {
    if [ -x "$LOCAL_BIN/delta" ] && "$LOCAL_BIN/delta" --version 2>/dev/null | grep -q "$DELTA_VERSION"; then
        ok "delta $DELTA_VERSION already installed"
        return
    fi
    log "Installing delta $DELTA_VERSION..."
    local url="https://github.com/dandavison/delta/releases/download/${DELTA_VERSION}/delta-${DELTA_VERSION}-x86_64-unknown-linux-musl.tar.gz"
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp/delta-${DELTA_VERSION}-x86_64-unknown-linux-musl/delta" "$LOCAL_BIN/delta"
    chmod +x "$LOCAL_BIN/delta"
    rm -rf "$tmp"
    ok "delta $DELTA_VERSION installed"
}

install_fd() {
    if [ -x "$LOCAL_BIN/fd" ] && "$LOCAL_BIN/fd" --version 2>/dev/null | grep -q "$FD_VERSION"; then
        ok "fd $FD_VERSION already installed"
        return
    fi
    log "Installing fd $FD_VERSION..."
    local url="https://github.com/sharkdp/fd/releases/download/v${FD_VERSION}/fd-v${FD_VERSION}-x86_64-unknown-linux-musl.tar.gz"
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp/fd-v${FD_VERSION}-x86_64-unknown-linux-musl/fd" "$LOCAL_BIN/fd"
    chmod +x "$LOCAL_BIN/fd"
    rm -rf "$tmp"
    ok "fd $FD_VERSION installed"
}

install_fzf() {
    if [ -x "$LOCAL_BIN/fzf" ] && "$LOCAL_BIN/fzf" --version 2>/dev/null | grep -q "$FZF_VERSION"; then
        ok "fzf $FZF_VERSION already installed"
        return
    fi
    log "Installing fzf $FZF_VERSION..."
    local url="https://github.com/junegunn/fzf/releases/download/v${FZF_VERSION}/fzf-${FZF_VERSION}-linux_amd64.tar.gz"
    curl -sSL "$url" | tar xz -C "$LOCAL_BIN" fzf
    chmod +x "$LOCAL_BIN/fzf"
    ok "fzf $FZF_VERSION installed"
}

# Generates CHANGELOG.md and release notes from conventional commits.
install_git_cliff() {
    if [ -x "$LOCAL_BIN/git-cliff" ] && "$LOCAL_BIN/git-cliff" --version 2>/dev/null | grep -q "$GIT_CLIFF_VERSION"; then
        ok "git-cliff $GIT_CLIFF_VERSION already installed"
        return
    fi
    log "Installing git-cliff $GIT_CLIFF_VERSION..."
    local url="https://github.com/orhun/git-cliff/releases/download/v${GIT_CLIFF_VERSION}/git-cliff-${GIT_CLIFF_VERSION}-x86_64-unknown-linux-musl.tar.gz"
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp/git-cliff-${GIT_CLIFF_VERSION}/git-cliff" "$LOCAL_BIN/git-cliff"
    chmod +x "$LOCAL_BIN/git-cliff"
    rm -rf "$tmp"
    ok "git-cliff $GIT_CLIFF_VERSION installed"
}

# Scans the tree and full history for credentials; this repo is public.
install_gitleaks() {
    if [ -x "$LOCAL_BIN/gitleaks" ] && "$LOCAL_BIN/gitleaks" version 2>/dev/null | grep -q "$GITLEAKS_VERSION"; then
        ok "gitleaks $GITLEAKS_VERSION already installed"
        return
    fi
    log "Installing gitleaks $GITLEAKS_VERSION..."
    local url="https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS_VERSION}/gitleaks_${GITLEAKS_VERSION}_linux_x64.tar.gz"
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp/gitleaks" "$LOCAL_BIN/gitleaks"
    chmod +x "$LOCAL_BIN/gitleaks"
    rm -rf "$tmp"
    ok "gitleaks $GITLEAKS_VERSION installed"
}

install_gitmux() {
    if [ -x "$LOCAL_BIN/gitmux" ] && [ -f "$LOCAL_BIN/.gitmux-version" ] && grep -q "$GITMUX_VERSION" "$LOCAL_BIN/.gitmux-version"; then
        ok "gitmux $GITMUX_VERSION already installed"
        return
    fi
    log "Installing gitmux $GITMUX_VERSION..."
    local url="https://github.com/arl/gitmux/releases/download/v${GITMUX_VERSION}/gitmux_v${GITMUX_VERSION}_linux_amd64.tar.gz"
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp/gitmux" "$LOCAL_BIN/gitmux"
    chmod +x "$LOCAL_BIN/gitmux"
    echo "$GITMUX_VERSION" >"$LOCAL_BIN/.gitmux-version"
    rm -rf "$tmp"
    ok "gitmux $GITMUX_VERSION installed"
}

# Toolchain for building apps/ (currently the Go agentbar).
install_go() {
    if [ -x "$LOCAL_BIN/go" ] && "$LOCAL_BIN/go" version 2>/dev/null | grep -q "go${GO_VERSION} "; then
        ok "Go $GO_VERSION already installed"
        return
    fi
    log "Installing Go $GO_VERSION..."
    rm -rf "$HOME/.local/go"
    local url="https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp/go" "$HOME/.local/go"
    ln -sf "$HOME/.local/go/bin/go" "$LOCAL_BIN/go"
    ln -sf "$HOME/.local/go/bin/gofmt" "$LOCAL_BIN/gofmt"
    rm -rf "$tmp"
    ok "Go $GO_VERSION installed"
}

install_hunk() {
    if [ -x "$LOCAL_BIN/hunk" ] && "$LOCAL_BIN/hunk" --version 2>/dev/null | grep -q "$HUNK_VERSION"; then
        ok "hunk $HUNK_VERSION already installed"
    else
        log "Installing hunk $HUNK_VERSION..."
        npm install -g --prefix "$HOME/.local" "hunkdiff@$HUNK_VERSION"
        ok "hunk $HUNK_VERSION installed"
    fi

    # Expose hunk's bundled Claude Code review skill as a user skill, so agents
    # can drive a live hunk review session. Symlinked (not copied) so it tracks
    # the installed hunk version. Runs on every bootstrap, install or not.
    local skill_md
    skill_md="$("$LOCAL_BIN/hunk" skill path 2>/dev/null)" || return 0
    if [ -f "$skill_md" ]; then
        mkdir -p "$HOME/.claude/skills"
        ln -sfn "$(dirname "$skill_md")" "$HOME/.claude/skills/hunk-review"
        ok "hunk-review Claude skill linked"
    fi
}

install_lazydocker() {
    if [ -x "$LOCAL_BIN/lazydocker" ] && "$LOCAL_BIN/lazydocker" --version 2>/dev/null | grep -q "$LAZYDOCKER_VERSION"; then
        ok "lazydocker $LAZYDOCKER_VERSION already installed"
        return
    fi
    log "Installing lazydocker $LAZYDOCKER_VERSION..."
    local url="https://github.com/jesseduffield/lazydocker/releases/download/v${LAZYDOCKER_VERSION}/lazydocker_${LAZYDOCKER_VERSION}_Linux_x86_64.tar.gz"
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp/lazydocker" "$LOCAL_BIN/lazydocker"
    chmod +x "$LOCAL_BIN/lazydocker"
    rm -rf "$tmp"
    ok "lazydocker $LAZYDOCKER_VERSION installed"
}

install_lazygit() {
    if [ -x "$LOCAL_BIN/lazygit" ] && "$LOCAL_BIN/lazygit" --version 2>/dev/null | grep -q "$LAZYGIT_VERSION"; then
        ok "lazygit $LAZYGIT_VERSION already installed"
        return
    fi
    log "Installing lazygit $LAZYGIT_VERSION..."
    local url="https://github.com/jesseduffield/lazygit/releases/download/v${LAZYGIT_VERSION}/lazygit_${LAZYGIT_VERSION}_Linux_x86_64.tar.gz"
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp/lazygit" "$LOCAL_BIN/lazygit"
    chmod +x "$LOCAL_BIN/lazygit"
    rm -rf "$tmp"
    ok "lazygit $LAZYGIT_VERSION installed"
}

install_neovim() {
    if [ -x "$LOCAL_BIN/nvim" ] && "$LOCAL_BIN/nvim" --version 2>/dev/null | grep -q "v${NEOVIM_VERSION}"; then
        ok "neovim $NEOVIM_VERSION already installed"
        return
    fi
    log "Installing neovim $NEOVIM_VERSION..."
    rm -rf "$HOME/.local/nvim"
    local url="https://github.com/neovim/neovim/releases/download/v${NEOVIM_VERSION}/nvim-linux-x86_64.tar.gz"
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp"/nvim-linux-x86_64 "$HOME/.local/nvim"
    ln -sf "$HOME/.local/nvim/bin/nvim" "$LOCAL_BIN/nvim"
    rm -rf "$tmp"
    ok "neovim $NEOVIM_VERSION installed"
}

# Lints the one Python script (dictate) for real bugs, not style.
install_ruff() {
    if [ -x "$LOCAL_BIN/ruff" ] && "$LOCAL_BIN/ruff" --version 2>/dev/null | grep -q "$RUFF_VERSION"; then
        ok "ruff $RUFF_VERSION already installed"
        return
    fi
    log "Installing ruff $RUFF_VERSION..."
    local url="https://github.com/astral-sh/ruff/releases/download/${RUFF_VERSION}/ruff-x86_64-unknown-linux-gnu.tar.gz"
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp/ruff-x86_64-unknown-linux-gnu/ruff" "$LOCAL_BIN/ruff"
    chmod +x "$LOCAL_BIN/ruff"
    rm -rf "$tmp"
    ok "ruff $RUFF_VERSION installed"
}

# Pinned: apt lags at 0.9.0. NB a comment starting with this tool's name is
# read as a directive and fails to parse - never start one that way.
install_shellcheck() {
    if [ -x "$LOCAL_BIN/shellcheck" ] && "$LOCAL_BIN/shellcheck" --version 2>/dev/null | grep -q "$SHELLCHECK_VERSION"; then
        ok "shellcheck $SHELLCHECK_VERSION already installed"
        return
    fi
    log "Installing shellcheck $SHELLCHECK_VERSION..."
    local url="https://github.com/koalaman/shellcheck/releases/download/v${SHELLCHECK_VERSION}/shellcheck-v${SHELLCHECK_VERSION}.linux.x86_64.tar.xz"
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xJ -C "$tmp"
    mv "$tmp/shellcheck-v${SHELLCHECK_VERSION}/shellcheck" "$LOCAL_BIN/shellcheck"
    chmod +x "$LOCAL_BIN/shellcheck"
    rm -rf "$tmp"
    ok "shellcheck $SHELLCHECK_VERSION installed"
}

# Finds shell files by shebang for the lint gate; not used to reformat.
install_shfmt() {
    if [ -x "$LOCAL_BIN/shfmt" ] && "$LOCAL_BIN/shfmt" --version 2>/dev/null | grep -q "$SHFMT_VERSION"; then
        ok "shfmt $SHFMT_VERSION already installed"
        return
    fi
    log "Installing shfmt $SHFMT_VERSION..."
    curl -sSL -o "$LOCAL_BIN/shfmt" \
        "https://github.com/mvdan/sh/releases/download/v${SHFMT_VERSION}/shfmt_v${SHFMT_VERSION}_linux_amd64"
    chmod +x "$LOCAL_BIN/shfmt"
    ok "shfmt $SHFMT_VERSION installed"
}

# Runs this repo's Taskfile: `task check`, `task stow`, `task changelog`.
install_task() {
    if [ -x "$LOCAL_BIN/task" ] && "$LOCAL_BIN/task" --version 2>/dev/null | grep -q "$TASK_VERSION"; then
        ok "task $TASK_VERSION already installed"
        return
    fi
    log "Installing task $TASK_VERSION..."
    local url="https://github.com/go-task/task/releases/download/v${TASK_VERSION}/task_linux_amd64.tar.gz"
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp/task" "$LOCAL_BIN/task"
    chmod +x "$LOCAL_BIN/task"
    rm -rf "$tmp"
    ok "task $TASK_VERSION installed"
}

install_yazi() {
    if [ -x "$LOCAL_BIN/yazi" ] && "$LOCAL_BIN/yazi" --version 2>/dev/null | grep -q "$YAZI_VERSION"; then
        ok "yazi $YAZI_VERSION already installed"
        return
    fi
    log "Installing yazi $YAZI_VERSION..."
    local url="https://github.com/sxyazi/yazi/releases/download/v${YAZI_VERSION}/yazi-x86_64-unknown-linux-gnu.zip"
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" -o "$tmp/yazi.zip"
    unzip -o "$tmp/yazi.zip" -d "$tmp" >/dev/null
    mv "$tmp/yazi-x86_64-unknown-linux-gnu/yazi" "$LOCAL_BIN/yazi"
    mv "$tmp/yazi-x86_64-unknown-linux-gnu/ya" "$LOCAL_BIN/ya"
    chmod +x "$LOCAL_BIN/yazi" "$LOCAL_BIN/ya"
    rm -rf "$tmp"
    ok "yazi $YAZI_VERSION installed"
}

install_zoxide() {
    if [ -x "$LOCAL_BIN/zoxide" ] && "$LOCAL_BIN/zoxide" --version 2>/dev/null | grep -q "$ZOXIDE_VERSION"; then
        ok "zoxide $ZOXIDE_VERSION already installed"
        return
    fi
    log "Installing zoxide $ZOXIDE_VERSION..."
    local url="https://github.com/ajeetdsouza/zoxide/releases/download/v${ZOXIDE_VERSION}/zoxide-${ZOXIDE_VERSION}-x86_64-unknown-linux-musl.tar.gz"
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp/zoxide" "$LOCAL_BIN/zoxide"
    chmod +x "$LOCAL_BIN/zoxide"
    rm -rf "$tmp"
    ok "zoxide $ZOXIDE_VERSION installed"
}

# =============================================================================
# JetBrainsMono Nerd Font
# =============================================================================

install_nerd_font() {
    local font_dir="$HOME/.local/share/fonts"
    if [ -f "$font_dir/.nerd-font-version" ] && grep -q "$NERD_FONT_VERSION" "$font_dir/.nerd-font-version"; then
        ok "JetBrainsMono Nerd Font $NERD_FONT_VERSION already installed"
        return
    fi
    log "Installing JetBrainsMono Nerd Font $NERD_FONT_VERSION..."
    mkdir -p "$font_dir"
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "https://github.com/ryanoasis/nerd-fonts/releases/download/v${NERD_FONT_VERSION}/JetBrainsMono.tar.xz" -o "$tmp/JetBrainsMono.tar.xz"
    tar xf "$tmp/JetBrainsMono.tar.xz" -C "$font_dir"
    fc-cache -fv "$font_dir" >/dev/null 2>&1
    echo "$NERD_FONT_VERSION" >"$font_dir/.nerd-font-version"
    rm -rf "$tmp"
    ok "JetBrainsMono Nerd Font $NERD_FONT_VERSION installed"
}

# =============================================================================
# TPM (Tmux Plugin Manager)
# =============================================================================

install_tpm() {
    local tpm_dir="$HOME/.tmux/plugins/tpm"
    if [ -d "$tpm_dir" ]; then
        ok "TPM already installed"
        return
    fi
    log "Installing TPM..."
    git clone https://github.com/tmux-plugins/tpm "$tpm_dir"
    ok "TPM installed - run 'prefix + I' in tmux to install plugins"
}

# =============================================================================
# Ghostty
# =============================================================================

install_ghostty() {
    if command -v ghostty &>/dev/null; then
        ok "Ghostty already installed"
        return
    fi
    log "Installing Ghostty..."
    if ! grep -rq mkasberg/ghostty-ubuntu /etc/apt/sources.list.d/ 2>/dev/null; then
        sudo add-apt-repository -y ppa:mkasberg/ghostty-ubuntu
        sudo apt-get update -qq
    fi
    sudo apt-get install -y -qq ghostty
    ok "Ghostty installed"
}

# =============================================================================
# Stow packages
# =============================================================================

backup_if_not_symlink() {
    # $1: target path. $2 (optional): the repo source path.
    # Behavior:
    #   * target doesn't exist or is a symlink → nothing to do.
    #   * target resolves (via realpath, catching parent-dir symlinks)
    #     to the same inode as src → nothing to do; stow has already
    #     folded the parent directory.
    #   * src given AND content matches → silently remove target so stow
    #     can replace it with a symlink (no noisy .pre-dotfiles).
    #   * otherwise → move target aside to ${target}.pre-dotfiles.
    local target="$1" src="${2:-}"
    [ -e "$target" ] && [ ! -L "$target" ] || return 0
    if [ -n "$src" ] && [ "$(readlink -f -- "$target")" = "$(readlink -f -- "$src")" ]; then
        return 0
    fi
    if [ -n "$src" ] && [ -f "$target" ] && [ -f "$src" ] && cmp -s "$target" "$src"; then
        rm "$target"
    else
        warn "Backing up $target -> ${target}.pre-dotfiles"
        mv "$target" "${target}.pre-dotfiles"
    fi
}

# Per-file backup for directories shared with the user's own files.
# Only backs up files we're about to stow in - user's other files stay put.
backup_pkg_files() {
    local pkg_src="$1" target_dir="$2"
    [ -d "$pkg_src" ] || return
    local f
    for f in "$pkg_src"/*; do
        [ -e "$f" ] || continue
        backup_if_not_symlink "$target_dir/$(basename "$f")" "$f"
    done
}

stow_packages() {
    local packages=(bash bat claude dictate ghostty git hunk nvim theme tmux trace yazi)

    # Single files we own outright: back up the file itself.
    backup_if_not_symlink "$HOME/.tmux.conf" "$DOTFILES_DIR/tmux/.tmux.conf"
    backup_if_not_symlink "$HOME/.gitmux.conf" "$DOTFILES_DIR/tmux/.gitmux.conf"
    backup_if_not_symlink "$HOME/.local/bin/tmux-gitlab.sh" "$DOTFILES_DIR/tmux/.local/bin/tmux-gitlab.sh"
    backup_if_not_symlink "$HOME/.claude/settings.json" "$DOTFILES_DIR/claude/.claude/settings.json"
    backup_if_not_symlink "$HOME/.claude/statusline-command.sh" "$DOTFILES_DIR/claude/.claude/statusline-command.sh"

    # Directories the user may share with their own files: back up only the
    # specific files we ship, leaving the rest of the directory intact.
    backup_pkg_files "$DOTFILES_DIR/bash/.bashrc.d" "$HOME/.bashrc.d"

    # Directories we own outright: back up the whole directory.
    backup_if_not_symlink "$HOME/.config/nvim"
    backup_if_not_symlink "$HOME/.config/bat"
    backup_if_not_symlink "$HOME/.config/ghostty"
    backup_if_not_symlink "$HOME/.config/hunk"
    backup_if_not_symlink "$HOME/.config/yazi"

    cd "$DOTFILES_DIR"
    for pkg in "${packages[@]}"; do
        log "Stowing $pkg..."
        stow --restow "$pkg"
    done
    ok "All packages stowed"

    # Install yazi plugins from package.toml
    if command -v ya &>/dev/null; then
        log "Installing yazi plugins..."
        ya pkg install
        ok "Yazi plugins installed"
    fi
}

enable_tmux_resurrect_timer() {
    # tmux-continuum only autosaves while a client is attached (its save hook
    # rides the status-bar redraw), so a long detached stretch before a reboot
    # loses everything since the last attached save. This user timer saves on a
    # schedule regardless of attachment. Units ship with the tmux stow package.
    if ! command -v systemctl &>/dev/null; then
        return
    fi
    log "Enabling tmux-resurrect autosave timer (systemd --user)..."
    systemctl --user daemon-reload 2>/dev/null || true
    if systemctl --user enable --now tmux-resurrect-save.timer 2>/dev/null; then
        ok "tmux-resurrect-save.timer enabled"
    else
        warn "Could not enable tmux-resurrect-save.timer (no user systemd session?)"
    fi
    # Keep the timer running even with no active login session.
    loginctl enable-linger "$USER" 2>/dev/null || true
}

# =============================================================================
# Patch ~/.bashrc
# =============================================================================

# The ~ in the messages below is for the reader, not a path to expand.
# shellcheck disable=SC2088
patch_bashrc() {
    local marker="# Load dotfiles shell customizations"
    if grep -qF "$marker" "$HOME/.bashrc"; then
        ok "~/.bashrc already patched"
        return
    fi
    log "Patching ~/.bashrc..."
    cp "$HOME/.bashrc" "$HOME/.bashrc.pre-dotfiles"
    cat >>"$HOME/.bashrc" <<'EOF'

# Load dotfiles shell customizations
for f in ~/.bashrc.d/*.bash; do [ -r "$f" ] && source "$f"; done
EOF
    ok "~/.bashrc patched (backup at ~/.bashrc.pre-dotfiles)"
}

copy_if_absent() {
    # copy_if_absent SRC DEST - seed a file only when it doesn't exist yet, so a re-run
    # never clobbers edits the user (or an agent) has made in the live vault.
    local src="$1" dest="$2"
    if [ -e "$dest" ]; then return; fi
    mkdir -p "$(dirname "$dest")"
    cp "$src" "$dest"
}

seed_vault() {
    # Seed one notes vault from vault-template/ as REAL files (not stow symlinks): the
    # scaffolding is committed into the vault's own private repo, so it stays portable
    # and self-contained on any machine. obsidian.nvim errors on startup if the
    # workspace path is missing, so the PARA + capture folders are always ensured;
    # everything else is copy-if-absent - idempotent, never overwriting live edits.
    # Never commits (that stays the user's call, same as the sync hints below).
    local vault="$1" type="$2"
    local tpl="$DOTFILES_DIR/vault-template" d f
    for d in archive areas assets dailies inbox projects resources templates; do
        mkdir -p "$vault/$d"
    done
    copy_if_absent "$tpl/common/.gitignore"  "$vault/.gitignore"
    copy_if_absent "$tpl/common/.prettierrc" "$vault/.prettierrc"
    copy_if_absent "$tpl/common/Home.md"     "$vault/Home.md"
    copy_if_absent "$tpl/$type/CLAUDE.md"    "$vault/CLAUDE.md"
    copy_if_absent "$tpl/$type/README.md"    "$vault/README.md"
    for f in "$tpl"/common/templates/*; do
        copy_if_absent "$f" "$vault/templates/$(basename "$f")"
    done
    copy_if_absent "$tpl/common/.claude/settings.json" "$vault/.claude/settings.json"
    copy_if_absent "$tpl/common/.claude/vault-check.sh" "$vault/.claude/vault-check.sh"
    for f in "$tpl"/common/.claude/hooks/*.sh; do
        copy_if_absent "$f" "$vault/.claude/hooks/$(basename "$f")"
    done
    copy_if_absent "$tpl/common/.githooks/pre-commit" "$vault/.githooks/pre-commit"
    # vault-check.sh must be +x - the pre-commit runs it only under `[ -x ]`, else integrity is silently skipped.
    chmod +x "$vault/.claude/vault-check.sh" "$vault/.claude/hooks/"*.sh "$vault/.githooks/pre-commit" 2>/dev/null || true
    # git ignores empty dirs, so a fresh skeleton has nothing to commit and the first
    # push fails. Keep each still-empty capture dir trackable with a .gitkeep.
    for d in archive areas assets dailies inbox projects resources templates; do
        [ -n "$(ls -A "$vault/$d" 2>/dev/null)" ] || touch "$vault/$d/.gitkeep"
    done
    # Route git hooks at the tracked .githooks/ dir (the secrets pre-commit guard).
    # Harmless to set repeatedly; only applies once the vault is a git repo. Kept as an
    # if (not `&&`) so a not-yet-a-repo vault leaves the function returning 0 under set -e.
    if [ -d "$vault/.git" ]; then
        git -C "$vault" config core.hooksPath .githooks
    fi
}

create_personal_vault() {
    # Personal knowledge vault (private GitHub). If it isn't a git repo yet, just flag
    # it - the wiring steps are shown together at the very end (print_vault_sync_hints)
    # so they aren't buried mid-run. Idempotent.
    local vault="$HOME/vaults/personal"
    seed_vault "$vault" personal
    if [ -d "$vault/.git" ]; then
        ok "personal vault ready at $vault"
    else
        ok "personal vault skeleton ready at $vault (not synced to a remote)"
        PERSONAL_VAULT_UNWIRED=1
    fi
}

create_work_vault() {
    # Work knowledge vault, an independent sibling on its own separate remote (private
    # GitLab). Same skeleton; if it isn't a git repo yet, flag it for the end-of-run
    # hints rather than printing steps mid-run. Idempotent.
    local vault="$HOME/vaults/work"
    seed_vault "$vault" work
    if [ -d "$vault/.git" ]; then
        ok "work vault ready at $vault"
    else
        ok "work vault skeleton ready at $vault (not synced to a remote)"
        WORK_VAULT_UNWIRED=1
    fi
}

print_vault_sync_hints() {
    # Printed at the very end so it isn't buried in the build logs. Optional: a vault
    # works locally without a remote; this only shows how to back one up / sync it.
    # We never create the remote or store identity here (these dotfiles are public),
    # so the user runs the steps himself. One independent block per vault by design.
    [ -n "${PERSONAL_VAULT_UNWIRED:-}" ] || [ -n "${WORK_VAULT_UNWIRED:-}" ] || return 0
    echo ""
    warn "Optional - notes vault(s) not yet synced to a git remote."
    warn "Set up only the ones you want backed up / synced (run these yourself):"
    if [ -n "${PERSONAL_VAULT_UNWIRED:-}" ]; then
        cat <<'EOF'

  personal vault -> your PRIVATE personal remote:
    cd ~/vaults/personal
    git init -b main
    git remote add origin <private-remote-url>   # e.g. a private <user>/vaults-personal
    git add -A
    git commit -m "Initialize personal vault"
    git push -u origin main
EOF
    fi
    if [ -n "${WORK_VAULT_UNWIRED:-}" ]; then
        cat <<'EOF'

  work vault -> your PRIVATE work remote:
    cd ~/vaults/work
    git init -b main
    git remote add origin <private-remote-url>   # e.g. a private <user>/vaults-work
    git add -A
    git commit -m "Initialize work vault"
    git push -u origin main
EOF
    fi
}

install_bat_themes() {
    # Vendored Catppuccin bat themes (gitignored - fetched here). bat's config dir is
    # stowed; drop the .tmTheme files in and rebuild the cache. The `theme` switcher
    # selects them via BAT_THEME. Idempotent. Must run after stow_packages.
    command -v bat &>/dev/null || return
    local dir changed=0 t enc
    dir="$(bat --config-dir)/themes"
    mkdir -p "$dir"
    for t in "Catppuccin Latte" "Catppuccin Mocha"; do
        [ -f "$dir/$t.tmTheme" ] && continue
        enc="${t// /%20}"
        if curl -sfL "https://raw.githubusercontent.com/catppuccin/bat/main/themes/${enc}.tmTheme" -o "$dir/$t.tmTheme"; then
            changed=1
        else
            warn "Could not fetch bat theme: $t"
            rm -f "$dir/$t.tmTheme"
        fi
    done
    if [ "$changed" = 1 ]; then
        bat cache --build >/dev/null 2>&1 || true
    fi
    ok "bat Catppuccin themes ready"
}

install_nvim_plugins() {
    # Pre-install plugins headlessly so the first interactive launch is ready.
    # Must run after the nvim config is stowed. Idempotent: `install` only
    # fetches missing plugins, `restore` pins them to the committed lazy-lock.json.
    if [ ! -x "$LOCAL_BIN/nvim" ]; then
        return
    fi
    log "Installing Neovim plugins (headless)..."
    timeout 300 "$LOCAL_BIN/nvim" --headless "+Lazy! install" "+Lazy! restore" +qa >/dev/null 2>&1 || true
    ok "Neovim plugins installed"
}

# =============================================================================
# Apps (built from source under apps/)
# =============================================================================

build_apps() {
    # Uniform contract for every project under apps/, regardless of language:
    # a Makefile with a `build` target. Install its toolchain above (e.g. Go)
    # and add nothing here. The built binary lives in the app's own bin/.
    [ -d "$DOTFILES_DIR/apps" ] || return
    local app name
    for app in "$DOTFILES_DIR"/apps/*/; do
        [ -f "$app/Makefile" ] || continue
        name=$(basename "$app")
        log "Building app: $name..."
        if make -C "$app" build >/dev/null 2>&1; then
            ok "Built $name"
        else
            warn "Build failed for $name (toolchain missing?)"
        fi
    done
}

# =============================================================================
# Opt-in: dictate deps (uv + parec/pactl)
# =============================================================================

# Not part of the default run - the dictate package is opt-in. Install its
# non-stock deps only when asked:  ./bootstrap.sh dictate-deps
install_dictate_deps() {
    mkdir -p "$LOCAL_BIN"
    # uv runs the PEP 723 dictate script. Unpinned (latest): opt-in convenience,
    # self-updating, and tolerant of the script's inline deps.
    if command -v uv &>/dev/null; then
        ok "uv already installed"
    else
        log "Installing uv..."
        curl -LsSf https://astral.sh/uv/install.sh \
            | env UV_INSTALL_DIR="$LOCAL_BIN" INSTALLER_NO_MODIFY_PATH=1 sh
        ok "uv installed"
    fi
    # parec (record) + pactl (audio ducking), both from pulseaudio-utils. Absent
    # on a fresh Ubuntu base, usually present on the desktop. Guarded = no-op when
    # already there.
    if command -v parec &>/dev/null && command -v pactl &>/dev/null; then
        ok "parec/pactl already installed"
    else
        log "Installing pulseaudio-utils (parec + pactl)..."
        sudo apt-get update -qq
        sudo apt-get install -y -qq pulseaudio-utils
        ok "pulseaudio-utils installed"
    fi
}

# =============================================================================
# Main
# =============================================================================

# Opt-in extras (run a single named step, then exit):
#   ./bootstrap.sh dictate-deps   # uv + pulseaudio-utils for the dictate package
if [ "${1:-}" = "dictate-deps" ]; then
    install_dictate_deps
    exit 0
fi

log "Starting dotfiles bootstrap..."
mkdir -p "$LOCAL_BIN"
# ~/.local/bin isn't on a fresh machine's PATH yet (login adds it only if it already
# exists); put it there so this run's `command -v` guards + Go build see our installs.
export PATH="$LOCAL_BIN:$PATH"

install_apt_packages
install_nodejs
install_delta
install_fd
install_fzf
install_ghostty
install_git_cliff
install_gitleaks
install_gitmux
install_go
install_hunk
install_lazydocker
install_lazygit
install_neovim
install_nerd_font
install_ruff
install_shellcheck
install_shfmt
install_task
install_tpm
install_yazi
install_zoxide
stow_packages
install_bat_themes
enable_tmux_resurrect_timer
patch_bashrc
create_personal_vault
create_work_vault
install_nvim_plugins
build_apps

echo ""
ok "Done! Restart your shell or run: source ~/.bashrc"
print_vault_sync_hints
