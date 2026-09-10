# folio - design

_A markdown reader for the terminal that reads like a page in a browser, not like a terminal tool._

This is the spec for the first version. It says what folio is, how it is built, and what a style file looks like.
Everything in it was decided against the mockups in `terminal-page-looks.html` (the GitHub look was chosen) and the
design language in `../../design/README.md`.

## What it is

- A viewer for one markdown file, or stdin, rendered to a reading column with the typography of a browser-rendered page:
  rhythm, rules, tinted code, quiet tables, links in one accent.
- **Styles are files.** A look is a TOML file of per-element rules. GitHub is the first built-in; adding a look is
  adding a file, and the engine knows nothing about any of them.
- **Colours are roles, never hexes.** A style names `accent`, `surface`, `muted`; the flavor from `design/palette.toml`
  says what those are. Every style works in every flavor, and the `theme` switcher drives folio the way it drives hunk.
- Built to become an editor later without a rewrite: a rope holds the text, every node carries its byte span, and every
  rendered line knows the source range it came from. Nothing in v1 edits.

Not a browser (no history stack), not a file manager, not a notes app. leaf stays installed until folio reaches parity
with what it renders today.

## Scope

**v1 - the reader**

- CommonMark plus GFM: tables, task lists, strikethrough, autolinks, footnotes, alerts (`> [!NOTE]`), front matter, math
  delimiters left as text, `==mark==` ignored.
- Vault syntax: `[[wikilinks]]` (with `|alias` and `#heading`), `#tags`. Both render; a wikilink opens the note if the
  vault holds one file of that name, else says so.
- Code blocks highlighted with bat's grammars and themes (two-face), language label from the fence.
- Links: OSC 8 on every link for the mouse; `f` shows hint labels on visible links, a letter follows one. `.md` targets
  open in folio, `#anchors` scroll, anything else goes to `xdg-open`.
- Outline (`t`) as an overlay, heading jumps, in-document search (`/`, `n`, `N`), help (`?`).
- Watch and reload, `--inline` to stdout for fzf and yazi previews, stdin.
- One built-in style (`github`), user styles from `~/.config/folio/styles/`, `--style`, `S` cycles at runtime.
- Theme by role: `--theme <flavor>`, `FOLIO_THEME`, default `solarized-light`.

**v2 - beyond the grid**

- Images via the Kitty protocol with unicode placeholders and tmux passthrough; halfblocks as fallback.
- Mermaid rendered as an image; math as an image with a Unicode fallback.
- Larger headings when Ghostty renders the text sizing protocol (1.3 parses it, does not draw it; tmux drops it).
- Reading position memory, a file picker.

**Never in the viewer**: editing UI. Editing is a separate mode built on the same model when it is wanted.

## Architecture

Five layers, one direction of data, no layer reaching past its neighbour:

```
source (rope) -> parse (comrak) -> Document -> layout (measure) -> Lines -> paint (ratatui) -> screen
                                       ^                              |
                                       +--- style: rules per element --+
```

| Layer    | Module   | Owns                                                                                  |
| -------- | -------- | ------------------------------------------------------------------------------------- |
| Buffer   | `buffer` | `ropey::Rope`; the only copy of the text. v1 loads and reloads; edits come later.     |
| Document | `doc`    | `Block` tree with `Span` (byte range) on every block and inline. Built from comrak.   |
| Style    | `style`  | Loads TOML into `Style`: one `Rule` per element, resolved against defaults/`extends`. |
| Layout   | `layout` | `Document` × `Style` × width → `Vec<Line>`; each `Line` has cells, and a `Span`.      |
| Theme    | `theme`  | Role → colour for a flavor. Generated from `design/palette.toml`.                     |
| Paint    | `ui`     | ratatui widgets: the page, outline overlay, search bar, hints, help, status line.     |
| App      | `app`    | Elm loop: `Model`, `Msg`, `update`, `view`. Keys, mouse, watch events, mode.          |
| CLI      | `main`   | clap: args, `--inline`, stdin, exit codes. Thin.                                      |

The two facts that make the editor possible later are already in the table: the rope is the only text, and a `Line`
carries the span it was laid out from. Hit-testing a click is a lookup, not a search.

### Document model

