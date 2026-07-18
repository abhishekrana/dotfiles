# dictate

Toggle-key local speech-to-text into tmux. Tap a key to start, tap again to stop - the clip is transcribed offline
with [faster-whisper](https://github.com/SYSTRAN/faster-whisper) and typed into your active tmux pane. CPU-only, fully
local, zero elevated privilege.

## How it works

- A GNOME custom shortcut runs `dictate --toggle`. First press records the mic via `parec`; second press ships the
  clip to a local model server for transcription and injects the text with `tmux send-keys`.
- Runs entirely in userspace - no sudo, no groups, no `/dev/*` access. tmux owns the pane's pty, so writing to it is
  ordinary I/O.
- **Lazy model server:** the first dictation spawns a background `dictate --serve` that loads faster-whisper once and
  keeps it resident, listening on a Unix socket in `$XDG_RUNTIME_DIR`. Later dictations reuse it and skip the ~1 s
  model-load - the toggle just streams PCM in and reads text back, so it never imports faster-whisper itself. The
  server self-exits after `DICTATE_IDLE` seconds of inactivity to free its RAM (~0.5-1 GB), and respawns on the next
  dictation. If the server can't be reached, the toggle falls back to loading the model in-process, so dictation always
  works. The mic is open only between toggle-on and toggle-off.
- **Auto-stop on silence:** while recording, a background `dictate --watch` samples the audio and stops for you after
  `DICTATE_SILENCE` seconds of trailing silence (only once you've actually spoken), or the `DICTATE_MAXSECS` cap. A
  second press still stops early; set `DICTATE_SILENCE=0` for manual-only.
- **Mutes other audio while recording:** the default output sink is muted on toggle-on and restored on toggle-off, so
  your speakers can't bleed into the mic; the prior state is saved so a crash can't leave it muted. `DICTATE_DUCK=0`
  disables it.
- Feedback & mouse control: a clickable `● dictate` segment sits dead-centre in the tmux status bar - grey idle, red
  recording, amber transcribing. Click it to toggle, same as the key. Same width in every state, so nothing shifts. A
  `⏎` button beside it submits - it presses Enter in the pane the transcript went to, so the whole loop is mouse-only.

## Targeting

The transcript goes to the pane you're focused on if it's running Claude; otherwise a `claude` pane in your current
session; otherwise the most-recently-active `claude` pane. Check it with `dictate --target`. Force a specific pane with
`DICTATE_TMUX_TARGET` (a pane id like `%7`, or `session:win.pane`).

## Requirements

`uv`, `tmux`, and `parec` (from `pipewire-pulse`/`pulseaudio-utils`). First run downloads the model (~250 MB, cached).

## Setup

```bash
dictate --install-shortcut    # bind Super+\ and Super+Z to `dictate --toggle`
dictate --check               # verify parec + tmux, and show the target pane
```

## Usage

```bash
dictate --toggle              # start/stop (this is what the shortcut runs)
dictate --target              # show which pane the transcript goes to
dictate --test                # record 5 s and print the transcript
dictate --serve-stop          # stop the model server (e.g. to pick up new config)
```

The server picks up its config (model, prompt, etc.) at spawn time, so after changing a `DICTATE_*` env var run
`dictate --serve-stop` (or wait for the idle timeout) so the next dictation starts a fresh server.

Dictated newlines are collapsed to spaces, so speech never submits a prompt - you press Enter yourself (or click the
`⏎` button in the status bar).

## Config (env vars)

| Var                   | Default        | Notes                                                       |
| --------------------- | -------------- | ----------------------------------------------------------- |
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

## Uninstall

```bash
cd ~/dotfiles && stow -D dictate
# remove the GNOME shortcut in Settings ▸ Keyboard ▸ Custom Shortcuts
```
