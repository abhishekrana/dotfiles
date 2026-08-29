# CLAUDE.md

## Project overview

Personal dotfiles managed with GNU Stow on Ubuntu 24.04.

## Stow packages

- `bash/` → `~/.bashrc.d/` (shell customizations)
- `bat/` → `~/.config/bat/`
- `claude/` → `~/.claude/settings.json`, `~/.claude/statusline-command.sh`. `settings.json` wires the agentbar hook on
  every Claude lifecycle event plus the statusLine; agent state comes from that plugin's `@agent_*` pane options, not
  local hook scripts. Claude Code does not load a user-level `~/.claude/settings.local.json`, so anything that must take
  effect goes in `settings.json`. **Its TUI theme is deliberately not switched by `theme`** - Claude owns that setting
  and `/theme` writes it into `settings.json` itself, so a switcher write is a second writer that fights it and every
  `/theme` dirties this tracked file. Pick `light-ansi` / `dark-ansi` there: those paint from the terminal's own 16
  colours, so the TUI follows this palette anyway. Plain `light`/`dark` are built-in hexes nothing here can reach, and
  no theme at all leaves every accent faded on Solarized cream.
- `clip/` → `~/.local/bin/clip` (copy stdin to the clipboard; picks wl-copy, xclip or pbcopy). Every copy path - tmux
  `copy-command`, `tmux-yank.sh`, fzf's Ctrl-Y, nvim - goes through it, so the backend is chosen in one place.
- `dictate/` → `~/.local/bin/dictate` (toggle-key local Whisper dictation into tmux). Has its own nested `CLAUDE.md` -
  read it before touching the script. `bootstrap.sh` installs its deps (`uv`, `pulseaudio-utils`); the model is not
  prefetched, so the first dictation downloads it. Two backends, named for the hardware and picked by what is installed
  rather than an env var: `gpu` (whisper.cpp via Vulkan, on an AMD or Intel iGPU or NVIDIA) once a GPU is visible, else
  `cpu` (faster-whisper). `bootstrap.sh` installs the GPU build when it can and falls back otherwise. Same `small.en`,
  measured 2.7× faster on the GPU. Dictating into a tmux on another machine needs no setup - the ssh sessions open now
  are the candidates, it routes to whichever tmux holds focus, and the transcript crosses over ssh, never the audio.
- `ghostty/` → `~/.config/ghostty/` (Ghostty terminal config)
- `git/` → `~/.config/git/config` (delta pager, merge settings)
- `hunk/` → `~/.config/hunk/` (hunk diff viewer config, Solarized Light theme)
- `leaf/` → `~/.config/leaf/` (leaf markdown previewer config). Carries a full Solarized Light palette as
  `[themes.solarized-light]`, since leaf ships only `solarized-dark`; that registration is what lets the theme switcher
  drive leaf by name via `LEAF_THEME`. **leaf writes this file itself** - a first run with no config seeds upstream's
  sample there, which blocks `stow leaf`, so the backup step in `bootstrap.sh` is load-bearing.
- `nvim/` → `~/.config/nvim/` (LazyVim config)
- `theme/` → `~/.local/bin/theme` (theme switcher; re-skins the terminal stack across the four flavors from
  `design/palette.toml`, writing per-tool files into `~/.config/theme/`)
