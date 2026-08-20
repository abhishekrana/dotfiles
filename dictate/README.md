# dictate

Toggle-key local speech-to-text into tmux. Tap a key to start, tap again to stop - the clip is transcribed offline with
[faster-whisper](https://github.com/SYSTRAN/faster-whisper) and typed into your active tmux pane. Fully local, zero
elevated privilege; CPU by default, and on the GPU once the opt-in backend is installed (see below).

## How it works

- A GNOME custom shortcut runs `dictate --toggle --send`. First press records the mic via `parec`; second press ships
  the clip to a local model server for transcription, injects the text with `tmux send-keys`, and presses Enter.
- Runs entirely in userspace - no sudo, no groups, no `/dev/*` access. tmux owns the pane's pty, so writing to it is
  ordinary I/O.
- **Lazy model server:** the first dictation spawns a background `dictate --serve` that loads faster-whisper once and
  keeps it resident, listening on a Unix socket in `$XDG_RUNTIME_DIR`. Later dictations reuse it and skip the ~1 s
  model-load - the toggle just streams PCM in and reads text back, so it never imports faster-whisper itself. The server
  self-exits after `DICTATE_IDLE` seconds of inactivity to free its RAM (~0.5-1 GB), and respawns on the next dictation.
  If the server can't be reached, the toggle falls back to loading the model in-process, so dictation always works. The
  mic is open only between toggle-on and toggle-off.
- **Auto-stop on silence:** while recording, a background `dictate --watch` samples the audio and stops for you after
  `DICTATE_SILENCE` seconds of trailing silence (only once you've actually spoken), or the `DICTATE_MAXSECS` cap. A
  second press still stops early; set `DICTATE_SILENCE=0` for manual-only.
- **Mutes other audio while recording:** the default output sink is muted on toggle-on and restored on toggle-off, so
  your speakers can't bleed into the mic; the prior state is saved so a crash can't leave it muted. `DICTATE_DUCK=0`
  disables it.
- Feedback & mouse control: a clickable `● dictate` segment sits dead-centre in the tmux status bar - grey idle, red
  recording, amber transcribing. Click it to toggle, same as the key. Same width in every state, so nothing shifts. A
  `⏎ send` button beside it submits - it presses Enter in the pane the transcript went to. Next to those,
  `● dictate+send` does the whole loop in one press-pair: it records, types the transcript, and presses Enter for you,
  so you never reach for the keyboard; it rests in teal, the row's one lit chip. A `⇡ commit+push` button types "commit
  and push" + Enter into the same pane, a one-click way to tell the agent to commit and push, and `◧ changes` opens the
  diff pane.

## Targeting

The transcript goes to the pane you're focused on if it's running Claude; otherwise a `claude` pane in your current
session; otherwise the most-recently-active `claude` pane. Check it with `dictate --target`. Force a specific pane with
`DICTATE_TMUX_TARGET` (a pane id like `%7`, or `session:win.pane`).

## Requirements

`uv`, `tmux`, and `parec` + `pactl` (both from `pulseaudio-utils`). First run downloads the model (~250 MB, cached).

`tmux` ships with the default bootstrap; the rest are opt-in, like the package itself. Install them in one step:

```bash
cd ~/dotfiles && ./bootstrap.sh dictate-deps
```

## Setup

```bash
dictate --install-shortcut    # bind the Copilot key to `dictate --toggle --send`
dictate --check               # verify parec + tmux, and show the target pane
```

One key is bound: the **Copilot key**, between AltGr and right Ctrl. Its string is `<Shift><Super>XF86TouchpadOff` - the
key emits `LeftMeta`+`LeftShift`+`F23`, and `KEY_F23`'s keycode carries the `XF86TouchpadOff` keysym, so `F23` does not
match. Pass keys to bind others; the dconf list ends up matching the arguments exactly.

## Usage

```bash
dictate --toggle              # start/stop
dictate --toggle --send       # same, but presses Enter once the transcript lands (the shortcut runs this)
dictate --target              # show which pane the transcript goes to
dictate --test                # record 5 s and print the transcript
dictate --serve-stop          # stop the model server (e.g. to pick up new config)
```

The server picks up its config (model, prompt, etc.) at spawn time, so after changing a `DICTATE_*` env var run
`dictate --serve-stop` (or wait for the idle timeout) so the next dictation starts a fresh server.

Dictated newlines are collapsed to spaces, so the speech itself can never submit a prompt - only `--send` presses Enter,
whether from the key, the `dictate+send` chip, or the `⏎ send` chip.

## Config (env vars)

| Var                   | Default        | Notes                                                       |
| --------------------- | -------------- | ----------------------------------------------------------- |
| `DICTATE_BACKEND`     | auto           | `gpu` when installed, else `cpu`                            |
| `DICTATE_MODEL`       | `small.en`     | see models below                                            |
| `DICTATE_COMPUTE`     | `int8`         | ctranslate2 compute type                                    |
| `DICTATE_IDLE`        | `300`          | seconds before the model server self-exits                  |
| `DICTATE_LANG`        | `en`           | language                                                    |
| `DICTATE_SOURCE`      | system default | PipeWire/Pulse source name                                  |
| `DICTATE_PROMPT`      | coding terms   | `initial_prompt` to bias vocabulary                         |
| `DICTATE_BEAM`        | `1`            | beam size (1 = fast, 5 = more accurate)                     |
| `DICTATE_LATENCY`     | `50`           | `parec` latency ms (low = flush promptly)                   |
| `DICTATE_SILENCE`     | `2.0`          | auto-stop after this much trailing silence (`0` = off)      |
| `DICTATE_MAXSECS`     | `120`          | hard cap on a single recording                              |
| `DICTATE_VAD_RMS`     | `300`          | int16 RMS above which audio counts as speech (tune per mic) |
| `DICTATE_DUCK`        | `1`            | mute other audio while recording (`0`/`off` disables)       |
| `DICTATE_TMUX_CMD`    | `claude`       | pane command treated as the target app                      |
| `DICTATE_TMUX_TARGET` | _(unset)_      | force a pane (pane id or `session:win.pane`)                |
| `DICTATE_TEST_SECS`   | `5`            | seconds recorded by `--test`                                |

Put per-machine overrides in `~/.bashrc.d/local.bash` (untracked), e.g. `export DICTATE_SOURCE=...`.

### Models

`small.en` is the CPU sweet spot (~95% of large-v3 accuracy at ~6× the speed). Alternatives: `base.en` (faster),
`distil-small.en` (fast English), `large-v3-turbo` (most accurate, slower on CPU - multilingual, so keep
`DICTATE_LANG=en`). English-only `.en` models beat the same-size multilingual model for English.

### GPU backend (`DICTATE_BACKEND=gpu`)

Whisper pads every clip to a 30-second window, so on CPU a two-second "yes, do that" costs the same as a long sentence.
Any Vulkan-capable GPU removes most of that - an AMD or Intel iGPU is enough. Installing it _is_ the switch; there is no
env var to set:

```sh
./install.sh whisper-vulkan          # apt deps + builds whisper.cpp with Vulkan + model (~260MB)
dictate --serve-stop                 # the running server still holds the CPU backend
```

`DICTATE_BACKEND` is resolved from what is installed, not from the environment, because the two launchers that matter -
the GNOME shortcut and the tmux status chip - never source `~/.bashrc.d`, so an export there would reach only a fresh
interactive shell. Set `DICTATE_BACKEND=cpu` to force the CPU back.

The two backends are named for the hardware, not the projects behind them (`gpu` is whisper.cpp via Vulkan, `cpu` is
[faster-whisper](https://github.com/SYSTRAN/faster-whisper)) - "faster-whisper" is upstream's name for being quicker
than OpenAI's reference implementation, and using it as a backend label read as a claim about _this_ machine, where it
is the slower of the two. The old values still work.

**The GPU is an optimisation, never a dependency.** Nothing here is AMD-specific: `mesa-vulkan-drivers` covers Intel and
AMD, NVIDIA works through its own ICD, and the install step checks a real GPU is visible (`llvmpipe`, Mesa's software
rasterizer, does not count) before spending minutes compiling. `bootstrap.sh` never runs it, so a machine that skips it
simply dictates on the CPU. And if whisper.cpp cannot start at runtime - driver gone, GPU disabled in a VM, model
truncated - dictation falls back to faster-whisper and says so, rather than failing.

Measured on a Radeon 860M (RDNA 3.5), warm server, five dictated clips of 16-24s (102s of audio in total):

| Backend                           | Total | Notes                                          |
| --------------------------------- | ----- | ---------------------------------------------- |
| `gpu` - `small.en-q8_0` (default) | 3.6s  | 2.7× faster than the CPU backend               |
| `cpu` - `small.en` faster-whisper | 9.7s  | the fallback when whisper.cpp is not installed |
| `gpu` - `large-v3-turbo-q5_0`     | 10.7s | slowest, no better on the terms that mattered  |

`large-v3-turbo` is the interesting negative result: on real-length clips it is slower than the CPU it was meant to
beat, and it read the same technical vocabulary no better. Fetch it (`ggml-large-v3-turbo-q5_0.bin`, ~570MB) and set
`DICTATE_WHISPERCPP_MODEL` if your own audio disagrees - noisy input or unusual proper nouns are where it should win.

| Var                        | Default                                                    |
| -------------------------- | ---------------------------------------------------------- |
| `DICTATE_WHISPERCPP_BIN`   | `~/.local/bin/whisper-server`                              |
| `DICTATE_WHISPERCPP_MODEL` | `~/.local/share/whisper-cpp/models/ggml-small.en-q8_0.bin` |
| `DICTATE_WHISPERCPP_PORT`  | `8178`                                                     |

The step installs what a fresh Ubuntu lacks (`cmake`, `glslc`, `libvulkan-dev`, `mesa-vulkan-drivers`, `vulkan-tools`)
and stops before compiling if no GPU is visible. It is **not** worth using on CPU - whisper.cpp's CPU build measured
slower than faster-whisper's, so the CPU default stays `faster-whisper`.

## Uninstall

```bash
cd ~/dotfiles && stow -D dictate
# remove the GNOME shortcut in Settings ▸ Keyboard ▸ Custom Shortcuts
```
