# dotfiles

Development environment for Ubuntu 24.04 - shell, tmux, Neovim, terminal, and CLI tooling - managed
with [GNU Stow](https://www.gnu.org/software/stow/) and reproducible on a fresh machine from a single
`bootstrap.sh`.

> **Built for [Ghostty](https://ghostty.org/).** The tmux, theme switcher, and shaders assume it.
> `bootstrap.sh` installs it; use it as your terminal. Other terminals work but aren't themed.

## Contents

- [What's included](#whats-included)
- [Setup on a new machine](#setup-on-a-new-machine)
- [Usage](#usage)
- [Managing configs](#managing-configs)
- [Notes](#notes)
- [License](#license)

## What's included

### Configs (stow packages)

| Package   | Description                                                                              | Target                           |
| --------- | ---------------------------------------------------------------------------------------- | -------------------------------- |
| `bash`    | Shell customizations, aliases, direnv/fzf/zoxide hooks, vi mode                          | `~/.bashrc.d/`                   |
| `bat`     | Syntax highlighter theme                                                                 | `~/.config/bat/`                 |
| `claude`  | Claude Code settings.json (agentbar hook + statusLine wiring) and statusline script      | `~/.claude/`                     |
| `dictate` | Toggle-key local speech-to-text (faster-whisper) into tmux                               | `~/.local/bin/`                  |
| `ghostty` | Ghostty terminal config (Solarized Light, block cursor, cursor trail shader)             | `~/.config/ghostty/`             |
| `git`     | Git tool settings (delta pager, staging/blame, merge)                                    | `~/.config/git/config`           |
| `hunk`    | hunk diff viewer config (Solarized Light theme, side-by-side)                            | `~/.config/hunk/`                |
| `nvim`    | Neovim config (LazyVim, LSP, plugins)                                                    | `~/.config/nvim/`                |
| `theme`   | Theme switcher - re-skins the terminal stack across four flavors (`design/palette.toml`) | `~/.local/bin/theme`             |
| `tmux`    | Tmux config, gitmux, CI status script                                                    | `~/.tmux.conf`, `~/.gitmux.conf` |
| `yazi`    | Yazi file manager config with zoxide plugin                                              | `~/.config/yazi/`                |

### Apps (built from source)

Binaries built from source under `apps/` (not stow packages). Each has a `Makefile` with a `make build` target; `bootstrap.sh` installs the toolchain and builds them.

| App        | Description                                                          | Language |
| ---------- | -------------------------------------------------------------------- | -------- |
| `agentbar` | tmux sidebar showing every Claude Code agent's state across sessions | Go       |

The sidebar loads from here via a `run-shell` line in `tmux/.tmux.conf`; edit it and reload tmux to pick up changes.

### System dependencies

Installed via `bootstrap.sh` (apt + `~/.local/bin`):

- [bat](https://github.com/sharkdp/bat) - cat with syntax highlighting
- [delta](https://github.com/dandavison/delta) - git diff pager with syntax highlighting
- [direnv](https://direnv.net/) - per-directory environment variables
- [fd](https://github.com/sharkdp/fd) - fast find (powers fzf file search)
- [fzf](https://github.com/junegunn/fzf) - fuzzy finder
- [Ghostty](https://ghostty.org/) - terminal emulator
- [gitmux](https://github.com/arl/gitmux) - git status in tmux
- [GNU Stow](https://www.gnu.org/software/stow/) - symlink manager
- [Go](https://go.dev/) - toolchain for building `apps/` (agentbar)
- [hunk](https://github.com/modem-dev/hunk) - interactive diff viewer (via `gd`/`gds` aliases)
- [JetBrainsMono Nerd Font](https://www.nerdfonts.com/) - terminal/editor font
- [jq](https://github.com/jqlang/jq) - JSON processor
- [lazydocker](https://github.com/jesseduffield/lazydocker) - terminal Docker UI
- [lazygit](https://github.com/jesseduffield/lazygit) - terminal git UI
- [Neovim](https://neovim.io/) - editor
- [ripgrep](https://github.com/BurntSushi/ripgrep) - fast recursive search
- [tmux](https://github.com/tmux/tmux) - terminal multiplexer
- [tree](https://gitlab.com/OldManProgrammer/unix-tree) - directory listing utility
- [yazi](https://github.com/sxyazi/yazi) - terminal file manager
- [zoxide](https://github.com/ajeetdsouza/zoxide) - smarter cd

## Setup on a new machine

### 1. Clone the repo

```bash
git clone <repo-url> ~/dotfiles
```

### 2. Run the bootstrap script

This installs all dependencies, stows configs, and patches `~/.bashrc`:

```bash
cd ~/dotfiles
./bootstrap.sh
```

### 3. Restart your shell

```bash
source ~/.bashrc
```

Neovim plugins will auto-install on first launch via lazy.nvim.
Run `prefix + I` in tmux to install tmux plugins.

### 4. Switch to Ghostty

Bootstrap installed it - open Ghostty and use it going forward; the configs are tuned for it.
(`bootstrap.sh` itself runs from any terminal.)

## Usage

Day-to-day keybindings and commands - shell aliases, tmux, Neovim (LazyVim), hunk, Ghostty - live in
**[CHEATSHEET.md](CHEATSHEET.md)** (also viewable in the terminal via the `cheat` alias). Re-skin the
whole terminal stack with `theme <flavor>` (`solarized-light` · `solarized-dark` · `catppuccin-latte` ·
`catppuccin-mocha`); see [`design/theme-switcher.md`](design/theme-switcher.md).

## Managing configs

### Add a new config

```bash
# 1. Create the package directory (mirrors home dir structure)
mkdir -p ~/dotfiles/newpkg/.config/newapp

# 2. Move the existing config into the package
mv ~/.config/newapp/config.toml ~/dotfiles/newpkg/.config/newapp/config.toml

# 3. Create the symlink
cd ~/dotfiles && stow newpkg
```

### Add a new shell customization

Create a new `.bash` file in the bash package:

```bash
# Edit directly in dotfiles - symlink means it takes effect immediately
vim ~/dotfiles/bash/.bashrc.d/my-feature.bash
```

### Add private/work-specific aliases

Create `~/.bashrc.d/local.bash` (not tracked in git):

```bash
vim ~/.bashrc.d/local.bash
```

### Machine-specific Claude Code settings

The committed `~/.claude/settings.json` is the full baseline - hooks, `statusLine`, `permissions`,
plugins, and prefs. Claude Code has no user-level `settings.local.json` (only a project's is read), and
the file is a stowed symlink, so runtime `/config` edits write into this repo: commit what you want to
keep, or `git checkout` to discard. An existing file is backed up to `*.pre-dotfiles` on first bootstrap.

### Stow commands

```bash
stow <package>       # Link a package
stow -D <package>    # Unlink a package
stow -R <package>    # Re-link (unlink + link)
```

## Notes

- **System `.bashrc` is never overwritten** - customizations live in `~/.bashrc.d/*.bash`, sourced by a loop `bootstrap.sh` appends (with a backup).
- **Private/work aliases** go in `~/.bashrc.d/local.bash` (not tracked).
- **Notes vaults**: `bootstrap.sh` scaffolds two Obsidian vault skeletons (`~/vaults/personal`, `~/vaults/work`) and, at the end, prints optional git-remote wiring steps for any not yet synced; each vault's contents live in its own private repo, never here.
- **Neovim plugins**: `lazy-lock.json` pins versions - commit it to keep installs reproducible.
- **Python venvs**: direnv auto-activates `.venv` per directory.
- **Idempotent**: `bootstrap.sh` is safe to re-run (skips what's installed).
- **Smoke test**: `test/bootstrap-fresh.sh` runs bootstrap in a clean Ubuntu 24.04 container (checks binaries, symlinks, idempotency); run before touching `bootstrap.sh`.

## License

[MIT](LICENSE).
