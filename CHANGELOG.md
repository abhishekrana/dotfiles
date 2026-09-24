# Changelog

Notable changes to these dotfiles, in [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) spirit and versioned with
[SemVer](https://semver.org/).

This repo stays on **0.x** by choice. Pre-1.0 SemVer puts the breaking signal on the MINOR, so a release that needs
manual steps on the machine - a re-login, a re-stow, a GNOME shortcut, a systemd unit - bumps **0.x** and says which;
PATCH is for everything else.

## [0.3.0] - 2026-09-24

### Added

- **claude**: Light the dictation chip for a dictation from another machine (5de9862)
- **claude**: Count down to the weekly reset, as the 5-hour window does (9b02e05)
- **claude**: Show context and usage limits as meters in the status line (72d72ab)
- **herdr**: Draw the popup's buttons as soft blocks, and stop the toolbar jumping (bdedc7a)
- **herdr**: Draw the popup in Solarized's own colours and roles (167db57)
- **herdr**: Put the popup's own actions in a toolbar on its top line (cb79b4b)
- **herdr**: Refresh tabs in place, and open all three from the popup (a8c2f2e)
- **herdr**: Open each popup item in a herdr tab of its own (723d7be)
- **needs manual steps** - **herdr**: Show the ticket, MR and pipeline in three clickable columns (7c6242d)
- **needs manual steps** - **herdr**: Open the focused branch's MR, ticket or pipeline from a popup (e644aeb)
- **needs manual steps** - **herdr**: Show the focused branch's ticket, MR and pipeline in the tab bar (fc66da1)
- **needs manual steps** - **git**: Track the global gitignore (305ff44)
- **needs manual steps** - **vault**: One vault, seeded as memory planes (f3d989e)
- **claude**: Show the dictation state in the status line (8f39fc4)
- **claude**: Link the branch's issue and pipeline in the status line (7e6faa4)
- **claude**: Show the rate limit windows in the status line (4ac8c6e)
- **claude**: Allow the Slack CLI auth check (5910dba)
- **folio**: Select text with the mouse (c4e568c)
- **needs manual steps** - **dictate**: Move the shortcut to right Alt (119fc71)
- **needs manual steps** - **claude**: Record the agent's worktree without tmux (025b718)
- **needs manual steps** - **claude**: Name the worktree Claude is working in on the status line (aeae142)
- **install**: Follow the herdr-dictate pin, and never clobber a linked worktree (55d9748)
- **install**: Generate the herdr Claude skill (9d2d011)
- **install**: Herdr, its Claude integration and the dictation plugin (9068515)
- **bash**: Fzf previews markdown as a folio page (3cf3b23)
- **folio**: Live reload, reading positions, trace edges (492d090)
- **folio**: Navigation - outline, search, link hints and clicks, back, copy, edit, help (addbb03)
- **folio**: T cycles the theme, logs rotate daily (2d038a9)
- **folio**: --inline width follows fzf, then the terminal, default 120 (f1b4a80)
- **folio**: Nerd Font callout icons and image placeholder, icons live in the style (7c21519)
- **folio**: Nerd Font checkboxes for tasks in the GitHub style (c9a481c)
- **folio**: Syntax highlighting in code blocks (6f70be1)
- **folio**: No zebra tint on table rows in the GitHub style (f4bb86b)
- **folio**: Inline code is colour alone, no tint and no padding (21a868d)
- **folio**: No blank row after a ruled heading in the GitHub style (6da4356)
- **folio**: The column fills the pane; a fixed measure stays a style option (98d0958)
- **folio**: The column starts at the left, and align is a style switch (3d14ee3)
- **theme**: The switcher drives folio via FOLIO_THEME (35f31a6)
- **folio**: A markdown reader that reads like a page (fca3322)
- **workdesk**: The merge request sheet gets a clickable ◧ diff (28f0b3e)
- **design**: The changes hue reaches the Go theme (23d743b)
- **workdesk**: The inbox opens on a week, and w widens it (a8d9fc6)
- **workdesk**: A merge request's diff opens side by side (b98d50b)
- **needs manual steps** - **workdesk**: D reads the fetched diff, and its window closes (03a4044)
- **workdesk**: D reads a merge request's diff, F reads it with the files (f1a4efa)
- **design**: A float is its own role in the palette (a6fc909)
- **tmux**: The workdesk float stops looking like a pane (fb09ec8)
- **workdesk**: A link in the preview is clickable, references included (0bfc27f)
- **needs manual steps** - **workdesk**: A click looks at a row, and that is all it does (dc96ce6)
- **workdesk**: Render the ticket body as markdown, not as its source (896fbbf)
- **workdesk**: An issue preview carries the ticket, not a link to it (627a58d)
- **workdesk**: The issues list runs the other way up (8adfe6c)
- **workdesk**: The issues tab is the board, for both your accounts (cca31bc)
- **workdesk**: A sync says which leg it is on, and how far it got (caeb353)
- **needs manual steps** - **workdesk**: A float, not a popup, so the chip closes what it opens (4b626aa)
- **tmux**: ▤ workdesk leads the toolbar, and ⏎ send is gone (c0b331c)
- **workdesk**: A ✕ in the corner, and alt+n closes as well as opens (6e57668)
- **tmux**: A ▤ workdesk chip, and one command behind it and Alt+n (234026f)
- **workdesk**: Make the pointer do what the keys do (b0100c1)
- **needs manual steps** - **workdesk**: Replace the bash GitLab work tool with a Go binary (e55bc4e)
- **work**: Mirror the GitLab work you own into a local board (cd62d5e)
- **claude**: Set Claude Code's TUI theme to dark (fce34b4)
- **dictate**: Discover ssh sessions as hosts, and stop at the focused one (b3abec2)
- **dictate**: Route dictation to the tmux holding focus, local or remote (153a0ec)
- **agentbar**: P pins, a holds active, d sends dormant (269ca9d)
- **agentbar**: Sink quiet sessions into dormant on the clock (53e462d)
- **tmux**: Give the picker three columns, and its preview the agents (9df56d4)
- **needs manual steps** - **agentbar**: Nest agents under their session, two lines each (3e072e4)
- **agentbar**: Default to Claude's title, not the branch (f16833e)
- **agentbar**: Head rows with a title, move notify into the settings (4cde4e8)
- **dictate**: Bind Pause to dictate+send as well (9a7a1a7)
- **agentbar**: Head rows with Claude's session name behind a toggle (e5e477a)
- **theme**: Drive the settings dialogue with the mouse (1af0368)
- **needs manual steps** - **theme**: A settings dialogue on the footer gear (d221841)
- **needs manual steps** - **theme**: A settings chip in the footer, picking a theme applies it (a9284ad)
- **dictate**: Install by default, with the GPU backend when one is visible (b0b80fd)
- **needs manual steps** - **dictate**: Bind dictate+send to the Copilot key (e90507e)

### Build

- **install**: Herdr-dictate 0.2.1 (8af28a9)
- **install**: Pin herdr 0.9.1 (af753eb)
- **install**: Pin the dictation plugin to a release tag (be2ac71)
- **install**: Install a pinned Rust toolchain (8802769)

### Changed

- **herdr**: Drop Split, leaving Browser and New tab (beadcb2)
- **claude**: Drop the GitLab row from the status line (b6f8e85)
- **folio**: Drop link hints; a click follows a link (42a8393)
- **workdesk**: Order the view ring by the work, and define it once (9b6d971)
- **needs manual steps** - **workdesk**: Replace the fzf picker with a Bubble Tea UI (70677cb)
- **agentbar**: Drop the hand-placed band marker (4e538c5)
- **agentbar**: One key, one band - p pinned, a active, d dormant (1f0ac5d)
- **agentbar**: Generate the sidebar flavors from palette.toml (1435083)
- **theme**: Show every setting value, drop the submenu (c718485)

### Documentation

- **herdr**: The tab bar line runs every second (1468e51)
- **herdr**: Name alt+u as the chooser's key, prefix+u as its fallback (70e5144)
- Stop the theme header listing adapters it will fall behind (22df966)
- Correct the docs the code had moved past (14e0492)
- Audit the docs against the code, and scrub work data from fixtures (536186e)
- Split CLAUDE.md along how Claude actually loads it (5b187bd)
- **workdesk**: Correct the comments the fzf removal left behind (184805e)
- Keep work-forge data out of this repo (7718715)
- Rewrap markdown to the 120-column gate (3bbff45)
- Correct what the audit found stale (de449a5)

### Fixed

- **herdr**: Show the MR's diff live, with the file list in its tab (74be852)
- **herdr**: Open the chooser's entries on a plain click (b7c080a)
- **claude**: Drop the herdr SessionStart hook 0.9.1 superseded (8a108c4)
- **claude**: Grey the issue link, not blacken it (0e46b16)
- **claude**: Find the agent's pane without $TMUX_PANE (00b998b)
- **claude**: Name the subagent on its status line row (4e37a1b)
- **claude**: Stop dimming the branch on the status line (70882ef)
- **install**: Keep one machine-agnostic herdr hook in settings.json (631302e)
- **dictate**: Prompt herdr, which transcribes as header or harder (ed7a284)
- **folio**: Table cells wrap inside their column instead of clipping (9ea7f14)
- **folio**: List depth saturates, and the reader refuses a stdout that is not a terminal (3c6d772)
- **folio**: List depth, footnote numbers, and a row can never exceed the measure (66014a1)
- **workdesk**: A detail fetch asks for the rows it names (7fb8ecc)
- **claude**: The slack plugin and notion's newer writes are denied (db54110)
- **agentbar**: A window can decline the sidebar (2d97918)
- **workdesk**: R resyncs the project the mirror holds, not the cwd's (f76d6a0)
- **workdesk**: O opens the row, instead of reporting that it did (e139cdb)
- **tmux**: Place the workdesk float, or tmux walks it down the screen (ebc172a)
- **tmux**: The workdesk chip wears a list mark, not a boxed one (ed47e99)
- **tmux**: Run the workdesk popup with run-shell, not if-shell (d4fcfe3)
- **install**: Pin whisper.cpp 1.9.3 and make its guard version-aware (821c259)
- **workdesk**: Sort newest first, in all six places that sort (afa5b4d)
- **workdesk**: Filter the todo feed, and bind Alt+n (fbe6b86)
- **workdesk**: Read the flavor from the file the switcher actually writes (03b2a8f)
- **theme**: Stop switching Claude Code's TUI theme, pin light-ansi (c963b34)
- **theme**: Let Claude Code's TUI follow the flavor's light/dark (31e5a15)
- **theme**: Route every remaining hardcoded colour through the palette (5dc7e68)
- **theme**: Repaint the host badge with the flavor (87a3a2c)
- **tmux**: Match the MR glyphs to CI's tick and cross (616e2f0)
- **agentbar**: Work ends a forced dormant, so d is a one-shot (14ab167)
- **tmux**: Make the preview's window block list windows (dff50f9)
- **tmux**: Size the picker's branch column to the branch names (1c7c328)
- **agentbar**: Strip whatever glyph Claude marks its title with (8b6dd23)
- **theme**: Keep the cursor on the row you picked (f275ad3)
- **theme**: Give popups their own ground, not the server's frozen one (c0f8695)
- **theme**: Re-run the diff pane on a switch, so hunk follows the flavor (937f284)
- **theme**: Read the flavor per call, so a switch reaches hunk (daba73b)

### Maintenance

- **claude**: Set Opus 5.5 to high effort and drop the footer link regexes (0a854c5)
- **vault**: Sync the collapsed type field (852b807)
- **vault**: Sync vault-check (8a653ae)
- **vault**: Sync vault-check and the work template (c3f5f33)
- **vault**: Carry kind and parent into the work-note template (2311b34)
- **claude**: Pin auto-compact on (e38ab08)
- **claude**: Opus 5 at high, its documented default (845e6f7)
- **claude**: Drop the global effort fallback (a25f344)
- **claude**: Unpinned models fall back to high (852d084)
- **claude**: Opus at xhigh for demanding agentic work (c953720)
- **claude**: Both models at the documented default effort (8c05384)
- **folio**: Keep prettier off the test fixtures (14d9a02)
- **folio**: Ignore pending insta snapshots (191ed85)
- **claude**: Set the TUI theme to light (c471792)
- **install**: Upgrade hunk to 0.19.0 (1791a2d)
- **needs manual steps** - **install**: Upgrade tmux to 3.7c (66d8db0)

### Performance

- **workdesk**: A manifest first, so a sync fetches only what moved (a24e537)

### Tests

- **claude**: Count the meter row's width in characters, whatever the locale (32d1c2d)
- Wait for the tmux server to exit before starting the next (fc270e8)
- **herdr**: Wrap the forge test's longest line to 120 columns (cc3de93)
- **claude**: Wrap the claude-hooks fixture to 120 columns (c5cf86c)
- **folio**: Keep the hostile-input literal within 120 columns (5ee20ec)
- **folio**: Hardening and CLI contract tests (183bc5a)
- **folio**: An edge-case fixture and reader scrolling tests (2acdff2)
- **theme**: Guard the settings dialogue's interaction (a95184c)
- **theme**: Assert every flavor reaches every generated file (3caa1c4)
- **agentbar**: Wait for the server to die before the next new-session (ea4fc06)
- **bootstrap**: Prove a fresh Ubuntu end to end (b5c77d1)
- **task**: Gate shell formatting with shfmt (53639f6)

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

- **yazi**: Drop the zoxide dep, it is bundled in yazi core (f08d11a)
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
