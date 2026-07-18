# Theme design language — remaining work (handoff)

_Companion to [README.md](./README.md) (the design language), [gap-analysis.md](./gap-analysis.md), [palette.toml](./palette.toml) (the source of truth) and [theme-switcher.md](./theme-switcher.md). Started as a handoff list; all roadmap items are now implemented — kept as the build record + the short live-verification list. Snapshot 2026-07-18._

## Goal

One command — `theme <flavor>` — re-skins the **entire** terminal environment to one of four flavors:
`solarized-light` (default) · `solarized-dark` · `catppuccin-latte` · `catppuccin-mocha`. Every tool
derives its colors from [`palette.toml`](./palette.toml); nothing hardcodes a hex. The theme is
user-selected and persists; it never auto-switches on OS/time.

## Current state (done, committed)

| Piece                         | State                                                                                                                                                                                                                                                 |
| ----------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Design docs                   | ✅ README, gap-analysis, palette.toml, theme-switcher — **pushed**                                                                                                                                                                                    |
| Agent sidebar (separate repo) | ✅ 4 flavors + 5-state split, **deployed** (reads `@agent-sidebar-theme`; verified live)                                                                                                                                                              |
| Sidebar e2e                   | ✅ `TestHoverLightsRow` skipped w/ reason (suite green) — **unpushed** in the sidebar repo                                                                                                                                                            |
| `theme` switcher v1           | ✅ `theme/.local/bin/theme` drives sidebar + ghostty from palette — **unpushed** (dotfiles)                                                                                                                                                           |
| Session picker                | ✅ `tmux-session-picker.sh` now uses the 5-state glyphs/colors (emoji retired) — **unpushed**                                                                                                                                                         |
| tmux frame (item A)           | ✅ switcher regenerates the frame from palette; `.tmux.conf` sources it — verified on a private socket — **unpushed**                                                                                                                                 |
| fzf (item B)                  | ✅ switcher writes `~/.config/theme/fzf.sh`; `fzf.bash` sources it (new shells) — verified — **unpushed**                                                                                                                                             |
| bat / env (items D+H)         | ✅ switcher writes `env.sh` (`BAT_THEME`); `theme.bash` sources it. All 4 flavors — Catppuccin `.tmTheme` fetched by `bootstrap.sh install_bat_themes` (gitignored), cache built, rendering verified — **unpushed**                                   |
| git-delta (item F)            | ✅ switcher writes full `[delta]` block to `~/.config/theme/delta.gitconfig`, included from git config (overrides solarized-light default). delta **does** follow includes (verified isolated); Catppuccin syntax from bat. No install — **unpushed** |
| ghostty (item C)              | ✅ default in tracked config; switcher writes `~/.config/theme/ghostty.conf` (optional `config-file` include, `~` expands, missing ignored — verified). No more per-switch repo churn — **unpushed**                                                  |
| hunk (item E)                 | ✅ `hunk()` wrapper in `theme.bash` appends `--theme $THEME` for `hunk diff` only; hunk falls back gracefully on a flavor it lacks (e.g. solarized-dark). No install/no repo dirty — **unpushed**                                                     |
| nvim (item G)                 | ✅ `catppuccin/nvim` installed; `colorscheme.lua` reads `~/.config/theme/nvim.lua`; solarized overrides made flavor-aware. All 4 flavors load headless — **unpushed**                                                                                 |
| yazi (item I)                 | ✅ Catppuccin flavors (in `package.toml`, `ya pkg install` on bootstrap, dirs gitignored); `theme.toml` auto-picks Mocha/Latte by terminal light/dark — **unpushed**                                                                                  |

**Status: all roadmap items implemented.** The switcher (`theme/.local/bin/theme`) drives sidebar,
ghostty, tmux frame, fzf, bat, delta, nvim; hunk + yazi follow via wrapper/auto. Sidebar unit + e2e
suites pass (hover test skipped, documented).

