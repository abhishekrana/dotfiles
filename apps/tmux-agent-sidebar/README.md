# tmux-agent-sidebar

A left sidebar for tmux that shows every Claude Code agent across all your sessions and what state each one is in —
so you always know which agents are working, which need your attention, and which are done.

```
 tmux agents            2/5 ⠼
──────────────────────────────

api-server
   ⠼ claude  working      2m
     feat/rate-limit-rollout
     ⤷ 2 subagents
   ◔ claude  permission  40s
     fix/csrf-rotation

blog
   ✓ claude  done        12m
     draft/tmux-agents-post

dotfiles
   ? claude  asking       4m
     main

scratch             no agents
──────────────────────────────
 ⚠ 2 need attention
 j/k · tab ⚠ · ⏎ jump · q
```

The row under the mouse lights up (hover), and the row you last clicked — session or agent — stays highlighted;
that highlight marks where you are, so there's no separate "here" tag.

States are driven by Claude Code hooks — no pane scraping, no fragile regexes. The instant an agent changes state
(starts working, hits a permission prompt, asks a question, finishes) the hook stamps the state onto its tmux pane,
and the sidebar picks it up on its 1s tick.

## Requirements

- tmux ≥ 3.2
- Claude Code ≥ 2.x (hooks)
- Go ≥ 1.25 (to build; only needed once)
- git (for the branch line)
- notify-send / libnotify (optional, only for desktop notifications)

## Install (TPM)

```tmux
set -g @plugin 'abhishekrana/tmux-agent-sidebar'
```

Press `prefix + I` to install. The plugin builds its binary on first load (requires Go), then wire up the Claude
Code hooks once:

```bash
~/.tmux/plugins/tmux-agent-sidebar/bin/tmux-agent-sidebar install-hooks
```

This appends the plugin's hook entries to `~/.claude/settings.json`. It is idempotent, preserves everything else in
the file (including any hooks you already have — Claude runs all entries for an event), writes through symlinks
(stow-safe), and uses `$HOME`-relative paths. Use `--target <file>` to write somewhere else, e.g. a project's
`.claude/settings.local.json`.

Agents started before the hooks were installed are picked up on their next restart.

## Use

| key            | action                                       |
| -------------- | -------------------------------------------- |
| `prefix + e`     | toggle the sidebar in **all** sessions        |
| `j`/`k`, wheel   | move between sessions and agents              |
| `Enter`, click   | on an agent: jump to its pane; on a session name: switch to that session |
| `g` / `G`        | first / last row                              |
| `Tab`            | jump to the next agent waiting on you (permission/asking), cycling across sessions — the work queue |
| `n`, click chip  | toggle desktop notifications (footer shows the state) |
| `q`              | hide the sidebar everywhere (same as toggle)  |

