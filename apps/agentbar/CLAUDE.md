# CLAUDE.md

Left tmux sidebar showing every Claude Code agent across all sessions, state driven by Claude Code hooks. Go +
Bubble Tea TUI with shell glue. README covers usage and architecture — read its "Notes for hacking" before touching
anything that talks to tmux. Line width ≤120 everywhere.

## Commands

```bash
make build              # bin/agentbar
make unit               # go test -short ./...
make e2e                # full lifecycle against throwaway tmux servers
make test               # everything
go test ./e2e/ -run TestName -v -count=1   # single e2e test
bin/agentbar mockup              # UI preview with fake data (needs a TTY)
```

Preview loop for `render.go`/`theme.go` — build, render `mockup` (fake data, no live sessions read) on a
throwaway socket, capture. Never the live server. Pinning the width keeps an `attach` faithful to ~30 cols:

```bash
make build
tmux -L agentbar-mock -f /dev/null kill-server 2>/dev/null
tmux -L agentbar-mock -f /dev/null new-session -d -s v -x 36 -y 34 "$PWD/bin/agentbar mockup"
tmux -L agentbar-mock -f /dev/null set -g window-size manual \; set -g status off \; resize-window -t v -x 36 -y 34
tmux -L agentbar-mock -f /dev/null capture-pane -p -e -t v    # -e keeps colors; plain -p to eyeball layout
tmux -L agentbar-mock -f /dev/null send-keys -t v G           # j/k/g/G navigate, Enter flashes the action
tmux -L agentbar-mock attach -t v                             # optional live-test; detach C-b d, then kill-server
```

`NewMockup` (`internal/ui/app.go`) is the sample fixture — when you change the layout, keep it representative:
every state (working/permission/asking/done/done-seen/idle) plus one multi-Claude-on-one-branch session.

## Layout

- `cmd/agentbar` — subcommands: `run`, `mockup`, `status`, `hook`
- `internal/hook` — event JSON → `@agent_*` pane options; `Decide()` is pure
- `internal/tmux` — exec wrapper, `list-panes -a` snapshot, branch cache, status segment
- `internal/ui` — Bubble Tea TUI: `app.go` (state, mouse, selection sync), `render.go` (blocks, highlight), `theme.go`
- `scripts/` — `toggle.sh` (global), `open.sh`, `follow.sh`, `resurrect-save.sh`, `common.sh` (shared helpers)
- `agentbar.tmux` — TPM entry point

## Rules

- Never touch the live tmux server. Tests and manual checks run on private sockets: `tmux -L <name> -f /dev/null`.
- Git branch is per worktree: one checkout = one branch, and multiple Claudes in the same worktree share it. The
  sidebar reads each pane's branch from its cwd and draws the branch headline once per run of consecutive
  same-branch agents (colored by the most-urgent when several share it). A session's panes usually sit in one
  worktree (so one branch), but tmux doesn't enforce that — don't assume one branch per session; just collapse the
  agents that actually match.
- Detection is hooks + pane options only — never scrape pane content.
- `hook` must never exit non-zero or block; Claude Code waits on it.
- Sidebar liveness is `#{pane_current_command} == agentbar` everywhere; never wrap the binary in a shell
  (breaks it).
- Mouse actions fire on release, not press.
- Comments: one short line, only for what the code can't say.
- After changing behavior, add or extend an e2e test that fails without the change.
- This project lives in the dotfiles monorepo at `apps/agentbar/` and loads via a `run-shell` line in
  `tmux/.tmux.conf` (not TPM) — it builds and runs straight from this checkout. Portability comes from the dotfiles
  themselves: `bootstrap.sh` installs Go and runs `make build`. Local changes reach a running sidebar via
  `make build` → reload tmux → restart (`prefix+e` twice); no push or clone-pull in the loop.

## Deploy

"Deploy" (aka "make it live", "get it working on my system") means: get the change running in the user's
**live tmux**, not just built. Do every step yourself except the last:

1. Commit to the dotfiles repo (push to its `main` only when the user asks).
2. `make build` here — the binary lands in `bin/`, which both the `run-shell` entry and the Claude hooks point at.
3. Verify the rebuilt binary headlessly (private-socket mockup) — never touch the live server.
4. Then tell the user to restart the sidebars: **`prefix + e` twice**. This is the one manual step — the sidebar
   is a long-lived process and this project never drives the live tmux server. Always state it explicitly.
