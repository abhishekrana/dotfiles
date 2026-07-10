# dotfiles

Personal development environment managed with [GNU Stow](https://www.gnu.org/software/stow/).

## What's included

### Configs (stow packages)

| Package      | Description                                                                  | Target                           |
| ------------ | ---------------------------------------------------------------------------- | -------------------------------- |
| `bash`       | Shell customizations, aliases, direnv/fzf/zoxide hooks, vi mode              | `~/.bashrc.d/`                   |
| `bat`        | Syntax highlighter theme                                                     | `~/.config/bat/`                 |
| `claude`     | Claude Code hooks, statusline, and settings.json (hooks/statusLine wiring)   | `~/.claude/hooks/`, `~/.claude/` |
| `dictate`    | Push-to-talk local speech-to-text (faster-whisper) for Wayland               | `~/.local/bin/`                  |
| `git`        | Git tool settings (delta pager, staging/blame, merge)                        | `~/.config/git/config`           |
| `ghostty`    | Ghostty terminal config (Solarized Light, block cursor, cursor trail shader) | `~/.config/ghostty/`             |
| `hunk`       | hunk diff viewer config (Solarized Light theme, side-by-side)                | `~/.config/hunk/`                |
| `nvim`       | Neovim config (LazyVim, LSP, plugins)                                        | `~/.config/nvim/`                |
| `terminator` | Terminal emulator (Solarized theme, JetBrainsMono Nerd Font)                 | `~/.config/terminator/`          |
| `tmux`       | Tmux config, gitmux, CI status script                                        | `~/.tmux.conf`, `~/.gitmux.conf` |
| `yazi`       | Yazi file manager config with zoxide plugin                                  | `~/.config/yazi/`                |

### System dependencies

Installed via `bootstrap.sh` (apt + `~/.local/bin`):

- [bat](https://github.com/sharkdp/bat) — cat with syntax highlighting
- [delta](https://github.com/dandavison/delta) — git diff pager with syntax highlighting
- [direnv](https://direnv.net/) — per-directory environment variables
- [fd](https://github.com/sharkdp/fd) — fast find (powers fzf file search)
- [fzf](https://github.com/junegunn/fzf) — fuzzy finder
- [Ghostty](https://ghostty.org/) — terminal emulator
- [gitmux](https://github.com/arl/gitmux) — git status in tmux
- [GNU Stow](https://www.gnu.org/software/stow/) — symlink manager
- [hunk](https://github.com/modem-dev/hunk) — interactive diff viewer (via `gd`/`gds` aliases)
- [JetBrainsMono Nerd Font](https://www.nerdfonts.com/) — terminal/editor font
- [jq](https://github.com/jqlang/jq) — JSON processor
- [lazydocker](https://github.com/jesseduffield/lazydocker) — terminal Docker UI
- [lazygit](https://github.com/jesseduffield/lazygit) — terminal git UI
- [Neovim](https://neovim.io/) — editor
- [ripgrep](https://github.com/BurntSushi/ripgrep) — fast recursive search
- [tmux](https://github.com/tmux/tmux) — terminal multiplexer
- [tree](https://gitlab.com/OldManProgrammer/unix-tree) — directory listing utility
- [yazi](https://github.com/sxyazi/yazi) — terminal file manager
- [zoxide](https://github.com/ajeetdsouza/zoxide) — smarter cd

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
# Edit directly in dotfiles — symlink means it takes effect immediately
vim ~/dotfiles/bash/.bashrc.d/my-feature.bash
```

### Add private/work-specific aliases

Create `~/.bashrc.d/local.bash` (not tracked in git):

```bash
vim ~/.bashrc.d/local.bash
```

### Machine-specific Claude Code settings

The dotfiles ship a minimal `~/.claude/settings.json` (hooks + statusline only).
Machine-specific preferences — `theme`, `voiceEnabled`, `effortLevel`,
`enabledPlugins`, `skipAutoPermissionPrompt`, etc. — belong in
`~/.claude/settings.local.json`, which Claude Code merges on top of the shipped
file and which stays out of the dotfiles repo.

If you already have a `~/.claude/settings.json` with preferences when you run
`bootstrap.sh`, it gets backed up to `~/.claude/settings.json.pre-dotfiles`.
Migrate the keys you want to keep:

```bash
jq '{autoCompactEnabled, theme, voiceEnabled, effortLevel, enabledPlugins, skipAutoPermissionPrompt}' \
  ~/.claude/settings.json.pre-dotfiles > ~/.claude/settings.local.json
```

### Edit a config

Edit files directly in `~/dotfiles/` — the symlinks mean changes take effect immediately.

### Stow commands

```bash
stow <package>       # Link a package
stow -D <package>    # Unlink a package
stow -R <package>    # Re-link (unlink + link)
```

## Directory structure

```
dotfiles/
├── bash/.bashrc.d/
│   ├── 00-path.bash          # PATH (loads first)
│   ├── aliases.bash           # shell options, git/docker aliases
│   ├── direnv.bash
│   ├── fzf.bash               # fzf + fd + Solarized opts, bat preview, `rfv` live-grep
│   ├── python.bash            # pyright-init function
│   ├── ssh-agent.bash
│   ├── tools.bash
│   └── zoxide.bash
├── claude/.claude/
│   ├── hooks/
│   │   ├── on-notification.sh   # 🔴 Claude needs your reply
│   │   ├── on-prompt-submit.sh  # clears session indicator
│   │   └── on-stop.sh           # 🟢 Claude finished
│   ├── settings.json            # hooks + statusLine wiring (portable, uses $HOME)
│   └── statusline-command.sh
├── git/.config/git/config         # delta pager, staging/blame, merge settings
├── nvim/.config/nvim/
│   ├── init.lua
│   └── lua/{config,plugins}/
├── tmux/
│   ├── .tmux.conf
│   ├── .gitmux.conf
│   └── .local/bin/tmux-gitlab.sh
├── ghostty/.config/ghostty/
│   ├── config
│   └── shaders/                 # vendored cursor trail shaders
├── yazi/.config/yazi/
├── terminator/.config/terminator/config
├── bat/.config/bat/config
├── hunk/.config/hunk/config.toml
├── bootstrap.sh
├── test/bootstrap-fresh.sh    # docker smoke test (fresh Ubuntu 24.04)
├── CHEATSHEET.md
└── README.md
```

## Notes

- **System `.bashrc` is never overwritten.** All customizations live in `~/.bashrc.d/*.bash` and are sourced from the system `.bashrc`. The bootstrap script appends the sourcing loop with a backup.
- **No personal info in repo.** Work-specific aliases go in `~/.bashrc.d/local.bash` (not tracked).
- **Neovim plugins**: Managed by lazy.nvim. `lazy-lock.json` pins plugin versions — commit it to keep installs reproducible.
- **Python venvs**: direnv auto-activates `.venv` per project directory.
- **Idempotent**: `bootstrap.sh` is safe to re-run — it skips what's already installed.
- **Smoke test**: `test/bootstrap-fresh.sh` runs the bootstrap in a clean Ubuntu 24.04 Docker container and verifies binaries, symlinks, and idempotency. Run before bumping pinned versions or touching bootstrap.
