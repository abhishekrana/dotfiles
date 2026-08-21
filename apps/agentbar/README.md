# agentbar

A left sidebar for tmux that shows every Claude Code agent across all your sessions and what state each one is in - so
you always know which agents are working, which need your attention, and which are done.

```
 agentbar               1/5 ⠼
──────────────────────────────
 pinned ·2 ───────────────────

 dotfiles  ⎇ main
   Sidebar label toggle
     ? asking              4m

 payments  ⎇ 2091-refund-idempo…
   Refund idempotency keys
     ◔ permission         40s
   Retry the dropped webhooks
     ✓ done               11m

 active ·2 ───────────────────

 api-server  ⎇ feat/rate-limit-…
   Rate limit middleware rollout
     ⠼ working             2m
     ⤷ 2 subagents

 blog  ⎇ draft/tmux-agents-post
   Draft the parallel agents po…
     ✓ done               12m

 dormant ·2 ──────────────────

 notes
 scratch
──────────────────────────────
 ⚠ 2 need attention
 j/k · tab ⚠ · p pin · q
```

The row under the mouse lights up (hover), and the row you last clicked - session or agent - stays highlighted; that
highlight marks where you are, so there's no separate "here" tag.

States are driven by Claude Code hooks - no pane scraping, no fragile regexes. The instant an agent changes state
(starts working, hits a permission prompt, asks a question, finishes) the hook stamps the state onto its tmux pane, and
the sidebar picks it up on its 1s tick.

## Requirements

- tmux 3.7+, pinned in `install.sh` and built by `bootstrap.sh` - Ubuntu 24.04 ships 3.4, which this suite fails on
- Claude Code ≥ 2.x (hooks)
- Go ≥ 1.25 (to build; only needed once)
- git (for the branch line)
- notify-send / libnotify (optional, only for desktop notifications)

## Install

This plugin lives in the dotfiles monorepo at `apps/agentbar/` and the dotfiles wire it up end to end:

- `bootstrap.sh` installs Go and runs `make build`.
- `tmux/.tmux.conf` loads it (after TPM) with `run-shell '~/dotfiles/apps/agentbar/agentbar.tmux'`.
- The Claude Code lifecycle hooks already live in the stowed `~/.claude/settings.json`, pointing at
  `$HOME/dotfiles/apps/agentbar/bin/agentbar`.

So a fresh machine gets the sidebar from `./bootstrap.sh` alone. Agents started before the hooks were installed are
picked up on their next restart.

## Use

| key             | action                                                                                              |
| --------------- | --------------------------------------------------------------------------------------------------- |
| `prefix + e`    | toggle the sidebar in **all** sessions                                                              |
| `Alt-h`/`Alt-l` | switch to the session one row up / down this list, wrapping (dotfiles binding, works outside it)    |
| `prefix + R`    | refresh this session's sidebar and reset the window layout (dotfiles' UI reset)                     |
| `j`/`k`, wheel  | move between sessions and agents                                                                    |
| `Enter`, click  | on an agent: jump to its pane; on a session name: switch to that session                            |
| `g` / `G`       | first / last row                                                                                    |
| `Tab`           | jump to the next agent waiting on you (permission/asking), cycling across sessions - the work queue |
| `p`             | pin the selected session - pinned sessions float to a band at the top                               |
| `a` / `d`       | put the session in the active / dormant band by hand; another key moves it                          |
| `q`             | hide the sidebar everywhere (same as toggle)                                                        |

