#!/usr/bin/env bash
# install.sh - installs the software this environment needs. Every version is
# pinned here; nothing else in the repo downloads a tool.
#
#   ./install.sh                 list the steps
#   ./install.sh all             everything (bootstrap.sh calls this)
#   ./install.sh gate-tools      only what `task check` needs (CI calls this)
#   ./install.sh install_tmux    one step by name
#   ./install.sh whisper-vulkan  whisper.cpp on the GPU for dictate (opt-in)
#
# Sourced by bootstrap.sh, which adds the machine wiring (stow, vaults, bashrc).
set -euo pipefail

LOCAL_BIN="$HOME/.local/bin"

# The only place a platform is decided. Release-asset names are grouped by
# upstream naming convention, so supporting another platform is one more arm
# here and no change in any install function. OS_KIND is exported for
# bootstrap.sh, which sources this file.
case "$(uname -s)" in
    Linux)
        export OS_KIND="linux"
        RUST_MUSL="x86_64-unknown-linux-musl" # delta fd git-cliff zoxide
        RUST_GNU="x86_64-unknown-linux-gnu"   # ruff yazi
        GO_SLUG="linux_amd64"                 # fzf gitmux shfmt task
        GOREL_SLUG="Linux_x86_64"             # lazydocker lazygit
        GITLEAKS_SLUG="linux_x64"
        GO_DIST="linux-amd64"
        LEAF_SLUG="linux-x86_64"
        NEOVIM_SLUG="linux-x86_64"
        SHELLCHECK_SLUG="linux.x86_64"
        ;;
    *)
        echo "dotfiles supports Linux (Ubuntu 24.04). Detected: $(uname -s)." >&2
        echo "See 'Platform support' in README.md." >&2
        exit 1
        ;;
esac

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
LEAF_VERSION="1.27.1"
NEOVIM_VERSION="0.12.4"
NERD_FONT_VERSION="3.4.0"
RUFF_VERSION="0.16.0"
SHELLCHECK_VERSION="0.11.0"
SHFMT_VERSION="3.13.1"
# Ubuntu 24.04 ships no SPIRV-Headers package, and whisper.cpp's Vulkan backend
# needs the spv:: constants; pinned here like everything else it downloads.
SPIRV_HEADERS_VERSION="vulkan-sdk-1.4.357.0"
TASK_VERSION="3.52.0"
TMUX_VERSION="3.7b"
WHISPER_CPP_VERSION="1.9.2"
WHISPER_CPP_MODEL="ggml-large-v3-turbo-q5_0.bin" # what dictate's whispercpp backend loads
YAZI_VERSION="26.5.6"
ZOXIDE_VERSION="0.10.0"

# URL of a GitHub release asset: gh_url <owner/repo> <tag> <asset>
gh_url() { printf 'https://github.com/%s/releases/download/%s/%s' "$1" "$2" "$3"; }

log() { echo -e "\033[1;34m[dotfiles]\033[0m $*"; }
warn() { echo -e "\033[1;33m[dotfiles]\033[0m $*"; }
ok() { echo -e "\033[1;32m[dotfiles]\033[0m $*"; }

