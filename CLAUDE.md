# CLAUDE.md

## Project overview

Personal dotfiles managed with GNU Stow on Ubuntu 24.04.

## Stow packages

- `bash/` → `~/.bashrc.d/` (shell customizations)
- `bat/` → `~/.config/bat/`
- `claude/` → `~/.claude/settings.json`, `~/.claude/statusline-command.sh`, `~/.claude/skills/`. `settings.json` wires
  the agentbar hook on every Claude lifecycle event plus the statusLine; agent state comes from that plugin's `@agent_*`
  pane options, not local hook scripts. Claude Code does not load a user-level `~/.claude/settings.local.json`, so
  anything that must take effect goes in `settings.json`.
- `clip/` → `~/.local/bin/clip` (copy stdin to the clipboard; picks wl-copy, xclip or pbcopy). Every copy path - tmux
  `copy-command`, `tmux-yank.sh`, fzf's Ctrl-Y, nvim - goes through it, so the backend is chosen in one place.
- `dictate/` → `~/.local/bin/dictate` (toggle-key local Whisper dictation into tmux). Has its own nested `CLAUDE.md` -
  read it before touching the script. `bootstrap.sh` installs its deps (`uv`, `pulseaudio-utils`); the model is not
  prefetched, so the first dictation downloads it. Two backends, named for the hardware and picked by what is installed
  rather than an env var: `gpu` (whisper.cpp via Vulkan, on an AMD or Intel iGPU or NVIDIA) once a GPU is visible, else
  `cpu` (faster-whisper). `bootstrap.sh` installs the GPU build when it can and falls back otherwise. Same `small.en`,
  measured 2.7× faster on the GPU.
- `ghostty/` → `~/.config/ghostty/` (Ghostty terminal config)
- `git/` → `~/.config/git/config` (delta pager, merge settings)
- `hunk/` → `~/.config/hunk/` (hunk diff viewer config, Solarized Light theme)
- `leaf/` → `~/.config/leaf/` (leaf markdown previewer config). Carries a full Solarized Light palette as
  `[themes.solarized-light]`, since leaf ships only `solarized-dark`; that registration is what lets the theme switcher
  drive leaf by name via `LEAF_THEME`. **leaf writes this file itself** - a first run with no config seeds upstream's
  sample there, which blocks `stow leaf`, so the backup step in `bootstrap.sh` is load-bearing.
- `nvim/` → `~/.config/nvim/` (LazyVim config)
- `theme/` → `~/.local/bin/theme` (theme switcher; re-skins the terminal stack across the four flavors from
  `design/palette.toml`, writing per-tool files into `~/.config/theme/`)
