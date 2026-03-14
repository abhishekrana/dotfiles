# dotfiles

Personal development environment managed with [GNU Stow](https://www.gnu.org/software/stow/).

## What's included

### Configs (stow packages)

| Package | Description | Target |
|---------|-------------|--------|
| `bash` | Shell customizations, aliases, direnv/fzf/zoxide hooks, venv prompt | `~/.bashrc.d/` |
| `nvim` | Neovim config (LazyVim, LSP, plugins) | `~/.config/nvim/` |
| `tmux` | Tmux config, gitmux, CI status script | `~/.tmux.conf`, `~/.gitmux.conf` |
| `terminator` | Terminal emulator (Solarized theme, JetBrainsMono Nerd Font) | `~/.config/terminator/` |
| `bat` | Syntax highlighter theme | `~/.config/bat/` |

### System dependencies

Installed via `bootstrap.sh` (apt + `~/.local/bin`):

- [Neovim](https://neovim.io/) — editor
- [tmux](https://github.com/tmux/tmux) — terminal multiplexer
- [direnv](https://direnv.net/) — per-directory environment variables
- [fzf](https://github.com/junegunn/fzf) — fuzzy finder
- [zoxide](https://github.com/ajeetdsouza/zoxide) — smarter cd
- [gitmux](https://github.com/arl/gitmux) — git status in tmux
- [Terminator](https://gnome-terminator.org/) — terminal emulator
- [JetBrainsMono Nerd Font](https://www.nerdfonts.com/) — terminal/editor font
- [GNU Stow](https://www.gnu.org/software/stow/) — symlink manager
- [bat](https://github.com/sharkdp/bat) — cat with syntax highlighting

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

### 3. Set up git identity

```bash
# Create your local git config (not tracked in repo)
cat > ~/.gitconfig.local << 'EOF'
[user]
    name = Your Name
    email = your@email.com
EOF
```

### 4. Restart your shell

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
vim ~/dotfiles/bash/.bashrc.d/90-my-feature.bash
```

### Add private/work-specific aliases

Create `~/.bashrc.d/local.bash` (not tracked in git):

```bash
vim ~/.bashrc.d/local.bash
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
├── bash/
│   └── .bashrc.d/
│       ├── 00-path.bash
│       ├── 10-prompt.bash
│       ├── 20-aliases.bash
│       ├── 30-completions.bash
│       ├── 40-direnv.bash
│       ├── 50-fzf.bash
│       ├── 60-zoxide.bash
│       ├── 70-ssh-agent.bash
│       └── 80-tools.bash
├── nvim/
│   └── .config/
│       └── nvim/
│           ├── init.lua
│           ├── lazy-lock.json
│           ├── lazyvim.json
│           ├── stylua.toml
│           └── lua/
│               ├── config/
│               │   ├── autocmds.lua
│               │   ├── keymaps.lua
│               │   ├── lazy.lua
│               │   └── options.lua
│               └── plugins/
│                   ├── colorscheme.lua
│                   ├── diffview.lua
│                   ├── example.lua
│                   ├── indent.lua
│                   └── python.lua
├── tmux/
│   ├── .tmux.conf
│   ├── .gitmux.conf
│   └── .local/
│       └── bin/
│           └── tmux-ci-status.sh
├── terminator/
│   └── .config/
│       └── terminator/
│           └── config
├── bat/
│   └── .config/
│       └── bat/
│           └── config
├── bootstrap.sh
└── README.md
```

## Notes

- **System `.bashrc` is never overwritten.** All customizations live in `~/.bashrc.d/*.bash` and are sourced from the system `.bashrc`. The bootstrap script appends the sourcing loop with a backup.
- **No personal info in repo.** Git identity goes in `~/.gitconfig.local`. Work-specific aliases go in `~/.bashrc.d/local.bash`. Neither is tracked.
- **Neovim plugins**: Managed by lazy.nvim. `lazy-lock.json` pins plugin versions — commit it to keep installs reproducible.
- **Python venvs**: direnv auto-activates `.venv` per project directory.
- **Idempotent**: `bootstrap.sh` is safe to re-run — it skips what's already installed.
