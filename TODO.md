# TODO

Open questions and deferred work. One heading per area, newest first within it.

## theme

**Should the `⛭` settings chip switch Claude Code's TUI theme too, and can it without dirtying a tracked file?**

Switching the flavor from the chip re-skins everything except Claude Code, so a switch to a dark flavor leaves the TUI
light until you run Claude's own `/theme`. Making the switcher do it is what we just removed, for a reason that has not
gone away: Claude persists `theme` into `~/.claude/settings.json`, which is a stow symlink into this repo, so **any**
write there - ours or Claude's own `/theme` - leaves the tracked file dirty. Two writers on one tracked file is the
problem, not which value wins.

What we know:

- `settings.json` beats `~/.claude.json`; the value pinned there is what Claude uses.
- Valid values are `light` · `dark` · `light-ansi` · `dark-ansi` · `light-daltonized` · `dark-daltonized`. The `-ansi`
  pair paints from the terminal's 16 ANSI colours, so it already follows this palette - which is why `light-ansi` is
  pinned and why the fade is gone.
- Because `-ansi` inherits the terminal palette, a same-mode flavor switch (solarized-light → catppuccin-latte) needs no
  Claude change at all. Only a light↔dark switch actually does.
- Claude Code reads the theme at startup, so any mechanism needs `/theme` or a restart to show up in running sessions.

Options not yet weighed properly:

- Leave it manual (today). One `/theme` after a light↔dark switch, and the dirty diff is yours to commit or revert.
- Untrack the theme key - move `settings.json` out of stow, or generate it, so Claude can own it without touching git.
- Have the switcher write it and accept the dirt, with a `task` target that reverts the key.
- Have the switcher only warn on a light↔dark switch: "run /theme", no write, no dirt.

## work

- No test suite. Every other tool under `test/` has one and `task check` lists them explicitly, so `work` is uncovered.
  `test/commit-walk.sh` (in a stash) is the model: stub the binary so no network or daemon is touched.
- `glab` is not pinned in `install.sh`, and `work` now depends on it plus `jq`. The rule is that `install.sh` holds
  every version pin.
- Issue→MR links are missing from the board. GitLab serves `closesIssues` and `relatedMergeRequests` one item at a time,
  so the whole chain costs one call per MR - worth an opt-in `work sync --links`, not the default path.
