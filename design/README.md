# Design language

_One terminal workspace, read as one product — in any theme._

This is the shared visual language for the terminal environment in this repository: the agent
sidebar, the tmux frame, the diff reviewer, the file manager, the editor, and the small status
widgets that live between them. It exists so that a signal — a color, a glyph, a state — means the
same thing everywhere, and so that switching the whole environment to a new theme is a change of
_skin_, never a change of _structure_.

It is a description of intent, not an implementation guide. It says what the system is and why;
where each tool reads its colors from and how a theme is wired up lives with that tool. When a new
feature or tool joins this environment, it should be designed against this document.

**Scope:** everything that renders in the terminal — sidebar · tmux status bar & panes · hunk diff
reviewer · delta · fzf · nvim · ghostty · dictate and notification widgets.

---

## Principles

1. **Calm by default.** Most agents are working most of the time, so _working_ must never shout.
   Attention is earned by exception — a question, a block — not by ordinary activity. A quiet screen
   is a healthy screen.

2. **One signal, one meaning.** A hue or glyph carries a single meaning across every tool. Red is
   always a hard stop. Green is always done-and-good. Nothing is recolored for local convenience.

3. **Glanceable before readable.** These surfaces are scanned, not read top to bottom. State must be
   legible from color, glyph, and position _before_ any word is parsed.

4. **Terminal-native.** Monospace type, character-cell alignment, and Nerd Font glyphs are the
   medium — not a limitation to design around. If a terminal can't draw it, it isn't in the language.

5. **Theme-agnostic, user-owned.** One structure wears many skins. The person chooses the flavor
   (default: Solarized Light) and the system never switches itself based on time or OS setting.

6. **Coherent as a product.** The sidebar, the frame, the diff, and the widgets are one thing. A
   decision made once — a token, a state color, a glyph — is honored everywhere without exception.

---

## Foundations

### Color

Color is defined in two tiers so that meaning stays stable while values are free to change per theme.

- **Primitives** are a theme's raw palette — Solarized's sixteen, Catppuccin's twenty-six, and so on.
  They are values, not decisions. Nothing in the environment refers to a primitive directly.
- **Semantic tokens** are the vocabulary everything speaks. Each names a _role_ — a surface, a kind
  of text, an accent, a state — and each theme maps that role onto one of its primitives. A role
  name (`blocked`, `surface`, `accent`) stays constant even as its hex changes from flavor to flavor.

Because every surface consumes the semantic layer, re-theming is a matter of remapping roles to
values. The layout, the hierarchy, and the meaning of each color never move.

**Surface & structure**

| Token       | Intent                                                          |
| ----------- | --------------------------------------------------------------- |
| `bg`        | The base canvas — the terminal ground every surface sits on.    |
| `surface`   | Raised or secondary areas: status bar, panels, card fills.      |
| `overlay`   | Inset chips and buttons resting on `surface` (dictate, submit). |
| `border`    | Hairlines and separators — the quietest structural line.        |
| `selection` | The fill behind the selected or hovered row.                    |

**Content**

| Token      | Intent                                                            |
| ---------- | ----------------------------------------------------------------- |
| `fg`       | Body text — the default reading color.                            |
| `emphasis` | Identity and headings: session names, branch headlines.           |
| `muted`    | At-rest and secondary: idle agents, hints, timestamps, seen work. |

**Interaction**

| Token    | Intent                                                                     |
| -------- | -------------------------------------------------------------------------- |
| `accent` | Selection rail, active pane, links, "you are here." Exactly one per theme. |