```rust
pub struct Span { pub start: usize, pub end: usize }          // byte offsets into the rope

pub enum Block {
    FrontMatter { fields: Vec<(String, String)>, span },
    Heading { level: u8, inlines: Vec<Inline>, id: String, span },
    Paragraph { inlines, span },
    List { ordered: bool, items: Vec<ListItem>, span },        // ListItem { task: Option<bool>, blocks }
    Quote { blocks: Vec<Block>, span },
    Callout { kind: CalloutKind, title: Option<String>, blocks, span },
    Code { lang: Option<String>, text: String, span },
    Table { align: Vec<Align>, head: Vec<Vec<Inline>>, rows: Vec<Vec<Vec<Inline>>>, span },
    Rule { span },
    Footnote { label: String, blocks, span },
    Image { alt: String, src: String, span },                  // v1 draws a placeholder line
}

pub enum Inline {
    Text(String, Span), Code(String, Span), Strong(Vec<Inline>, Span), Emph(Vec<Inline>, Span),
    Strike(Vec<Inline>, Span), Link { text: Vec<Inline>, href: String, span },
    WikiLink { target: String, alias: Option<String>, span }, Tag(String, Span),
    FootnoteRef(String, Span), SoftBreak, HardBreak,
}
```

comrak is configured with `sourcepos` and the GFM, footnotes, front matter, alerts and wikilinks extensions. Tags are
not a comrak extension; a small pass over `Inline::Text` splits `#word` out, skipping headings and code.

### Layout

- **Measure.** The style sets `measure` (GitHub: 80) and `align` (`left`, the default: a 2-cell gutter; or `center`). A
  pane narrower than the measure forces the column to its width minus 2 cells each side. The outline rail, when a style
  turns it on, takes its width from the left first.
- **Rhythm.** A style says how many blank rows precede each element; the engine never inserts its own.
- **Wrapping** is per grapheme cluster using `unicode-width`, greedy, with a `Cell` per column so wide characters and
  emoji occupy two. Code never wraps: long lines are clipped with `→` in the last cell and pan with `h`/`l`.
- **Tables** get fair-share column widths. A table wider than the measure shrinks its widest column with an ellipsis
  first, then scrolls horizontally as a unit. No modal in v1.
- **Inline styling is per span**, so a link, a code span or an emphasis is styled on exactly its cells, and every cell
  keeps the id of the inline it came from.
- **Relayout** happens on resize and on reload only, all blocks, cached by `(block hash, width)`. At document scale this
  is milliseconds; measured before anything smarter is added.

### Style files

A style is TOML. Keys are element names; values are rules. Colours are palette roles. Unknown keys fail loudly, as the
workdesk config does, so a typo cannot silently fall back to a default.

```toml
# ~/.config/folio/styles/github.toml (the built-in, verbatim)
name    = "GitHub"
extends = "base"  # every built-in inherits base; a user style may extend any built-in
measure = "full"  # or a cell count, e.g. 80
align   = "left"  # or "center", for a fixed measure
rail    = false   # the outline rail is a layout switch, not an element

h1 = { fg = "emphasis", bold = true, rule = "column", above = 2, below = 1 }
h2 = { fg = "emphasis", bold = true, rule = "column", above = 1, below = 1 }
h3 = { fg = "emphasis", bold = true, rule = "none", above = 1, below = 1 }
h4 = { fg = "emphasis", bold = true, rule = "none", above = 1, below = 0 }

paragraph = { fg = "fg", below = 1 }
strong    = { fg = "emphasis", bold = true }
emph      = { italic = true }
strike    = { fg = "muted", strike = true }

link      = { fg = "accent", underline = true }
wikilink  = { fg = "accent", underline = true }
tag       = { fg = "muted" }
code_span = { fg = "changes", pad = 0 }

code    = { bg = "surface", pad = 1, label = "none", above = 0, below = 1 }
quote   = { bar = "▌", bar_fg = "muted", fg = "muted" }

list = { bullet = "•", bullet_fg = "muted", nested = "◦", indent = 2 }
task = { done = "󰄲", todo = "󰄱", done_fg = "done", todo_fg = "muted", done_text = "muted" }

table        = { lines = "box", header_bold = true, header_rule = true, line_fg = "border" }
rule         = { glyph = "─", fg = "border", width = "column" }
front_matter = { as = "table" }
footnote     = { marker = "superscript", fg = "accent", text_fg = "muted" }
image        = { placeholder = "󰥶", fg = "muted" }  # v1 draws "󰥶 alt (640×360)" on one line

[callout]
bar = "▌"
bar_fg = "accent"
title_fg = "accent"
icon = true
kinds = { note = "accent", tip = "done", important = "changes", warning = "asking", caution = "blocked" }
icons = { note = "󰋽", tip = "󰌶", important = "󰨄", warning = "󰀪", caution = "󰳦" }
```

Rule fields are a closed set per element, checked at load. `rule` on headings is `none | words | column`; `lines` on
tables is `none | rules | box`; `label` on code is `none | above | right`; `front_matter.as` is
`hidden | line | list | table`. That vocabulary is exactly the difference between the four mockups, so Air, Minimal and
Docs are each a file of overrides on `base`.

Resolution order: `base` (compiled in), then the named style's `extends` chain, then the style itself.
`~/.config/folio/styles/` is searched before the built-ins, so a user file named `github.toml` replaces the built-in.

