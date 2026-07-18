# CLAUDE.md

## Project overview

Personal dotfiles managed with GNU Stow on Ubuntu 24.04.

## Stow packages

- `bash/` → `~/.bashrc.d/` (shell customizations)
- `bat/` → `~/.config/bat/`
- `claude/` → `~/.claude/settings.json`, `~/.claude/statusline-command.sh` (settings.json wires the agentbar hook on every Claude lifecycle event + the statusLine; agent state for both the sidebar and the `M-;` session picker comes from that plugin's `@agent_*` pane options, not local hook scripts. Note: Claude Code does NOT load a user-level `~/.claude/settings.local.json` — hooks/settings there are silently ignored; anything that must take effect goes in `settings.json`)
- `dictate/` → `~/.local/bin/dictate` (toggle-key local Whisper dictation into tmux; opt-in)
- `ghostty/` → `~/.config/ghostty/` (Ghostty terminal config)
- `git/` → `~/.config/git/config` (delta pager, merge settings)
- `hunk/` → `~/.config/hunk/` (hunk diff viewer config, Solarized Light theme)
- `nvim/` → `~/.config/nvim/` (LazyVim config)
- `theme/` → `~/.local/bin/theme` (theme switcher; re-skins the terminal stack across the four flavors from `design/palette.toml`, writing per-tool files into `~/.config/theme/`)
- `tmux/` → `~/.tmux.conf`, `~/.gitmux.conf`, `~/.local/bin/` scripts (`tmux-gitlab.sh` GitLab status, session picker, resurrect guard, yank)
- `yazi/` → `~/.config/yazi/` (yazi file manager config)

## Apps (built from source)

Buildable projects live under `apps/` — these are **not** stow packages and are never passed to `stow`. Each is self-contained with its own `Makefile` exposing a uniform `build` target, so `bootstrap.sh` builds any language the same way: `build_apps` loops over `apps/*/` running `make build`, and the toolchain gets a pinned `install_*` step (e.g. `install_go`). Add more apps by dropping a project with a `Makefile` under `apps/`.

- `apps/agentbar/` → Go tmux plugin (the Claude agent sidebar). Loaded by a `run-shell` line at the end of `tmux/.tmux.conf`, so it builds and runs straight from the repo. The Claude lifecycle hooks in `claude/.claude/settings.json` invoke its binary at `$HOME/dotfiles/apps/agentbar/bin/agentbar`. It has its own nested `CLAUDE.md` — read that before touching the code.

## Rules

- **Never commit personal info**: no names, emails, IP addresses, work-specific paths, or company references (alpha-ignis, alpha-collection, etc.)
- **Audit before committing**: `git diff --cached | grep -iE '10\.\d+\.\d+|172\.\d+|abhishek|alpha'` must return empty
- **Only track customizations**: don't add stock Ubuntu defaults (prompt, bash-completion, color aliases) — those belong in the system `.bashrc`
- **Prefer `~/.local/bin`** for tool installations over system-wide installs
- **Keep it simple**: no unnecessary abstractions, no over-engineering

## Conventions

- Bash files in `.bashrc.d/` use `.bash` extension
- Only `00-path.bash` has a numeric prefix (must load first for PATH); all other files use plain names
- Each tool init file guards with `command -v tool &>/dev/null || return`
- Private/work-specific config goes in `~/.bashrc.d/local.bash` (not tracked)
- `bootstrap.sh` must be idempotent (safe to re-run)
- Keep lists alphabetically sorted (stow packages, apt packages, pinned versions, bootstrap calls, docs)

## Deploy

When I say "deploy": **first commit and push, then make it live** on the running system.

1. **Commit & push** to `main` (run the secrets audit first).
2. **Make it live:**
   - **tmux** (`tmux/.tmux.conf`): `tmux source-file ~/.tmux.conf`. One server is shared by all sessions, so a single reload updates every existing session at once.
   - **Stowed scripts** (symlinks — `dictate/`, `bash/`, etc.): live the moment the repo file is saved; no step needed.
   - **New stow package or file**: `cd ~/dotfiles && stow <pkg>`, then reload the relevant tool.
   - **`apps/agentbar`** (Go): `make -C ~/dotfiles/apps/agentbar build`, then reload tmux (`tmux source-file ~/.tmux.conf`). The sidebar is a long-lived process, so restarting it is a manual step you must list: **`prefix + e` twice**. Hook-path edits to the stowed `settings.json` take effect on the next agent lifecycle event.
3. Always list any steps I must run by hand — things that can't be scripted (re-login, `gsettings`/GNOME shortcut install, `systemctl --user …`, opening a fresh shell).

## Commits

- Do not add `Co-Authored-By` lines to commit messages

## Formatting

- Format markdown files with `npx prettier --write <file>.md`
- Always use a plain hyphen (`-`), never em or en dashes
