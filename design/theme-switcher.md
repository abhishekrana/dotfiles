# Theme switcher — sketch

_A design sketch, not a finished tool. It defines how one command re-skins the whole environment
from [`palette.toml`](./palette.toml). The per-tool mechanics come from [`gap-analysis.md`](./gap-analysis.md)._

## Goal

```
theme                      # show the current theme; pick a new one with fzf
theme catppuccin-mocha     # switch the whole stack to a named flavor
theme --list               # list the four flavors
```

One name in, every tool re-skinned, live where possible. The chosen theme **persists** and applies
to new shells, new windows, and the running tmux. It is **explicit** — the switcher never picks a
theme from the OS appearance or the time of day (see the design language: _theme-agnostic,
user-owned_). Default is `solarized-light`.

## Two kinds of tools

The switcher does two different things depending on how a tool is themed:

- **Name-driven** (ghostty, tmux plugins, nvim, bat, delta, hunk-Catppuccin, yazi, terminator): the
  switcher sets a **theme identifier** and reloads. No hexes flow through the switcher.
- **Hex-driven** (our tmux status bar, the sidebar option, `tmux-gitlab.sh`, `fzf`, session picker,
  and the hand-authored Solarized cases): the switcher reads hexes from `palette.toml` and writes a
  small generated config the tool consumes.

`palette.toml` is the source of truth for the hex-driven half; the name-driven half maps a flavor to
each tool's own identifier in the **adapter table** below.

## Adapter table

| Tool               | Themed by       | Apply                                                                  | Reload                                    |
| ------------------ | --------------- | ---------------------------------------------------------------------- | ----------------------------------------- |
| **ghostty**        | named           | write `theme = <Name>` into the config                                 | `killall -SIGUSR2 ghostty` (Linux)        |
| **tmux frame**     | hex → generated | emit `~/.config/theme/tmux-<id>.conf` (`status-style`, …) from palette | `tmux source-file …`                      |
| **tmux plugin**    | named           | `@catppuccin_flavor` _or_ source `seebi` light/dark conf               | `tmux source-file …`                      |
| **agent sidebar**  | option (id)     | `tmux set -g @agent-sidebar-theme <id>`                                | restart (`prefix + e` ×2) — see note      |
| **nvim**           | named           | write `~/.config/theme/nvim.lua` (`colorscheme` + `background`)        | live `:colorscheme` / next launch         |
| **fzf**            | hex → export    | rewrite the `--color` string in a sourced `theme-fzf.sh`               | new shells (re-`export FZF_DEFAULT_OPTS`) |
| **bat**            | named           | set `BAT_THEME` in a sourced file                                      | next invocation                           |
| **git-delta**      | named + feature | set `delta.features` (Catppuccin) / Solarized feature                  | next `git` invocation                     |
| **hunk**           | named / custom  | write `theme =` (Catppuccin) or a `theme="custom"` block (Solarized)   | in-app selector / next launch             |
| **yazi**           | flavor          | write `theme.toml` `[flavor]`                                          | next launch                               |
| **terminator**     | profile         | switch active profile (GUI-only; document, don't script)               | GUI                                       |
| **session picker** | hex             | derive icon colors from palette (retire emoji)                         | next launch                               |

Two hand-authored gaps ride along here (from the gap analysis): **hunk + Solarized** needs a checked-in
`theme="custom"` block, and **delta + Solarized** needs a checked-in `[delta "solarized-*"]` feature.
The switcher just selects them; the definitions live in the repo.

## Where the choice lives

- `~/.config/theme/current` — the selected flavor id (one line). The source of truth for _"what is
  active."_
- `~/.config/theme/env.sh` — sourced by `.bashrc.d`; exports `THEME`, `BAT_THEME`, `FZF_DEFAULT_OPTS`
  so new shells inherit the theme.
- `tmux set -g @theme <id>` + `@agent-sidebar-theme <id>` — so tmux and the sidebar can read it.

## Skeleton

Lives at `~/.local/bin/theme` (a `theme/` stow package, mirroring `dictate/`). Pseudo-bash:

```bash
#!/usr/bin/env bash
# theme — re-skin the terminal environment from ~/dotfiles/design/palette.toml
set -euo pipefail
PALETTE="$HOME/dotfiles/design/palette.toml"
STATE="${XDG_CONFIG_HOME:-$HOME/.config}/theme"

# palette_get <theme-id> <token> -> hex   (small awk reader; TOML here is flat tables)
palette_get() { awk -v t="[themes.$1]" -v k="$2" '…'; "$PALETTE"; }

apply_ghostty()  { set_kv "$GHOSTTY_CONF" theme "$(name_of "$1")"; reload_ghostty; }
apply_tmux()     { render_tmux_conf "$1" > "$STATE/tmux.conf"; tmux source-file "$STATE/tmux.conf"; }
apply_sidebar()  { tmux set -g @agent-sidebar-theme "$1"; }        # restart handled separately
apply_fzf()      { render_fzf_colors "$1" > "$STATE/fzf.sh"; }     # picked up by new shells
apply_bat()      { echo "export BAT_THEME='$(bat_name "$1")'" >> "$STATE/env.sh"; }
apply_nvim()     { render_nvim_lua "$1" > "$STATE/nvim.lua"; }
apply_delta()    { git config --global delta.features "$(delta_feature "$1")"; }
apply_hunk()     { set_kv "$HUNK_CONF" theme "$(hunk_id "$1")"; }  # custom block for Solarized
apply_yazi()     { render_yazi_flavor "$1" > "$YAZI_THEME"; }

main() {
  local id="${1:-$(fzf_pick)}"
  validate "$id"                       # must be one of palette [meta].order
  echo "$id" > "$STATE/current"
  for t in ghostty tmux sidebar fzf bat nvim delta hunk yazi; do "apply_$t" "$id"; done
  echo "→ $(name_of "$id"). New shells & windows inherit it; run prefix+e twice to restart sidebars."
}
main "$@"
```

## Open decisions

- **Sidebar reload.** The sidebar is a long-lived process; `@agent-sidebar-theme` changes but the
  running instance needs `prefix + e` twice (the project never drives the live server). The switcher
  should print that reminder rather than try to restart it.
- **base16 vs. bespoke.** [`tinty`](https://github.com/tinted-theming/tinty) could drive the
  name-driven 80% (fzf/yazi/nvim/tmux/terminal templates exist for both families). The bespoke
  switcher is still needed for the hex-driven half and the two Solarized hand-authored gaps — so the
  realistic build is _bespoke, optionally delegating clean tools to tinty_. Decide before writing v1.
- **TOML in bash.** `palette.toml` is flat per table; a ~15-line awk reader avoids adding a TOML
  dependency. If a richer format is wanted later, generate a bash-sourceable `palette.sh` from it.
- **terminator** stays manual (GUI-only profile switch) — documented, not scripted.

## Status

**Implemented.** `theme/.local/bin/theme` drives sidebar, ghostty, tmux frame, fzf, bat (`BAT_THEME`),
git-delta, and nvim from `palette.toml`; hunk follows via a `theme.bash` wrapper and yazi via auto
light/dark. Per-tool results and the live-verification checklist are in
[remaining-work.md](./remaining-work.md). This sketch is kept as the design rationale.
