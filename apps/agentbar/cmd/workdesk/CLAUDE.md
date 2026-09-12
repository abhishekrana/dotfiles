# CLAUDE.md

`workdesk` - a GitLab work inbox. It shares the `apps/agentbar/` Go module with the sidebar (Go forbids importing
another module's `internal/`, and a second module would mean a third trace writer plus a second copy of the tmux reader)
but is a separate product: separate command, separate UI, nothing of the sidebar in it. The sidebar's own rules are in
`../../CLAUDE.md` and do not apply here.

## Commands

It mirrors the GitLab work you own into `~/.local/state/dotfiles/workdesk/`. `sync` fetches, `open` is the Bubble Tea UI
(inbox · issues · merge requests · agents, `1`-`4` and tab, `?` for help), `board` is the whole queue, `mr <iid>` one
merge request end to end, `matrix` one row per MR and one column per gate, `ready` the actionable rows for an agent.
`bootstrap.sh` links it into `~/.local/bin`, as it does folio: both are CLIs you type, and agentbar is not.

Separate commands, not subcommands: the sidebar runs on every Claude lifecycle event and must not carry a forge client,
so a GitLab failure can never be a sidebar failure.

**`Alt+n` and the `≡ workdesk` chip** toggle the float, both through the dotfiles' `tmux-workdesk.sh` so the two cannot
drift, by absolute path so the binding never depends on tmux's PATH. Bare `workdesk` opens it too.

### The UI

- **Bubble Tea, not fzf.** fzf re-invoked a process per cursor movement, so previews had to be pre-rendered and cat'd,
  band headers had to be fake rows the cursor skipped, and key hints were truncated silently. Here the model is held:
  headers are derived at render time so the cursor is always on a real row, previews are built from the snapshot, and
  `?` renders the keymap so no hint can go missing. The markdown documents are still written and are what
  `workdesk mr <iid>`, `issue <iid>` and `board` print for an agent. Palette is `internal/ui`, generated from
  `design/palette.toml`.
- **The UI never acts.** It records which key was pressed on which row and quits; the caller runs the action. That is
  what keeps every action a plain `workdesk act <key> <ref>` with no terminal, and why write confirms live outside the
  render loop.
- **A click is how you look, and that is all it does.** The wheel walks whichever pane it is over and stops at the ends,
  where `j`/`k` wrap. A click selects a row; on a ticket the preview beside the list is already the detail, so nothing
  is left for a second click. **The agents view is the exception** - a row is a place, not a document, so `↵` and the
  second click go to its pane, which is why `↵` lives in the `?` overlay and not the footer. The guard is in `request`,
  not the caller: a pending action tears the UI down, so a second click would flash and lose the cursor. Tabs and the
  `synced` marker are clickable, a band header answers with the first row under it, and everything fires on release -
  terminals eat the press of a click that also focuses their window. `listItems` is the one pass the renderer, the
  scroll window and the hit test share.
- **The sheet keeps a head line, and it carries the `◧ diff` chip.** The row's name is pinned above the scrolling
  preview. The chip is the only control that is not a row or a tab, so it is held to the same rules: `diffChipSpan` is
  the single geometry the renderer and the hit test share, it is drawn only on a merge request sheet, and a pane too
  narrow to also name the MR draws none rather than offering a click on text never rendered.
- **A link in the preview is clicked, and workdesk answers.** tmux holds the mouse while the float is up, so the
  terminal never gets to make a URL clickable. `findLinks` indexes the _rendered_ preview - every `https://`, and every
  `#1234`/`!1234`, which no terminal could linkify because a bare reference is not a URL. Columns are display columns,
  not bytes; the target comes from the mirror, which is what keeps a host out of this repo.
- **Where you were survives an action.** Every action rebuilds the UI, so `CurrentRef`/`PreviewOffset` go out and
  `Restore` puts cursor and scroll back. A row that has left the view leaves both alone rather than guessing.
- **A float, not a popup.** A popup swallows every click outside its box, so a second click on the chip never reached
  `MouseUp1Status` and it could only ever open. A float (tmux 3.7 `new-pane`) is a pane, so the click lands. One float
  per window. **A float is a column to `select-layout`** - evening a window holding one shrinks it, so `tmux-reset.sh`
  skips those windows whole, and `P` (promote) targets `{last}` because tmux refuses to split a floating pane.
- **The view ring follows the work**: `1` inbox, `2` issues, `3` merge requests, `4` agents - not started, in flight,
  who is doing it, inbox first because it cuts across all three. The `View` const block is the only place that order
  lives; tab bar, digits, help text and `tab`/`shift+tab` all derive from it. It was encoded in five places before,
  which is how three views came to sort one way and three the other.

### Rows and ordering

- **Newest first inside a band, in all six places that sort.** The band is already the priority signal. `prio::` is a
  label on the row, not a second sort - one list cannot be ordered by two things without disagreeing with the other
  five. **The index is a stored artifact, so changing a sort needs `workdesk render`** (no network) before `list`
  reflects it.
- **The model holds no presentation.** Titles stored unpadded, ages not at all - only an epoch. A pre-padded title grows
  an ellipsis it never earned once a UI re-pads it. `Row.TSV()` is the one place a fixed column belongs, because its
  consumer is not this program.
- **Band names are GitLab's own**, from the merge request homepage, so this view and the web UI say the same words. Its
  active/inactive split is modelled too: the picker draws a line where the bands stop asking anything of you.
- **Issues band by GitLab's status, and the lifecycle is read, never written down.** The bands are
  `project.workItemTypes` → the `STATUS` widget's `allowedStatuses`, so a column added upstream appears on the next sync
  and nothing here names a workflow. **Not from a board**: a project carries dozens, they disagree, and picking one
  would be picking a workflow rather than reading it. **The sequence is read from the far end** - furthest along at the
  top, backlog at the bottom, because a list is read top down and the backlog is the biggest band there is. Finished
  categories keep the bottom, since the divider needs them contiguous. The active flag comes from _position_ (where
  `done`/`canceled` begin), not each status's own category, which guarantees the single transition the divider draws. An
  unknown status, and an issue with `no status`, sort after everything known. `s` lists the lifecycle as GitLab declares
  it: stable numbers are what `workdesk act` takes.
- **Labels are a column, never a grouping.** An issue carries several, so any one of them puts it in two places or
  neither. A scoped label shows its value alone (`high · chore`); the preview keeps the full titles. The column is sized
  against the title and given up on a narrow pane.
- **The sprint is a marker**: `◆` between title and age, with two cells reserved on every issue row so the age column
  does not move as the sprint changes.

### Previews

- **A preview carries the ticket, not a link to it.** Description, comments and assignees are fetched and rendered,
  system notes dropped. Comments are whole on an issue and first lines only on a merge request: an issue's argument is
  the content, where a merge request's annotates a diff you can go and read. **Bodies are wrapped to the pane**
  (`renderPreview` wraps once, at the end) - the viewport truncates what it cannot fit, so the right-hand half of every
  long line was silently absent before. `workdesk preview` on a command line has no pane and stays unwrapped.
- **The body is rendered markdown, not its source**, via glamour - what glab renders with, and a parser rather than a
  set of patterns, which is what makes nested lists, tables and reflow come out right. It costs ~8.7MB of binary (most
  of it chroma's lexers) and ~1.3ms a render. `markdownStyle` is built in Go from the theme, because a second palette
  that only approximated `design/palette.toml` would show. Two renderers are held and rebuilt on resize alone, the
  second an indent narrower for a comment body. No renderer falls back to the wrapped source.

### Fetching

- **"Can I merge it" comes from `mergeabilityChecks`, never inferred.** `detailedMergeStatus` names one blocker and is
  computed lazily (`UNCHECKED` for much of any real queue); `mergeabilityChecks` returns every gate with its own state.
  The identifier→message map is deliberately open: GitLab adds checks and does not document the set, so an unknown one
  degrades to a readable label.
- **`approvalState.rules` is what explains a stuck MR**: an approval count can read as satisfied while GitLab refuses
  the merge, because the approver was not eligible for the rule that gates it.
- **The mirror has two tiers.** `index.json` is a few kilobytes and holds only what rows need; the full snapshot and
  pre-rendered documents are read by nothing interactive. Ages are epochs formatted at read time - a baked-in "3d" is
  wrong by morning.
- **A manifest first, then only what moved.** A detail node costs GitLab a fraction of a second on its own, so a sync
  opens with one cheap call per collection (`iid updatedAt`, 100 at a time) and fetches in full only the rows whose
  `updatedAt` moved since the mirror was written. `updatedAt` is the only change token on offer - there is no content
  hash, and the GraphQL endpoint sends an `ETag` but ignores `If-None-Match` (REST does honour it). **The manifest also
  keeps the snapshot full**: it is the authority on which rows are open, so a merged MR falls out by being absent from
  it, and that property has its own test.
- **Detail fetches go out several at a time.** GitLab charges per node either way, so a single long request buys only a
  single slow one - measured, splitting one call into four concurrent chunks cut it to roughly a third. Hence
  `detailChunk`/`detailAtOnce` rather than a cursor walk, which could not overlap at all.
- **A detail fetch asks for the rows it names, and the chunk is capped by price, not latency.** GitLab prices a query by
  the page size asked for, not what comes back, and refuses one above complexity 250 - a small batch asked for as
  `first: 100` was declined outright and failed the whole sync on its first page. `first` is therefore the batch, and 36
  rows is the ceiling `detailChunk` can have. `workdesk schema-check` probes the detail query's shape for that reason.
- **A null `project` is not an empty project.** GitLab answers `project(fullPath:)` for a path it cannot see with a null
  rather than an error - once read as zero rows, silently replacing a good board with an empty one. A remote whose host
  is not glab's is refused for the same reason. `schema-check` validates against the live schema by probing a path that
  cannot exist.
- **A row is yours by author OR assignee, for every account**, and the union is done here: they are different queues, so
  the manifest is asked once per account per relation and deduped by iid in `refresh`. GitLab's own `or:` filter would
  do it in one call but was measured returning a fraction of what its parts return, so it is not used. `meta.user` is
  the token's identity; `meta.users` is every account fetched for.
- A full snapshot every sync, so a merged MR disappears with no cursor state to drift, and the mirror is derived, so
  deleting it costs nothing. It lives outside any repo because MR bodies can carry credentials. Project comes from the
  git remote and identity from glab's token, so nothing here holds a host, group or username.
- **A sync says what it is doing**, because it is slow and the UI is down: `progressLine` draws one row, rewritten in
  place - the project, every leg (identity, merge requests, issues, todos, workflow, writing) with the rows it brought
  back, and the seconds so far. Naming the legs is the point; "syncing…" says nothing about whether it is stuck. Silent
  when stdout is not a terminal.
- **`r` refreshes what is on screen, so the mirror is the fallback.** The working directory wins when it names a GitLab
  project - that is what lets a `cd` point the board elsewhere - but it usually names none, since the float inherits the
  cwd of the pane it opened from. The trace's `via=cwd|mirror` says which answered.
- **The todo feed is asked for by action, not filtered after the fact.** GitLab never marks todos done, so the pending
  list is an accumulating log, overwhelmingly machine notifications about state the bands already report. Only the
  actions the bands cannot derive are wanted (`assigned`, `mentioned`, `directly_addressed`, `marked`), so `TodoActions`
  asks for exactly those, one call each and concurrent. `informativeActions` stays as the local net and is the same set,
  so request and filter cannot drift. `TodoMaxAge` drops stale ones at render, and the band header owns up to anything
  left out.
- **How far back the inbox reaches is a window, and `w` widens it.** It opens on the last week (`inbox_since`,
  `WORKDESK_WINDOW` overrides) and `w` cycles that stop, a month, then the whole queue - only ever widening until it
  wraps. **The window is the inbox's alone**: views 2 and 3 are the complete lists you widen into, and `list`/`ready`
  are never windowed, since an agent handed a shortened queue is one quietly missing work. **It is a lens, not a
  reclassification** - membership is settled before the window, so an item ageing out cannot look like it changed bands.
  Two things say so on screen: the window in the tab bar, accented only while holding rows back, and one pinned line at
  the foot naming the count and the next stop. **The count is one total, not a tally per band**, because a band with no
  rows left has no header to carry one.
- **Whose work it is, is configurable, and the file is not in this repo.** `~/.config/workdesk/config.toml` holds
  `accounts` and `inbox_since` (`WORKDESK_CONFIG` overrides the path); `@me` is whoever glab authenticates as, so the
  file never carries a generated username, and no config means that identity alone. A username is exactly what must
  never be committed here. The parser **refuses a line it does not understand by name**: a silently ignored table header
  is a board that looks complete while holding one account's work.

### Diffs and writes

- **`D` reads a merge request's diff, fetched, with the files around it.** It fetches `refs/merge-requests/<iid>/head`
  (FETCH_HEAD only, so nothing is written into the clone) and runs `hunk diff <base>...<head>`, where **the base is
  GitLab's own `diff_refs.base_sha`, never a local merge-base** - a working clone's `origin/main` is routinely months
  behind, and computing the base against it reported thousands of files changed for a two-file merge request.
- **The patch alone is a command, not a key.** `glab mr diff` into `hunk patch -` needs no clone and no fetch, and is
  the only form that can answer for a project never cloned here - so it stays `workdesk diff <iid> --patch`, which is
  what the missing-clone error points at. Two keys for one question is a question you answer every time. (GitLab's patch
  carries `---`/`+++` pairs but no `diff --git` headers; hunk splits on those alone.)
- **A diff opens in a window of its own, and that window declines the sidebar.** The sidebar follows the session's
  active window, so without declining it moves in - and when hunk exits the window survives holding nothing but a
  full-width sidebar. The window is opened **detached**, marked `@agentbar-skip 1`, and only then selected, so the mark
  is in place before `session-window-changed` fires. `follow.sh` honours that mark on any window. It runs
  `workdesk diff <iid>` rather than a quoted shell pipeline, so there is one command to get right.
- **`d` needs a directory, not a branch.** The diff pane's helper takes a worktree path; handed a branch it answered
  "not a git repo" _and_ exited 0, so the fallback never fired either. `worktreeOn` resolves the branch to the checkout
  holding it, and a branch nothing holds says which key does want it.
- Five keys write to GitLab - `a` assign, `e` auto-merge, `M` merge on a merge request; `s` status and `i` sprint on an
  issue - each behind the one `confirm` gate. `WORKDESK_DRY=1` prints the command and stops, which is what the mockup
  sets. Everything else is read-only.
- **`s` and `i` are one mutation, and `i` is one key both ways.** Neither has a glab subcommand, so both go through
  `workItemUpdate`, addressed as `gid://gitlab/WorkItem/<n>` - the same n GitLab hands out as `gid://gitlab/Issue/<n>`.
  `i` reads the row's `◆` and goes the other way. **A refused mutation is a 200**: GitLab puts its complaint in the
  payload's own `errors` array, which glab does not read, so `Do` reads it - without that, a move GitLab declined
  printed as one that worked.
