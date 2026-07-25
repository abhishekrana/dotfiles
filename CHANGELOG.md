# Changelog

Notable changes to these dotfiles, in [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) spirit and versioned with
[SemVer](https://semver.org/).

A **MAJOR** bump means the release needs manual steps on the machine - a re-login, a re-stow, a GNOME shortcut, a
systemd unit - and the notes say which.

## [0.1.0] - unreleased

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