Clicking a session name switches to it - the one way to reach a session with no agents running (it just
`switch-client`s, leaving the target's window and pane where you left them).

## Bands & pinning

Sessions are grouped into three bands so your working set stays together and dead sessions get out of the way:

- **`pinned`** - sessions you pinned with `p`, floated to the top; the label reads gold.
- **`active`** - sessions whose agents are working, blocked on you, or last changed state within `@agentbar-active-for`
  (default 1h).
- **`dormant`** - sessions with no agents **and** sessions whose agents have all gone quiet for longer than
  `@agentbar-active-for`, dimmed grey and sunk to the bottom, name only. A quiet session sinks on its own an hour after
  you stop; nothing runs in the background to do it, since the band is `now - @agent_since` evaluated on the sidebar's
  existing poll. A pinned session never moves, however quiet.

A labelled divider heads each band - all three named, on one rule: it appears when the band has a non-empty neighbour to
divide it from, so a single-band list shows no dividers at all. The names are `model.BandLabel`, the same tokens
`agentbar order` prints, so a label you read is a token you can grep. Within every band sessions stay **alphabetical**,
so positions never shuffle as agents change state; they move only when you pin or unpin. Pins live in the global
`@agentbar-pins` option (tab-separated session names - tmux allows spaces in a session name but never tabs), so every
session's sidebar shows the same bands at once. Every pin write also mirrors the set to
`${XDG_STATE_HOME:-~/.local/state}/dotfiles/agentbar-pins` and drops names whose session is gone; a tmux server restart
drops the option, and the next read restores it from that mirror.

This order is the **only** session order in the stack. `agentbar order` prints it as `band<TAB>name` lines, and both the
`Alt-h`/`Alt-l` keys (`agentbar prev` / `next`) and the dotfiles session picker popup (`Alt-;`) walk exactly that, so
the list you see is the list they move through. Pinning is the one thing that reorders it - and the popup's `p` writes
the same set this sidebar's `p` does.

The sidebar is on by default: it opens in every session at tmux server start, and any session created while it's on gets
one automatically (set `@agentbar-autostart 'off'` to start closed). The toggle is global: one press closes them all
everywhere, the next reopens them. While on, each session's sidebar follows you: switching windows moves the sidebar
pane into the active window (one long-lived pane, so selection and scroll position survive).

Every session runs its own sidebar, but the selection is shared: jump to an agent in another session and the sidebar you
land in already highlights it (published via a global option, signalled over a `wait-for` channel so it's instant, not
next-tick). Session switches made outside the sidebar move the highlight too - even to an agent you only start after
switching.

Agent states: `working` (teal spinner) · `permission` (red) · `asking` (amber) · `done` (green until you visit the pane,
then gray) · `idle` (gray). Each session shows its branch; each agent its title and live subagent count.

## What a row says

The list nests: a **session** carries its name and, dim beside it, the branch of the worktree it works in. Each
**agent** under it carries the title Claude gave that session, with its state a step deeper.

```text
 api-server  ⎇ feat/rate-limit-…
   Rate limit middleware rollout
     ⠼ working                2m
   Backfill the refund ledger
     ◔ permission            40s
```

That split is the whole design: one checkout is one branch, so the branch belongs to the session; each Claude titles its
own session, so the title belongs to the agent. Both facts fit at once, which is why nothing has to choose between them.

Claude Code generates the title itself and publishes it in its pane title - nothing to configure. A session it has not
titled yet (or whose pane title is still the hostname tmux seeded) shows its state line alone. The state line does not
spell out `claude`: it is the same word on every row, and dropping it is what buys the indent.

## Notifications

Off by default. Set it from the `⛭` dialogue's Notify row (dotfiles) or with `tmux set -g @agent_notify on` to get
desktop notifications for the whole server. When on, the instant any agent needs you - a permission prompt or a
question - the plugin fires a `notify-send` notification (`Claude · permission` / `Claude · asking`, with the
`session:window`). It rides the same Claude Code hooks as the sidebar (no pane scraping) and only fires on the
transition _into_ an attention state, so a working agent never spams you. The footer chip mirrors the state (`notify on`
/ `notify off`), held in the global `@agent_notify` tmux option; it needs `notify-send` (libnotify) installed and no-ops
harmlessly without it.

## Tip: window-tab clicks that need a second try

Unrelated to this plugin but easy to blame on it - two stock-tmux reasons a status-line tab click gets dropped: rapid
clicks chain into `SecondClick`/`TripleClick` events (unbound by default), and terminals eat the _press_ of a click that
also focuses their window, delivering only the release. Make every click count:

```tmux
bind -n SecondClick1Status switch-client -t =
bind -n TripleClick1Status switch-client -t =
bind -n MouseUp1Status switch-client -t =
```

The sidebar itself jumps on release for the same reason.

## tmux-resurrect / continuum

Whitelist the sidebar (and, to resume Claude conversations, `claude`) so restores relaunch them:

```tmux
set -g @resurrect-processes '"~agentbar run" "~claude"'
```

The rest is automatic. Resurrect can't see the sidebar's command (it's the pane's root process), so the plugin sets
resurrect's post-save hook to stamp the command into each save; on restore the whitelist relaunches it and the sidebar
re-registers its own options and follow hook. Without the whitelist, the restored slot is a dead shell pane you can
close.

Autostart also sets resurrect's pre/post-restore hooks: auto-open is suspended while a restore runs (so a
freshly-recreated session doesn't get a second sidebar racing the restored one, nor a corrupted layout) and re-run
afterwards, adopting the restored sidebars.

The same post-save hook rewrites each saved `claude` pane into `claude --resume <session-id>`, using the id the hook
already stamped on the pane (`@agent_session_id`). With `"~claude"` whitelisted, a restore reopens the conversation
where you left it instead of a blank prompt; a pane whose id wasn't captured falls back to a plain `claude`.

## Status line segment

Add the placeholder wherever you want a compact summary:

```tmux
set -g status-right '#{agentbar_status} ... the rest ...'
```

It renders like `⚠2 ●3` (attention / working) and disappears entirely when no agents are running.

## Options

```tmux
set -g @agentbar-key 'e'                # toggle key (after prefix)
set -g @agentbar-width '30'             # sidebar width in columns
set -g @agentbar-theme 'solarized-light' # or 'dark'
set -g @agentbar-focus 'off'            # 'on' focuses sidebar on open
set -g @agentbar-autostart 'on'         # 'off' starts with the sidebar closed
set -g @agentbar-active-for '1h'        # how long a quiet session stays in the active band
# @agentbar-pins / @agentbar-bands hold the p and a/d choices (mirrored under XDG_STATE_HOME)
```

## Development

```bash
make build          # build bin/agentbar
make unit           # unit tests (hook state machine, installer, snapshot, selection)
make e2e            # end-to-end: real tmux servers on throwaway sockets
make test           # everything
bin/agentbar mockup   # render the UI with fake data in any pane
bin/agentbar doctor   # audit live Claude panes vs the hook trace for state desync
```

### Checking the UI headlessly

`mockup` needs a TTY, but you can render _and_ inspect it without one - on a throwaway tmux server
(`tmux -L <socket> -f /dev/null`, the same isolation the e2e suite uses, so it never touches your live server). Drive it
with `send-keys` and read it back with `capture-pane`; `-e` keeps the escape codes, so you can confirm colors and the
full-width selection highlight, not just the text layout. This is the fast loop for iterating on `render.go`:

```bash
sock=agentbar-mock; bin=$PWD/bin/agentbar
tmux -L $sock -f /dev/null new-session -d -s v -x 30 -y 24 "$bin mockup"   # -x 30 = default width
sleep 1
tmux -L $sock -f /dev/null send-keys -t v G                # navigate: j/k/g/G move, Enter flashes
tmux -L $sock -f /dev/null capture-pane -p -e -t v         # -p text, -e keeps colors; drop -e for plain
tmux -L $sock -f /dev/null kill-server                     # clean up
```

Tracing: agentbar always records action edges (start, click, jump/switch + latency, hook events, dropped hook events) to
the shared dotfiles trace log at `${XDG_STATE_HOME:-~/.local/state}/dotfiles/trace.log` (view with
`dotfiles-trace tail -f`). `tmux set -g @agentbar-trace-verbose on` additionally logs mouse motion + spinner ticks; it
takes effect within ~1s (no restart) and `off` stops them again. `DOTFILES_TRACE=0` disables all tracing. Hook events
carry the session id (and `source` on SessionStart); a paneless drop logs the `cwd`/`sid` it carried, and a pane
recovered via the cwd fallback is tagged `via=cwd`. `bin/agentbar doctor` rolls all this into a per-pane health check -
the fast way to spot a stale sidebar.

The e2e suite (`e2e/`) spins up an isolated tmux server per test (`tmux -L <socket> -f /dev/null`, never your live
server or config), fakes agents with a renamed sleep(1) so `#{pane_current_command}` matches, drives real `hook` events,
and asserts against `capture-pane`. It injects raw SGR mouse sequences for real clicks - into panes for the sidebar and
into a pty-attached client for status-line tabs - including the release-only click a terminal produces when its focus
click eats the press.

For a local checkout instead of TPM, add to `~/.tmux.conf`:

```tmux
run-shell ~/path/to/agentbar/agentbar.tmux
```

Notes for hacking:

- `hook` must never exit non-zero or block - Claude Code waits on it.
- Hooks only load from `~/.claude/settings.json` (user level) or `.claude/settings{,.local}.json` (project level). A
  user-level `settings.local.json` is silently ignored by Claude Code.
- Anchor every tmux call explicitly (`-t`/`-c`): `run-shell` does not set `$TMUX_PANE`, and bare commands resolve
  against the attached client - the wrong session when triggered from elsewhere.
- Only trim newlines from `list-panes` output; trimming whitespace eats trailing empty format fields of the last line.
- Act on mouse _release_: terminals eat the press of a click that also focuses their window.
- Never wrap the sidebar in a plain `sh -c`: without job control the pane's `#{pane_current_command}` becomes `sh` and
  every liveness check breaks.
- resurrect saves the pane shell's _child_ command - for a pane whose root process is the program, that's empty (hence
  the post-save hook). Its `restore.sh` only works from server context (`run-shell`).

## How it works

- The stowed `~/.claude/settings.json` registers `agentbar hook` for the Claude Code lifecycle events (SessionStart,
  UserPromptSubmit, PreToolUse, PermissionRequest, Notification, Stop, SubagentStart/Stop, SessionEnd).
- The hook reads the event JSON and finds its pane via `$TMUX_PANE`, falling back to matching the event's `cwd` to a
  Claude pane when it's absent - resumed / `claude daemon run` sessions fire hooks with no `TMUX_PANE`. It stamps
  pane-scoped user options (`@agent_state`, `@agent_since`, `@agent_subagents`, ...); pane options die with the pane, so
  cleanup is automatic, and a guard on the pane's current command filters zombies.
- The sidebar TUI (Go, Bubble Tea) snapshots `list-panes -a` once a second and renders sessions in three bands - pinned,
  active, dormant - alphabetical within each. Jumping runs `switch-client` + `select-window` + `select-pane`, publishes
  the selection, and signals a `wait-for` channel every sidebar blocks on.
- `agentbar order` / `next` / `prev` / `pin` expose that same grouping (`model.Arrange`) to the keys and the picker
  popup, so nothing outside the TUI has to reimplement the bands. They skip the per-pane git lookups the sidebar does
  for branch names - order needs none, and they run on a keypress.
- A `session-window-changed` hook moves the sidebar pane into whichever window becomes active (`join-pane -d`), with a
  re-entrancy guard and self-healing if the pane died.
- A `window-resized` hook (`scripts/pin.sh`) holds the sidebar at `@agentbar-width`. tmux has no fixed-size pane and
  takes a shrink evenly from every pane in the row rather than in proportion, so the narrow sidebar is the first thing
  it loses - moving to a smaller screen would leave it a few columns wide. Width only, so pane sizes you set by hand are
  left alone. (It is a window hook, so it only shows up under `show-hooks -gw`.)
- A global `client-session-changed` hook signals the same channel, so the highlight follows session switches made
  outside the sidebar - to the newly attached session's agent, or to the first agent started there afterwards.
- The TUI registers its own session options and follow hook at startup, so sidebars started outside `open.sh` (resurrect
  restores) just work, and `open.sh` adopts any pane already running the sidebar.

## License

MIT
