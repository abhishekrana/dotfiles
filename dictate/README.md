# dictate

Toggle-key local speech-to-text for Wayland (GNOME), CPU-only. Tap a key to start, tap again to stop — the clip is
transcribed offline with [faster-whisper](https://github.com/SYSTRAN/faster-whisper) and typed into the focused window.
No cloud, no daemon, no clipboard.

## How it works

- A GNOME custom shortcut runs `dictate --toggle`. First press records the mic via `parec` (to a temp file); second
  press stops, transcribes with faster-whisper, and injects the text through a synthetic **uinput** keyboard.
- **Daemonless:** each press is a short-lived `dictate --toggle` invocation — there is no resident process and no
  autostart. The mic is open only between toggle-on and toggle-off.
- **No `/dev/input` access:** because GNOME owns the trigger key, nothing here reads raw input. There is no keylogger
  surface and no `input` group — the only elevated capability is write to `/dev/uinput` (for typing), via a dedicated
  `uinput` group. See the security note.
- uinput is the reliable injection path on GNOME Wayland (`wtype`'s virtual-keyboard protocol is unsupported there and
  packaged `ydotool` is stale).

## Requirements

- `uv`, `parec` (from `pipewire-pulse`/`pulseaudio-utils`, default on Ubuntu desktop), and `gsettings` (GNOME).
  `notify-send` (libnotify) is optional, for on-screen recording feedback. First run compiles `evdev` and downloads the
  model (~250 MB, cached).

## One-time setup (opt-in, per machine)

```bash
dictate-setup                 # sudo: dedicated 'uinput' group + /dev/uinput udev rule
# log out and back in         # activates the 'uinput' group
dictate --install-shortcut    # bind Ctrl+Alt+D to `dictate --toggle`
dictate --check               # expect: /dev/uinput writable, parec present
```

`dictate-setup` is intentionally **not** run by `bootstrap.sh`: it changes your security posture and needs a re-login.
See the security note below.

## Usage

```bash
dictate --toggle                     # start/stop (this is what the shortcut runs)
dictate --install-shortcut '<Super>x'  # bind a different key
dictate --test                       # record 5 s and print the transcript (no setup needed)
dictate --check                      # verify permissions/devices
```

Dictated newlines are collapsed to spaces so speech can never auto-submit a prompt — you press Enter yourself.

## Config (env vars)

| Var                 | Default        | Notes                                      |
| ------------------- | -------------- | ------------------------------------------ |
| `DICTATE_MODEL`     | `small.en`     | see models below                           |
| `DICTATE_COMPUTE`   | `int8`         | ctranslate2 compute type                   |
| `DICTATE_LANG`      | `en`           | language                                   |
| `DICTATE_SOURCE`    | system default | PipeWire/Pulse source name                 |
| `DICTATE_PROMPT`    | coding terms   | `initial_prompt` to bias vocabulary        |
| `DICTATE_BEAM`      | `1`            | beam size (1 = fast, 5 = more accurate)    |
| `DICTATE_LATENCY`   | `50`           | `parec` latency ms (low = flush promptly)  |
| `DICTATE_NOTIFY`    | `1`            | desktop notifications on record/transcribe |
| `DICTATE_TEST_SECS` | `5`            | seconds recorded by `--test`               |

Put per-machine overrides in `~/.bashrc.d/local.bash` (untracked), e.g. `export DICTATE_SOURCE=...`.

### Models

`small.en` is the CPU sweet spot (~95% of large-v3 accuracy at ~6× the speed). Alternatives: `base.en` (faster),
`distil-small.en` (fast English), `large-v3-turbo` (most accurate, slower on CPU — multilingual, so keep
`DICTATE_LANG=en`). English-only `.en` models beat the same-size multilingual model for English.

### Transcription tuning

`condition_on_previous_text=False` (the key anti-hallucination flag for dictation) plus `vad_filter=True` and
faster-whisper's default logprob/compression/no-speech thresholds are enabled in the script.

### Latency

The model loads on each stop (daemonless), so stop→text is ~1–2 s. If that ever matters, a resident server mode
(model kept warm, `--toggle` as a thin client) would cut it to well under a second — not implemented yet by choice,
to keep it daemonless.

## Security note

The trigger is a GNOME shortcut, so dictate never reads `/dev/input` — there is **no keylogging surface** and you are
**not** in the `input` group. The one elevated capability is membership in the dedicated `uinput` group, which lets
processes running as you write `/dev/uinput`, i.e. **inject** keystrokes. That is inherent to auto-typing; it cannot log
what you type.

To shrink the remaining surface: run untrusted code (npm/pip installs, unknown repos, `docker` with host access) as a
different user or in a container/VM, so a compromised dependency isn't inheriting your `uinput` access. To remove
injection too, switch to clipboard + manual paste (drops the `uinput` group entirely, at the cost of auto-typing).

Upgrade path: GNOME 48+ ships the GlobalShortcuts portal (press **and** release events), which would allow hold-to-talk
again without any `/dev/input` access. Ubuntu 24.04 is GNOME 46, so that isn't available here yet.

## Uninstall

```bash
cd ~/dotfiles && stow -D dictate
sudo rm -f /etc/udev/rules.d/99-uinput-dictate.rules /etc/modules-load.d/uinput.conf
sudo gpasswd -d "$USER" uinput    # optional: leave the uinput group
# remove the GNOME shortcut in Settings ▸ Keyboard ▸ Custom Shortcuts
```
