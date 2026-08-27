# dotfiles

Development environment for Ubuntu 24.04 - shell, tmux, Neovim, terminal, and CLI tooling - managed with
[GNU Stow](https://www.gnu.org/software/stow/) and reproducible on a fresh machine from a single `bootstrap.sh`.

> **Built for [Ghostty](https://ghostty.org/).** The tmux, theme switcher, and shaders assume it. `bootstrap.sh`
> installs it; use it as your terminal. Other terminals work but aren't themed.

## Contents

- [What's included](#whats-included)
- [Platform support](#platform-support)
- [Setup on a new machine](#setup-on-a-new-machine)
- [Usage](#usage)
- [Managing configs](#managing-configs)
- [Notes](#notes)
- [License](#license)

## What's included

### Configs (stow packages)

| Package   | Description                                                                                                                 | Target                        |
| --------- | --------------------------------------------------------------------------------------------------------------------------- | ----------------------------- |
| `bash`    | Shell customizations, aliases, direnv/fzf/zoxide hooks, vi mode                                                             | `~/.bashrc.d/`                |
| `bat`     | Syntax highlighter theme                                                                                                    | `~/.config/bat/`              |
| `claude`  | Claude Code settings.json (agentbar hook, statusLine, permissions), statusline script, and shared skills (`managing-vault`) | `~/.claude/`                  |
| `clip`    | Copy stdin to the clipboard - picks wl-copy (Wayland), xclip (X11) or pbcopy (macOS)                                        | `~/.local/bin/clip`           |
| `dictate` | Toggle-key local speech-to-text into tmux (faster-whisper on CPU, or whisper.cpp on the GPU)                                | `~/.local/bin/`               |
| `ghostty` | Ghostty terminal config (Solarized Light, block cursor, cursor trail shader)                                                | `~/.config/ghostty/`          |
| `git`     | Git tool settings (delta pager, staging/blame, merge)                                                                       | `~/.config/git/config`        |
| `hunk`    | hunk diff viewer config (stacked/unified, themed by the switcher)                                                           | `~/.config/hunk/`             |
| `leaf`    | leaf markdown previewer config, carrying a full Solarized Light palette (leaf ships only `solarized-dark`)                  | `~/.config/leaf/`             |
| `nvim`    | Neovim config (LazyVim, LSP, plugins)                                                                                       | `~/.config/nvim/`             |
| `theme`   | Theme switcher - re-skins the terminal stack across four flavors (`design/palette.toml`)                                    | `~/.local/bin/theme`          |
| `tmux`    | Tmux config, CI status script, `prefix + R` UI reset, agent-following diff pane                                             | `~/.tmux.conf`                |
| `trace`   | Shared always-on trace log for the tmux/agent stack                                                                         | `~/.local/bin/dotfiles-trace` |
| `yazi`    | Yazi file manager config with zoxide plugin                                                                                 | `~/.config/yazi/`             |

### Apps (built from source)

Binaries built from source under `apps/` (not stow packages). Each has a `Makefile` with a `make build` target;
`bootstrap.sh` installs the toolchain and builds them.

| App        | Description                                                            | Language |
| ---------- | ---------------------------------------------------------------------- | -------- |
| `agentbar` | tmux sidebar showing every Claude Code agent's state across sessions   | Go       |
| `workdesk` | GitLab work inbox: merge requests, issues, todos and agents in a popup | Go       |

Both binaries build from the one module under `apps/agentbar/`. `workdesk` is linked into `~/.local/bin`; `agentbar` is
invoked by absolute path from tmux and the Claude hooks.

The sidebar loads from here via a `run-shell` line in `tmux/.tmux.conf`. `prefix + R` picks up changes: it reloads the
config, rebuilds the binary if the source moved, and restarts that session's sidebar.

### System dependencies

Installed by `install.sh` (apt + `~/.local/bin`), which `bootstrap.sh` calls and CI reuses so both install the same
pinned versions:

- [bat](https://github.com/sharkdp/bat) - cat with syntax highlighting
- [delta](https://github.com/dandavison/delta) - git diff pager with syntax highlighting
- [direnv](https://direnv.net/) - per-directory environment variables
- [fd](https://github.com/sharkdp/fd) - fast find (powers fzf file search)
- [fzf](https://github.com/junegunn/fzf) - fuzzy finder
- [Ghostty](https://ghostty.org/) - terminal emulator
- [git-cliff](https://git-cliff.org/) - changelog and release notes from conventional commits
- [gitleaks](https://github.com/gitleaks/gitleaks) - secret scanning over the tree and history
- [GNU Stow](https://www.gnu.org/software/stow/) - symlink manager
- [Go](https://go.dev/) - toolchain for building `apps/` (agentbar)
- [hunk](https://github.com/modem-dev/hunk) - interactive diff viewer (via `gd`/`gds` aliases)
- [JetBrainsMono Nerd Font](https://www.nerdfonts.com/) - terminal/editor font
- [jq](https://github.com/jqlang/jq) - JSON processor
- [lazydocker](https://github.com/jesseduffield/lazydocker) - terminal Docker UI
- [lazygit](https://github.com/jesseduffield/lazygit) - terminal git UI
- [leaf](https://github.com/rivolink/leaf) - terminal markdown previewer
- [Neovim](https://neovim.io/) - editor
- [ripgrep](https://github.com/BurntSushi/ripgrep) - fast recursive search
- [ruff](https://docs.astral.sh/ruff/) - Python linter (gates the `dictate` script)
- [shellcheck](https://www.shellcheck.net/) - shell linter (gates every script here)
- [shfmt](https://github.com/mvdan/sh) - finds shell files by shebang for the lint gate
- [Task](https://taskfile.dev/) - task runner for this repo's `Taskfile.yml`
- [tmux](https://github.com/tmux/tmux) - terminal multiplexer (pinned, built from source: 24.04 ships 3.4)
- [tree](https://gitlab.com/OldManProgrammer/unix-tree) - directory listing utility
- [yazi](https://github.com/sxyazi/yazi) - terminal file manager
- [zoxide](https://github.com/ajeetdsouza/zoxide) - smarter cd

## Platform support

**Ubuntu 24.04 is the supported platform** - it is what this repo is developed and used on. `install.sh` assumes `apt`,
Linux release assets and a GNU userland, so it refuses to run anywhere else rather than failing half way through.

macOS is **not supported**. The configs are largely portable and each platform-varying piece is kept in one place, so
adding it would be tractable rather than a rewrite:

| Layer                                               | Today               | Would need                                                              |
| --------------------------------------------------- | ------------------- | ----------------------------------------------------------------------- |
| Configs (nvim, bat, git, yazi, hunk, leaf, ghostty) | portable            | nothing                                                                 |
| Clipboard                                           | portable via `clip` | nothing - it already picks `pbcopy`                                     |
| Shell config (`bash/.bashrc.d/`)                    | bash only           | zsh guards, since macOS defaults to zsh                                 |
| Release assets (`install.sh`)                       | one `case` arm      | one more arm - no install function changes                              |
| System packages                                     | `apt`               | Homebrew; `brew bundle` cannot pin versions the way `apt` does          |
| `agentbar`                                          | Go core is portable | `osascript` notify fallback, BSD `script` in the e2e suite              |
| `trace`                                             | Linux only          | `date '+%3N'`, `date -d`, `stat -c` and `flock` have no BSD equivalents |
| `dictate`                                           | Linux only          | out of scope - different audio stack, unscriptable mic permission       |

`task portability` prints every Linux-only primitive and the files holding it - the inventory another platform would
have to answer for.

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

Neovim plugins will auto-install on first launch via lazy.nvim. Run `prefix + I` in tmux to install tmux plugins.

### 4. Switch to Ghostty

Bootstrap installed it - open Ghostty and use it going forward; the configs are tuned for it. (`bootstrap.sh` itself
runs from any terminal.)

## Usage

Day-to-day keybindings and commands - shell aliases, tmux, Neovim (LazyVim), hunk, Ghostty - live in
**[CHEATSHEET.md](CHEATSHEET.md)** (also viewable in the terminal via the `cheat` alias). Re-skin the whole terminal
stack with the ⛭ chip at the far right of the status bar, or `theme <flavor>` (`solarized-light` · `solarized-dark` ·
`catppuccin-latte` · `catppuccin-mocha`); see [`design/theme-switcher.md`](design/theme-switcher.md).

## Development

`task` lists everything this repo can do. `task check` is the gate CI runs on every push - shellcheck, ruff, prettier,
gitleaks and the agentbar test suite - and `task check-ci` reruns that suite in a container mirroring the runner (older
tmux, no `LANG`, `CI` set). Commits follow [Conventional Commits](https://www.conventionalcommits.org/), and releases
are cut by pushing an annotated `v*` tag: `.github/workflows/release.yml` re-runs the gate, runs the container
fresh-install test, and publishes a GitHub Release with notes generated from the commit history. See
[CHANGELOG.md](CHANGELOG.md) and the "Releasing" section of [CLAUDE.md](CLAUDE.md).

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

The committed `~/.claude/settings.json` is the full baseline - hooks, `statusLine`, `permissions`, plugins, and prefs.
Claude Code has no user-level `settings.local.json` (only a project's is read), and the file is a stowed symlink, so
runtime `/config` edits write into this repo: commit what you want to keep, or `git checkout` to discard. An existing
file is backed up to `*.pre-dotfiles` on first bootstrap.

### Stow commands

```bash
stow <package>       # Link a package
stow -D <package>    # Unlink a package
stow -R <package>    # Re-link (unlink + link)
```

## Notes

- **System `.bashrc` is never overwritten** - customizations live in `~/.bashrc.d/*.bash`, sourced by a loop
  `bootstrap.sh` appends (with a backup).
- **Private/work aliases** go in `~/.bashrc.d/local.bash` (not tracked).
- **Notes vaults**: `bootstrap.sh` seeds two plain-markdown PARA vault skeletons (`~/vaults/personal`, `~/vaults/work`)
  from `vault-template/`, each with an agent layer - the global `managing-vault` skill adds and maintains notes, and a
  deterministic `.claude/vault-check.sh` integrity gate runs in the git pre-commit hook (which also blocks secrets). It
  prints optional git-remote wiring steps for any not yet synced; each vault's contents live in its own private repo,
  never here.
- **Neovim plugins**: `lazy-lock.json` pins versions - commit it to keep installs reproducible.
- **Python venvs**: direnv auto-activates `.venv` per directory.
- **Idempotent**: `bootstrap.sh` is safe to re-run (skips what's installed).
- **Smoke test**: `task fresh` runs bootstrap in a clean Ubuntu 24.04 container (checks binaries, symlinks,
  idempotency); run before touching `bootstrap.sh`. The release workflow runs it too, so no release ships without it.

## License

[MIT](LICENSE).
