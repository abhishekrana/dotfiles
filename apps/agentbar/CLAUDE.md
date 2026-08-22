# CLAUDE.md

Left tmux sidebar showing every Claude Code agent across all sessions, state driven by Claude Code hooks. Go + Bubble
Tea TUI with shell glue. README covers usage and architecture - read its "Notes for hacking" before touching anything
that talks to tmux. Line width ≤120 everywhere.

## Commands

From the repo root, `task agentbar:build` / `agentbar:test` / `agentbar:unit` / `agentbar:mockup` / `agentbar:doctor`
delegate to these targets; `task check` at the root runs the suite as part of the release gate.

```bash
make gen                # theme_gen.go from design/palette.toml (build/test run it)
make build              # bin/agentbar
make unit               # go test -short ./...
make e2e                # full lifecycle against throwaway tmux servers
make test               # everything
go test ./e2e/ -run TestName -v -count=1   # single e2e test
bin/agentbar mockup              # UI preview with fake data (needs a TTY)
bin/agentbar doctor              # audit live Claude panes vs the hook trace for state desync
```

Preview loop for `render.go`/`theme.go` - build, render `mockup` (fake data, no live sessions read) on a throwaway
socket, capture. Never the live server. Pinning the width keeps an `attach` faithful to ~30 cols:

```bash
make build
tmux -L agentbar-mock -f /dev/null kill-server 2>/dev/null
tmux -L agentbar-mock -f /dev/null new-session -d -s v -x 36 -y 34 "$PWD/bin/agentbar mockup"
tmux -L agentbar-mock -f /dev/null set -g window-size manual \; set -g status off \; resize-window -t v -x 36 -y 34
tmux -L agentbar-mock -f /dev/null capture-pane -p -e -t v    # -e keeps colors; plain -p to eyeball layout
tmux -L agentbar-mock -f /dev/null send-keys -t v G           # j/k/g/G navigate, Enter flashes the action
tmux -L agentbar-mock attach -t v                             # optional live-test; detach C-b d, then kill-server
```