install_apt_packages() {
    # chafa: image previews for yazi
    # imagemagick: convert/identify, used by image.nvim to render images in nvim
    local pkgs=(
        bat bison build-essential chafa curl direnv fontconfig imagemagick jq
        libevent-dev libncurses-dev pkg-config ripgrep software-properties-common
        stow tree unzip wget wl-clipboard xclip
    )
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
        local url="https://raw.githubusercontent.com/catppuccin/bat/main/themes/${enc}.tmTheme"
        if curl -sfL "$url" -o "$dir/$t.tmTheme"; then
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

install_delta() {
    if [ -x "$LOCAL_BIN/delta" ] && "$LOCAL_BIN/delta" --version 2>/dev/null | grep -q "$DELTA_VERSION"; then
        ok "delta $DELTA_VERSION already installed"
        return
    fi
    log "Installing delta $DELTA_VERSION..."
    local url
    url=$(gh_url dandavison/delta "${DELTA_VERSION}" \
        "delta-${DELTA_VERSION}-${RUST_MUSL}.tar.gz")
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp/delta-${DELTA_VERSION}-${RUST_MUSL}/delta" "$LOCAL_BIN/delta"
    chmod +x "$LOCAL_BIN/delta"
    rm -rf "$tmp"
    ok "delta $DELTA_VERSION installed"
}

install_dictate_deps() {
    mkdir -p "$LOCAL_BIN"
    # uv runs the PEP 723 dictate script. Unpinned (latest): opt-in convenience,
    # self-updating, and tolerant of the script's inline deps.
    if command -v uv &>/dev/null; then
        ok "uv already installed"
    else
        log "Installing uv..."
        curl -LsSf https://astral.sh/uv/install.sh |
            env UV_INSTALL_DIR="$LOCAL_BIN" INSTALLER_NO_MODIFY_PATH=1 sh
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

install_fd() {
    if [ -x "$LOCAL_BIN/fd" ] && "$LOCAL_BIN/fd" --version 2>/dev/null | grep -q "$FD_VERSION"; then
        ok "fd $FD_VERSION already installed"
        return
    fi
    log "Installing fd $FD_VERSION..."
    local url
    url=$(gh_url sharkdp/fd "v${FD_VERSION}" "fd-v${FD_VERSION}-${RUST_MUSL}.tar.gz")
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp/fd-v${FD_VERSION}-${RUST_MUSL}/fd" "$LOCAL_BIN/fd"
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
    local url
    url=$(gh_url junegunn/fzf "v${FZF_VERSION}" "fzf-${FZF_VERSION}-${GO_SLUG}.tar.gz")
    curl -sSL "$url" | tar xz -C "$LOCAL_BIN" fzf
    chmod +x "$LOCAL_BIN/fzf"
    ok "fzf $FZF_VERSION installed"
}

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

install_git_cliff() {
    if [ -x "$LOCAL_BIN/git-cliff" ] &&
        "$LOCAL_BIN/git-cliff" --version 2>/dev/null | grep -q "$GIT_CLIFF_VERSION"; then
        ok "git-cliff $GIT_CLIFF_VERSION already installed"
        return
    fi
    log "Installing git-cliff $GIT_CLIFF_VERSION..."
    local url
    url=$(gh_url orhun/git-cliff "v${GIT_CLIFF_VERSION}" \
        "git-cliff-${GIT_CLIFF_VERSION}-${RUST_MUSL}.tar.gz")
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp/git-cliff-${GIT_CLIFF_VERSION}/git-cliff" "$LOCAL_BIN/git-cliff"
    chmod +x "$LOCAL_BIN/git-cliff"
    rm -rf "$tmp"
    ok "git-cliff $GIT_CLIFF_VERSION installed"
}

install_gitleaks() {
    if [ -x "$LOCAL_BIN/gitleaks" ] && "$LOCAL_BIN/gitleaks" version 2>/dev/null | grep -q "$GITLEAKS_VERSION"; then
        ok "gitleaks $GITLEAKS_VERSION already installed"
        return
    fi
    log "Installing gitleaks $GITLEAKS_VERSION..."
    local url
    url=$(gh_url gitleaks/gitleaks "v${GITLEAKS_VERSION}" "gitleaks_${GITLEAKS_VERSION}_${GITLEAKS_SLUG}.tar.gz")
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp/gitleaks" "$LOCAL_BIN/gitleaks"
    chmod +x "$LOCAL_BIN/gitleaks"
    rm -rf "$tmp"
    ok "gitleaks $GITLEAKS_VERSION installed"
}

install_gitmux() {
    if [ -x "$LOCAL_BIN/gitmux" ] && [ -f "$LOCAL_BIN/.gitmux-version" ] &&
        grep -q "$GITMUX_VERSION" "$LOCAL_BIN/.gitmux-version"; then
        ok "gitmux $GITMUX_VERSION already installed"
        return
    fi
    log "Installing gitmux $GITMUX_VERSION..."
    local url
    url=$(gh_url arl/gitmux "v${GITMUX_VERSION}" "gitmux_v${GITMUX_VERSION}_${GO_SLUG}.tar.gz")
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp/gitmux" "$LOCAL_BIN/gitmux"
    chmod +x "$LOCAL_BIN/gitmux"
    echo "$GITMUX_VERSION" >"$LOCAL_BIN/.gitmux-version"
    rm -rf "$tmp"
    ok "gitmux $GITMUX_VERSION installed"
}

install_go() {
    if [ -x "$LOCAL_BIN/go" ] && "$LOCAL_BIN/go" version 2>/dev/null | grep -q "go${GO_VERSION} "; then
        ok "Go $GO_VERSION already installed"
        return
    fi
    log "Installing Go $GO_VERSION..."
    rm -rf "$HOME/.local/go"
    local url="https://go.dev/dl/go${GO_VERSION}.${GO_DIST}.tar.gz"
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
    if [ -x "$LOCAL_BIN/lazydocker" ] &&
        "$LOCAL_BIN/lazydocker" --version 2>/dev/null | grep -q "$LAZYDOCKER_VERSION"; then
        ok "lazydocker $LAZYDOCKER_VERSION already installed"
        return
    fi
    log "Installing lazydocker $LAZYDOCKER_VERSION..."
    local url
    url=$(gh_url jesseduffield/lazydocker "v${LAZYDOCKER_VERSION}" \
        "lazydocker_${LAZYDOCKER_VERSION}_${GOREL_SLUG}.tar.gz")
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
    local url
    url=$(gh_url jesseduffield/lazygit "v${LAZYGIT_VERSION}" \
        "lazygit_${LAZYGIT_VERSION}_${GOREL_SLUG}.tar.gz")
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp/lazygit" "$LOCAL_BIN/lazygit"
    chmod +x "$LOCAL_BIN/lazygit"
    rm -rf "$tmp"
    ok "lazygit $LAZYGIT_VERSION installed"
}

install_leaf() {
    if [ -x "$LOCAL_BIN/leaf" ] && "$LOCAL_BIN/leaf" --version 2>/dev/null | grep -q "$LEAF_VERSION"; then
        ok "leaf $LEAF_VERSION already installed"
        return
    fi
    log "Installing leaf $LEAF_VERSION..."
    local url
    # leaf tags releases without the usual `v` prefix, and ships a bare binary
    # rather than an archive - no tar step here.
    url=$(gh_url rivolink/leaf "$LEAF_VERSION" "leaf-${LEAF_SLUG}")
    curl -sSL -o "$LOCAL_BIN/leaf" "$url"
    chmod +x "$LOCAL_BIN/leaf"
    ok "leaf $LEAF_VERSION installed"
}

install_neovim() {
    if [ -x "$LOCAL_BIN/nvim" ] && "$LOCAL_BIN/nvim" --version 2>/dev/null | grep -q "v${NEOVIM_VERSION}"; then
        ok "neovim $NEOVIM_VERSION already installed"
        return
    fi
    log "Installing neovim $NEOVIM_VERSION..."
    rm -rf "$HOME/.local/nvim"
    local url
    url=$(gh_url neovim/neovim "v${NEOVIM_VERSION}" "nvim-${NEOVIM_SLUG}.tar.gz")
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp/nvim-${NEOVIM_SLUG}" "$HOME/.local/nvim"
    ln -sf "$HOME/.local/nvim/bin/nvim" "$LOCAL_BIN/nvim"
    rm -rf "$tmp"
    ok "neovim $NEOVIM_VERSION installed"
}

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
    local url
    url=$(gh_url ryanoasis/nerd-fonts "v${NERD_FONT_VERSION}" JetBrainsMono.tar.xz)
    curl -sSL "$url" -o "$tmp/JetBrainsMono.tar.xz"
    tar xf "$tmp/JetBrainsMono.tar.xz" -C "$font_dir"
    fc-cache -fv "$font_dir" >/dev/null 2>&1
    echo "$NERD_FONT_VERSION" >"$font_dir/.nerd-font-version"
    rm -rf "$tmp"
    ok "JetBrainsMono Nerd Font $NERD_FONT_VERSION installed"
}

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

install_ruff() {
    if [ -x "$LOCAL_BIN/ruff" ] && "$LOCAL_BIN/ruff" --version 2>/dev/null | grep -q "$RUFF_VERSION"; then
        ok "ruff $RUFF_VERSION already installed"
        return
    fi
    log "Installing ruff $RUFF_VERSION..."
    local url="https://github.com/astral-sh/ruff/releases/download/${RUFF_VERSION}/ruff-${RUST_GNU}.tar.gz"
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp/ruff-${RUST_GNU}/ruff" "$LOCAL_BIN/ruff"
    chmod +x "$LOCAL_BIN/ruff"
    rm -rf "$tmp"
    ok "ruff $RUFF_VERSION installed"
}

install_shellcheck() {
    if [ -x "$LOCAL_BIN/shellcheck" ] &&
        "$LOCAL_BIN/shellcheck" --version 2>/dev/null | grep -q "$SHELLCHECK_VERSION"; then
        ok "shellcheck $SHELLCHECK_VERSION already installed"
        return
    fi
    log "Installing shellcheck $SHELLCHECK_VERSION..."
    local url
    url=$(gh_url koalaman/shellcheck "v${SHELLCHECK_VERSION}" \
        "shellcheck-v${SHELLCHECK_VERSION}.${SHELLCHECK_SLUG}.tar.xz")
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xJ -C "$tmp"
    mv "$tmp/shellcheck-v${SHELLCHECK_VERSION}/shellcheck" "$LOCAL_BIN/shellcheck"
    chmod +x "$LOCAL_BIN/shellcheck"
    rm -rf "$tmp"
    ok "shellcheck $SHELLCHECK_VERSION installed"
}

install_shfmt() {
    if [ -x "$LOCAL_BIN/shfmt" ] && "$LOCAL_BIN/shfmt" --version 2>/dev/null | grep -q "$SHFMT_VERSION"; then
        ok "shfmt $SHFMT_VERSION already installed"
        return
    fi
    log "Installing shfmt $SHFMT_VERSION..."
    curl -sSL -o "$LOCAL_BIN/shfmt" \
        "https://github.com/mvdan/sh/releases/download/v${SHFMT_VERSION}/shfmt_v${SHFMT_VERSION}_${GO_SLUG}"
    chmod +x "$LOCAL_BIN/shfmt"
    ok "shfmt $SHFMT_VERSION installed"
}

install_task() {
    if [ -x "$LOCAL_BIN/task" ] && "$LOCAL_BIN/task" --version 2>/dev/null | grep -q "$TASK_VERSION"; then
        ok "task $TASK_VERSION already installed"
        return
    fi
    log "Installing task $TASK_VERSION..."
    local url
    url=$(gh_url go-task/task "v${TASK_VERSION}" "task_${GO_SLUG}.tar.gz")
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp/task" "$LOCAL_BIN/task"
    chmod +x "$LOCAL_BIN/task"
    rm -rf "$tmp"
    ok "task $TASK_VERSION installed"
}

install_tmux() {
    if [ -x "$LOCAL_BIN/tmux" ] && "$LOCAL_BIN/tmux" -V 2>/dev/null | grep -q "tmux $TMUX_VERSION"; then
        ok "tmux $TMUX_VERSION already installed"
        return
    fi
    log "Building tmux $TMUX_VERSION from source..."
    local url tmp
    url=$(gh_url tmux/tmux "$TMUX_VERSION" "tmux-${TMUX_VERSION}.tar.gz")
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    (cd "$tmp/tmux-${TMUX_VERSION}" && ./configure --prefix="$HOME/.local" >/dev/null &&
        make -j"$(nproc)" >/dev/null && make install >/dev/null)
    rm -rf "$tmp"
    ok "tmux $TMUX_VERSION installed"
}

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

# whisper.cpp built against Vulkan, for dictate's `whispercpp` backend: the
# Radeon iGPU runs the same Whisper models several times faster than the CPU
# does, which is what buys large-v3-turbo at small.en's latency. Opt-in (not in
# all_tools) - it needs apt packages, compiles for a few minutes, and pulls a
# ~570MB model. Built static, so what lands on PATH is one self-contained binary.
install_whisper_cpp() {
    local models="$HOME/.local/share/whisper-cpp/models" src
    if [ -x "$LOCAL_BIN/whisper-server" ] && [ -f "$models/$WHISPER_CPP_MODEL" ]; then
        ok "whisper.cpp (Vulkan) already installed"
        return
    fi
    # glslc compiles the shaders, libvulkan-dev supplies headers + link lib, and
    # vulkan-tools is how you check the GPU is seen (`vulkaninfo --summary`).
    # The RADV driver itself comes with mesa on any AMD desktop install.
    if ! command -v glslc &>/dev/null || [ ! -f /usr/include/vulkan/vulkan.h ]; then
        log "Installing Vulkan build deps (glslc, libvulkan-dev, vulkan-tools)..."
        sudo apt-get update -qq
        sudo apt-get install -y -qq glslc libvulkan-dev vulkan-tools
    fi
    src=$(mktemp -d)
    log "Building whisper.cpp $WHISPER_CPP_VERSION with Vulkan (a few minutes)..."
    git clone -q --depth 1 --branch "$SPIRV_HEADERS_VERSION" \
        https://github.com/KhronosGroup/SPIRV-Headers.git "$src/spirv-src"
    cmake -S "$src/spirv-src" -B "$src/spirv-build" -DCMAKE_INSTALL_PREFIX="$src/spirv" >/dev/null
    cmake --install "$src/spirv-build" >/dev/null
    git clone -q --depth 1 --branch "v$WHISPER_CPP_VERSION" \
        https://github.com/ggml-org/whisper.cpp.git "$src/whisper.cpp"
    # CMAKE_CXX_FLAGS as well as CMAKE_PREFIX_PATH: whisper.cpp find_package()s
    # SPIRV-Headers but never puts its include dir on the compile line, so the
    # spv:: constants go missing without this.
    cmake -S "$src/whisper.cpp" -B "$src/build" -DCMAKE_BUILD_TYPE=Release \
        -DGGML_VULKAN=1 -DBUILD_SHARED_LIBS=OFF \
        -DCMAKE_PREFIX_PATH="$src/spirv" -DCMAKE_CXX_FLAGS="-I$src/spirv/include" \
        -DWHISPER_BUILD_TESTS=OFF -DWHISPER_BUILD_EXAMPLES=ON >/dev/null
    cmake --build "$src/build" -j "$(nproc)" --target whisper-server >/dev/null
    install -Dm755 "$src/build/bin/whisper-server" "$LOCAL_BIN/whisper-server"
    rm -rf "$src"
    ok "whisper-server installed"
    if [ ! -f "$models/$WHISPER_CPP_MODEL" ]; then
        log "Downloading $WHISPER_CPP_MODEL (~570MB)..."
        mkdir -p "$models"
        curl -sSLo "$models/$WHISPER_CPP_MODEL" \
            "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/$WHISPER_CPP_MODEL"
    fi
    ok "whisper.cpp ready - enable with DICTATE_BACKEND=whispercpp"
}

install_yazi() {
    if [ -x "$LOCAL_BIN/yazi" ] && "$LOCAL_BIN/yazi" --version 2>/dev/null | grep -q "$YAZI_VERSION"; then
        ok "yazi $YAZI_VERSION already installed"
        return
    fi
    log "Installing yazi $YAZI_VERSION..."
    local url
    url=$(gh_url sxyazi/yazi "v${YAZI_VERSION}" yazi-${RUST_GNU}.zip)
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" -o "$tmp/yazi.zip"
    unzip -o "$tmp/yazi.zip" -d "$tmp" >/dev/null
    mv "$tmp/yazi-${RUST_GNU}/yazi" "$LOCAL_BIN/yazi"
    mv "$tmp/yazi-${RUST_GNU}/ya" "$LOCAL_BIN/ya"
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
    local url
    url=$(gh_url ajeetdsouza/zoxide "v${ZOXIDE_VERSION}" \
        "zoxide-${ZOXIDE_VERSION}-${RUST_MUSL}.tar.gz")
    local tmp
    tmp=$(mktemp -d)
    curl -sSL "$url" | tar xz -C "$tmp"
    mv "$tmp/zoxide" "$LOCAL_BIN/zoxide"
    chmod +x "$LOCAL_BIN/zoxide"
    rm -rf "$tmp"
    ok "zoxide $ZOXIDE_VERSION installed"
}

# Everything `task check` needs. Kept here, not in the workflows, so CI installs
# with the same code and the same pins a developer machine does.
gate_tools() {
    install_apt_packages
    install_git_cliff
    install_gitleaks
    install_ruff
    install_shellcheck
    install_shfmt
    install_task
    install_tmux
}

# Every tool, in dependency order (node before hunk, which installs via npm).
all_tools() {
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
    install_leaf
    install_neovim
    install_nerd_font
    install_ruff
    install_shellcheck
    install_shfmt
    install_task
    install_tmux
    install_tpm
    install_yazi
    install_zoxide
}

run_step() {
    case $1 in
        all) all_tools ;;
        gate-tools) gate_tools ;;
        dictate-deps) install_dictate_deps ;;
        whisper-vulkan) install_whisper_cpp ;;
        install_*)
            declare -F "$1" >/dev/null || {
                warn "no such step: $1"
                exit 2
            }
            "$1"
            ;;
        *)
            warn "unknown step: $1"
            exit 2
            ;;
    esac
}

usage() {
    echo "usage: ./install.sh <step>...   (all | gate-tools | dictate-deps | whisper-vulkan | install_*)"
    echo "steps:"
    declare -F | awk '{print $3}' | grep '^install_' | sed 's/^/  /'
}

# ~/.local/bin isn't on a fresh machine's PATH yet; put it there so this run's
# `command -v` guards and the Go build see what we install.
mkdir -p "$LOCAL_BIN"
export PATH="$LOCAL_BIN:$PATH"

# CLI only when executed directly - bootstrap.sh sources this file.
if [ "${BASH_SOURCE[0]}" = "$0" ]; then
    [ $# -gt 0 ] || {
        usage
        exit 0
    }
    for step in "$@"; do run_step "$step"; done
fi
