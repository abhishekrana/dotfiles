# Changelog

Notable changes to these dotfiles, in [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) spirit and versioned with
[SemVer](https://semver.org/).

This repo stays on **0.x** by choice. Pre-1.0 SemVer puts the breaking signal on the MINOR, so a release that needs
manual steps on the machine - a re-login, a re-stow, a GNOME shortcut, a systemd unit - bumps **0.x** and says which;
PATCH is for everything else.

## [0.2.0] - 2026-08-20

### Added

- **dictate**: Make dictate+send the row's one highlight (c1bff82)
- **dictate**: Prompt for "skill" (5e7ffa6)
- **bash**: Gwtm merges, never rebases (e88578a)
- **tmux**: Show whether the merge request is open, merged or closed (13e1415)
- **dictate**: A dictate+send chip, and a footer that reads as a toolbar (ccf7d08)
- **dictate**: Drop gitleaks from the prompt (a95d5cc)
- **dictate**: The prompt vocabulary you actually speak, in the order that measured clean (f48424d)
- **dictate**: Pick the backend from what is installed, not the environment (634121f)
- **dictate**: --test reports its backend and cost (5f74e89)
- **dictate**: Run Whisper on the GPU, behind a backend switch (cc6eacb)
- **bash**: Gwtm finishes the job instead of aborting (6cbbfe3)
- **needs manual steps** - **leaf**: Add leaf markdown previewer, themed by the switcher (1cc116b)
- **needs manual steps** - **tmux**: Pane rails, and a diff pane that follows the agent (c39e18e)
- **agentbar**: Stamp the worktree an agent is writing in (3bf0092)

### Changed

- **tmux**: Drop the issue/MR words, and a fixed-width CI glyph (a855eef)
- **tmux**: Drop gitmux, and abbreviate the footer sha to 7 (d915c2f)
- **dictate**: Name the backends for the hardware, and stop --test eating a live dictation (27ebf92)
- **bash**: Cut gwtm back to the one rule that matters (ab938fd)

### Documentation

- State the rules in CLAUDE.md, drop the narration (c87b5bf)
- **dictate**: State the chip rule, drop the prose around it (00f3978)

### Fixed

- **agentbar**: Drop the agent's workdir at a session boundary (ba62426)
- **dictate**: Light up only the chip you clicked (623c651)
- **dictate**: The GPU is an optimisation, never a dependency (c9c99d3)
- **dictate**: Drop committed **pycache** and ignore it (c4524d5)
- **dictate**: Never share a whisper-server, never orphan one (80a0229)
- **install**: Make the Vulkan step work on a fresh Ubuntu (0baa2fa)
- **tmux**: Name a session by its own worktree, and use the whole popup (18a1d1a)
- **tmux**: Keep the diff pane where you pointed it (a8081bd)
- **agentbar**: Never move the agent's workdir on a cwd change (1cfcc1f)
- **tmux**: Draw both rail zones in the accent (fec9630)
- **tmux**: Size the rail's branch cap to the pane (f9ae1c4)
- **tmux**: Leave the sidebar's rail blank (666db84)

### Maintenance

- **install**: Bump pinned tool versions (fea1ebd)
- **lint**: Ignore **pycache** so importing dictate cannot dirty the tree (1a2a138)
- **claude**: Pin the model to opus[1m] (1da0a78)

### Performance

- **tmux**: Make the rail's memo hit fork nothing (8ce9356)
- **tmux**: Halve the GitLab segment's cost on every status redraw (e2c80b3)
- **dictate**: Default the GPU backend to small.en, and prompt only for words heard failing (e25677e)

## [0.1.4] - 2026-08-05

### Fixed

- **tmux**: Match a picker row without closing the pipe early (225fbe6)

### Tests

- **task**: Give the CI mirror the runner's gawk and the shell gates (0d5f57e)

## [0.1.3] - 2026-08-05

Tagged but never published: the release gate failed on the runner (see 0.1.4). Everything below ships in 0.1.4.

### Added

- **theme**: Drive the session popup's colors from the palette (7a241b9)
- **hunk**: Default to the stacked layout with wrapped lines (d87df95)
- **tmux**: Fit every band in the session popup and space them apart (60df65f)
- **agentbar**: Name the middle band, so all three read the same (702d8c9)
- **tmux**: Walk the agent bar's order with Alt-h/Alt-l, pin from the picker (f7d6968)
- **agentbar**: Publish the sidebar's session order as order/next/prev/pin (016d25c)

### Fixed

- **agentbar**: Draw the working count in the working colour (c3e85e2)
- **tmux**: One agent-state language for the picker and its preview (34aefd4)

## [0.1.2] - 2026-07-28

### Added

- **tmux**: Reset the UI to its defaults on prefix + R (018d8f1)
- **agentbar**: Hold the sidebar width, restart one sidebar in place (ae4002e)
- **clip**: Verify the clipboard actually took the copy (8b772c8)
- **agentbar**: Report theme drift in doctor (abafed0)
- **task**: Add a portability report (baaaf57)
- **clip**: Add a clipboard wrapper with tracing, covering wayland, x11 and macos (e887baa)
- **install**: Detect the platform and lift release-asset naming into one block (13c6bad)

### Build

- **install**: Add xclip so clip has an X11 clipboard backend (82c13c6)

### Changed

- **tmux**: Route every copy path through clip (1088feb)

### Documentation

- Trim CLAUDE.md to what carries weight (e869c27)
- Describe the UI reset and the sidebar width pin (54f7b7e)
- State the platform support policy (ef7f652)

### Fixed

- **bash**: Stop forcing TERM, which hid Ghostty from tmux (7706c06)
- **tmux**: Route every copy trigger through copy-command (dcb9b89)

### Maintenance

- **task**: Stop tracking the task build cache (3a93b4f)

### Tests

- **clip**: Guard the copy path against silent regressions (2c78ac1)
- **ci**: Add `task check-ci` to run the suite as the runner sees it (54d120a)

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