- `tmux/` → `~/.tmux.conf`, `~/.local/bin/` scripts - see [tmux](#tmux) below
- `trace/` → `~/.local/bin/dotfiles-trace` (shared always-on trace log for the tmux/agent stack; see "Debugging")
- `yazi/` → `~/.config/yazi/` (yazi file manager config). `package.toml` is machine-managed: `ya pkg install` rewrites
  it, so commit exactly what `ya` writes and never comment it. `zoxide` is bundled in yazi core - listing it as a dep
  fails.

### tmux

Scripts in `tmux/.local/bin/`:

- `tmux-gitlab.sh` - GitLab status, `#issue !mr CI ✓`. No words: the sigils are GitLab's own notation, and the CI glyph
  is fixed-width so a flipping pipeline never shifts the clock.
- `tmux-session-preview.sh` - the picker's fzf preview: the session's dir, repo, branch, aggregate state, its agents
  (one line each: glyph, title, state, age, and the `window.pane` it sits in) and its windows (one line each: index,
  name, pane count, and the flags the status line shows - current, zoomed, bell; never activity, which
  `monitor-activity on` puts on every window). It resolves the directory the same way the rows do - most of the
  session's non-sidebar panes, an agent's worktree winning - never the session's active pane. **tmux's own words**: the
  bottom line is the status line, `1:claude 2:bash` is the window list, and each entry is a window status of index
  (`#I`), name (`#W`) and flags (`#F`); panes are the splits inside a window and never appear there.
- `tmux-agent-state.sh` - sourced agent-state language: glyphs, colors, state ranking and `agent_title` (Claude's own
  title for a session, from its pane title), shared by the session picker and its preview, mirroring the sidebar's.
  Colors come from the theme switcher, never hardcoded.
- `tmux-settings.sh` - the `⛭` chip's dialogue: the owning area down the left, its settings beside them, their values
  down the right, one value per row. No submenu - fzf has one cursor and its preview cannot be focused, so showing every
  value is what removes the need for two. Add a setting with a `group()` call in `list_rows` and an arm in `do_apply`;
  areas and settings stay alphabetical. It is also the only home for a setting with no business being a sidebar key -
  agentbar's Notify lives here, not on the bar.
- `tmux-reset.sh` - the `prefix + R` UI reset: reload + default geometry, nothing killed.
- `tmux-workdesk.sh` - the one command behind `Alt+n` and the `▤ workdesk` chip: close this window's workdesk float if
  one is up, else open what `@workdesk_open` says. Matching is floating + a command named `workdesk`, so the mockup's
  fixture override toggles the same way.
- `tmux-mockup.sh` - `task mockup`, previews the whole frame with fake data on a private server.
- Session picker, resurrect guard, yank.

Rules:

- **One session order everywhere.** The agentbar sidebar's bands (pinned / active / dormant, alphabetical inside each)
  are the order, `Alt-h`/`Alt-l` walk it row by row and wrap, and the `Alt-;` picker popup renders the same bands. All
  three read `agentbar order`, and `p` (pin) in either view is the only thing that moves a session.
- **Every pane carries a rail** (`pane-border-status top`, `tmux-rail.sh`), same rule for all of them. LEFT, always:
  this pane's folder and branch. RIGHT, only when that pane runs Claude: the worktree it is _writing_ in, which its cwd
  never follows, since the Bash tool's `cd` does not move a pane. Two zones via `#[align=right]` - the left anchored at
  one column, the right growing leftward into rule - so nothing shifts as state changes.
- **A fresh diff pane follows the agent, not the pane.** `tmux-diff-pane.sh` targets `@agent_workdir` (stamped by the
  agentbar hook from each Edit/Write) and records what is on screen in `@diff_target`.
- **The target then sticks.** A mode item changes what you see, not where you look; only `f` (follow),
  `tmux-worktree-picker.sh` (the menu's `W`) and per-window auto-follow (`F`, off by default) re-point a live pane.
- An amber `◧ changes` chip means the worktree is in no agent's `@agent_workdirs`, and reports nothing else. A click
  only opens the menu.
- **Three keys, three bands.** `p` pins, `a` puts a session in the active band, `d` sends it to dormant now rather than
  waiting out the window. One key, one destination: pressing it again changes nothing, and pressing another moves it -
  so `a` on a pinned session unpins and holds it active. A row never says how it got into its band; a hand-placed
  session looks exactly like one the clock put there. All three keys work in the sidebar and the `Alt-;` picker, which
  share both stores, so `agentbar order` - what `Alt-h`/`Alt-l` walk - always agrees with what you see. A session nobody
  has placed is left to the clock.
- **`d` is a one-shot, and work is what ends it.** A session being worked in - an agent working, or blocked on you - is
  never buried by a placement: it comes straight back up and the forced dormant is dropped, so it obeys the clock from
  there rather than sinking again every time it falls quiet. So `d` on a session you are working in moves nothing, and a
  stray `d` heals itself the moment you go back to that session. A pin is not a one-shot - `p` is yours until you press
  another key.
- **The active band is what you are working on now.** A session with no agents is dormant, and so is one whose agents
  have all gone quiet for longer than `@agentbar-active-for` (default 1h, a `⛭` row). An agent that is working or
  blocked on you keeps its session active however long it has been at it. Nothing runs in the background to make this
  happen: the band is `now - @agent_since` evaluated on the sidebar's existing 1s poll, so there is no timer and nothing
  to go stale. A pinned session never moves - pins are yours - and a session only sinks on its own; coming back up takes
  a prompt, a permission or a new agent.
- **The sidebar nests: session, then its agents.** A session line carries its name and its branch (dim, `⎇` as on the
  pane rail); each agent under it carries the title Claude gave that session, with its state a step deeper. Both facts
  are on screen at once, which is why there is no setting choosing between them. A session Claude has not titled yet
  shows its state line alone, and so does one whose pane title is still the hostname tmux seeded it with.
- **The `Alt-;` popup says the same things in three fixed columns**: session, branch, title. A row is one line and one
  session - that is what `agentbar order`, pin, kill and rename all act on - so when a session holds several agents the
  title shown is the most urgent one's, the same agent the row's glyph describes, and `+N` owns up to the rest. The
  preview lists them all. Fixed columns are the point: cycling lands your eye in the same place on every row.
- **Picking a theme is applying it.** The `⛭` chip opens a centred dialogue with every value on screen; a click or Enter
  applies one and the dialogue stays open, so there is no save step and nothing to drill into. `theme <flavor>` re-skins
  tmux and ghostty and re-runs the current session's sidebar and diff pane - hunk takes the flavor as a startup flag, so
  the pane has to be respawned; other sessions recolour on `prefix + R`, because restarting every sidebar at once storms
  this client.
- **The footer holds no per-pane facts** - the work's commit (7-char sha), its CI, the clock, and the `⚙` settings chip
  at the far right, where its fixed width cannot reflow the clock. Dropping the git-status plugin took ~72ms of git off
  every status redraw.
- **The centre band is a toolbar, and every chip is fixed width.** `▤ workdesk`, `● dictate`, `● dictate+send`,
  `⇡ commit+push`, `◧ changes` - one space of padding inside each and one between, so the row reads as a toolbar rather
  than a ragged sentence, and a chip that changed width would clip the right-aligned segments. The order is the order of
  the work: pick it up, say what to do, then commit it and look at the diff. Labels name what happens, never the key;
  `▤ workdesk` is the one that names its tool instead, because it is the only thing in the row you also type. Colour is
  the only feedback a status bar has, so a chip is coloured only when it has state to report - `▤ workdesk` and
  `● dictate` rest muted for that reason.
- **A chip and its key run one command.** `Alt+n` and `▤ workdesk` both run `tmux-workdesk.sh`, which opens what
  `@workdesk_open` says or closes the float already up, so neither the geometry nor the toggle can drift between them -
  and `tmux-mockup.sh` overrides that option rather than rebinding the key, which is what keeps a click in the mock off
  the real queue.
- **An option holding a command is run by `run-shell`, never `if-shell`.** `if-shell` does not expand `#{}` in its
  command argument: it reports success and runs nothing, so the config parses, `list-keys` looks right, the click even
  logs its range - and nothing happens. `run-shell -b "tmux #{@opt}"` is the form that works; `task conf` fails the gate
  on the other one.

## Apps (built from source)

Buildable projects live under `apps/` - these are **not** stow packages and are never passed to `stow`. Each carries its
own `Makefile` with a uniform `build` target, so `bootstrap.sh` builds any language the same way and the toolchain gets
a pinned `install_*` step. Add one by dropping a project with a `Makefile` under `apps/`.

- `apps/agentbar/` → **two binaries from one Go module.** `bin/agentbar` is the tmux plugin (the Claude agent sidebar);
  `bin/workdesk` is the GitLab work inbox. Separate commands, not subcommands: agentbar runs on every Claude lifecycle
  event and must not carry a forge client, so a GitLab failure can never be a sidebar failure. One module, because Go
  forbids importing another module's `internal/`, and a second module would mean a **third** trace writer (see
  "Debugging") plus a second copy of the tmux reader that already resolves agent → worktree → branch.
  - `workdesk` mirrors the GitLab work you own into `~/.local/state/dotfiles/workdesk/`: `sync` fetches, `open` is the
    Bubble Tea UI (inbox · issues · merge requests · agents, `1`-`4` and tab, `?` for help), `board` is the whole queue,
    `mr <iid>` is one merge request end to end, `matrix` is one row per MR and one column per gate with a totals row,
    and `ready` prints the actionable rows for an agent. `bootstrap.sh` links it into `~/.local/bin` - the one app
    binary that does get a link, because it is a CLI you type.
  - **`Alt+n` and the `▤ workdesk` chip** toggle it (`tmux/.tmux.conf`), both through `tmux-workdesk.sh` so the two
    cannot drift. Invoked by absolute path like the agentbar bindings - `apps/` binaries are not stow packages, so there
    is no `~/.local/bin` symlink to rely on inside tmux. Bare `workdesk` opens it too.
  - **The todo feed is filtered, and the count says so.** GitLab never marks todos done, so the pending list is an
    accumulating log: measured on a real account, 427 of 453 were `review_submitted`, `build_failed`, `unmergeable` or
    `merge_train_removed` - machine notifications about state `mergeabilityChecks` and the pipeline already report for a
    merge request you own, and noise for the far larger number you do not. Not one was a mention. So only the actions
    the bands cannot derive are kept (`assigned`, `mentioned`, `directly_addressed`, `marked`), nothing older than
    `TodoMaxAge`, and the band header reports how many were left out. Unfiltered, the inbox opened with 469 rows; it
    opens with 61.
  - **Bubble Tea, not fzf, and the difference is structural.** fzf re-invoked a process per cursor movement, so previews
    had to be markdown pre-rendered at sync time and cat'd, band headers had to be smuggled into the row list as fake
    items the cursor was taught to skip, and the key hints had to fit ~52 columns or fzf truncated them silently. Here
    the model is held: headers are derived at render time so the cursor is always on a real row, previews are built from
    the snapshot with colour on the gates and a real table for the approval rules, the preview scrolls, and `?` renders
    the keymap so no hint can go missing. The palette is `internal/ui`, generated from `design/palette.toml`, so it
    matches tmux rather than approximating it.
  - **The pointer does what the keys do, and a click is how you look.** The wheel walks whichever pane it is over - the
    list by a row, the preview by lines - and stops at the ends, where `j`/`k` deliberately wrap. A click selects a row;
    the second click on it opens the sheet. The sidebar jumps on the first click because it has no preview; here the
    preview is the reason to click at all. The tabs and the `synced` marker are clickable, a band header answers with
    the first row under it, and everything fires on release - terminals eat the press of a click that also focuses their
    window. `listItems` is the one pass the renderer, the scroll window and the hit test share.
  - **A float, not a popup, and the chip is the reason.** A popup is an overlay: it swallows every click outside its own
    box, so a second click on `▤ workdesk` never reached `MouseUp1Status` and the chip could only ever open. A float
    (tmux 3.7 `new-pane`) is a pane, so the click lands and `tmux-workdesk.sh` closes what it opened. Probed both ways,
    not assumed. `Alt+n` and `✕` still close it from the keyboard and the pointer, and one float per window - the toggle
    reads the current window's panes.
  - **A float is a column to `select-layout`.** Evening a window that holds one shrinks it to a share of the width, so
    `tmux-reset.sh` skips those windows whole rather than repairing geometry by breaking it; `prefix + R` picks them up
    once the float is closed. `workdesk`'s own `P` (promote to a pane) targets `{last}` for the same family of reason -
    tmux refuses to split a floating pane at all.
  - **The UI never acts.** It records which key was pressed on which row and quits; the caller runs the action. That is
    what keeps every action a plain function `workdesk act <key> <ref>` can run with no terminal, and it is why the
    write confirms do not live inside the render loop.
  - **The view ring follows the work, not the volume.** `1` inbox, `2` issues, `3` merge requests, `4` agents: not
    started, then in flight, then who is doing it, with the inbox first because it cuts across all three. The `View`
    const block is the only place that order lives - the tab bar, the digits, their help text, `tab` and `shift+tab` all
    derive from it, so a reorder is one line. It was encoded in five places before, which is how three views came to
    sort one way and three the other.
  - **Newest first inside a band, in all six places that sort.** The band is already the priority signal, so within one
    the useful order is what you touched most recently. Oldest-first was the first attempt - the longest-waiting item is
    the most forgotten - but it opened a band with a merge request from seven months ago, and it silently disagreed with
    the issue, todo and agent views, which were newest-first all along. **The index is a stored artifact, so changing a
    sort needs `workdesk render`** (no network) before `list` reflects it.
  - **The model holds no presentation.** Titles are stored unpadded and ages not at all - only an epoch. A pre-padded
    title looks harmless until a UI sizes the column to the terminal and re-pads it, at which point every row grows an
    ellipsis it never earned. `Row.TSV()` is the one place a fixed column belongs, because its consumer is not this
    program.
  - **Band names are GitLab's own**, from the merge request homepage that has shipped by default since 18.2, so this
    view and the web UI say the same words. Its active/inactive split is modelled too: the picker draws a line where the
    bands stop asking anything of you.
  - **"Can I merge it" comes from `mergeabilityChecks`, never inferred.** `detailedMergeStatus` names one blocker and is
    computed lazily (`UNCHECKED` for much of any real queue); `mergeabilityChecks` returns every gate with its own
    state, so an MR with three problems says so instead of revealing them one at a time. The identifier→message map is
    deliberately open: GitLab adds checks and does not document the set, so an unknown one degrades to a readable label.
  - **`approvalState.rules` is the one that explains a stuck MR.** An approval count can read as satisfied while GitLab
    refuses the merge, because the approver was not eligible for the rule that gates it.
  - **The mirror has two tiers, and that is what makes the popup instant.** `index.json` is a few kilobytes and holds
    only what rows need; the full snapshot and the pre-rendered documents are read by nothing interactive. fzf re-runs
    the preview command on every cursor movement, so decoding the snapshot per keystroke would cost ~7ms against ~0.2ms.
    Ages are stored as epochs and formatted at read time - a baked-in "3d" is wrong by morning.
  - A full snapshot every sync, so a merged MR disappears with no cursor state to drift, and the mirror is derived, so
    deleting it costs nothing. It lives outside any repo because MR bodies can carry credentials. Project comes from the
    git remote and identity from glab's token, so nothing here holds a host, group or username.
  - **A null `project` is not an empty project.** GitLab answers `project(fullPath:)` for a path it cannot see with a
    null rather than an error - once read as zero rows that silently replaced a good board with an empty one. A remote
    whose host is not glab's is refused for the same reason. `workdesk schema-check` validates the query against the
    live schema by probing a path that cannot exist, so a GitLab upgrade that moves a field is one command.
  - Three keys write to GitLab - `a` assign, `e` auto-merge, `M` merge - each behind a typed confirm. `WORKDESK_DRY=1`
    prints the command and stops, which is what the mockup sets. Everything else is read-only.
- `apps/agentbar/` (the sidebar itself) is loaded by a `run-shell` line at the end of `tmux/.tmux.conf`, so it builds
  and runs straight from the repo. The Claude lifecycle hooks in `claude/.claude/settings.json` invoke its binary at
  `$HOME/dotfiles/apps/agentbar/bin/agentbar`. It has its own nested `CLAUDE.md` - read that before touching the code.

## Installing software

`install.sh` is the only thing in this repo that downloads a tool, and it holds every version pin. It doubles as a CLI
so CI installs with the same code a machine does:

```sh
./install.sh                 # list the steps
./install.sh all             # every tool (bootstrap.sh calls this)
./install.sh gate-tools      # just what `task check` needs (CI calls this)
./install.sh install_tmux    # one step by name
./install.sh dictate-deps    # uv + pulseaudio-utils (also part of `all`)
./install.sh whisper-vulkan  # whisper.cpp built against Vulkan for dictate's GPU backend (also part of `all`)
```

`bootstrap.sh` sources it and adds the machine wiring: stow, the `.bashrc` patch, the vaults, the resurrect timer and
`apps/` builds. It takes no arguments - run a single step through `install.sh`. Steps that need the stowed configs in
place (`install_bat_themes`, `install_nvim_plugins`) run after `stow_packages`.

**tmux is pinned and built from source.** Ubuntu 24.04 ships 3.4, which the sidebar's e2e suite fails on.

## Release furniture

- `.github/workflows/` - `ci.yml` on every push/PR, `release.yml` on a `v*` tag
- `cliff.toml` - git-cliff config: Conventional Commits to CHANGELOG.md and release notes

## Debugging (trace log)

**Read the trace log first** when anything in the tmux/agent workflow misbehaves. It is always on and records action
_edges_ across the whole interactive stack.

- **Where:** `${XDG_STATE_HOME:-~/.local/state}/dotfiles/trace.log` (outside the repo, never committed). Size-capped at
  1 MiB with one rotation (`trace.log.1`).
- **View:** `dotfiles-trace tail -f`, or
  `dotfiles-trace show --since 5m --src <tmux|agentbar|clip|hook|sidebar|picker|dictate|resurrect|yank> --grep <pat>`.
  `dotfiles-trace path` prints the file.
- **Format:** logfmt - `ts=<iso ms> src=… evt=… pid=… k=v …`. The on-screen status clock is `%H:%M:%S`, so a screenshot
  anchors to a log window.

What to read, by symptom:

- **A click did nothing.** `src=tmux evt=click range=…` means tmux received it, so the failure is downstream - our bug.
  No line for a click you made means the terminal dropped the event first: the known Ghostty+tmux status-click bug, not
  fixable here.
- **A copy did nothing.** Every copy path goes through `clip`, leaving
  `src=clip evt=copy backend=… sel=… bytes=… rc=… wl=… dsp=…`; a mouse yank adds
  `src=yank evt=copy bytes=… rc=… rc_primary=…` for the tmux edge above it. Read in this order: **no `src=yank` line**
  means tmux never fired the yank (the selection or the binding, not the clipboard); **`bytes=0`** means an empty drag;
  **`rc` non-zero** means the backend failed, and `wl=`/`dsp=` say why. A long-lived tmux server keeps the
  `WAYLAND_DISPLAY` it started with, so after a re-login `wl-copy` cannot reach the compositor until the server is
  restarted or its environment updated.
- **The layout drifted.** A squeezed sidebar or skewed split is a screen change: tmux takes a shrink evenly from every
  pane and has no fixed-size pane. `src=sidebar evt=pin` is the `window-resized` hook fixing the width itself.
  `prefix + R` logs `src=tmux evt=reset … changed=N floated=N` plus one `evt=layout win=… before=… after=…` per window
  changed, and nothing when nothing had drifted. A `floated=` above zero is windows the reset skipped whole: a float
  counts as a column to `select-layout`, so those wait until it is closed.
- **The ▤ workdesk chip did nothing.** `src=tmux evt=workdesk action=open|close rc=…` is every press of the chip and of
  `Alt+n`, which run the same script. `action=close` with no float on screen means the match found someone else's
  floating `workdesk`; `rc` non-zero on `open` means `@workdesk_open` failed, and `err=no_command` that the option is
  unset - a config that was never sourced.
- **A session jump landed wrong.** `Alt-h`/`Alt-l` log `src=agentbar evt=switch session=… from=… key=prev|next ms=…`. No
  line means the binary never ran, so the binding fell through to tmux's alphabetical `switch-client` - rebuild it. A
  `session=` that is not the neighbouring row means the bands moved under you; `agentbar order` prints the list the keys
  walk, and `src=agentbar evt=pin session=… pinned=…` is every pin change from either view.
- **The sidebar state looks stale.** `src=hook evt=event name=… prev=… new=… sid=…` is ground truth of what Claude told
  the sidebar (`via=cwd` means the pane was recovered by the cwd fallback - a resumed or `claude daemon run` session
  that fired the hook with no `$TMUX_PANE`). `src=hook evt=drop reason=no_pane cwd=… sid=…` flags a hook that arrived
  with no pane to land on. `$HOME/dotfiles/apps/agentbar/bin/agentbar doctor` rolls this into a per-pane health check -
  the one-command way to spot a stale sidebar.
- **The diff pane shows the wrong tree.** `src=hook evt=workdir pane=… before=… after=…` is every move of an agent's
  worktree; absent means the agent has only read files, or a hook is not wired - the pane then falls back to its own
  cwd. An empty `after=` is a session boundary dropping it, so a new agent cannot inherit the last one's target.
  `src=tmux evt=diff action=create|respawn target=…` is what the pane was pointed at, and `action=follow from=… to=…`
  every catch-up; a `to=` you did not expect means `@agent_workdir` is stale, so read the `evt=workdir` line above it.
  `bg=bg` marks an auto-follow (no focus change), `bg=0` an explicit one.

Writing to it:

- **Two writers, one format:** the `dotfiles-trace` CLI (`trace/`, used by all shell/tmux callers) and the Go
  `apps/agentbar/internal/trace` package (used by the sidebar, the hook and workdesk). Keep them in sync on timestamp,
  escaping, and rotation.
- **Log edges only, never hot loops** (mouse motion, ticks, status redraws, the dictate silence poll, fzf preview/list,
  statusline) - that keeps it free.
- **Toggles:** `tmux set -g @agentbar-trace-verbose on` adds the noisy sidebar events for a live hunt (effect within
  ~1s, no restart). `DOTFILES_TRACE=0` disables entirely; for tmux `run-shell` children use
  `tmux set-environment -g DOTFILES_TRACE 0`.

### Tracing a new feature

- One `dotfiles-trace log <src> <evt> k=v ...` per action edge, never in a hot loop. Reuse an existing `src`.
- Log the outcome, not the intent - `rc=`, `bytes=` are what separate "it ran" from "it worked".
- Changing state: `before=… after=…` on one line, only when it actually changed. Values are capped at 200 chars.
- Tests run under `DOTFILES_TRACE=0`, never into the live log.

## Vault template

`vault-template/` holds the boilerplate for the two notes vaults (`~/vaults/personal`, `~/vaults/work`). Like `apps/`,
it is **not** a stow package - `bootstrap.sh` copies it into each vault as **real files**, so the scaffolding is
committed into that vault's own private repo. `common/` is shared by both vaults (skeleton, templates, `.claude/`
guardrail hooks + `vault-check`, `.githooks/` pre-commit guard); `personal/` and `work/` carry their own `CLAUDE.md` +
`README.md`. Copies are seed-if-missing, so re-running bootstrap never clobbers live edits. Vault _content_ never lives
here - this repo is public.