Clicking a session name switches to it — the one way to reach a session with no agents running (it just
`switch-client`s, leaving the target's window and pane where you left them).

The toggle is global: one press opens a sidebar in every session (and sessions created while it's on get one
automatically); the next press closes them all. While on, each session's sidebar follows you: switching windows
moves the sidebar pane into the active window (one long-lived pane, so selection and scroll position survive).

Every session runs its own sidebar, but the selection is shared: jump to an agent in another session and the
sidebar you land in already highlights it (published via a global option, signalled over a `wait-for` channel so
it's instant, not next-tick). Session switches made outside the sidebar move the highlight too — even to an agent
you only start after switching.

Agent states: `working` (yellow, spinner) · `permission` (red) · `asking` (orange) · `done` (green until you visit
the pane, then gray) · `idle` (gray). Each agent shows its git branch and live subagent count.

## Notifications

Off by default. Press `n` (or click the `notify` chip in the footer) to toggle desktop notifications for the whole
server. When on, the instant any agent needs you — a permission prompt or a question — the plugin fires a
`notify-send` notification (`Claude · permission` / `Claude · asking`, with the `session:window`). It rides the same
Claude Code hooks as the sidebar (no pane scraping) and only fires on the transition *into* an attention state, so a
working agent never spams you. The footer chip mirrors the state (`notify on` / `notify off`), held in the global
`@agent_notify` tmux option; it needs `notify-send` (libnotify) installed and no-ops harmlessly without it.

## Tip: window-tab clicks that need a second try

Unrelated to this plugin but easy to blame on it — two stock-tmux reasons a status-line tab click gets dropped:
rapid clicks chain into `SecondClick`/`TripleClick` events (unbound by default), and terminals eat the *press* of a
click that also focuses their window, delivering only the release. Make every click count:

```tmux
bind -n SecondClick1Status switch-client -t =
bind -n TripleClick1Status switch-client -t =
bind -n MouseUp1Status switch-client -t =
```

The sidebar itself jumps on release for the same reason.

## tmux-resurrect / continuum

Whitelist the sidebar (and, to resume Claude conversations, `claude`) so restores relaunch them:

```tmux
set -g @resurrect-processes '"~tmux-agent-sidebar run" "~claude"'
```

The rest is automatic. Resurrect can't see the sidebar's command (it's the pane's root process), so the plugin sets
resurrect's post-save hook to stamp the command into each save; on restore the whitelist relaunches it and the
sidebar re-registers its own options and follow hook. Without the whitelist, the restored slot is a dead shell pane
you can close.

The same post-save hook rewrites each saved `claude` pane into `claude --resume <session-id>`, using the id the hook
already stamped on the pane (`@agent_session_id`). With `"~claude"` whitelisted, a restore reopens the conversation
where you left it instead of a blank prompt; a pane whose id wasn't captured falls back to a plain `claude`.

## Status line segment

Add the placeholder wherever you want a compact summary:

```tmux
set -g status-right '#{agent_sidebar_status} ... the rest ...'
```

It renders like `⚠2 ●3` (attention / working) and disappears entirely when no agents are running.

## Options

```tmux
set -g @agent-sidebar-key 'e'                # toggle key (after prefix)
set -g @agent-sidebar-width '30'             # sidebar width in columns
set -g @agent-sidebar-theme 'solarized-light' # or 'dark'
set -g @agent-sidebar-focus 'off'            # 'on' focuses sidebar on open
```

## Development

```bash
make build          # build bin/tmux-agent-sidebar
make unit           # unit tests (hook state machine, installer, snapshot, selection)
make e2e            # end-to-end: real tmux servers on throwaway sockets
make test           # everything
bin/tmux-agent-sidebar mockup   # render the UI with fake data in any pane
```

### Checking the UI headlessly

`mockup` needs a TTY, but you can render *and* inspect it without one — on a throwaway tmux server
(`tmux -L <socket> -f /dev/null`, the same isolation the e2e suite uses, so it never touches your live
server). Drive it with `send-keys` and read it back with `capture-pane`; `-e` keeps the escape codes, so
you can confirm colors and the full-width selection highlight, not just the text layout. This is the fast
loop for iterating on `render.go`:

```bash
sock=tas-mock; bin=$PWD/bin/tmux-agent-sidebar
tmux -L $sock -f /dev/null new-session -d -s v -x 30 -y 24 "$bin mockup"   # -x 30 = default width
sleep 1
tmux -L $sock -f /dev/null send-keys -t v G                # navigate: j/k/g/G move, Enter flashes
tmux -L $sock -f /dev/null capture-pane -p -e -t v         # -p text, -e keeps colors; drop -e for plain
tmux -L $sock -f /dev/null kill-server                     # clean up
```

`tmux set -g @agent-sidebar-debug /path/to/log` makes newly started sidebars log mouse events and jumps there.

The e2e suite (`e2e/`) spins up an isolated tmux server per test (`tmux -L <socket> -f /dev/null`, never your live
server or config), fakes agents with a renamed sleep(1) so `#{pane_current_command}` matches, drives real `hook`
events, and asserts against `capture-pane`. It injects raw SGR mouse sequences for real clicks — into panes for the
sidebar and into a pty-attached client for status-line tabs — including the release-only click a terminal produces
when its focus click eats the press.

For a local checkout instead of TPM, add to `~/.tmux.conf`:

```tmux
run-shell ~/path/to/tmux-agent-sidebar/agent-sidebar.tmux
```

Notes for hacking:

- `hook` must never exit non-zero or block — Claude Code waits on it.
- Hooks only load from `~/.claude/settings.json` (user level) or `.claude/settings{,.local}.json` (project level).
  A user-level `settings.local.json` is silently ignored by Claude Code.
- Anchor every tmux call explicitly (`-t`/`-c`): `run-shell` does not set `$TMUX_PANE`, and bare commands resolve
  against the attached client — the wrong session when triggered from elsewhere.
- Only trim newlines from `list-panes` output; trimming whitespace eats trailing empty format fields of the last line.
- Act on mouse *release*: terminals eat the press of a click that also focuses their window.
- Never wrap the sidebar in a plain `sh -c`: without job control the pane's `#{pane_current_command}` becomes `sh`
  and every liveness check breaks.
- resurrect saves the pane shell's *child* command — for a pane whose root process is the program, that's empty
  (hence the post-save hook). Its `restore.sh` only works from server context (`run-shell`).

## How it works

- `install-hooks` registers `tmux-agent-sidebar hook` for the Claude Code lifecycle events (SessionStart,
  UserPromptSubmit, PreToolUse, PermissionRequest, Notification, Stop, SubagentStart/Stop, SessionEnd).
- The hook reads the event JSON, finds its pane via `$TMUX_PANE`, and stamps pane-scoped user options
  (`@agent_state`, `@agent_since`, `@agent_subagents`, ...). Pane options die with the pane, so cleanup is
  automatic; a guard on the pane's current command filters zombies.
- The sidebar TUI (Go, Bubble Tea) snapshots `list-panes -a` once a second and renders sessions alphabetically with
  the current one marked. Jumping runs `switch-client` + `select-window` + `select-pane`, publishes the selection,
  and signals a `wait-for` channel every sidebar blocks on.
- A `session-window-changed` hook moves the sidebar pane into whichever window becomes active (`join-pane -d`),
  with a re-entrancy guard and self-healing if the pane died.
- A global `client-session-changed` hook signals the same channel, so the highlight follows session switches made
  outside the sidebar — to the newly attached session's agent, or to the first agent started there afterwards.
- The TUI registers its own session options and follow hook at startup, so sidebars started outside `open.sh`
  (resurrect restores) just work, and `open.sh` adopts any pane already running the sidebar.

## License

MIT
