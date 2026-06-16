#!/usr/bin/env bash
set -euo pipefail

DOTFILES_DIR="$(cd "$(dirname "$0")" && pwd)"
LOCAL_BIN="$HOME/.local/bin"
ARCH="$(uname -m)"

# Platform detection: OS_KIND is "linux" or "macos"; everything else fails fast.
case "$(uname -s)" in
    Linux)  OS_KIND="linux" ;;
    Darwin) OS_KIND="macos" ;;
    *) echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

is_linux() { [ "$OS_KIND" = "linux" ]; }
is_macos() { [ "$OS_KIND" = "macos" ]; }

# Pinned versions (Linux tarballs only — macOS uses Homebrew)
DELTA_VERSION="0.19.2"
FD_VERSION="10.4.2"
FZF_VERSION="0.72.0"
GITMUX_VERSION="0.11.5"
LAZYDOCKER_VERSION="0.25.2"
LAZYGIT_VERSION="0.61.1"
NEOVIM_VERSION="0.12.2"
NERD_FONT_VERSION="3.4.0"
YAZI_VERSION="26.5.6"
ZOXIDE_VERSION="0.9.9"

log() { echo -e "\033[1;34m[dotfiles]\033[0m $*"; }
warn() { echo -e "\033[1;33m[dotfiles]\033[0m $*"; }
ok() { echo -e "\033[1;32m[dotfiles]\033[0m $*"; }

# =============================================================================
# System packages
# =============================================================================