**State** — see [The state language](#the-state-language). The tokens `working`, `asking`, `blocked`,
and `done` carry meaning through color; `idle` reuses `muted`.

**Diff** — `add` and `remove` are change tints for the reviewer, derived from each theme's own green
and red so the diff belongs to the same world as the pane beside it.

### Typography

Monospace is the medium, not merely the code font. One face — **JetBrains Mono Nerd Font** — carries
the entire environment.

- **Alignment is a design tool.** Because every glyph is one cell wide, columns line up for free;
  the system uses that grid rather than fighting it. Fixed-width fields (name · state · elapsed) read
  as columns without rules or boxes.
- **Weight signals role, not size.** Regular is body. Bold marks identity and headlines — session
  names, branch headlines, the loudest state. Italic is reserved for the secondary and the aside:
  subagent counts, hints, acknowledged work.
- **Nerd Font glyphs are part of the type system.** Host badges, git status marks, and the state
  glyphs below all come from the same font and are treated as first-class characters.

### Iconography & glyphs

Every state pairs a color with a glyph, so meaning survives when color can't be trusted (color
blindness, a stripped-down terminal, a screenshot). The glyph set is small and consistent:

| Glyph       | Meaning                                        |
| ----------- | ---------------------------------------------- |
| `⠋ ⠙ ⠹ ⠸ …` | Working — an animated braille spinner.         |
| `?`         | Asking — a soft question awaiting your answer. |
| `◔`         | Blocked — a hard stop awaiting permission.     |
| `✓`         | Done — finished, ready to review.              |
| `·`         | Idle — a live agent at rest.                   |
| `⚠`         | Attention tally — how many need you.           |
| `⤷`         | Subordinate detail, e.g. a subagent count.     |

### Space & density

- **The sidebar lives at roughly thirty columns.** Space is scarce and spent deliberately.
- **Density is a first-class choice.** _Dense_ is the default — it keeps the most agents on one
  screen, the right trade for a monitor you glance at. _Cards_ is an opt-in that trades some of that
  count for breathing room and a rounded, panel-per-branch feel.
- **A single-column gutter carries selection.** The selected row shows an accent rail in that gutter
  rather than a heavy full-width block, so focus is clear without shouting.
- **Repetition collapses.** A branch shared by several agents is named once, as a headline, colored
  by its most urgent agent; the agents list quietly beneath it.

### Motion

Motion means _liveness_, never decoration.

- Only **working** animates — a braille spinner at roughly ten frames a second. It is the one moving
  thing on the screen, and it moves precisely because that agent is alive and busy.
- Everything else is still. A screen at rest looks at rest.
- Motion always yields to `prefers-reduced-motion`: the spinner freezes to a single static frame and
  loses nothing but the movement.

---

## The state language

The heart of the system. Every agent is in exactly one of five states, and each state is expressed
the same way in the sidebar, the session picker, and the tmux frame.

| State       | Glyph | Color     | Means                                         | Behavior                                       |
| ----------- | ----- | --------- | --------------------------------------------- | ---------------------------------------------- |
| **Working** | `⠋`   | `working` | Actively processing — the common case.        | Calm and cool. Animates. Never shouts.         |
| **Asking**  | `?`   | `asking`  | A soft question is waiting on you.            | Warm amber. Counts as _needs you_.             |
| **Blocked** | `◔`   | `blocked` | A hard stop — waiting on permission/approval. | Red, the loudest state. Counts as _needs you_. |
| **Done**    | `✓`   | `done`    | Finished, ready to review.                    | Green. Mutes to grey once you've seen it.      |
| **Idle**    | `·`   | `muted`   | A live agent at rest.                         | Quiet grey. Present, not competing.            |

**Needs you = asking + blocked.** These two states, and only these two, drive every "needs
attention" affordance: the sidebar's footer tally, and a tmux window that glows to flag the agent
inside it. Working and done never demand you.

**Urgency ordering.** When one line must represent a group (a branch with several agents), it takes
the color of its most urgent member. Urgency runs: **blocked → asking → working → done (unseen) →
done (seen) / idle**. The loudest thing in a group is what you see first.

**Done fades.** Finished work is green so it reads as ready — but once acknowledged it drops to
`muted`, so completed-and-seen agents stop competing for attention while still showing they exist.

**Why working stays cool.** Working is the state an agent is in most of the time. Coloring it a warm
or alarm hue would keep the whole screen shouting and drown the exceptions that matter. Keeping it
calm is what makes _asking_ and _blocked_ pop.

### Where the states come from

