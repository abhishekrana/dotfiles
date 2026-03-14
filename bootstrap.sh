#!/usr/bin/env bash
set -euo pipefail

DOTFILES_DIR="$(cd "$(dirname "$0")" && pwd)"
LOCAL_BIN="$HOME/.local/bin"
ARCH="$(uname -m)"

log()  { echo -e "\033[1;34m[dotfiles]\033[0m $*"; }
warn() { echo -e "\033[1;33m[dotfiles]\033[0m $*"; }
ok()   { echo -e "\033[1;32m[dotfiles]\033[0m $*"; }

# =============================================================================
# APT packages
# =============================================================================

install_apt_packages() {
    local pkgs=(stow tmux direnv terminator git-lfs curl wget unzip fontconfig bat)
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
# Binary tools -> ~/.local/bin
# =============================================================================

install_neovim() {
    if [ -x "$LOCAL_BIN/nvim" ]; then
        ok "neovim already installed"
        return
    fi
    log "Installing neovim..."
    local url="https://github.com/neovim/neovim/releases/latest/download/nvim-linux-x86_64.tar.gz"
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp"/nvim-linux-x86_64 "$HOME/.local/nvim"
    ln -sf "$HOME/.local/nvim/bin/nvim" "$LOCAL_BIN/nvim"
    rm -rf "$tmp"
    ok "neovim installed"
}

install_fzf() {
    if [ -x "$LOCAL_BIN/fzf" ]; then
        ok "fzf already installed"
        return
    fi
    log "Installing fzf..."
    local version
    version=$(curl -sSL "https://api.github.com/repos/junegunn/fzf/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
    local url="https://github.com/junegunn/fzf/releases/download/v${version}/fzf-${version}-linux_amd64.tar.gz"
    curl -sSL "$url" | tar xz -C "$LOCAL_BIN" fzf
    chmod +x "$LOCAL_BIN/fzf"
    ok "fzf $version installed"
}

install_zoxide() {
    if [ -x "$LOCAL_BIN/zoxide" ]; then
        ok "zoxide already installed"
        return
    fi
    log "Installing zoxide..."
    curl -sSfL https://raw.githubusercontent.com/ajeetdsouza/zoxide/main/install.sh | sh
    ok "zoxide installed"
}

install_gitmux() {
    if [ -x "$LOCAL_BIN/gitmux" ]; then
        ok "gitmux already installed"
        return
    fi
    log "Installing gitmux..."
    local version
    version=$(curl -sSL "https://api.github.com/repos/arl/gitmux/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
    local url="https://github.com/arl/gitmux/releases/download/v${version}/gitmux_v${version}_linux_amd64.tar.gz"
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp/gitmux" "$LOCAL_BIN/gitmux"
    chmod +x "$LOCAL_BIN/gitmux"
    rm -rf "$tmp"
    ok "gitmux $version installed"
}

# =============================================================================
# JetBrainsMono Nerd Font
# =============================================================================

install_nerd_font() {
    if fc-list | grep -qi "JetBrainsMono Nerd" &>/dev/null; then
        ok "JetBrainsMono Nerd Font already installed"
        return
    fi
    log "Installing JetBrainsMono Nerd Font..."
    local font_dir="$HOME/.local/share/fonts"
    mkdir -p "$font_dir"
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "https://github.com/ryanoasis/nerd-fonts/releases/latest/download/JetBrainsMono.tar.xz" -o "$tmp/JetBrainsMono.tar.xz"
    tar xf "$tmp/JetBrainsMono.tar.xz" -C "$font_dir"
    fc-cache -fv "$font_dir" >/dev/null 2>&1
    rm -rf "$tmp"
    ok "JetBrainsMono Nerd Font installed"
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
    local packages=(bash nvim tmux terminator bat)

    # Backup existing configs that would conflict
    backup_if_not_symlink "$HOME/.bashrc.d"
    backup_if_not_symlink "$HOME/.config/nvim"
    backup_if_not_symlink "$HOME/.tmux.conf"
    backup_if_not_symlink "$HOME/.gitmux.conf"
    backup_if_not_symlink "$HOME/.config/terminator"
    backup_if_not_symlink "$HOME/.config/bat"

    cd "$DOTFILES_DIR"
    for pkg in "${packages[@]}"; do
        log "Stowing $pkg..."
        stow --restow "$pkg"
    done
    ok "All packages stowed"
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
    cat >> "$HOME/.bashrc" << 'EOF'

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
install_neovim
install_fzf
install_zoxide
install_gitmux
install_nerd_font
install_tpm
stow_packages
patch_bashrc

echo ""
ok "Done! Restart your shell or run: source ~/.bashrc"
ok "Don't forget to set up ~/.gitconfig.local with your [user] name and email"
