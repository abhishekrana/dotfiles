# Theme gap analysis

_A planning companion to [the design language](./README.md). Where that document is the "what," this
is the "can we, and what's missing." Snapshot taken 2026-07-18._

**Question.** Can every tool in this environment wear the four supported flavors — **Solarized Light,
Solarized Dark, Catppuccin Latte, Catppuccin Mocha** — and can they all be switched together as one
product?

**Answer.** Yes, nothing is truly blocked. But the gaps cluster in two places: a few tools can't do
_Solarized_ without hand-authoring (Catppuccin has org-backed ports; Solarized doesn't), and — the
larger problem — today there is no shared palette and the state language is already inconsistent
between two of our own tools.

---

## Support matrix

Legend: ● built-in · ○ install a port/snippet · ⨯ hand-author

| Tool           | Sol Light | Sol Dark | Catp Latte | Catp Mocha | Verdict                                                      |
| -------------- | :-------: | :------: | :--------: | :--------: | ------------------------------------------------------------ |
| **ghostty**    |     ●     |    ●     |     ●      |     ●      | Cleanest — all four built-in.                                |
| **tmux**       |     ○     |    ○     |     ○      |     ○      | Plugins: `catppuccin/tmux` + `seebi/tmux-colors-solarized`.  |
| **nvim**       |     ○     |    ○     |     ○      |     ○      | Two plugins: `catppuccin/nvim` + `maxmx03/solarized.nvim`.   |
| **fzf**        |     ○     |    ○     |     ○      |     ○      | `--color` snippets (`catppuccin/fzf`, tinted-fzf).           |
| **terminator** |     ●     |    ●     |     ○      |     ○      | Solarized built-in; Catppuccin port (GUI-switch only).       |
| **bat**        |    ●\*    |   ●\*    |     ○      |     ○      | Catppuccin needs `bat cache --build`; built-in Sol is stale. |
| **git-delta**  |   ○ / ⨯   |  ○ / ⨯   |     ○      |     ○      | Catppuccin port; **Solarized +/- tints hand-authored**.      |
| **yazi**       |     ○     |    ○     |     ○      |     ○      | Catppuccin flavor; **Solarized only a base16 approx**.       |
| **hunk**       |     ⨯     |    ⨯     |     ●      |     ●      | Catppuccin built-in; **Solarized must be hand-authored**.    |

\* bat's bundled Solarized is outdated / pink-heavy ([bat #941](https://github.com/sharkdp/bat/issues/941)).

The **agent sidebar** (`tmux-agent-sidebar`) is not in the matrix because it is _ours_: it ships only
two palettes today (`solarized-light` + a generic `dark`), compiled into the binary. It needs the
four-flavor set and the five-state model added — expected work, not a blocker.

---

## The real theme gaps (ranked)

1. **hunk + Solarized — the standout.** hunk has a _fixed_ built-in theme registry
   (`github-{light,dark}-default`, `catppuccin-{latte,frappe,macchiato,mocha}`, `zenburn`). Catppuccin
   Latte/Mocha ship built-in; **there is no Solarized theme and no port.** Solarized means a
   hand-authored `theme = "custom"` block (chrome + syntax hexes) in `~/.config/hunk/config.toml`, and
   there is no shortcut to point syntax highlighting at a Shiki `solarized` theme.
2. **git-delta + Solarized.** Syntax colors come free from bat, but the **+/- background tints** have
   no Solarized feature — they're hand-authored as a `[delta "solarized-*"]` feature. (We already
   hardcode these tints today, so this is a lateral move, not new debt.)
3. **bat + Solarized fidelity.** Built-in `Solarized (light/dark)` exists but is stale; a faithful one
   needs a fresh `.tmTheme`. Catppuccin needs its theme files dropped in + a cache rebuild.