## Rules

- **Never commit personal info**: no names, emails, IP addresses, work-specific paths, or employer / product / project
  names. This includes anything read out of a work forge - **no ticket or MR numbers, branch names, CODEOWNERS paths, or
  queue statistics** (counts of open MRs, ages, approval numbers). Those are findings about the employer's codebase, not
  facts about these dotfiles; they belong in the work vault. Tool docs describe behaviour, never the data it returned.
- **Audit before committing**: `task secrets` (gitleaks over the tree and the full history) must pass, and eyeball the
  diff for your name, employer, and project names - a scanner won't catch those
- **Only track customizations**: don't add stock Ubuntu defaults (prompt, bash-completion, color aliases) - those belong
  in the system `.bashrc`
- **Prefer `~/.local/bin`** for tool installations over system-wide installs
- **Keep it simple**: no unnecessary abstractions, no over-engineering
- **Keep comments and docs terse**: state the rule, not the story around it. History belongs in the commit message.

## Conventions

- Bash files in `.bashrc.d/` use `.bash` extension
- Only `00-path.bash` has a numeric prefix (must load first for PATH); all other files use plain names
- Each tool init file guards with `command -v tool &>/dev/null || return`
- Scripts under a `.local/bin/` are executable; a fragment meant to be **sourced** says so in its header comment and
  stays non-executable (`task perms` enforces both - tmux swallows a `#()` it cannot execute as empty output, so a
  missing `+x` makes a rail or a status segment silently vanish)
