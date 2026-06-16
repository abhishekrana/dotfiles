# CLAUDE.md

## Project overview

Personal dotfiles managed with GNU Stow. Primary target: Ubuntu 24.04. Also supported: macOS (Apple Silicon and Intel).

## Stow packages

- `bash/` → `~/.bashrc.d/` (shell customizations)
- `bat/` → `~/.config/bat/`
- `claude/` → `~/.claude/hooks/`, `~/.claude/settings.json`, `~/.claude/statusline-command.sh` (machine-specific prefs go in `~/.claude/settings.local.json`, not tracked)
- `claude-indicator/` → `~/.local/bin/claude-indicator`, `~/.config/autostart/` (Linux only — GNOME top-bar indicator for Claude Code notifications)
- `git/` → `~/.config/git/config` (delta pager, merge settings)
- `ghostty/` → `~/.config/ghostty/` (Ghostty terminal config)
- `nvim/` → `~/.config/nvim/` (LazyVim config)
- `screenshot-watcher/` → `~/.local/bin/screenshot-watcher`, `~/.config/autostart/` (Linux only — auto-copy screenshots to clipboard)
- `terminator/` → `~/.config/terminator/`
- `tmux/` → `~/.tmux.conf`, `~/.gitmux.conf`, `~/.local/bin/tmux-ci-status.sh`
- `yazi/` → `~/.config/yazi/` (yazi file manager config)

Linux-only packages (`claude-indicator`, `screenshot-watcher`, `terminator`) are skipped automatically by `bootstrap.sh` on macOS.

## Rules

- **Never commit personal info**: no names, emails, IP addresses, work-specific paths, or company references (alpha-ignis, alpha-collection, etc.)
- **Audit before committing**: `git diff --cached | grep -iE '10\.\d+\.\d+|172\.\d+|abhishek|alpha'` must return empty
- **Only track customizations**: don't add stock Ubuntu/macOS defaults (prompt, bash-completion, color aliases) — those belong in the system `.bashrc` / `.zshrc`
- **Prefer `~/.local/bin`** for tool installations over system-wide installs
- **Keep it simple**: no unnecessary abstractions, no over-engineering
- **Cross-platform**: guard OS-specific code with `is_linux` / `is_macos` helpers in `bootstrap.sh`; in shell config, branch on `$OSTYPE` (`linux-gnu*` vs `darwin*`)

## Conventions

- Bash files in `.bashrc.d/` use `.bash` extension
- Only `00-path.bash` has a numeric prefix (must load first for PATH); all other files use plain names
- Each tool init file guards with `command -v tool &>/dev/null || return`
- Private/work-specific config goes in `~/.bashrc.d/local.bash` (not tracked)
- `bootstrap.sh` must be idempotent (safe to re-run)
- Keep lists alphabetically sorted (stow packages, apt/brew packages, pinned versions, bootstrap calls, docs)

## Commits

- Do not add `Co-Authored-By` lines to commit messages

## Formatting

- Format markdown files with `npx prettier --write <file>.md`
