---
paths:
  - "claude/**"
  - "clip/**"
  - "dictate/**"
  - "herdr/**"
  - "hunk/**"
  - "leaf/**"
  - "yazi/**"
---

# Per-package pitfalls

- **`claude/`** - the status line names the session's place, and the `⚠` names the other place: the worktree Claude last
  wrote in, shown only while its root differs. Roots are compared, never paths, so a subdirectory is not a move.
  **Nothing in the row depends on tmux** - the payload never says which file was written, so `statusline-workdir.sh`
  (this package's `PostToolUse` hook) records the edited file's repo root in
  `$XDG_STATE_HOME/dotfiles/claude-workdir/<session id>` and the row reads that. `refreshInterval` is 1s: no event fires
  when a hook writes that file, and the dictation chip has to light while you are still talking. That cadence is only
  affordable because the script forks nothing - helpers write a named global instead of printing into a `$( )` subshell,
  and jq reading the payload is the one process a run starts (5ms; a subshell per segment cost 14ms). The branch's
  ticket, MR and pipeline are herdr's tab bar (`herdr/`), not a row here. `test/statusline.sh` is the guard, tmux off
  PATH. The second row's rate limits ride the same stdin payload, so they cost no process and no network; each window is
  absent before the session's first API response and after its own reset, and an absent window shows nothing. The
  dictation chip reads the herdr plugin's state file
  (`$XDG_STATE_HOME/herdr/plugins/abhishekrana.dictate/recording.json`) and never writes it, so polling cannot disturb a
  recording; a pid with no process is a recorder that died, not a recording. Its label never changes, only its colour -
  grey idle, red recording, amber transcribing, the tmux footer chip's rule - because the meters sit beside it and must
  not shift as you speak.
- **`claude/` has three writers**: this repo, `herdr integration install claude`, and Claude's own `/theme`. Its TUI
  theme is therefore **deliberately not switched by `theme`**. Prefer `light-ansi`/`dark-ansi`, which paint from the
  terminal's own 16 colours and so follow this palette; the tracked value drifts to whatever `/theme` last wrote, which
  is the open question in `TODO.md`. Claude Code does not load a user-level `settings.local.json`, so anything that must
  take effect goes in `settings.json`.
- **`clip/`** - every copy path (tmux `copy-command`, `tmux-yank.sh`, fzf's Ctrl-Y, nvim) goes through it, so the
  backend is chosen in one place.
- **`dictate/`** - has its own nested `CLAUDE.md`; read it first. Backends are named for the hardware and picked by what
  is installed, never an env var. The model is not prefetched, so the first dictation downloads it.
- **`herdr/`** - `herdr-forge line` is a `tab_bar_right` command entry in `~/.config/herdr/config.toml`, which stays
  untracked because `herdr-dictate setup` appends to it. Herdr strips colour from that entry, so state is words and
  glyphs. `line` runs every 1s (about 7 ms) and forks only git; GitLab is one GraphQL call per branch per TTL, detached.
  Pass the branch as `-f b=<name>`: glab's `-F 'b[]=…'` form drops the filter and returns the project's newest MR. Herdr
  cannot make tab bar text clickable, so `herdr-forge open` hands off to `herdr-forge-popup` (stdlib Python), an `alt+u`
  popup with `prefix+u` as a fallback (`[[keys.command]]`, same file): ticket, MR and pipeline in three columns, stacked
  below 156 columns, each with Browser, Split and New tab buttons. New tab names the herdr tab for what it shows
  (`diff !45`) and focuses it rather than opening a second; only Browser and Split have keys. A popup hands every click
  to its program and never to herdr's Ctrl+click link handling (v0.9.1), so it reads SGR mouse reports and maps the
  clicked row and column itself. Its GraphQL is two queries asked in parallel, because one exceeds GitLab's complexity
  limit of 250. Most branches carry no ticket number, so the ticket falls back to what the MR closes. MR pipelines run
  on `refs/merge-requests/N/merge`, so the pipeline opens with `glab ci view -p <id>`, never `-b <branch>`; the MR diff
  is live: `hunk diff <merge-base> --watch`, the working tree against where the branch left its target, with `--sidebar`
  added only in its own tab (hunk's config keeps the file list off). Never pipe a patch into hunk there: piped input is
  pager mode, which drops the file list and cannot watch. `test/herdr-forge.sh` stubs glab, herdr and xdg-open, and
  drives the popup through `--dump` and `--act`.
- **`hunk/`** - `mode = "stack"` is deliberate: full width per line for the diff pane beside your work. Workdesk's `D`
  overrides it to `split` at the call, because an MR diff gets a window of its own. hunk reads the file at startup, so
  an open pane keeps its layout until respawned.
- **`leaf/`** - **leaf writes this file itself**; a first run with no config seeds upstream's sample there, which blocks
  `stow leaf`, so the backup step in `bootstrap.sh` is load-bearing. It carries a Solarized Light palette because leaf
  ships only the dark one.
- **`yazi/`** - `package.toml` is machine-managed: commit exactly what `ya` writes and never comment it. `zoxide` is
  bundled in yazi core; listing it as a dep fails.
