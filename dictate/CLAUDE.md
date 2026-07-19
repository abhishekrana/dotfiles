# CLAUDE.md

Toggle-key local speech-to-text into tmux (faster-whisper, CPU-only). See `README.md` for usage and config; this file
is for hacking on the script. Keep it a single file - do not split it into a package.

## Packaging (why it is a stow package, not an app)

- One self-contained Python script run via `#!/usr/bin/env -S uv run --script`; deps are declared inline (PEP 723:
  `faster-whisper`, `numpy`). There is nothing to build, so it is **not** an `apps/` build target.
- It is a **stow package** -> `~/.local/bin/dictate` (a symlink). It must stay on PATH under the short name `dictate`,
  because the GNOME shortcut, `tmux/.tmux.conf` (`$HOME/.local/bin/dictate`), and the script's own self-re-exec
  (`shutil.which("dictate")`, used to spawn `--serve`/`--watch`) all resolve it by that name. Stow is what provides it;
  moving to `apps/` (run-in-place) would only reintroduce a symlink.

## Process model

- The **toggle client** never imports faster-whisper. It ships raw PCM to a model server and reads text back.
- **`--serve`**: a lazy model server, spawned on first dictation, holding the model resident on a Unix socket in
  `$XDG_RUNTIME_DIR`. Self-exits after `DICTATE_IDLE`s. Binding the socket only after the model loads doubles as the
  readiness signal (connect == ready).
- **`--watch PID`**: a silence-watcher subprocess that auto-stops recording after trailing silence / the cap.
- If the server is unreachable the client falls back to loading the model **in-process**, so dictation always works.

## tmux coupling (change both sides together)

- The script sets `@dictate` to `"rec"` (red) / `"work"` (amber) / unset (idle grey). `tmux/.tmux.conf` renders that via
  `@dictate_seg`, and its status bar defines the mouse ranges `dictate` -> `dictate --toggle`, `submit` ->
  `dictate --send` (⏎ enter), and `push` -> `dictate --type 'commit and push'` (types the phrase + Enter). Renaming a
  state string or a range means editing `tmux.conf` too. The `@*_seg` chips are also regenerated per-flavor by
  `theme/.local/bin/theme` (`apply_tmux`), so relabel/recolor a chip in both. Chip colors also appear in `design/*.md`.

## Tracing

- `log()` prints to stderr, which is **discarded** in every real launch context (tmux `run-shell -b`, the GNOME
  shortcut, `--serve`/`--watch` DEVNULL). The `trace(evt, **kv)` helper is the only durable record: it fires
  edges (`toggle`, `rec`, `transcribe`, `send`, `result` with outcome/chars/ms) into the shared dotfiles trace log
  (`dotfiles-trace`). Call it at action edges only - **never** in `watch()` (150ms poll) or the `--serve` loop. View
  with `dotfiles-trace show --src dictate`; see the repo-root CLAUDE.md "Debugging" section.

## Config, deploy, smoke test

- All config is `DICTATE_*` env vars, read at **process start**. The server captures its config at spawn, and a running
  server holds the **old code** - so after editing config _or_ the script, run `dictate --serve-stop` so the next
  dictation starts a fresh server. The stowed symlink itself is live the moment the file is saved.
- Audio ducking mutes the default sink via `pactl` while recording; `DUCK_FILE` persists the prior mute state so a crash
  cannot strand it muted.
- Smoke test: `dictate --check` (parec + tmux + server state), `dictate --test` (record 5s, print transcript). Clean-env
  install of the deps: `./bootstrap.sh dictate-deps` (see `bootstrap.sh`).