install_apt_packages() {
    # chafa: image previews for yazi
    # gir1.2-appindicator3-0.1 + python3-gi: GNOME top bar indicator for claude-indicator
    # imagemagick: convert/identify, used by snacks.image to render images in nvim
    # inotify-tools: screenshot-watcher (auto-copy screenshots to clipboard)
    local pkgs=(bat build-essential chafa curl direnv fontconfig gir1.2-appindicator3-0.1 imagemagick inotify-tools jq python3-gi ripgrep software-properties-common stow terminator tmux tree unzip wl-clipboard wget)
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

install_brew() {
    if command -v brew &>/dev/null; then
        ok "Homebrew already installed"
        return
    fi
    log "Installing Homebrew..."
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    # Add brew to PATH for this script's session (shellenv prints PATH exports).
    if [ -x /opt/homebrew/bin/brew ]; then
        eval "$(/opt/homebrew/bin/brew shellenv)"
    elif [ -x /usr/local/bin/brew ]; then
        eval "$(/usr/local/bin/brew shellenv)"
    fi
    ok "Homebrew installed"
}

install_brew_packages() {
    # Mirrors the Linux apt + binary-tool lists. Brew gives us up-to-date
    # versions and handles macOS-specific quirks (e.g. neovim from formula).
    local pkgs=(bat chafa direnv fd fzf imagemagick jq lazydocker lazygit neovim node ripgrep stow tmux tree yazi zoxide)
    local missing=()
    for pkg in "${pkgs[@]}"; do
        brew list --formula "$pkg" &>/dev/null || missing+=("$pkg")
    done
    if [ ${#missing[@]} -gt 0 ]; then
        log "Installing brew packages: ${missing[*]}"
        brew install "${missing[@]}"
    else
        ok "Brew packages already installed"
    fi

    # delta and gitmux live in taps / separate formulae.
    brew list --formula git-delta &>/dev/null || { log "Installing git-delta..."; brew install git-delta; }
    brew list --formula gitmux    &>/dev/null || { log "Installing gitmux...";    brew install arl/arl/gitmux 2>/dev/null || brew install gitmux; }

    # Cask: JetBrainsMono Nerd Font (no fc-cache needed; Font Book picks it up).
    brew list --cask font-jetbrains-mono-nerd-font &>/dev/null || {
        log "Installing JetBrainsMono Nerd Font..."
        brew tap homebrew/cask-fonts 2>/dev/null || true
        brew install --cask font-jetbrains-mono-nerd-font
    }

    # Cask: Ghostty.
    if [ ! -d "/Applications/Ghostty.app" ] && ! brew list --cask ghostty &>/dev/null; then
        log "Installing Ghostty..."
        brew install --cask ghostty
    else
        ok "Ghostty already installed"
    fi
}

# =============================================================================
# Node.js (Linux only — macOS gets it via brew)
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
# Linux binary tools -> ~/.local/bin
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
# JetBrainsMono Nerd Font (Linux only — macOS uses brew cask)
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
# Ghostty (Linux — Ubuntu PPA; macOS handled by brew_packages)
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
    # macOS' readlink lacks -f; coreutils' greadlink covers it. Fall back to
    # python if neither is available.
    local resolve
    if command -v greadlink &>/dev/null; then
        resolve="greadlink -f"
    elif readlink -f / &>/dev/null; then
        resolve="readlink -f"
    else
        resolve="python3 -c 'import os,sys;print(os.path.realpath(sys.argv[1]))'"
    fi
    if [ -n "$src" ] && [ "$($resolve -- "$target" 2>/dev/null || $resolve "$target")" = "$($resolve -- "$src" 2>/dev/null || $resolve "$src")" ]; then
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
# Only backs up files we're about to stow in — user's other files stay put.
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
    local packages=(bash bat claude git ghostty nvim tmux yazi)
    if is_linux; then
        # Linux-only packages: GNOME indicator, screenshot watcher (inotify),
        # and the GTK terminator config.
        packages+=(claude-indicator screenshot-watcher terminator)
    fi

    # Single files we own outright: back up the file itself.
    backup_if_not_symlink "$HOME/.tmux.conf"                    "$DOTFILES_DIR/tmux/.tmux.conf"
    backup_if_not_symlink "$HOME/.gitmux.conf"                  "$DOTFILES_DIR/tmux/.gitmux.conf"
    backup_if_not_symlink "$HOME/.local/bin/tmux-ci-status.sh"  "$DOTFILES_DIR/tmux/.local/bin/tmux-ci-status.sh"
    backup_if_not_symlink "$HOME/.claude/settings.json"         "$DOTFILES_DIR/claude/.claude/settings.json"
    backup_if_not_symlink "$HOME/.claude/statusline-command.sh" "$DOTFILES_DIR/claude/.claude/statusline-command.sh"

    # Directories the user may share with their own files: back up only the
    # specific files we ship, leaving the rest of the directory intact.
    backup_pkg_files "$DOTFILES_DIR/bash/.bashrc.d"          "$HOME/.bashrc.d"
    backup_pkg_files "$DOTFILES_DIR/claude/.claude/hooks"    "$HOME/.claude/hooks"

    # Directories we own outright: back up the whole directory.
    backup_if_not_symlink "$HOME/.config/nvim"
    backup_if_not_symlink "$HOME/.config/bat"
    backup_if_not_symlink "$HOME/.config/ghostty"
    backup_if_not_symlink "$HOME/.config/yazi"
    is_linux && backup_if_not_symlink "$HOME/.config/terminator"

    cd "$DOTFILES_DIR"
    for pkg in "${packages[@]}"; do
        log "Stowing $pkg..."
        stow --restow -t "$HOME" "$pkg"
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
# Patch shell rc (~/.bashrc on Linux, ~/.zshrc on macOS)
# =============================================================================

patch_shell_rc() {
    local rc marker="# Load dotfiles shell customizations"
    if is_macos; then
        rc="$HOME/.zshrc"
        [ -f "$rc" ] || touch "$rc"
    else
        rc="$HOME/.bashrc"
    fi
    if grep -qF "$marker" "$rc"; then
        ok "$rc already patched"
        return
    fi
    log "Patching $rc..."
    cp "$rc" "${rc}.pre-dotfiles"
    # Same loop for bash and zsh. Each .bashrc.d/*.bash file is responsible for
    # guarding bash-only or zsh-only sections internally (via $BASH_VERSION /
    # $ZSH_VERSION) so a single source line works for both shells.
    cat >>"$rc" <<'EOF'

# Load dotfiles shell customizations
for f in ~/.bashrc.d/*.bash; do [ -r "$f" ] && source "$f"; done
EOF
    ok "$rc patched (backup at ${rc}.pre-dotfiles)"
}

create_notes_vault() {
    # obsidian.nvim errors on startup if its workspace path is missing and does
    # not create it itself. Layout matches nvim's obsidian.lua.
    local vault="$HOME/notes"
    mkdir -p "$vault/notes" "$vault/dailies" "$vault/templates"
    ok "notes vault ready at $vault"
}

install_nvim_plugins() {
    # Pre-install plugins headlessly so the first interactive launch is ready.
    # Must run after the nvim config is stowed. Idempotent: `install` only
    # fetches missing plugins, `restore` pins them to the committed lazy-lock.json.
    local nvim_bin
    if [ -x "$LOCAL_BIN/nvim" ]; then
        nvim_bin="$LOCAL_BIN/nvim"
    elif command -v nvim &>/dev/null; then
        nvim_bin=$(command -v nvim)
    else
        return
    fi
    log "Installing Neovim plugins (headless)..."
    # macOS lacks GNU timeout by default — use perl as a portable fallback.
    if command -v timeout &>/dev/null; then
        timeout 300 "$nvim_bin" --headless "+Lazy! install" "+Lazy! restore" +qa >/dev/null 2>&1 || true
    else
        perl -e 'alarm shift; exec @ARGV' 300 "$nvim_bin" --headless "+Lazy! install" "+Lazy! restore" +qa >/dev/null 2>&1 || true
    fi
    ok "Neovim plugins installed"
}

# =============================================================================
# Main
# =============================================================================

log "Starting dotfiles bootstrap on $OS_KIND..."
mkdir -p "$LOCAL_BIN"

if is_linux; then
    install_apt_packages
    install_nodejs
    install_delta
    install_fd
    install_fzf
    install_ghostty
    install_gitmux
    install_lazydocker
    install_lazygit
    install_nerd_font
    install_neovim
    install_yazi
    install_zoxide
else
    install_brew
    install_brew_packages
fi

install_tpm
stow_packages
patch_shell_rc
create_notes_vault
install_nvim_plugins

echo ""
if is_macos; then
    ok "Done! Restart your shell or run: source ~/.zshrc"
else
    ok "Done! Restart your shell or run: source ~/.bashrc"
fi