- `tmux/` → `~/.tmux.conf`, `~/.local/bin/` scripts - see [tmux](#tmux) below
- `trace/` → `~/.local/bin/dotfiles-trace` (shared always-on trace log for the tmux/agent stack; see "Debugging")
- `yazi/` → `~/.config/yazi/` (yazi file manager config). `package.toml` is machine-managed: `ya pkg install` rewrites
  it, so commit exactly what `ya` writes and never comment it. `zoxide` is bundled in yazi core - listing it as a dep
  fails.

### tmux

Scripts in `tmux/.local/bin/`:

- `tmux-gitlab.sh` - GitLab status, `#issue !mr CI ✓`. No words: the sigils are GitLab's own notation, and the CI glyph
  is fixed-width so a flipping pipeline never shifts the clock.
- `tmux-agent-state.sh` - sourced agent-state language: glyphs, colors and state ranking shared by the session picker
  and its preview, mirroring the sidebar's. Colors come from the theme switcher, never hardcoded.
- `tmux-settings.sh` - the `⛭` chip's dialogue: setting names down the left, their values down the right, one value per
  row. No submenu - fzf has one cursor and its preview cannot be focused, so showing every value is what removes the
  need for two. Add a setting with a `group()` call in `list_rows` and an arm in `do_apply`.
- `tmux-reset.sh` - the `prefix + R` UI reset: reload + default geometry, nothing killed.
- `tmux-mockup.sh` - `task mockup`, previews the whole frame with fake data on a private server.
- Session picker, resurrect guard, yank.

Rules:

- **One session order everywhere.** The agentbar sidebar's bands (pinned / active / dormant, alphabetical inside each)
  are the order, `Alt-h`/`Alt-l` walk it row by row and wrap, and the `Alt-;` picker popup renders the same bands. All
  three read `agentbar order`, and `p` (pin) in either view is the only thing that moves a session.
- **Every pane carries a rail** (`pane-border-status top`, `tmux-rail.sh`), same rule for all of them. LEFT, always:
  this pane's folder and branch. RIGHT, only when that pane runs Claude: the worktree it is _writing_ in, which its cwd
  never follows, since the Bash tool's `cd` does not move a pane. Two zones via `#[align=right]` - the left anchored at
  one column, the right growing leftward into rule - so nothing shifts as state changes.
- **A fresh diff pane follows the agent, not the pane.** `tmux-diff-pane.sh` targets `@agent_workdir` (stamped by the
  agentbar hook from each Edit/Write) and records what is on screen in `@diff_target`.
- **The target then sticks.** A mode item changes what you see, not where you look; only `f` (follow),
  `tmux-worktree-picker.sh` (the menu's `W`) and per-window auto-follow (`F`, off by default) re-point a live pane.
- An amber `◧ diff` chip means the worktree is in no agent's `@agent_workdirs`, and reports nothing else. A click only
  opens the menu.
- **Picking a theme is applying it.** The `⛭` chip opens a centred dialogue with every value on screen; a click or Enter
  applies one and the dialogue stays open, so there is no save step and nothing to drill into. `theme <flavor>` re-skins
  tmux and ghostty and re-runs the current session's sidebar and diff pane - hunk takes the flavor as a startup flag, so
  the pane has to be respawned; other sessions recolour on `prefix + R`, because restarting every sidebar at once storms
  this client.
- **The footer holds no per-pane facts** - the work's commit (7-char sha), its CI, the clock, and the `⚙` settings chip
  at the far right, where its fixed width cannot reflow the clock. Dropping the git-status plugin took ~72ms of git off
  every status redraw.

## Apps (built from source)

Buildable projects live under `apps/` - these are **not** stow packages and are never passed to `stow`. Each carries its
own `Makefile` with a uniform `build` target, so `bootstrap.sh` builds any language the same way and the toolchain gets
a pinned `install_*` step. Add one by dropping a project with a `Makefile` under `apps/`.

- `apps/agentbar/` → Go tmux plugin (the Claude agent sidebar). Loaded by a `run-shell` line at the end of
  `tmux/.tmux.conf`, so it builds and runs straight from the repo. The Claude lifecycle hooks in
  `claude/.claude/settings.json` invoke its binary at `$HOME/dotfiles/apps/agentbar/bin/agentbar`. It has its own nested
  `CLAUDE.md` - read that before touching the code.

## Installing software

`install.sh` is the only thing in this repo that downloads a tool, and it holds every version pin. It doubles as a CLI
so CI installs with the same code a machine does:

```sh
./install.sh                 # list the steps
./install.sh all             # every tool (bootstrap.sh calls this)
./install.sh gate-tools      # just what `task check` needs (CI calls this)
./install.sh install_tmux    # one step by name
./install.sh dictate-deps    # uv + pulseaudio-utils (also part of `all`)
./install.sh whisper-vulkan  # whisper.cpp built against Vulkan for dictate's GPU backend (also part of `all`)
```

`bootstrap.sh` sources it and adds the machine wiring: stow, the `.bashrc` patch, the vaults, the resurrect timer and
`apps/` builds. It takes no arguments - run a single step through `install.sh`. Steps that need the stowed configs in
place (`install_bat_themes`, `install_nvim_plugins`) run after `stow_packages`.

**tmux is pinned and built from source.** Ubuntu 24.04 ships 3.4, which the sidebar's e2e suite fails on.

## Release furniture

- `.github/workflows/` - `ci.yml` on every push/PR, `release.yml` on a `v*` tag
- `cliff.toml` - git-cliff config: Conventional Commits to CHANGELOG.md and release notes

## Debugging (trace log)

**Read the trace log first** when anything in the tmux/agent workflow misbehaves. It is always on and records action
_edges_ across the whole interactive stack.

- **Where:** `${XDG_STATE_HOME:-~/.local/state}/dotfiles/trace.log` (outside the repo, never committed). Size-capped at
  1 MiB with one rotation (`trace.log.1`).
- **View:** `dotfiles-trace tail -f`, or
  `dotfiles-trace show --since 5m --src <tmux|agentbar|clip|hook|sidebar|picker|dictate|resurrect|yank> --grep <pat>`.
  `dotfiles-trace path` prints the file.
- **Format:** logfmt - `ts=<iso ms> src=… evt=… pid=… k=v …`. The on-screen status clock is `%H:%M:%S`, so a screenshot
  anchors to a log window.

What to read, by symptom:

- **A click did nothing.** `src=tmux evt=click range=…` means tmux received it, so the failure is downstream - our bug.
  No line for a click you made means the terminal dropped the event first: the known Ghostty+tmux status-click bug, not
  fixable here.
- **A copy did nothing.** Every copy path goes through `clip`, leaving
  `src=clip evt=copy backend=… sel=… bytes=… rc=… wl=… dsp=…`; a mouse yank adds
  `src=yank evt=copy bytes=… rc=… rc_primary=…` for the tmux edge above it. Read in this order: **no `src=yank` line**
  means tmux never fired the yank (the selection or the binding, not the clipboard); **`bytes=0`** means an empty drag;
  **`rc` non-zero** means the backend failed, and `wl=`/`dsp=` say why. A long-lived tmux server keeps the
  `WAYLAND_DISPLAY` it started with, so after a re-login `wl-copy` cannot reach the compositor until the server is
  restarted or its environment updated.
- **The layout drifted.** A squeezed sidebar or skewed split is a screen change: tmux takes a shrink evenly from every
  pane and has no fixed-size pane. `src=sidebar evt=pin` is the `window-resized` hook fixing the width itself.
  `prefix + R` logs `src=tmux evt=reset … changed=N` plus one `evt=layout win=… before=… after=…` per window changed,
  and nothing when nothing had drifted.
- **A session jump landed wrong.** `Alt-h`/`Alt-l` log `src=agentbar evt=switch session=… from=… key=prev|next ms=…`. No
  line means the binary never ran, so the binding fell through to tmux's alphabetical `switch-client` - rebuild it. A
  `session=` that is not the neighbouring row means the bands moved under you; `agentbar order` prints the list the keys
  walk, and `src=agentbar evt=pin session=… pinned=…` is every pin change from either view.
- **The sidebar state looks stale.** `src=hook evt=event name=… prev=… new=… sid=…` is ground truth of what Claude told
  the sidebar (`via=cwd` means the pane was recovered by the cwd fallback - a resumed or `claude daemon run` session
  that fired the hook with no `$TMUX_PANE`). `src=hook evt=drop reason=no_pane cwd=… sid=…` flags a hook that arrived
  with no pane to land on. `$HOME/dotfiles/apps/agentbar/bin/agentbar doctor` rolls this into a per-pane health check -
  the one-command way to spot a stale sidebar.
- **The diff pane shows the wrong tree.** `src=hook evt=workdir pane=… before=… after=…` is every move of an agent's
  worktree; absent means the agent has only read files, or a hook is not wired - the pane then falls back to its own
  cwd. An empty `after=` is a session boundary dropping it, so a new agent cannot inherit the last one's target.
  `src=tmux evt=diff action=create|respawn target=…` is what the pane was pointed at, and `action=follow from=… to=…`
  every catch-up; a `to=` you did not expect means `@agent_workdir` is stale, so read the `evt=workdir` line above it.
  `bg=bg` marks an auto-follow (no focus change), `bg=0` an explicit one.

Writing to it:

- **Two writers, one format:** the `dotfiles-trace` CLI (`trace/`, used by all shell/tmux callers) and the Go
  `apps/agentbar/internal/trace` package (used by the sidebar + hook). Keep them in sync on timestamp, escaping, and
  rotation.
- **Log edges only, never hot loops** (mouse motion, ticks, status redraws, the dictate silence poll, fzf preview/list,
  statusline) - that keeps it free.
- **Toggles:** `tmux set -g @agentbar-trace-verbose on` adds the noisy sidebar events for a live hunt (effect within
  ~1s, no restart). `DOTFILES_TRACE=0` disables entirely; for tmux `run-shell` children use
  `tmux set-environment -g DOTFILES_TRACE 0`.

### Tracing a new feature

- One `dotfiles-trace log <src> <evt> k=v ...` per action edge, never in a hot loop. Reuse an existing `src`.
- Log the outcome, not the intent - `rc=`, `bytes=` are what separate "it ran" from "it worked".
- Changing state: `before=… after=…` on one line, only when it actually changed. Values are capped at 200 chars.
- Tests run under `DOTFILES_TRACE=0`, never into the live log.

## Vault template

`vault-template/` holds the boilerplate for the two notes vaults (`~/vaults/personal`, `~/vaults/work`). Like `apps/`,
it is **not** a stow package - `bootstrap.sh` copies it into each vault as **real files**, so the scaffolding is
committed into that vault's own private repo. `common/` is shared by both vaults (skeleton, templates, `.claude/`
guardrail hooks + `vault-check`, `.githooks/` pre-commit guard); `personal/` and `work/` carry their own `CLAUDE.md` +
`README.md`. Copies are seed-if-missing, so re-running bootstrap never clobbers live edits. Vault _content_ never lives
here - this repo is public.

## Rules

- **Never commit personal info**: no names, emails, IP addresses, work-specific paths, or employer / product / project
  names
- **Audit before committing**: `task secrets` (gitleaks over the tree and the full history) must pass, and eyeball the
  diff for your name, employer, and project names - a scanner won't catch those
- **Only track customizations**: don't add stock Ubuntu defaults (prompt, bash-completion, color aliases) - those belong
  in the system `.bashrc`
- **Prefer `~/.local/bin`** for tool installations over system-wide installs
- **Keep it simple**: no unnecessary abstractions, no over-engineering
- **Keep comments and docs terse**: state the rule, not the story around it. History belongs in the commit message.

## Conventions

- Bash files in `.bashrc.d/` use `.bash` extension
- Only `00-path.bash` has a numeric prefix (must load first for PATH); all other files use plain names
- Each tool init file guards with `command -v tool &>/dev/null || return`
- Scripts under a `.local/bin/` are executable; a fragment meant to be **sourced** says so in its header comment and
  stays non-executable (`task perms` enforces both - tmux swallows a `#()` it cannot execute as empty output, so a
  missing `+x` makes a rail or a status segment silently vanish)
- Private/work-specific config goes in `~/.bashrc.d/local.bash` (not tracked)
- `bootstrap.sh` must be idempotent (safe to re-run)
- `bootstrap.sh` scaffolds the two notes vaults (see "Vault template"); a vault with no git remote is reported once at
  the end, and the remote/identity are never created or stored here
- Keep lists alphabetically sorted (stow packages, apt packages, pinned versions, bootstrap calls, docs)

## Deploy

When I say "deploy": **first commit and push, then make it live** on the running system.

1. **Commit & push** to `main` (run the secrets audit first).
2. **Make it live:**
   - **tmux** (`tmux/.tmux.conf`): `tmux source-file ~/.tmux.conf`. One server is shared by all sessions, so a single
     reload updates every existing session at once.
   - **Stowed scripts** (symlinks - `dictate/`, `bash/`, etc.): live the moment the repo file is saved; no step needed.
   - **New stow package or file**: `cd ~/dotfiles && stow <pkg>`, then reload the relevant tool. A new file in an
     already-stowed package needs this too - until then it has no symlink, and callers using `~/.local/bin` fail
     silently.
   - **`apps/agentbar`** (Go): `task agentbar:build`, then **`prefix + R`** per session to pick up the new binary
     (reloads and restarts that sidebar in place); **`prefix + e` twice** does all sessions, at the cost of the render
     storm. Hook-path edits to the stowed `settings.json` take effect on the next agent lifecycle event.
3. Always list any steps I must run by hand - things that can't be scripted (re-login, `gsettings`/GNOME shortcut
   install, `systemctl --user …`, opening a fresh shell).