4. **yazi + Solarized.** No first-class flavor exists anywhere ([yazi-rs/flavors #7](https://github.com/yazi-rs/flavors/issues/7));
   only a base16 (16-color) approximation via `tinted-yazi`.

**Takeaway:** Catppuccin is effectively free everywhere; **Solarized is what forces hand-authoring** —
which matters because Solarized Light is our default.

---

## The coherence gaps (these matter more)

These are gaps in what we have today, independent of any single theme.

1. **No single source of truth.** The same Solarized hexes are hand-copied across ~8 files
   (`.tmux.conf`, `theme.go`, `tmux-gitlab.sh`, `fzf.bash`, git-delta config, nvim overrides,
   terminator, session picker). Re-theming the stack today means editing eight files plus a Go
   rebuild. This is the gap that makes "coherent product" hard, and the one [`palette.toml`](./palette.toml)
   is meant to close.
2. **The sidebar's state model is behind the language.** It still conflates _permission_ and _asking_
   as one amber and paints _working_ cyan. It needs the five-state split (permission → red blocked,
   asking → amber) and the four flavors.
3. **The session picker disagrees with the sidebar.** `tmux-session-picker.sh` uses emoji
   🔴 permission / 🟠 asking / 🟡 working / 🟢 done — but the sidebar renders permission = amber,
   asking = amber, working = **cyan**. Only _done = green_ matches. Emoji also can't be themed. This is
   exactly the "one product, two languages" problem the design language exists to fix.

---

## Switching them together

The only mechanism that unifies **both** families is **base16 / base24** (tinted-theming): good
schemes exist for Solarized _and_ Catppuccin, applied by [`tinty`](https://github.com/tinted-theming/tinty)
or [`flavours`](https://github.com/Misterio77/flavours) with templates for fzf, yazi, and the terminal
palette. Its limits: hunk and delta's Solarized bits fall outside base16 templating, and base16 is a
16-color approximation of Catppuccin's higher-fidelity native ports.

The **Catppuccin ecosystem** is the highest-fidelity route for the two Catppuccin flavors, but it
ships **no Solarized**, so it can never be the unifier.

**Recommendation — a custom theme-switcher keyed off a shared palette.** One theme name that points
each tool at its theme (plugin flavor, `--color` export, `theme.toml`, delta feature, hunk custom
block) and reloads it. base16 can do the easy 80% (fzf / yazi / nvim / tmux / terminal); the switcher
wraps the 20% base16 can't (hunk + delta Solarized, and the sidebar). This _is_ the single source of
truth from the design language, made executable. Sketched in [`theme-switcher.md`](./theme-switcher.md).

---

## Recommended sequence

1. **Palette source of truth** — [`palette.toml`](./palette.toml). Everything else derives from it. _(done)_
2. **Sidebar** — read the four flavors, implement the five-state model. Self-contained, low-risk.
3. **Switcher** — the script in [`theme-switcher.md`](./theme-switcher.md), starting with the clean
   tools (ghostty, tmux, nvim, fzf, bat).
4. **Hand-authored gaps** — hunk Solarized custom theme, delta Solarized feature.
5. **Unify the picker** — move `tmux-session-picker.sh` onto the five-state colors (retire the emoji).

---

## Sources

- ghostty — [themes](https://ghostty.org/docs/features/theme) · [catppuccin/ghostty](https://github.com/catppuccin/ghostty)
- tmux — [catppuccin/tmux](https://github.com/catppuccin/tmux) · [seebi/tmux-colors-solarized](https://github.com/seebi/tmux-colors-solarized)
- nvim — [catppuccin/nvim](https://github.com/catppuccin/nvim) · [maxmx03/solarized.nvim](https://github.com/maxmx03/solarized.nvim)
- hunk — [modem-dev/hunk](https://github.com/modem-dev/hunk)
- delta — [catppuccin/delta](https://github.com/catppuccin/delta)
- bat — [catppuccin/bat](https://github.com/catppuccin/bat) · [bat #941](https://github.com/sharkdp/bat/issues/941)
- fzf — [catppuccin/fzf](https://github.com/catppuccin/fzf) · [fzf color schemes](https://github.com/junegunn/fzf/wiki/Color-schemes)
- terminator — [catppuccin/terminator](https://github.com/catppuccin/terminator)
- yazi — [catppuccin/yazi](https://github.com/catppuccin/yazi) · [flavors overview](https://yazi-rs.github.io/docs/flavors/overview/)
- orchestration — [tinted-theming/schemes](https://github.com/tinted-theming/schemes) · [tinty](https://github.com/tinted-theming/tinty) · [flavours](https://github.com/Misterio77/flavours)
