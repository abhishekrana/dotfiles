# CLAUDE.md

Left tmux sidebar showing every Claude Code agent across all sessions, state driven by Claude Code hooks. Go +
Bubble Tea TUI with shell glue. README covers usage and architecture — read its "Notes for hacking" before touching
anything that talks to tmux. Line width ≤120 everywhere.

## Commands

```bash
make build              # bin/tmux-agent-sidebar
make unit               # go test -short ./...
make e2e                # full lifecycle against throwaway tmux servers
make test               # everything
go test ./e2e/ -run TestName -v -count=1   # single e2e test
bin/tmux-agent-sidebar mockup              # UI preview with fake data (needs a TTY)
```

Preview loop for `render.go`/`theme.go` — build, render `mockup` (fake data, no live sessions read) on a
throwaway socket, capture. Never the live server. Pinning the width keeps an `attach` faithful to ~30 cols:

```bash
make build
tmux -L tas-mock -f /dev/null kill-server 2>/dev/null
tmux -L tas-mock -f /dev/null new-session -d -s v -x 36 -y 34 "$PWD/bin/tmux-agent-sidebar mockup"
tmux -L tas-mock -f /dev/null set -g window-size manual \; set -g status off \; resize-window -t v -x 36 -y 34
tmux -L tas-mock -f /dev/null capture-pane -p -e -t v    # -e keeps colors; plain -p to eyeball layout
tmux -L tas-mock -f /dev/null send-keys -t v G           # j/k/g/G navigate, Enter flashes the action
tmux -L tas-mock attach -t v                             # optional live-test; detach C-b d, then kill-server
```

`NewMockup` (`internal/ui/app.go`) is the sample fixture — when you change the layout, keep it representative:
every state (working/permission/asking/done/done-seen/idle) plus one multi-Claude-on-one-branch session.

## Layout

- `cmd/tmux-agent-sidebar` — subcommands: `run`, `mockup`, `status`, `hook`, `install-hooks`
- `internal/hook` — event JSON → `@agent_*` pane options; `Decide()` is pure, `Install()` merges into Claude
  settings (symlink-safe)
- `internal/tmux` — exec wrapper, `list-panes -a` snapshot, branch cache, status segment
- `internal/ui` — Bubble Tea TUI: `app.go` (state, mouse, selection sync), `render.go` (blocks, highlight), `theme.go`
- `scripts/` — `toggle.sh` (global), `open.sh`, `follow.sh`, `resurrect-save.sh`
- `agent-sidebar.tmux` — TPM entry point

## Rules

- Never touch the live tmux server. Tests and manual checks run on private sockets: `tmux -L <name> -f /dev/null`.
- Git branch is per worktree: one checkout = one branch, and multiple Claudes in the same worktree share it. The
  sidebar reads each pane's branch from its cwd and draws the branch headline once per run of consecutive
  same-branch agents (colored by the most-urgent when several share it). A session's panes usually sit in one
  worktree (so one branch), but tmux doesn't enforce that — don't assume one branch per session; just collapse the
  agents that actually match.
- Detection is hooks + pane options only — never scrape pane content.
- `hook` must never exit non-zero or block; Claude Code waits on it.
- Sidebar liveness is `#{pane_current_command} == tmux-agent-sidebar` everywhere; never wrap the binary in a shell
  (breaks it).
- Mouse actions fire on release, not press.
- Comments: one short line, only for what the code can't say.
- After changing behavior, add or extend an e2e test that fails without the change.
- The sidebar loads via TPM `@plugin` (portable across machines through the dotfiles) — never switch it to a
  `run-shell` on a local checkout; that path isn't portable. So local changes reach a running sidebar only via
  push → pull the TPM clone (`~/.tmux/plugins/tmux-agent-sidebar`) → `make build` → restart (`prefix+e` twice),
  and the clone's HEAD must actually reach `origin/main` (`prefix+U` can silently skip the pull).

## Deploy

"Deploy" (aka "make it live", "get it working on my system") means: get the change running in the user's
**live tmux**, not just built or pushed. Do every step yourself except the last:

1. Commit + push to `main`.
2. Pull the TPM clone to `origin/main` and `make build` there — both filesystem, don't touch the live server.
3. Verify the rebuilt binary headlessly (private-socket mockup) and confirm the clone HEAD reached `origin/main`.
4. Then tell the user to restart the sidebars: **`prefix + e` twice**. This is the one manual step — the sidebar
   is a long-lived process and this project never drives the live tmux server. Always state it explicitly.