## Commits

[Conventional Commits](https://www.conventionalcommits.org/): `type(scope): summary`. The changelog and the release
notes are generated from these, so the type and scope are the machine-readable part - get them right.

- **Types**: `feat` · `fix` · `docs` · `refactor` · `perf` · `test` · `build` · `ci` · `chore`
- **Scope** is the area, matching a stow package, an app, or a repo concern: `agentbar`, `bash`, `bat`, `bootstrap`,
  `claude`, `clip`, `design`, `dictate`, `ghostty`, `git`, `hunk`, `install`, `leaf`, `lint`, `nvim`, `release`, `task`,
  `theme`, `tmux`, `trace`, `vault`, `yazi`. Omit it only when a change genuinely spans everything.
- **Breaking = needs manual steps on the machine.** A `!` after the scope (`feat(tmux)!:`) or a `BREAKING CHANGE:`
  footer marks a release that can't just be pulled - a re-login, a re-stow, a GNOME shortcut, a systemd unit. It renders
  as "needs manual steps" in the changelog and, pre-1.0, drives the MINOR bump.
- `ci:` and `chore(release):` are filtered out of the changelog (see `cliff.toml`).
- Do not add `Co-Authored-By` lines to commit messages

## Releasing

`task check` must be green and pushed first. Then the tag is the trigger: `.github/workflows/release.yml` re-runs the
gate, runs the Docker fresh-install test, and publishes a GitHub Release with notes from `git cliff`. Never move a
published tag - bump the patch instead.

```sh
task check                              # gate: shellcheck, ruff, prettier, gitleaks, tests
task changelog V=v0.2.0                 # prepend the generated section to CHANGELOG.md
git commit -am "chore(release): v0.2.0" && git push
git tag -a v0.2.0 -m "dotfiles v0.2.0"  # annotated SemVer tag
git push origin v0.2.0                  # fires release.yml
```

SemVer, `v`-prefixed. **This repo stays on 0.x - do not bump to 1.0.** Pre-1.0 shifts the meanings down one: a release
needing manual steps bumps the MINOR, everything else bumps the PATCH. `task release-notes` previews what the next
release would publish; `task changelog` prepends to `CHANGELOG.md` rather than regenerating it.

**A tag with no published Release is not a release.** Before tagging, check the newest tag has one. If it does not, fix
the failure and ask whether to re-tag that version or bump - never tag over it.

**Green `task check` does not mean the release will publish.** `release.yml` also runs `test/bootstrap-fresh.sh`, which
the gate does not. Run `task fresh` before tagging.

## Tasks

`Taskfile.yml` holds the routine work - run `task` for the list. `task check` is the gate CI runs. The `tmux-*` tasks
are the ones that act on the live server (`tmux-reset` is `prefix + R`); the `agentbar:*` tasks delegate to that
project's own build files.

`task check-ci` reruns the tmux-driven suites - the shell gates and the agentbar tests - in a container mirroring the
runner: gawk as `awk`, no `LANG`, `CI` set. Run it before pushing anything that touches tmux, rendering or the pane
protocol. The rest of the gate reads files and is environment-blind. **The runner's `awk` is gawk and Ubuntu's is
mawk**: on `exit` gawk closes the pipe and SIGPIPEs the producer, so under `set -e` + `pipefail` a pipeline that reads
one line and quits dies on CI and passes here. Match a first row with a flag, never `exit`.

## Formatting

- Markdown: `task fmt` (prettier, pinned in `Taskfile.yml` so CI and local agree). `task fmt-check` checks without
  writing.
- Shell: `shfmt -i 4 -ci` (flags in `Taskfile.yml`, not `.editorconfig` - an extensionless script that matched no
  section would silently get tab indent). Never `--keep-padding`: it splits `cmd; cmd` and then aligns to the original
  column, which mangles the file.
- Lint gates on bugs, not style: `shellcheck -S warning` for shell, `ruff --select E9,F` for the Python in `dictate`.
  Deliberate idioms that a linter misreads carry a directive rather than being rewritten.
- **120 columns**, enforced per language: `task width` (shell, Go), `ruff` E501 (Python), prettier `printWidth`
  (markdown, yaml, json). Markdown table rows and long inline-code spans can exceed it - prettier will not break an
  unbreakable token.
- Always use a plain hyphen (`-`), never em or en dashes