**Needs live verification** (couldn't be checked from a headless session — launch each and eyeball):
nvim colorscheme colors per flavor · yazi flavor per light/dark · hunk render for solarized-dark /
catppuccin. **Known limitation:** yazi has no first-class Solarized flavor, so on the Solarized
flavors it shows Catppuccin (light/dark-matched), not true Solarized.

**Activation on a fresh pull:** `cd ~/dotfiles && stow theme bash` (switcher + `theme.bash`), open a
new shell, `bat cache --build` (or run `bootstrap.sh`), then `theme <flavor>`.

## The key insight (read this first)

**Inside tmux, everything looks Solarized even after `theme catppuccin-mocha` — because
`~/.tmux.conf` hardcodes `window-style … bg=#fdf6e3` (cream) on every pane, plus Solarized colors on
the status bar. That paints over ghostty's theme.** Outside tmux (a plain shell) the switch is
visible (ghostty goes dark); inside tmux, tmux repaints cream. So **wiring the tmux frame to the
palette is the single highest-impact task** — the user lives inside tmux. Ghostty theming already
works (v1); it's just being overpainted.

## Ground facts the next agent needs

- **Palette source of truth:** `~/dotfiles/design/palette.toml` — `[themes.<flavor>]` tables with
  tokens `bg surface selection border fg emphasis muted accent working asking blocked done add remove`.
  The switcher has a `palette_get <flavor> <token>` awk reader.
- **Switcher:** `~/dotfiles/theme/.local/bin/theme` (v1). Extend it — add an `apply_<tool>` per tool
  and call it from `main`. Generated per-flavor files should live in `~/.config/theme/` (NOT tracked
  in dotfiles), so switching never dirties the repo. Wired: `apply_sidebar`, `apply_ghostty`,
  `apply_tmux`, `apply_fzf`, `apply_env` (bat), `apply_delta`, `apply_nvim` (hunk + yazi need no
  per-switch write — wrapper / auto).
- **Stow-symlink constraint (important):** `~/.config/ghostty`, `~/.config/hunk`, and `~/.config/yazi`
  are **directory symlinks into the dotfiles repo** — so rewriting those configs edits tracked files
  and dirties the repo (that's the ghostty `theme =` diff seen when switching). Prefer a **non-repo
  override** per tool: an env var (`BAT_THEME`, `FZF_DEFAULT_OPTS`), a sourced file, or a
  `config-file`/include that points at `~/.config/theme/…`. Only rewrite a tracked config if the tool
  offers no other mechanism — and expect a diff. tmux and fzf already avoid this via generated files
  under `~/.config/theme/`. delta follows git `[include]` (verified), so its selection lives in an
  included `~/.config/theme/delta.gitconfig`; ghostty uses a `config-file = ?~/.config/theme/…` include.
- **Verified ghostty theme names** (`ghostty +list-themes`): `iTerm2 Solarized Light`,
  `iTerm2 Solarized Dark`, `Catppuccin Latte`, `Catppuccin Mocha`. (There is **no** plain
  "Solarized Light".) The switcher already maps these.
- **Sidebar:** deployed and reads `@agent-sidebar-theme`; a running sidebar only re-themes after
  `prefix + e` twice (the sidebar project never drives the live tmux server). It has **no explicit bg
  fill** — it blends into the tmux pane background, so once the tmux frame is themed the sidebar sits
  on the flavor's `bg` automatically. Sidebar deploy flow (separate repo): commit + push → in the TPM
  clone `~/.tmux/plugins/tmux-agent-sidebar` fetch+reset to `origin/main` and `make build` → verify
  headlessly on a private socket → tell the user to `prefix + e` twice. (No sidebar code changes are
  needed for the remaining work.)
- **hunk:** v0.17 **does** have a working `solarized-light` theme (verified live — renders real
  Solarized, not a fallback). This corrects gap-analysis.md, which guessed hunk had no Solarized. So
  **no hand-authored hunk theme is needed** — just confirm `solarized-dark` exists and wire the
  switcher to set hunk's built-in `theme` per flavor.

### Testing constraints (important)

- **Never touch the live tmux server** (the `default` socket). Use private sockets:
  `tmux -L <name> -f /dev/null …`, and `kill-server` when done.
- **Do NOT** wrap `tmux` in a shim that omits `-f /dev/null` — it will load the user's `~/.tmux.conf`
  (TPM, resurrect, the sidebar plugin) on the test socket and hang. Always pass `-f /dev/null`.
- The switcher's `tmux …` / `pkill -USR2 ghostty` calls hit the **live** server/terminal by design
  (it's a user-run live tool). To test it safely, run it with a PATH shim where `tmux` →
  `tmux -L <test> -f /dev/null` and `pkill` → no-op, and point `XDG_CONFIG_HOME` at a temp dir with a
  throwaway ghostty config. Palette reads are read-only.

### Dotfiles conventions (`~/dotfiles/CLAUDE.md`)

- Stow packages; **secrets audit before every commit** — run the `grep` documented in CLAUDE.md (it
  flags private IP ranges, the user's name, and company references) and confirm it returns nothing.
  **Never put personal info** (names, emails, IPs, work paths, company refs) in tracked files.
- Format markdown with `npx prettier --write <file>.md`. `MD013` (line length) is off.
- **No `Co-Authored-By`** lines. Commit style: `area: lowercase summary`. Commit after each item;
  push only when asked.
- Deploy = commit + push, then make live: `tmux source-file ~/.tmux.conf`; stowed scripts are live on
  save; a new stow package needs `stow <pkg>`.

---

## Remaining work

Ordered by impact. Do them one at a time, commit each, have the user test live.

### A. tmux frame → palette (highest impact, no install) — ✅ DONE

**Status:** implemented and committed. `apply_tmux` (below) is now in the switcher and `.tmux.conf`
has the source-file hook. Verified on a private socket (tmux parses the generated file; window/status
styles carry the flavor). Kept here for reference and in case the host badge / `tmux-gitlab.sh` polish
is picked up later.

**Why:** the "most things still Solarized inside tmux" bug (see key insight).

**How:** add `apply_tmux` to the switcher — generate `~/.config/theme/tmux.conf` from the palette and
`tmux source-file` it live. In `~/.tmux.conf`, after the hardcoded Solarized defaults, add:

```tmux
if-shell '[ -f ~/.config/theme/tmux.conf ]' 'source-file ~/.config/theme/tmux.conf'
```

so a fresh tmux still defaults to Solarized Light and the generated file overrides when present.
Starter `apply_tmux` (token → setting mapping already worked out):

```bash
apply_tmux() {
	command -v tmux >/dev/null || return 0
	local f="$STATE_DIR/tmux.conf"
	local bg surface fg muted accent block ask grn work
	bg=$(palette_get "$1" bg);       surface=$(palette_get "$1" surface)
	fg=$(palette_get "$1" fg);       muted=$(palette_get "$1" muted)
	accent=$(palette_get "$1" accent)
	block=$(palette_get "$1" blocked); ask=$(palette_get "$1" asking)
	grn=$(palette_get "$1" done);      work=$(palette_get "$1" working)
	cat >"$f" <<EOF
set -g status-style "fg=$fg,bg=$surface"
set -g window-style "fg=$fg,bg=$bg"
set -g window-active-style "fg=$fg,bg=$bg"
set -g pane-border-style "fg=$muted,bg=$bg"
set -g pane-active-border-style "fg=$accent,bg=$bg"
set -g status-left "#{@host_badge}#[fg=$bg,bg=$accent,bold] #S #[fg=$fg,bg=$surface] "
set -g window-status-current-format "#[fg=$bg,bg=$accent,bold] #I:#W#{?window_zoomed_flag, Z,} "
set -g window-status-format "#{?window_bell_flag,#[fg=$bg bg=$block bold],#[fg=$muted bg=$surface]} #I:#W "
set -g @dictate_seg "#{?#{==:#{@dictate},rec},#[fg=$block bg=$surface] ● dictate ,#{?#{==:#{@dictate},work},#[fg=$ask bg=$surface] ● dictate ,#[fg=$muted bg=$surface] ● dictate }}"
set -g @submit_seg "#{?@submit_flash,#[fg=$bg bg=$grn]   ⏎   ,#[fg=$work bg=$surface]   ⏎   }"
EOF
	tmux source-file "$f" 2>/dev/null || true
}
```

Design decision baked in: the **current window** uses `accent` (not the old amber), freeing amber/red
for the state language (see README). Left out for now: the **host badge** (`@host_badge`, set once at
server start via `run-shell` with SSH detection; theming it needs the hostname and is fiddly — leave
green/orange, or theme later using `@is_ssh` + a `%if`), and `tmux-gitlab.sh`'s issue/MR/CI hexes
(theme from palette in a later pass).

**Done when:** on a private socket, `source-file`ing the generated mocha file makes
`show -g window-style` report `bg=#1e1e2e` etc.; live, `theme catppuccin-mocha` turns the whole tmux
frame dark.

### B. fzf → palette (no install)

`~/dotfiles/bash/.bashrc.d/fzf.bash` hardcodes a Solarized `--color=` string. Add `apply_fzf` that
writes `~/.config/theme/fzf.sh` exporting `FZF_DEFAULT_OPTS`'s `--color` from the palette; source that
file from a `.bashrc.d` entry so new shells inherit it. Map palette → fzf color roles
(`bg,fg,hl,fg+,bg+,hl+,info,prompt,pointer,marker,border,header`). Reference:
`catppuccin/fzf`, `tinted-theming/tinted-fzf`.

### C. ghostty `config-file` include cleanup (no install)

v1 rewrites the **tracked** ghostty config's `theme =` line, so a non-default flavor shows a dotfiles
diff. Cleaner: main config `config-file = theme.conf`; `apply_ghostty` writes a non-tracked
`~/.config/ghostty/theme.conf` containing only `theme = <name>`. Then switching never dirties the repo.

### D. env inheritance (no install)

Per the switcher sketch: write `~/.config/theme/env.sh` (exports `THEME`, `BAT_THEME`,
`FZF_DEFAULT_OPTS`) and source it from a new `bash/.bashrc.d/theme.bash`, so new shells pick up the
current flavor. Persisted flavor lives in `~/.config/theme/current` (v1 already writes it).

### E. hunk → palette (built-in themes; minimal)

Add `apply_hunk`: rewrite the `theme = …` line in `~/.config/hunk/config.toml` per flavor —
`solarized-light`, `solarized-dark` (confirm it exists: `hunk --theme solarized-dark` on a diff),
`catppuccin-latte`, `catppuccin-mocha` (all built-in). No hand-authored theme needed. Applies on
hunk's next launch.

### F. git-delta → palette (Solarized feature by hand; Catppuccin needs a port)

Delta's syntax colors come from bat; its `+/-` tints are `[delta "<feature>"]` blocks. Add
`[delta "solarized-light"]` / `[delta "solarized-dark"]` features (syntax-theme `Solarized (light/dark)`

- `plus-style`/`minus-style` from palette `add`/`remove`). Install `catppuccin/delta` for the
  catppuccin features. `apply_delta` sets `git config --global delta.features <feature>` per flavor.

### G. nvim → palette (needs a plugin)

Install `catppuccin/nvim` (LazyVim). Solarized already via `maxmx03/solarized.nvim`. `apply_nvim`
writes `~/.config/theme/nvim.lua` (`colorscheme` + `vim.o.background`) read on nvim startup. Map:
solarized-light/dark → `solarized` (bg light/dark); catppuccin-latte/mocha → `catppuccin-latte`/`-mocha`.
Also move the hardcoded Solarized overrides in `nvim/.config/nvim/lua/plugins/colorscheme.lua` and
`diffview.lua` onto palette-derived / theme-aware values.

### H. bat → palette (Catppuccin needs install)

Solarized (light/dark) is built-in (stale but usable). Install `catppuccin/bat` `.tmTheme` files +
`bat cache --build` for Catppuccin. Set `BAT_THEME` per flavor via `~/.config/theme/env.sh` (item D).
Note bat also feeds delta's `syntax-theme`.

### I. yazi → palette (needs install)

Install `catppuccin/yazi` flavors (`ya pkg add …`). Solarized has no first-class flavor — use
`tinted-theming/tinted-yazi` base16 approximation or a palette-derived `flavor.toml`. `apply_yazi`
writes `~/.config/yazi/theme.toml`'s `[flavor]` per flavor.

### J. session picker — full theming (functional part already done)

The picker's glyphs/colors now match the sidebar, but its state hexes and its `fzf --color=` line are
still hardcoded Solarized. To re-skin per flavor, source palette hexes for the active flavor
(`~/.config/theme/current` + palette) instead of literals. Low priority.

### K. terminator (manual, optional)

Solarized is a built-in preset; `catppuccin/terminator` is a profile port. Theme switching is
GUI-only (not scriptable) — document, don't automate.

### L. docs & upkeep

- Add `theme/` to the stow-packages list in `~/dotfiles/CLAUDE.md` and the README table (keep the
  list sorted).
- Correct gap-analysis.md's hunk claim (hunk 0.17 **has** `solarized-light`).
- Update theme-switcher.md's "Status" and adapter table as items land.

---

## Suggested order

**A** (tmux frame) → **C** (ghostty include) → **B** (fzf) → **D** (env) → **E** (hunk) → **F** (delta)
→ **G** (nvim) → **H** (bat) → **I** (yazi) → **J** (picker full theming) → **L** (docs). A + C + B + D
alone make `theme <flavor>` visibly re-skin the whole tmux + terminal + shell stack (no installs).

## Acceptance (whole feature)

`theme catppuccin-mocha` (then `prefix + e` twice for the sidebar; new shell for fzf/env; next launch
for nvim/hunk/yazi; next invoke for bat/delta) leaves **no** Solarized surface visible; `theme
solarized-light` restores the default with no leftover dotfiles diff.