- Private/work-specific config goes in `~/.bashrc.d/local.bash` (not tracked)
- `bootstrap.sh` must be idempotent (safe to re-run)
- `bootstrap.sh` scaffolds the two notes vaults (see "Vault template"); a vault with no git remote is reported once at
  the end, and the remote/identity are never created or stored here
- Keep lists alphabetically sorted (stow packages, apt packages, pinned versions, bootstrap calls, docs)

## Deploy

When I say "deploy": **first commit and push, then make it live** on the running system.

1. **Commit & push** to `main` (run the secrets audit first).
2. **Make it live:**
   - **tmux** (`tmux/.tmux.conf`): `tmux source-file ~/.tmux.conf`. One server is shared by all sessions, so a single
     reload updates every existing session at once.
   - **Stowed scripts** (symlinks - `dictate/`, `bash/`, etc.): live the moment the repo file is saved; no step needed.
   - **New stow package or file**: `cd ~/dotfiles && stow <pkg>`, then reload the relevant tool. A new file in an
     already-stowed package needs this too - until then it has no symlink, and callers using `~/.local/bin` fail
     silently.
   - **`apps/agentbar`** (Go): `task agentbar:build`, then **`prefix + R`** per session to pick up the new binary
     (reloads and restarts that sidebar in place); **`prefix + e` twice** does all sessions, at the cost of the render
     storm. Hook-path edits to the stowed `settings.json` take effect on the next agent lifecycle event.