The five states are an abstraction over **Claude Code's own hook lifecycle** — they are stamped from
real hook events, not guessed from screen scraping, so they line up with what Claude Code actually
reports:

| Claude Code signal                                                            | State       |
| ----------------------------------------------------------------------------- | ----------- |
| `SessionStart`                                                                | **idle**    |
| `UserPromptSubmit`, `PreToolUse` (and tools running)                          | **working** |
| `PermissionRequest` — a tool awaiting approval                                | **blocked** |
| `PermissionRequest` (AskUserQuestion) · `Notification` (`elicitation_dialog`) | **asking**  |
| `Stop` · `Notification` (`agent_completed`)                                   | **done**    |
| `SessionEnd`                                                                  | _(cleared)_ |

Three deliberate readings keep the abstraction honest:

- **`Stop` means "this turn is finished," not "gone for good."** That is exactly _done_ — ready to
  review — which is why _done_ fades to `muted` once seen rather than disappearing.
- **Compaction is not its own state.** `PreCompact`/`PostCompact` bracket the agent doing work on its
  own context, so it reads as **working** — the screen shouldn't sprout a new color for housekeeping.
- **Noisy "waiting for input" nudges are ignored on purpose.** Claude's periodic idle/needs-input
  notifications are dropped so a reviewed-_done_ agent doesn't flip back into a nagging state; the
  reliable attention signals are the two above.

---

## Themes

A theme is a set of values for the semantic roles — a skin. Choosing one re-skins the entire
environment at once; nothing about layout, hierarchy, or meaning changes.

- **Default is Solarized Light.** It is the resting identity of this environment.
- **The person chooses.** A theme is selected explicitly and stays until changed. The system does
  **not** follow the OS light/dark setting or the time of day. Predictability over cleverness.
- **Adding a flavor adds a column, not a concept.** A new theme supplies new values for the roles
  that already exist — never a new role and never a layout change.

### Flavor catalog

Solarized ships a light and a dark flavor that share the same eight accent hues and only swap their
base tones — so `accent` and the four state colors are identical across the two, and the mood change
lives entirely in `bg` / `surface` / text.

| Token      | Solarized Light · _default_ | Solarized Dark · _dark_ | Catppuccin Latte · _light_ | Catppuccin Mocha · _dark_ |
| ---------- | --------------------------- | ----------------------- | -------------------------- | ------------------------- |
| `bg`       | `#fdf6e3`                   | `#002b36`               | `#eff1f5`                  | `#1e1e2e`                 |
| `surface`  | `#eee8d5`                   | `#073642`               | `#e6e9ef`                  | `#181825`                 |
| `border`   | `#cabf9e`                   | `#0c3a46`               | `#bcc0cc`                  | `#45475a`                 |
| `fg`       | `#657b83`                   | `#839496`               | `#4c4f69`                  | `#cdd6f4`                 |
| `emphasis` | `#586e75`                   | `#93a1a1`               | `#2e3047`                  | `#eef1fb`                 |
| `muted`    | `#93a1a1`                   | `#586e75`               | `#8c8fa1`                  | `#6c7086`                 |
| `accent`   | `#268bd2`                   | `#268bd2`               | `#1e66f5`                  | `#89b4fa`                 |
| `working`  | `#2aa198`                   | `#2aa198`               | `#179299`                  | `#94e2d5`                 |
| `asking`   | `#b58900`                   | `#b58900`               | `#df8e1d`                  | `#fab387`                 |
| `blocked`  | `#dc322f`                   | `#dc322f`               | `#d20f39`                  | `#f38ba8`                 |
| `done`     | `#859900`                   | `#859900`               | `#40a02b`                  | `#a6e3a1`                 |

### Diff tints

| Token    | Solarized Light | Solarized Dark | Catppuccin Latte | Catppuccin Mocha |
| -------- | --------------- | -------------- | ---------------- | ---------------- |
| `add`    | `#d8f0d8`       | `#0f2e21`      | `#dbebd6`        | `#24372c`        |
| `remove` | `#f5d6d3`       | `#35211f`      | `#f3d7dd`        | `#3a2531`        |

