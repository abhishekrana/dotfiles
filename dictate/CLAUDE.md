# CLAUDE.md

Toggle-key local speech-to-text into tmux. See `README.md` for usage and config; this file is for hacking on the script.
Keep it a single file - do not split it into a package.

## Backends (`DICTATE_BACKEND`)

Two, named for the hardware and resolved from what is installed rather than from the environment: **`gpu`** runs
`small.en` through whisper.cpp against Vulkan on whatever GPU is present when `./install.sh whisper-vulkan` has been
run, else **`faster-whisper`** runs `small.en` int8 on the CPU, in-process. Measured on a Radeon 860M over five dictated
clips (102s of audio): GPU `small.en` **3.6s** total, CPU `small.en` **9.7s**, GPU `large-v3-turbo` **10.7s**. Install
with `./install.sh whisper-vulkan`; `DICTATE_BACKEND=cpu` forces the CPU back, and the old `whispercpp`/`faster-whisper`
values still resolve.

- **The backend is detected, not configured, on purpose.** The GNOME shortcut and the tmux status chip both launch
  `dictate` without sourcing `~/.bashrc.d`, so an env var set there reaches only a fresh interactive shell - the path
  used least. Having installed the binary and model is the opt-in signal; do not replace this with an env var.
- **The GPU is an optimisation, never a dependency, and nothing here is AMD-specific.** File existence only says what to
  _try_: `load_model()` falls back to faster-whisper when whisper.cpp will not start, and every caller dispatches on the
  handle (`is_whispercpp()`) rather than on `BACKEND`, so the fallback routes itself. Keep it that way - a machine with
  no GPU, a broken driver or a VM without passthrough must still dictate.
- **Strip whisper.cpp's non-speech tokens.** It narrates silence as `[BLANK_AUDIO]`, `[END]`, `[SOUND]`; faster-whisper
  returns `""`, and a stray toggle must not type a token into the pane. The rule is shape (square-bracketed all-caps),
  not a name list, because the set is open-ended - but parentheses are dictated, so only known words go there.
- **`large-v3-turbo` lost on both axes** - slower than the CPU backend on real-length clips, and no better on the
  technical vocabulary. Do not assume the bigger model wins here; re-measure before switching to one.
- **The prompt moves accuracy more than the model does, in both directions.** Terms in `DICTATE_PROMPT` come out right;
  terms absent from it come out as "work tree", ".files", "source reuse port". But each entry is also a word that can be
  hallucinated into unclear audio: adding `origin/main` turned "diff pane" into "diff_main". **Add a term only after
  hearing it fail, then re-run the clips and check nothing common broke** - a vocabulary dump trades frequent words for
  rare ones. Prompting a term is not a guarantee either: `agentbar` never displaced "agent sidebar".
- **The prompt's word order is load-bearing, and is not alphabetical.** Sorting the same terms turned "task check" into
  "taskcheck" and appended a stray "merge request"; proven-failing terms first and the loosest term last measured clean.
  Do not tidy it into alphabetical order without re-running the clips.

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
  `dictate --send` (⏎ send), `dictsend` -> `dictate --toggle --send` (records, types, then presses Enter), and `push` ->
  `dictate --type 'commit and push'` (types the phrase + Enter). Renaming a state string or a range means editing
  `tmux.conf` too. The `@*_seg` chips are also regenerated per-flavor by `theme/.local/bin/theme` (`apply_tmux`), so
  relabel/recolor a chip in both. Chip colors also appear in `design/*.md`.

## Footer chips

- Five ranges, in this order: `dictate` · `submit` · `dictsend` · `push` · `diff`. **One space of padding inside every
  chip and one between them** - every gap is the same three columns. The old row mixed one- and two-space padding and
  read as ragged; keep it uniform when editing a label.
- **Labels name the effect, not the key.** `⏎ send` rather than "enter", `◧ changes` rather than "diff", because a
  status bar has no tooltips and a first-time reader gets one chance. Glyphs alone were rejected: a bare glyph is a 2-3
  column click target, too small to hit.
- `dictate+send` is one press-pair, not a second recorder: the first press drops `SEND_FILE` beside the PCM buffer and
  the second reads it _before_ `cleanup()` removes it, then presses Enter once the transcript is in the pane. Only the
  chip you clicked lights up - both read `@dictate`, but each colours itself only when `@dictate_src` names it.
- **One chip rests lit, and it is `dictate+send`** (`working` teal); `dictate` and `⏎ send` rest grey. It is the one
  used most, and grey reads as unavailable. Never a warm hue: the chip's own states are red (recording) and amber
  (transcribing), so orange blurs idle into busy. To recolor, move the highlight - do not add a second.
- **Hover is impossible** - tmux 3.7b rejects `MouseMoveStatus`; only Down/Up/Drag/Wheel exist for the status line.

## Key binding

- **One shortcut: the Copilot key, running `--toggle --send`.** `--install-shortcut` resets every `dictate*` keybinding
  it does not install, so the dconf list matches its arguments exactly.
- **The string is `<Shift><Super>XF86TouchpadOff`, not `F23`.** The key emits `LeftMeta`+`LeftShift`+`F23`, and
  `KEY_F23`'s keycode carries the `XF86TouchpadOff` keysym, so `F23` does not match. GNOME's static touchpad-off grab is
  on the bare keysym, which the held modifiers do not match either - the touchpad is unaffected.
- **The numpad's Backspace and `=` cannot be bound.** They emit the main-row scancodes (`0xe`, `0xd`), so no layer -
  hwdb, keyd, xkb, GNOME - can distinguish them.

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