### Theme

`theme.rs` is generated from `design/palette.toml` by `make gen`, exactly as agentbar's `theme_gen.go` is, so a flavor
is defined once. Roles available to styles:
`bg surface selection border fg emphasis muted accent changes working asking blocked done`. Syntax highlighting uses
two-face's Solarized and Catppuccin themes, mapped per flavor, so a code block matches bat in the pane beside it.

## Keys

vi and less, nothing to learn:

| Key                   | Action                                                      |
| --------------------- | ----------------------------------------------------------- |
| `j` `k` `↓` `↑`       | scroll a line; the wheel does the same                      |
| `d` `u` `PgDn` `PgUp` | half page                                                   |
| `g` `G`               | top, bottom                                                 |
| `]` `[`               | next, previous heading                                      |
| `t`                   | outline overlay; `↵` jumps, `Esc` closes                    |
| `/` `n` `N`           | search, next, previous                                      |
| `f`                   | link hints; a letter follows, `Esc` cancels                 |
| `y`                   | copy the code block under the cursor line (OSC 52 via clip) |
| `e`                   | open the file at the current line in `$EDITOR`              |
| `r` `w`               | reload; toggle watch                                        |
| `S` `T`               | cycle style; cycle theme                                    |
| `?` `q`               | help; quit                                                  |

The status line is one row on `surface`: file name, current section, percent, and three hints. It is the viewer's,
inside the pane; tmux keeps its own below.

## CLI

```
folio [FILE]                  # a file, or stdin when FILE is absent and stdin is not a TTY
folio --inline [FILE]         # render to stdout, no TUI; width: --width, else FZF_PREVIEW_COLUMNS, else the
                              # terminal, else 120 - so fzf and yazi previews fit without flags
folio --inline --format plain # force plain or ansi; auto (default) is ansi on a terminal, plain in a pipe
folio --style github --theme solarized-dark --width 88 --watch FILE
folio --list-styles
```

Exit codes: 0, 1 on a bad argument or unreadable file, 2 on a style file that fails to load (named, with the key).

## In this repo

- `apps/folio/` with a `Makefile` (`gen`, `build`, `test`, `lint`, `clean`) so `bootstrap.sh`'s `build_apps` picks it up
  unchanged. `link_app_clis` links `bin/folio` into `~/.local/bin` alongside workdesk: it is a CLI you type.
- **Toolchain.** `install.sh` gains `install_rust`: rustup into `~/.local`, the toolchain pinned by
  `rust-toolchain.toml` in the crate. The pin is the version; `install.sh` holds no second copy.
- **Gate.** `task check` runs `folio:lint` (`cargo fmt --check`, `cargo clippy --all-targets -- -D warnings`) and
  `folio:test`. `task width` adds `*.rs` to its 120-column sweep; rustfmt's `max_width` is set to 120 to match.
- **Theme switcher** gains a `folio` row: named, `export FOLIO_THEME` in `env.sh`, next launch. `theme-switcher.md` gets
  the line.
- **Previews.** yazi's opener and the fzf preview for `.md` move to `folio --inline` once it renders the sample note as
  well as leaf does; not before.
- **Trace.** Edges only (`open`, `style`, `theme`, `follow`), by calling the `dotfiles-trace` CLI. No third writer.

## Engineering conventions

Modern, boring Rust; the community's defaults, not ours.

- Edition 2024, MSRV in `rust-toolchain.toml`, `Cargo.lock` committed. rustfmt defaults except `max_width = 120`. clippy
  at `-D warnings` with `clippy::pedantic` enabled and a short, justified allow-list in `Cargo.toml`.
- `thiserror` in library modules, `anyhow` only in `main`. No `unwrap` outside tests. No `unsafe`.
- One crate, `src/` modules as in the table above; a module is one job. `lib.rs` exposes what `main.rs` and the tests
  use, nothing else.
- Prefer a maintained crate over our own: ratatui, crossterm, comrak, ropey, syntect + two-face, unicode-width, clap
  (derive), notify (watch), serde + toml, insta (snapshots).
- **Tests are snapshots.** The mockups' sample note is the fixture; each built-in style has an `--inline` snapshot at 80
  and 60 columns, so a rendering change is a reviewed diff. Wrapping, width and table allocation have unit tests.
  Nothing tests the live terminal.
- Comments say what, one line. History goes in commit messages, as the repo's `CLAUDE.md` says.

## Open questions

- Where a wikilink resolves when the file is outside a vault: the file's directory, then nothing. Good enough for v1.
- Whether `--inline` should emit OSC 8 (yes for a terminal, no for a pipe; detect the TTY).
- The Docs look's rail width and whether the rail scrolls with the page or stays fixed. Decided when that style is
  written, not now.