Across all flavors the role holds: `accent` is a blue, `working` a cool teal/cyan, `asking` a warm
amber, `blocked` a red, `done` a green. The mood shifts; the meaning does not.

---

## Components

Described here as intent and anatomy — the shared parts every tool reuses, not their code.

- **Session / group header.** A quiet `emphasis` label that names a session or a branch and groups
  the agents beneath it. It organizes; it does not compete.
- **Branch headline.** The scannable unit of the sidebar: a state glyph and the branch name, colored
  by the group's most urgent agent. One headline stands for a whole branch, however many agents share
  it.
- **Agent row.** A calm secondary line — `name · state · elapsed` — laid out on the character grid.
  It reports; the headline above it alarms.
- **Selection & hover.** A `selection` fill plus a one-cell `accent` rail. Focus, not decoration.
- **Status widgets.** The small indicators — dictate, notify toggle, agent tally, attention count —
  are one family: an `overlay` chip, a state dot, a right-aligned value. Recording is the only place
  red appears outside _blocked_; transcribing borrows the _asking_ amber.
- **Status-bar segments.** Identity segments (host, session) keep their own colors; window tabs use
  `accent` for the active one and the state language for any window whose agent needs you.
- **Diff rows.** Added and removed lines tinted from `add` / `remove`, a blue hunk marker, and an
  `emphasis` file path — the reviewer speaking the same language as the pane beside it.
- **Cards.** The opt-in dense alternative: one rounded pane per branch with a state-colored left
  rail and a right-aligned _needs you_ tag when it applies.

---

## Usage guidelines

**Do**

- Speak in semantic tokens — reach for `blocked`, `surface`, `accent`, never a raw hex.
- Keep _working_ cool and quiet; let _asking_ and _blocked_ be the loud ones.
- Reserve red for a genuine hard stop. If it isn't blocking, it isn't red.
- Pair every state with its glyph, so color is never the only carrier of meaning.
- Collapse repetition — name a shared branch once, report its agents beneath.
- Let the person pick the theme; keep the default Solarized Light.

**Don't**

- Don't hardcode a color in a single tool; that is how coherence rots.
- Don't invent a new meaning for an existing hue for local convenience.
- Don't animate anything but liveness — no decorative motion.
- Don't auto-switch themes on OS setting, time of day, or focus.
- Don't spend the thirty-column budget on chrome that carries no information.

---

## Accessibility

- **Legible on its own ground.** Every flavor must hold contrast on both light and dark backgrounds;
  state colors must stay readable against `bg` and against `selection`.
- **Never color alone.** Glyph and position carry the same signal as hue, so the system works for
  color-blind readers and in monochrome captures.
- **Respect reduced motion.** The spinner freezes to a static frame; no meaning is lost.
- **Keep the quiet legible.** `muted` and `idle` must stay clearly distinct from `fg`, not fade into
  the background.

---

## Evolving the system

- **One source of truth.** The palette — the semantic roles and each theme's values — is defined
  once and consumed by every tool. Coherence is a property of that single definition, not of
  discipline repeated in a dozen configs.
- **Add flavors freely; add states rarely.** A new theme is routine — a column of values. A new
  _state_ is a significant act: it must earn a distinct glyph and a distinct hue in every flavor, and
  find its place in the urgency order. Prefer reusing the five.
- **The document is the contract.** When a tool or theme changes, this file changes with it. Drift
  between what is written here and what renders in the terminal is a bug, not a detail.

---

## Lineage

The palettes stand on the shoulders of [Solarized](https://ethanschoonover.com/solarized/) and
[Catppuccin](https://catppuccin.com/).
The two-tier token model (primitive → semantic) follows the
[W3C Design Tokens](https://www.designtokens.org/) direction, and the idea of one style guide kept
coherent across many independent tools is borrowed from Catppuccin's
[style guide](https://github.com/catppuccin/catppuccin/blob/main/docs/style-guide.md) and its "ports"
model.