3. Always list any steps I must run by hand - things that can't be scripted (re-login, `gsettings`/GNOME shortcut
   install, `systemctl --user …`, opening a fresh shell).

## Commits

[Conventional Commits](https://www.conventionalcommits.org/): `type(scope): summary`. The changelog and the release
notes are generated from these, so the type and scope are the machine-readable part - get them right.

- **Types**: `feat` · `fix` · `docs` · `refactor` · `perf` · `test` · `build` · `ci` · `chore`
- **Scope** is the area, matching a stow package, an app, or a repo concern: `agentbar`, `bash`, `bat`, `bootstrap`,
  `claude`, `clip`, `design`, `dictate`, `ghostty`, `git`, `hunk`, `install`, `leaf`, `lint`, `nvim`, `release`, `task`,
  `theme`, `tmux`, `trace`, `vault`, `workdesk`, `yazi`. Omit it only when a change genuinely spans everything.
- **Breaking = needs manual steps on the machine.** A `!` after the scope (`feat(tmux)!:`) or a `BREAKING CHANGE:`
  footer marks a release that can't just be pulled - a re-login, a re-stow, a GNOME shortcut, a systemd unit. It renders
  as "needs manual steps" in the changelog and, pre-1.0, drives the MINOR bump.
- `ci:` and `chore(release):` are filtered out of the changelog (see `cliff.toml`).
- Do not add `Co-Authored-By` lines to commit messages

## Releasing

`task check` must be green and pushed first. Then the tag is the trigger: `.github/workflows/release.yml` re-runs the
gate, runs the Docker fresh-install test, and publishes a GitHub Release with notes from `git cliff`. Never move a
published tag - bump the patch instead.

```sh
task check                              # gate: shellcheck, ruff, prettier, gitleaks, tests
task changelog V=v0.2.0                 # prepend the generated section to CHANGELOG.md
git commit -am "chore(release): v0.2.0" && git push
git tag -a v0.2.0 -m "dotfiles v0.2.0"  # annotated SemVer tag
git push origin v0.2.0                  # fires release.yml
```

SemVer, `v`-prefixed. **This repo stays on 0.x - do not bump to 1.0.** Pre-1.0 shifts the meanings down one: a release
needing manual steps bumps the MINOR, everything else bumps the PATCH. `task release-notes` previews what the next
release would publish; `task changelog` prepends to `CHANGELOG.md` rather than regenerating it.

**A tag with no published Release is not a release.** Before tagging, check the newest tag has one. If it does not, fix
the failure and ask whether to re-tag that version or bump - never tag over it.

**Green `task check` does not mean the release will publish.** `release.yml` also runs `test/bootstrap-fresh.sh`, which
the gate does not. Run `task fresh` before tagging.

## Tasks

`Taskfile.yml` holds the routine work - run `task` for the list. `task check` is the gate CI runs. The `tmux-*` tasks
are the ones that act on the live server (`tmux-reset` is `prefix + R`); the `agentbar:*` tasks delegate to that
project's own build files.

`task check-ci` reruns the tmux-driven suites - the shell gates and the agentbar tests - in a container mirroring the
runner: gawk as `awk`, no `LANG`, `CI` set. Run it before pushing anything that touches tmux, rendering or the pane
protocol. The rest of the gate reads files and is environment-blind. **The runner's `awk` is gawk and Ubuntu's is
mawk**: on `exit` gawk closes the pipe and SIGPIPEs the producer, so under `set -e` + `pipefail` a pipeline that reads
one line and quits dies on CI and passes here. Match a first row with a flag, never `exit`.

## Formatting

- Markdown: `task fmt` (prettier, pinned in `Taskfile.yml` so CI and local agree). `task fmt-check` checks without
  writing.
- Shell: `shfmt -i 4 -ci` (flags in `Taskfile.yml`, not `.editorconfig` - an extensionless script that matched no
  section would silently get tab indent). Never `--keep-padding`: it splits `cmd; cmd` and then aligns to the original
  column, which mangles the file.
- Lint gates on bugs, not style: `shellcheck -S warning` for shell, `ruff --select E9,F` for the Python in `dictate`.
  Deliberate idioms that a linter misreads carry a directive rather than being rewritten.
- **120 columns**, enforced per language: `task width` (shell, Go), `ruff` E501 (Python), prettier `printWidth`
  (markdown, yaml, json). Markdown table rows and long inline-code spans can exceed it - prettier will not break an
  unbreakable token.
- Always use a plain hyphen (`-`), never em or en dashes
