# Theme switcher

_How one command re-skins the whole terminal stack from [`palette.toml`](./palette.toml). Companion to
[README.md](./README.md) (the design language)._

## Usage

```
theme                      # print the current flavor
theme --list               # list the four flavors
theme <flavor>             # re-skin the whole stack
```

Flavors: `solarized-light` (default) · `solarized-dark` · `catppuccin-latte` · `catppuccin-mocha`. The choice
**persists** (via `~/.config/theme/`) and applies to new shells, new windows, and the running tmux/ghostty. It is
**explicit** - the switcher never picks a theme from the OS appearance or the time of day.

## Two kinds of tools

- **Name-driven** (ghostty, bat, delta, nvim): the switcher sets a theme _identifier_ the tool already ships and lets
  the tool render it.
- **Hex-driven** (tmux frame, fzf, agent sidebar): the switcher reads hexes from `palette.toml` and writes a small
  generated config the tool consumes.

`palette.toml` is the source of truth for the hex-driven half; the name-driven half maps a flavor to each tool's own
identifier.

## Adapter table

Everything the switcher touches writes into `~/.config/theme/` and is consumed from there, so switching never edits a
tracked config.

| Tool           | Themed by       | What the switcher writes                                                                 | Reload                            |
| -------------- | --------------- | ---------------------------------------------------------------------------------------- | --------------------------------- |
| agent sidebar  | hex → generated | `@agentbar-theme`; colors from `theme_gen.go` (built from the palette)                   | this session's, immediate         |
| ghostty        | named           | `theme = <Name>` → `ghostty.conf` (a `config-file` include)                              | `pkill -USR2 -x ghostty`          |
| tmux frame     | hex → generated | `tmux.conf` (status/window/pane + the dictate/submit/push/diff/⛭ chips)                  | `tmux source-file` (immediate)    |
| fzf            | hex → export    | `fzf.sh` (`_fzf_color` `--color` block, sourced by fzf.bash)                             | new shells                        |
| bat / `$THEME` | named           | `env.sh` (`export THEME`, `export BAT_THEME`)                                            | new shells                        |
| leaf           | named           | `env.sh` (`export LEAF_THEME`)                                                           | next `leaf` launch                |
| git-delta      | hex + named     | `delta.gitconfig` (a `[delta]` block, git-included)                                      | next `git` invocation             |
| nvim           | named           | `nvim.lua` (`colorscheme` + `background`)                                                | live `:colorscheme` / next launch |
| session popup  | hex → generated | `agent-state.sh` (state colors + the popup's fzf palette, base mode and ground included) | next open                         |
| hunk           | named           | `--theme <flavor>` on the diff pane's own command line                                   | the diff pane is re-run           |

Tools that follow the flavor **without** being driven by the switcher:

- **hunk** - a `hunk()` wrapper in `bash/.bashrc.d/theme.bash` appends `--theme $THEME` to `hunk diff` (hunk falls back
  gracefully on a flavor it lacks).
- **yazi** - a static `theme.toml` picks Mocha/Latte automatically by terminal light/dark.
- **`tmux-gitlab.sh`** - still hardcodes Solarized hexes (not yet palette-driven).

## Where the choice lives

- `~/.config/theme/current` - the selected flavor id (one line); read back by `theme` to print the active flavor.
- `~/.config/theme/env.sh` - sourced by `bash/.bashrc.d/theme.bash`; exports `THEME`, `BAT_THEME` and `LEAF_THEME`.
  (fzf's colors are separate, in `fzf.sh`.)
- `~/.config/theme/agent-state.sh` - sourced by `tmux-agent-state.sh`, the shell side of the agent-state language (the
  session popup's glyph colors and its fzf palette). Falls back to the palette's solarized-light values when absent, so
  the popup works before the first `theme` run.
- `tmux set -g @agentbar-theme <flavor>` - so the sidebar can read the flavor at launch.

## Operational notes

**Activation on a fresh pull:** `cd ~/dotfiles && stow theme bash`, open a new shell, `bat cache --build` (or run
`bootstrap.sh`), then `theme <flavor>`.

**Known limitations:** yazi has no first-class Solarized flavor, so on the Solarized flavors it shows Catppuccin
(light/dark-matched), not true Solarized. leaf is the mirror image - it ships no Catppuccin, so those two flavors get
its closest light/dark built-in (`arctic` / `ocean`) instead. Its Solarized Light is not upstream either: leaf ships
only `solarized-dark`, so the light half is a full palette registered as `[themes.solarized-light]` in
`leaf/.config/leaf/config.toml`.

**leaf fails quietly, so keep the map total.** An unrecognised `LEAF_THEME` is not an error - leaf ignores its config's
`theme` and drops to its own default (`ocean`, a _dark_ theme), which on a light terminal looks like a bug rather than a
fallback. Every flavor in `apply_env` must therefore map to a name leaf knows. (`--theme` is the strict path by
contrast: it exits 1 with `Unknown theme "..."`.)

**Needs live verification** (not checkable headless - launch and eyeball): nvim colorscheme per flavor · yazi flavor per
light/dark · hunk render for solarized-dark / catppuccin.
