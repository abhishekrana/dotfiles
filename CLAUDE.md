# CLAUDE.md

## Project overview

Personal dotfiles managed with GNU Stow on Ubuntu 24.04.

## Stow packages

- `bash/` → `~/.bashrc.d/` (shell customizations)
- `bat/` → `~/.config/bat/`
- `claude/` → `~/.claude/hooks/`, `~/.claude/settings.json`, `~/.claude/statusline-command.sh` (note: Claude Code does NOT load a user-level `~/.claude/settings.local.json` — hooks/settings there are silently ignored; anything that must take effect goes in `settings.json`)
- `dictate/` → `~/.local/bin/dictate`, `~/.local/bin/dictate-setup` (push-to-talk local Whisper dictation; opt-in, run `dictate-setup` once)
- `git/` → `~/.config/git/config` (delta pager, merge settings)
- `ghostty/` → `~/.config/ghostty/` (Ghostty terminal config)
- `hunk/` → `~/.config/hunk/` (hunk diff viewer config, Solarized Light theme)
- `nvim/` → `~/.config/nvim/` (LazyVim config)
- `terminator/` → `~/.config/terminator/`
- `tmux/` → `~/.tmux.conf`, `~/.gitmux.conf`, `~/.local/bin/tmux-gitlab.sh` (GitLab issue/MR/CI status segments)
- `yazi/` → `~/.config/yazi/` (yazi file manager config)

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

## Commits

- Do not add `Co-Authored-By` lines to commit messages

## Formatting

- Format markdown files with `npx prettier --write <file>.md`
