# CLAUDE.md

Toggle-key local speech-to-text into tmux. See `README.md` for usage and config; this file is for hacking on the script.
Keep it a single file - do not split it into a package.

## Backends (`DICTATE_BACKEND`)

Two, and the default never changes without asking: **`faster-whisper`** (default) runs `small.en` int8 on the CPU,
in-process. **`whispercpp`** runs whisper.cpp against Vulkan on the AMD iGPU, which is what makes `large-v3-turbo`
affordable - measured on a Radeon 860M, warm, 3s clip: `small.en` CPU 2455ms, `large-v3-turbo` GPU ~2100ms, `small.en`
GPU ~400ms. Install it with `./install.sh whisper-vulkan`, switch with `DICTATE_BACKEND=whispercpp`.

- **Keep faster-whisper as the CPU path.** whisper.cpp's own CPU build measured _slower_ than faster-whisper here
  (3143ms vs 2455ms on the same clip) - CTranslate2's int8 kernels win on CPU. The GPU is the only reason to switch.
- whisper.cpp has no usable in-process binding, so `whispercpp` holds a **`whisper-server` child** and posts each clip
  to it over loopback. Per-call `whisper-cli` was 410ms slower - process start plus Vulkan context setup, every time.
  The child must die with `serve()` (it holds the GPU and the port), which is what `_bye()` handles.
- The multipart POST is built by hand from stdlib: the script keeps its two PEP 723 deps and adds no HTTP library.

## Packaging (why it is a stow package, not an app)

- One self-contained Python script run via `#!/usr/bin/env -S uv run --script`; deps are declared inline (PEP 723:
  `faster-whisper`, `numpy`). There is nothing to build, so it is **not** an `apps/` build target. The `whispercpp`
  backend's binary is a downloaded tool, so it is pinned and built in `install.sh` like every other one - not here.
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
  shortcut, `--serve`/`--watch` DEVNULL). The `trace(evt, **kv)` helper is the only durable record: it fires edges
  (`toggle`, `rec`, `transcribe`, `send`, `result` with outcome/chars/ms) into the shared dotfiles trace log
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
