# CLAUDE.md

## Project overview

Personal dotfiles managed with GNU Stow on Ubuntu 24.04.

## Stow packages

- `bash/` → `~/.bashrc.d/` (shell customizations)
- `nvim/` → `~/.config/nvim/` (LazyVim config)
- `tmux/` → `~/.tmux.conf`, `~/.gitmux.conf`, `~/.local/bin/tmux-ci-status.sh`
- `terminator/` → `~/.config/terminator/`
- `bat/` → `~/.config/bat/`

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

## Commits

- Do not add `Co-Authored-By` lines to commit messages

## Formatting

- Format `README.md` with `npx prettier --write README.md`
