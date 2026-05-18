#!/usr/bin/env bash
set -euo pipefail

DOTFILES_DIR="$(cd "$(dirname "$0")" && pwd)"
LOCAL_BIN="$HOME/.local/bin"
ARCH="$(uname -m)"

# Pinned versions (update these to upgrade)
DELTA_VERSION="0.19.2"
FD_VERSION="10.4.2"
FZF_VERSION="0.70.0"
GITMUX_VERSION="0.11.5"
LAZYDOCKER_VERSION="0.25.0"
LAZYGIT_VERSION="0.60.0"
NEOVIM_VERSION="0.12.0"
NERD_FONT_VERSION="3.4.0"
YAZI_VERSION="26.1.22"
ZOXIDE_VERSION="0.9.9"

log() { echo -e "\033[1;34m[dotfiles]\033[0m $*"; }
warn() { echo -e "\033[1;33m[dotfiles]\033[0m $*"; }
ok() { echo -e "\033[1;32m[dotfiles]\033[0m $*"; }

# =============================================================================
# APT packages
# =============================================================================

install_apt_packages() {
    # chafa: image previews for yazi
    # gir1.2-appindicator3-0.1: GNOME top bar indicator for claude-indicator
    local pkgs=(bat build-essential chafa curl direnv fontconfig gir1.2-appindicator3-0.1 jq python3-gi ripgrep stow terminator tmux unzip wl-clipboard wget)
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
    ok "TPM installed — run 'prefix + I' in tmux to install plugins"
}

# =============================================================================
# Herdr (terminal-native agent runtime)
# =============================================================================

install_herdr() {
    if ! command -v herdr &>/dev/null; then
        log "Installing herdr..."
        curl -fsSL https://herdr.dev/install.sh | sh
    else
        ok "herdr $(herdr --version 2>/dev/null | awk '{print $2}') already installed"
    fi
}

# Provision herdr's Claude integration: writes ~/.claude/hooks/herdr-agent-state.sh
# and adds PreToolUse/PostToolUse/Notification/PermissionRequest/PostToolUseFailure/
# SessionEnd entries to ~/.claude/settings.json. Idempotent.
# Must run AFTER stow_packages because ~/.claude/hooks must be the symlinked dir
# (otherwise herdr writes to a stale location). The hook file itself is gitignored
# -- herdr owns it and refreshes it on each install.
install_herdr_claude_integration() {
    command -v herdr &>/dev/null || return 0
    command -v claude &>/dev/null || { warn "claude CLI not found -- skipping herdr Claude integration"; return 0; }
    log "Provisioning herdr Claude integration..."
    herdr integration install claude >/dev/null
    ok "herdr Claude integration installed"
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
    local target="$1"
    if [ -e "$target" ] && [ ! -L "$target" ]; then
        warn "Backing up $target -> ${target}.pre-dotfiles"
        mv "$target" "${target}.pre-dotfiles"
    fi
}

stow_packages() {
    local packages=(bash bat claude claude-indicator git ghostty herdr nvim terminator tmux yazi)

    # Backup existing configs that would conflict
    backup_if_not_symlink "$HOME/.bashrc.d"
    backup_if_not_symlink "$HOME/.config/nvim"
    backup_if_not_symlink "$HOME/.tmux.conf"
    backup_if_not_symlink "$HOME/.gitmux.conf"
    backup_if_not_symlink "$HOME/.config/terminator"
    backup_if_not_symlink "$HOME/.config/bat"
    backup_if_not_symlink "$HOME/.config/ghostty"
    # Only back up the config file -- the dir holds live sockets/logs/session.json.
    backup_if_not_symlink "$HOME/.config/herdr/config.toml"
    backup_if_not_symlink "$HOME/.config/yazi"
    backup_if_not_symlink "$HOME/.local/bin/tmux-ci-status.sh"
    backup_if_not_symlink "$HOME/.claude/hooks"
    backup_if_not_symlink "$HOME/.claude/statusline-command.sh"

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

# =============================================================================
# Patch ~/.bashrc
# =============================================================================

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

# =============================================================================
# Main
# =============================================================================

log "Starting dotfiles bootstrap..."
mkdir -p "$LOCAL_BIN"

install_apt_packages
install_nodejs
install_delta
install_fd
install_fzf
install_ghostty
install_gitmux
install_herdr
install_lazydocker
install_lazygit
install_nerd_font
install_neovim
install_tpm
install_yazi
install_zoxide
stow_packages
install_herdr_claude_integration
patch_bashrc

echo ""
ok "Done! Restart your shell or run: source ~/.bashrc"
