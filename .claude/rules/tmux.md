---
paths:
  - "tmux/**"
  - "theme/**"
  - "apps/agentbar/**"
---

# tmux, the status bar and the sidebar

Scripts in `tmux/.local/bin/`. `tmux-agent-state.sh` and `tmux-settings.sh` are sourced fragments; the rest are
executables. Colours always come from the theme switcher, never hardcoded.

- **One session order everywhere.** The sidebar's bands are the order; `Alt-h`/`Alt-l` walk it and the `Alt-;` picker
  renders it. All three read `agentbar order`. The bands, the clock behind them and the three placement keys
  (`p`/`a`/`d`) are stated in `apps/agentbar/CLAUDE.md` - do not restate them elsewhere.
- **Every pane carries a rail** (`pane-border-status top`, `tmux-rail.sh`). LEFT, always: the pane's folder and branch.
  RIGHT, only for a Claude pane: the worktree it is _writing_ in, which its cwd never follows, since the Bash tool's
  `cd` does not move a pane. Two zones via `#[align=right]` so nothing shifts as state changes.
- **The float wears the review hue, bold, and names itself.** A float IS a pane, so it borrowed the focused-split accent
  the rail paints everywhere. `pane-active-border-style` and `pane-border-format` both expand formats, so
  `#{?pane_floating_flag,…}` states both cases in one setting each. **Bold is the only weight that isolates**:
  `heavy`/`double` come from `pane-border-lines`, which takes no format and resolves from the active pane for the whole
  window even under `set -p`, so every divider thickens with it. Attributes are honoured on the _active_ border despite
  what the manual says about `pane-border-style`. **Both settings live twice** - `.tmux.conf` and the `theme` switcher -
  so changing one alone survives only until the next `theme`.
- **A fresh diff pane follows the agent, not the pane**: `tmux-diff-pane.sh` targets `@agent_workdir` and records what
  is on screen in `@diff_target`. **The target then sticks** - only `f`, the worktree picker (`W`) and per-window
  auto-follow (`F`, off by default) re-point a live pane. An amber `◧ changes` chip means the worktree is in no agent's
  `@agent_workdirs`, and reports nothing else.
- **The centre band is a toolbar, and every chip is fixed width** - one space of padding inside, one between; a chip
  that changed width would clip the right-aligned segments. Glyphs must clear the top of their cell: a full-height box
  (`▤`) reads as a second status bar. The order is the order of the work. Labels name what happens, never the key -
  `≡ workdesk` names its tool instead, being the only thing in the row you also type. A chip is coloured only when it
  has state to report.
- **The footer holds no per-pane facts** - commit, CI, clock, and the `⛭` chip at the far right where its fixed width
  cannot reflow the clock. Dropping the git-status plugin took ~72ms of git off every redraw.
- **A chip and its key run one command.** `Alt+n` and `≡ workdesk` both run `tmux-workdesk.sh`; `tmux-mockup.sh`
  overrides `@workdesk_open` rather than rebinding the key, which keeps a click in the mock off the real queue.
- **A float is placed, or tmux cascades it.** `new-pane` with no `-X`/`-Y` steps every float 4 columns right and 2 rows
  down, so the fifth is half off the screen. `@workdesk_open` carries both offsets and `task conf` fails without them.
- **An option holding a command is run by `run-shell`, never `if-shell`.** `if-shell` does not expand `#{}` in its
  command argument: it reports success and runs nothing, so the config parses, `list-keys` looks right, the click logs
  its range - and nothing happens. `run-shell -b "tmux #{@opt}"` is the form that works; `task conf` gates it.
- **Picking a theme is applying it.** The `⛭` dialogue applies on click or Enter and stays open. `theme <flavor>`
  re-skins tmux and ghostty and re-runs the current session's sidebar and diff pane (hunk takes the flavor as a startup
  flag, so that pane is respawned); other sessions recolour on `prefix + R`, since restarting every sidebar at once
  storms this client.
- **`tmux-gitlab.sh` exits 2 on a subcommand it does not have.** The catch-all used to render a status segment and
  answer 0, which is how `o` came to report success and open nothing. `open-url` is the one place a link leaves this
  machine, so the ssh case (OSC 52, since the browser is at the other end) is decided once.
- **tmux's own words**, for anything that renders the picker preview: the bottom line is the status line, `1:claude` is
  a window status of index (`#I`), name (`#W`) and flags (`#F`); panes are the splits inside a window and never appear
  there. Never report activity - `monitor-activity on` puts it on every window.
