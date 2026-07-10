# dictate

Push-to-talk local speech-to-text for Wayland (GNOME), CPU-only. Hold a key, speak, release — the clip is transcribed
offline with [faster-whisper](https://github.com/SYSTRAN/faster-whisper) and typed into the focused window.
No cloud, no daemon, no clipboard.

## How it works

- A single `uv` script (`~/.local/bin/dictate`, PEP 723 inline deps — nothing to `pip install`).
- Watches the keyboard at the evdev layer for the push-to-talk key (default **Right Ctrl**, harmless when pressed
  alone).
- While held, records the mic via `parec`; on release, transcribes with faster-whisper and injects the text through a
  synthetic **uinput** keyboard (the reliable route on GNOME Wayland, where `wtype`'s virtual-keyboard protocol is
  unsupported and packaged `ydotool` is stale).
- The mic is opened **only while the key is held** — idle captures nothing. It is a foreground process you launch per
  session; there is **no autostart / systemd service** by design.

## Requirements

- `uv`, and `parec` (from `pipewire-pulse`/`pulseaudio-utils`, default on Ubuntu desktop). First run compiles `evdev`
  and downloads the model (~250 MB, cached).

## One-time setup (opt-in, per machine)

```bash
dictate-setup          # adds you to 'input', installs a udev rule for /dev/uinput
# log out and back in  (activates the 'input' group)
dictate --check        # expect: uinput writable, keyboard(s) found, parec present
```

`dictate-setup` is intentionally **not** run by `bootstrap.sh`: it changes your security posture and needs a re-login.
See the security note below.

## Usage

```bash
dictate                # start listening; focus a window, hold Right Ctrl, speak, release
dictate --test         # record 5 s and print the transcript (no sudo needed)
dictate --check        # verify permissions/devices
```

Dictated newlines are collapsed to spaces so speech can never auto-submit a prompt — you press Enter yourself.

## Config (env vars)

| Var                 | Default         | Notes                                     |
| ------------------- | --------------- | ----------------------------------------- |
| `DICTATE_KEY`       | `KEY_RIGHTCTRL` | evdev key to hold (e.g. `KEY_SCROLLLOCK`) |
| `DICTATE_MODEL`     | `small.en`      | see models below                          |
| `DICTATE_COMPUTE`   | `int8`          | ctranslate2 compute type                  |
| `DICTATE_LANG`      | `en`            | language                                  |
| `DICTATE_SOURCE`    | system default  | PipeWire/Pulse source name                |
| `DICTATE_PROMPT`    | coding terms    | `initial_prompt` to bias vocabulary       |
| `DICTATE_BEAM`      | `1`             | beam size (1 = fast, 5 = more accurate)   |
| `DICTATE_LATENCY`   | `50`            | `parec` latency ms (low = flush promptly) |
| `DICTATE_TEST_SECS` | `5`             | seconds recorded by `--test`              |

Put per-machine overrides in `~/.bashrc.d/local.bash` (untracked), e.g. `export DICTATE_SOURCE=...`.

### Models

`small.en` is the CPU sweet spot (~95% of large-v3 accuracy at ~6× the speed). Alternatives: `base.en` (faster),
`distil-small.en` (fast English), `large-v3-turbo` (most accurate, slower on CPU — multilingual, so keep
`DICTATE_LANG=en`). English-only `.en` models beat the same-size multilingual model for English.

### Transcription tuning

`condition_on_previous_text=False` (the key anti-hallucination flag for dictation) plus `vad_filter=True` and
faster-whisper's default logprob/compression/no-speech thresholds are enabled in the script.

## Security note

Membership in the `input` group lets any process you run read the raw keystroke stream (`/dev/input/event*`) and
synthesize keystrokes (`/dev/uinput`) — a real keylogger/injection surface. That is why enablement is opt-in per
machine rather than part of `bootstrap.sh`. Only run `dictate-setup` on a machine where you accept that trade-off.

## Uninstall

```bash
cd ~/dotfiles && stow -D dictate
sudo rm -f /etc/udev/rules.d/99-uinput-dictate.rules /etc/modules-load.d/uinput.conf
sudo gpasswd -d "$USER" input     # optional: leave the input group
```