`NewMockup` (`internal/ui/app.go`) is the sample fixture - when you change the layout, keep it representative: every
state (working/permission/asking/done/done-seen/idle), one multi-Claude-on-one-branch session, one untitled agent (so
title mode's branch fallback shows), and all three bands (pinned/active/dormant).

## Layout

- `cmd/agentbar` - subcommands: `run`, `mockup`, `status`, `order`, `next`/`prev`, `pin`, `hook`, `doctor`
- `internal/hook` - event JSON → `@agent_*` pane options; `Decide()` is pure; `ResolvePane()` finds the pane by the
  event `cwd` when `$TMUX_PANE` is absent; `workdir.go` stamps `@agent_workdir` (the worktree the agent is _writing_ in,
  which its pane's cwd never follows) at pane and window scope, from the `file_path` of an Edit/Write tool event - a
  write and nothing else, never a `CwdChanged` (Claude resets its shell cwd home after every Bash call, which would undo
  the edit that just moved the agent); `ClearWorkdir` drops it again at a session boundary
- `internal/tmux` - exec wrapper, `list-panes -a` snapshot, branch cache, status segment
- `internal/ui` - Bubble Tea TUI: `app.go` (state, mouse, selection sync), `render.go` (blocks, bands, highlight),
  `theme.go` (the struct and the state map) and `theme_gen.go` (the four flavors, generated); `model.Arrange` groups
  sessions into bands
- `internal/doctor` - `agentbar doctor` self-check: audits live Claude panes vs the hook trace; pure
  `ParsePanes`/`ParseHealth`/`Render`
- `internal/trace` - writer for the shared dotfiles trace log; must match the `dotfiles-trace` CLI byte-for-byte on
  ts/escaping/rotation
- `scripts/` - `toggle.sh` (global), `open.sh`, `restart.sh` (one session, in place - used by the dotfiles `prefix + R`
  reset), `pin.sh` (`window-resized` hook: holds the sidebar at `@agentbar-width`, since tmux takes a shrink evenly from
  every pane and has no fixed-size pane), `follow.sh`, `resurrect-save.sh`, `common.sh` (shared helpers)
- `agentbar.tmux` - TPM entry point

## Rules

- Never touch the live tmux server. Tests and manual checks run on private sockets: `tmux -L <name> -f /dev/null`.
- **The nesting is the layout, and it is what removed a setting.** A session line carries its name and, dim beside it,
  its branch; each agent under it carries Claude's title for that session (`agentIndent`) and a state line one step
  deeper (`stateIndent`). The branch belongs to the session because one checkout is one branch; the title belongs to the
  agent because each Claude has its own. They only ever competed for a line while the layout gave them the same slot -
  so there is no headline mode, no `@agentbar-headline`, and nothing to arbitrate.
- **The state line does not spell out the command.** It is `claude` on every row, and the title above already says whose
  row it is. Two indent steps plus the state label plus the elapsed time is exactly what 30 columns hold - dropping that
  word is what pays for the second step.
- **An untitled agent is one line**, its state line alone; `blockLineCount` must agree. A dormant session is its dim
  name alone - no agent means no worktree to read a branch from, so it draws none.
- `model.BranchOf` is the one implementation of a session's branch, used by the live snapshot and the mockup alike.
  Agents in one session almost always share a worktree; when they do not, the first wins and `+N` counts the rest, since
  one line cannot name several branches. Do not assume one branch per session - that suffix is the honest signal.
- **Two pane titles mean "not titled yet"**: Claude's `Claude Code` placeholder, and the **hostname** - what tmux seeds
  `pane_title` with until something sets one. Without that second guard, a title-mode row for a Claude that has not
  titled itself yet reads as the machine's name; `agentTitle` takes `#{host}` from the same `list-panes` call to test
  it.
- **Notifications are not a sidebar control.** `@agent_notify` is read by the hook and set from the `⛭` dialogue's
  Notify row - the sidebar has no key and no chip for it, so a setting has one home.
- Detection is hooks + pane options only - never scrape pane content. The one exception is `#{pane_title}`, where Claude
  Code publishes that title (`agentTitle` strips its glyph and reads the pre-prompt placeholder as no title) - a pane
  property, read for the headline alone. State stays hooks-only.
- The hook resolves its pane from `$TMUX_PANE`, else by matching the event `cwd` to a Claude pane (`ResolvePane`) -
  resumed / `claude daemon run` sessions fire hooks with no `TMUX_PANE`, and dropping them silently freezes the pane's
  state. A paneless hook that still can't be placed is dropped (traced with `cwd`/`sid`); `agentbar doctor` surfaces
  this.
- Sessions render in three bands - pinned (`@agentbar-pins`, the `p` key), active, dormant - alphabetical within each.
  `model.Arrange` is the pure grouping; section dividers are non-selectable blocks nav/clicks skip. All three carry a
  `<name> ·<count>` label (`model.BandLabel`, the tokens `order` prints) and show it whenever the band has a non-empty
  neighbour - never leave one unlabelled as "the rest".
- **Dormant is "no agents" OR "gone quiet"**, and the clock decides the second: `model.Fresh` calls a session active
  while any agent is working or blocked on you - however old its `@agent_since`, since that stamps the last state
  _change_ and a long turn would otherwise age out mid-work - else while the newest state change is inside
  `@agentbar-active-for` (`model.DefaultActiveFor`, 1h, read via `tmux.ActiveFor` so the sidebar and the short-lived
  `order`/`next`/`prev` processes agree). `Arrange` stamps `Session.Quiet`, so `Band()` and every caller stay
  clock-free.
- **Nothing owns that transition.** It is `now - timestamp` evaluated at render, on the 1s poll the sidebar already
  runs: no background process, no timer, no persisted state, and `agentbar order` computes the same bands from the same
  pane options. Do not add a watcher - that is the property that keeps this from going stale.
- **Three keys, three bands: `p` pinned, `a` active, `d` dormant.** One key, one destination - pressing it again lands
  the session where it already is, so nothing happens. `model.Place` is the whole transition, pure and tested: a pin and
  a forced band are one decision, so whichever key you press clears the other store. That is what makes `a` on a pinned
  session move it instead of being swallowed by `Band()`, which reads `Pinned` first. Pins live in
  `tmux.Pins`/`SetPins`, forced bands in `tmux.Bands`/`SetBands` (`@agentbar-bands`, `name=active|dormant`) on the same
  option-plus-XDG-mirror-plus-prune path, and `agentbar band <session> pinned|active|dormant` is the key as a command so
  the picker drives both stores through one place.
- **A session nobody has placed is left to the clock**, which is the normal case; a placed one stays where you put it
  until you press another key.
- **A forced dormant is a one-shot; a forced active is not.** `Session.Live` (working, or blocked on you) is what ends
  it: `Band()` lifts a live session on the spot, and `model.Expire` then drops the placement outright, so from there the
  session is back on the clock instead of re-sinking the instant it goes quiet - that bounce was the bug. `d` therefore
  cannot sink a session being worked in at all: an unplaced row does not move, and a pinned one loses its pin and is
  left to the clock, since `Place` runs first and a pin and a forced band are one decision. Pinned still beats
  everything. `Expire` takes one session name so the caller elects the writer: every sidebar renders every session, and
  all of them clearing every placement would be a write and a refresh storm per bar - so a sidebar expires only the
  session it lives in (`a.current`), and both `d` paths expire before they persist.
- **A row never says how it got into its band.** A session placed by hand renders exactly like one the clock put there;
  every row is the same shape whatever decided it. A marker was tried and rejected as noise.
- **Positions move on `p`/`a`/`d`, on that threshold, and when work ends a forced dormant - nothing else.** A pinned
  session never moves, however quiet - pins are the user's. A session only ever sinks on its own; coming back up takes a
  real event (a prompt, a permission, a new agent).
- **`setSnapshot` anchors an agent to its session as well as its pane.** Sinking a session drops its agent blocks, so
  without that fallback the cursor jumped to an unrelated row - and `d` then `a` acted on whatever it landed on.
- **A dormant session is its name alone**, no branch and no agent rows: reclaiming those rows is the point of sinking
  one. `buildBlocks` skips its agent blocks entirely rather than hiding them at render, so nav and clicks cannot land on
  a row that is not on screen.
- **`model.Arrange` is the only session order in the stack.** `agentbar order` publishes it as `band<TAB>name`, and
  `next`/`prev` (the dotfiles `Alt-h`/`Alt-l`) plus the dotfiles session picker popup consume it - never reimplement the
  banding in a caller, and never give a caller an order of its own. Those paths pass a nil `BranchCache` (no git calls,
  as `StatusSegment` does): order needs no branches and they run on a keypress.
- Pins go through `tmux.Pins`/`SetPins`, never a raw option write: a write also mirrors the set under `XDG_STATE_HOME`
  and prunes dead session names, and a read restores it when the option is empty. tmux drops user options when its
  server exits, and pins are now the only thing that reorders the bar, so losing them flattens it to alphabetical.
- `hook` must never exit non-zero or block; Claude Code waits on it.
- `@agent_workdir` is stamped at pane, window AND session scope, so one `#{@agent_workdir}` reference resolves by tmux's
  own hierarchy wherever it is read - the agent's pane, a sibling shell, the diff pane, another window of the session.
  `@agent_workdirs` (the last five, newest first, pipe-_wrapped_ so `#{m:*|dir|*,…}` tests membership without a shorter
  name matching a longer one) follows the same path. `@agent_elsewhere` is pane-scoped ONLY - it means "this agent
  writes outside its own pane's worktree", which a shell pane must not inherit. An edit inside the already-stamped
  worktree writes nothing and runs no `git`; a file outside any repo leaves the last known workdir alone rather than
  blanking it.
- **A new agent context drops all three** (`ClearWorkdir`: every `SessionStart` but `compact`, and `SessionEnd`). Pane
  options outlive the Claude session, so without it a fresh agent in a recycled pane - a `/clear`, a resume, a restart -
  inherits the dead one's write target and every reader names a worktree it has never written in. `compact` is the
  exception: same session, same task. Session scope goes only when the last registered agent does, since a sibling agent
  in another window shares that copy.
- A session-scope `set-option` cannot take a pane id: tmux errors and **abandons the rest of the command chain**, so the
  options after it silently never land. Resolve `#{session_name}` first and target that.
- `@agentbar-workdir-cmd` is this package's ONLY hook into anything outside it: an optional command run detached
  (`AGENTBAR_PANE`, `AGENTBAR_WORKDIR` in its environment) after the workdir changed. Unset by default; the dotfiles
  point it at `tmux-diff-pane.sh autofollow`. Keep the knowledge one-way - nothing here may know what the command does.
- Sidebar liveness is `#{pane_current_command} == agentbar` everywhere; never wrap the binary in a shell (breaks it).
- Mouse actions fire on release, not press.
- Trace action edges via `internal/trace` (`start`/`click`/`jump`/`switch`/`pin` + latency, hook `event`/`drop`) -
  always-on, best-effort, never fails the hook. Never trace hot paths (`Snapshot`, `StatusSegment`, the 200ms tick,
  mouse motion); motion + ticks are `Logv` (verbose-gated via `@agentbar-trace-verbose`). See the repo-root CLAUDE.md
  "Debugging" section.
- **Never edit `theme_gen.go`.** A flavor is defined once, in `design/palette.toml`; `scripts/gen-theme.sh` emits the
  Go. `make build`/`make test` run it, and the dotfiles `task check` fails if the committed file is stale.
- Comments: one short line, only for what the code can't say.
- After changing behavior, add or extend an e2e test that fails without the change.
- This project lives in the dotfiles monorepo at `apps/agentbar/` and loads via a `run-shell` line in `tmux/.tmux.conf`
  (not TPM) - it builds and runs straight from this checkout. Portability comes from the dotfiles themselves:
  `bootstrap.sh` installs Go and runs `make build`. Local changes reach a running sidebar via `prefix + R` (the dotfiles
  UI reset: reload + `restart.sh` for that session), or `make build` → reload tmux → `prefix+e` twice to restart them
  all; no push or clone-pull in the loop.

## Deploy

"Deploy" (aka "make it live", "get it working on my system") means: get the change running in the user's **live tmux**,
not just built. Do every step yourself except the last:

1. Commit to the dotfiles repo (push to its `main` only when the user asks).
2. `make build` here - the binary lands in `bin/`, which both the `run-shell` entry and the Claude hooks point at.
3. Verify the rebuilt binary headlessly (private-socket mockup) - never touch the live server.
4. Then tell the user to restart the sidebars: **`prefix + R`** in each session that should pick up the new binary (or
   **`prefix + e` twice** for all of them at once, accepting the render storm). This is the one manual step - the
   sidebar is a long-lived process and this project never drives the live tmux server. Always state it explicitly.
