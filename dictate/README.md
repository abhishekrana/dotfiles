# dictate

Toggle-key local speech-to-text into tmux. Tap a key to start, tap again to stop — the clip is transcribed offline
with [faster-whisper](https://github.com/SYSTRAN/faster-whisper) and typed into your active tmux pane. CPU-only, fully
local, zero elevated privilege.

## How it works

- A GNOME custom shortcut runs `dictate --toggle`. First press records the mic via `parec`; second press transcribes
  with faster-whisper and injects the text with `tmux send-keys`.
- Runs entirely in userspace — no sudo, no groups, no `/dev/*` access. tmux owns the pane's pty, so writing to it is
  ordinary I/O.
- Daemonless: each press is a short-lived invocation. The mic is open only between toggle-on and toggle-off.

## Targeting

The transcript goes to the pane you're focused on if it's running Claude; otherwise a `claude` pane in your current
session; otherwise the most-recently-active `claude` pane. Check it with `dictate --target`. Force a specific pane with
`DICTATE_TMUX_TARGET` (a pane id like `%7`, or `session:win.pane`).

## Requirements

`uv`, `tmux`, and `parec` (from `pipewire-pulse`/`pulseaudio-utils`). `notify-send` (libnotify) is optional, for
recording feedback. First run downloads the model (~250 MB, cached).

## Setup

```bash
dictate --install-shortcut    # bind Super+\ to `dictate --toggle`
dictate --check               # verify parec + tmux, and show the target pane
```

## Usage

```bash
dictate --toggle              # start/stop (this is what the shortcut runs)
dictate --target              # show which pane the transcript goes to
dictate --test                # record 5 s and print the transcript
```

Dictated newlines are collapsed to spaces, so speech never submits a prompt — you press Enter yourself.

## Config (env vars)

| Var                   | Default        | Notes                                        |
| --------------------- | -------------- | -------------------------------------------- |
| `DICTATE_MODEL`       | `small.en`     | see models below                             |
| `DICTATE_COMPUTE`     | `int8`         | ctranslate2 compute type                     |
| `DICTATE_LANG`        | `en`           | language                                     |
| `DICTATE_SOURCE`      | system default | PipeWire/Pulse source name                   |
| `DICTATE_PROMPT`      | coding terms   | `initial_prompt` to bias vocabulary          |
| `DICTATE_BEAM`        | `1`            | beam size (1 = fast, 5 = more accurate)      |
| `DICTATE_LATENCY`     | `50`           | `parec` latency ms (low = flush promptly)    |
| `DICTATE_NOTIFY`      | `1`            | desktop notifications on record/transcribe   |
| `DICTATE_TMUX_CMD`    | `claude`       | pane command treated as the target app       |
| `DICTATE_TMUX_TARGET` | _(unset)_      | force a pane (pane id or `session:win.pane`) |
| `DICTATE_TEST_SECS`   | `5`            | seconds recorded by `--test`                 |

Put per-machine overrides in `~/.bashrc.d/local.bash` (untracked), e.g. `export DICTATE_SOURCE=...`.

### Models

`small.en` is the CPU sweet spot (~95% of large-v3 accuracy at ~6× the speed). Alternatives: `base.en` (faster),
`distil-small.en` (fast English), `large-v3-turbo` (most accurate, slower on CPU — multilingual, so keep
`DICTATE_LANG=en`). English-only `.en` models beat the same-size multilingual model for English.

## Uninstall

```bash
cd ~/dotfiles && stow -D dictate
# remove the GNOME shortcut in Settings ▸ Keyboard ▸ Custom Shortcuts
```
