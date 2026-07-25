#!/usr/bin/env bash
set -euo pipefail

DOTFILES_DIR="$(cd "$(dirname "$0")" && pwd)"

# All software installs (and every version pin) live in install.sh.
# shellcheck source=install.sh
. "$DOTFILES_DIR/install.sh"

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
    copy_if_absent "$tpl/common/.gitignore" "$vault/.gitignore"
    copy_if_absent "$tpl/common/.prettierrc" "$vault/.prettierrc"
    copy_if_absent "$tpl/common/Home.md" "$vault/Home.md"
    copy_if_absent "$tpl/$type/CLAUDE.md" "$vault/CLAUDE.md"
    copy_if_absent "$tpl/$type/README.md" "$vault/README.md"
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
    chmod +x "$vault/.claude/vault-check.sh" "$vault/.claude/hooks/"*.sh \
        "$vault/.githooks/pre-commit" 2>/dev/null || true
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
# =============================================================================
# Main
# =============================================================================

log "Starting dotfiles bootstrap..."
all_tools

stow_packages
# These two need the stowed configs in place first.
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
