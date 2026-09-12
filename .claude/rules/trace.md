---
paths:
  - "tmux/**"
  - "trace/**"
  - "apps/agentbar/**"
  - "clip/**"
  - "dictate/**"
---

# Reading the trace log

Always on, action _edges_ only, across the whole interactive stack.
`dotfiles-trace show --since 5m --src <tmux|agentbar|clip|hook|sidebar|picker|dictate|resurrect|yank> --grep <pat>`;
`tail -f` follows. logfmt, `ts=<iso ms> src= evt= pid= k=v`; the status clock is `%H:%M:%S`, so a screenshot anchors to
a log window.

By symptom:

- **A click did nothing.** `src=tmux evt=click range=…` means tmux received it, so the failure is downstream - our bug.
  No line at all means the terminal dropped it first: the known Ghostty+tmux status-click bug, not fixable here.
- **A copy did nothing.** Read in order: no `src=yank` line means tmux never fired the yank (the selection or the
  binding); `bytes=0` is an empty drag; non-zero `rc` is the backend, with `wl=`/`dsp=` saying why. A long-lived tmux
  server keeps the `WAYLAND_DISPLAY` it started with, so after a re-login `wl-copy` cannot reach the compositor until
  the server is restarted.
- **The layout drifted.** tmux takes a shrink evenly from every pane and has no fixed-size pane; `src=sidebar evt=pin`
  is the `window-resized` hook fixing the width. `prefix + R` logs `evt=reset … changed=N floated=N`, and a `floated=`
  above zero is windows it skipped whole, since a float counts as a column to `select-layout`.
- **The ≡ workdesk chip did nothing.** `src=tmux evt=workdesk action=open|close rc=…`. `action=close` with no float on
  screen means the match found someone else's floating `workdesk`; `err=no_command` means the option is unset.
- **A session jump landed wrong.** `src=agentbar evt=switch`. No line means the binary never ran and the binding fell
  through to tmux's alphabetical `switch-client` - rebuild it.
- **The sidebar state looks stale.** `src=hook evt=event name=… prev=… new=…` is ground truth (`via=cwd` means the pane
  was recovered by the cwd fallback). `agentbar doctor` rolls this into a per-pane health check.
- **The diff pane shows the wrong tree.** `src=hook evt=workdir` is every move of an agent's worktree; absent means the
  agent has only read files, or a hook is not wired. `src=tmux evt=diff action=create|respawn|follow` is what the pane
  was pointed at.

Writing to it:

- **Two writers, one format**: the `dotfiles-trace` CLI (all shell/tmux callers) and Go `apps/agentbar/internal/trace`.
  Keep them in sync on timestamp, escaping and rotation.
- One `dotfiles-trace log <src> <evt> k=v …` per action edge, reusing an existing `src`. **Never in a hot loop** - mouse
  motion, ticks, status redraws, the dictate silence poll, fzf preview/list, the statusline.
- Log the outcome, not the intent: `rc=`, `bytes=` are what separate "it ran" from "it worked". `before=`/`after=` on
  one line, only when something actually changed. Values cap at 200 chars.
- Tests run under `DOTFILES_TRACE=0`, never into the live log. `@agentbar-trace-verbose on` adds the noisy sidebar
  events for a live hunt.
