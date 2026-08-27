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

## workdesk

- Issue→MR links are inferred from branch names and descriptions, not asked of GitLab. `closesIssues` is served one
  issue at a time, so forge-truth linkage costs a round trip per issue - worth an opt-in flag, never the default.
- No footer chip yet. A silent `⚑N` count of what is asking something of you, and no live `.tmux.conf` binding for the
  popup - both still to key.
- `sync` fetches merge requests, issues and todos concurrently, but the MR pages are cursor-chained and each node is
  heavy. Splitting into a light list query plus parallel per-MR detail fetches is the next lever; measure first.
