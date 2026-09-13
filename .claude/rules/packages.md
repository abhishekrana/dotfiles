---
paths:
  - "claude/**"
  - "clip/**"
  - "dictate/**"
  - "hunk/**"
  - "leaf/**"
  - "yazi/**"
---

# Per-package pitfalls

- **`claude/`** - the status line names the session's place, and the `⚠` names the other place: the worktree Claude last
  wrote in, shown only while its root differs. Roots are compared, never paths, so a subdirectory is not a move.
  **Nothing in the row depends on tmux** - the payload never says which file was written, so `statusline-workdir.sh`
  (this package's `PostToolUse` hook) records the edited file's repo root in
  `$XDG_STATE_HOME/dotfiles/claude-workdir/<session id>` and the row reads that. `refreshInterval` is 3s because no
  event fires when a hook writes that file. `test/statusline.sh` is the guard, tmux off PATH. The second row's rate
  limits ride the same stdin payload, so they cost no process and no network; each window is absent before the session's
  first API response and after its own reset, and an absent window shows nothing.
- **`claude/` has three writers**: this repo, `herdr integration install claude`, and Claude's own `/theme`. Its TUI
  theme is therefore **deliberately not switched by `theme`**. Prefer `light-ansi`/`dark-ansi`, which paint from the
  terminal's own 16 colours and so follow this palette; the tracked value drifts to whatever `/theme` last wrote, which
  is the open question in `TODO.md`. Claude Code does not load a user-level `settings.local.json`, so anything that must
  take effect goes in `settings.json`.
- **`clip/`** - every copy path (tmux `copy-command`, `tmux-yank.sh`, fzf's Ctrl-Y, nvim) goes through it, so the
  backend is chosen in one place.
- **`dictate/`** - has its own nested `CLAUDE.md`; read it first. Backends are named for the hardware and picked by what
  is installed, never an env var. The model is not prefetched, so the first dictation downloads it.
- **`hunk/`** - `mode = "stack"` is deliberate: full width per line for the diff pane beside your work. Workdesk's `D`
  overrides it to `split` at the call, because an MR diff gets a window of its own. hunk reads the file at startup, so
  an open pane keeps its layout until respawned.
- **`leaf/`** - **leaf writes this file itself**; a first run with no config seeds upstream's sample there, which blocks
  `stow leaf`, so the backup step in `bootstrap.sh` is load-bearing. It carries a Solarized Light palette because leaf
  ships only the dark one.
- **`yazi/`** - `package.toml` is machine-managed: commit exactly what `ya` writes and never comment it. `zoxide` is
  bundled in yazi core; listing it as a dep fails.
