# Changelog

Notable changes to these dotfiles, in [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) spirit and versioned with
[SemVer](https://semver.org/).

This repo stays on **0.x** by choice. Pre-1.0 SemVer puts the breaking signal on the MINOR, so a release that needs
manual steps on the machine - a re-login, a re-stow, a GNOME shortcut, a systemd unit - bumps **0.x** and says which;
PATCH is for everything else.

## [0.1.1] - 2026-07-25

### Changed

- **install**: Extract install.sh so CI installs the way a machine does (d3b48ad)

### Fixed

- **agentbar**: Keep CI out of the e2e environment so the sidebar renders colour (a0c64b0)
- **agentbar**: Force a UTF-8 locale on tmux calls that parse tabs (8c37c3d)
- **ci**: Build the pinned tmux instead of using Ubuntu's 3.4 (0482b8e)

## [0.1.0] - 2026-07-25

First tagged snapshot of a repo that had been running untagged since 2026-03-14. Written by hand: the history predates
Conventional Commits, so `git-cliff` has nothing to group. Generated notes take over from 0.2.0.

### Added

- **Stow packages** for bash, bat, claude, dictate, ghostty, git, hunk, nvim, theme, tmux, trace and yazi, installed by
  an idempotent `bootstrap.sh` that pins every tool version.
- **agentbar** (`apps/agentbar`) - a tmux sidebar showing every Claude Code agent across all sessions, driven by Claude
  Code lifecycle hooks rather than screen scraping. Go + Bubble Tea, with a `doctor` self-check and a 26-test e2e suite
  against throwaway tmux servers.
- **dictate** - toggle-key local speech-to-text into tmux via faster-whisper, CPU-only, with a lazy model server and a
  silence watcher.
- **theme** - one switcher that re-skins the whole terminal stack across four flavors from `design/palette.toml`.
- **dotfiles-trace** - one always-on, size-capped trace log shared by tmux, the sidebar, the hooks and dictate, so a
  misbehaving click or a stale state has evidence waiting.
- **Notes vaults** - `vault-template/` seeds two independent private vaults with guardrail hooks, an integrity check and
  a secrets pre-commit hook.
- **A release process** - `task check` as the gate (shellcheck, ruff, prettier, gitleaks, tests), CI on every push, and
  a tag-triggered GitHub Release with notes generated from Conventional Commits.
