# CLAUDE.md

## Project overview

Personal dotfiles managed with GNU Stow on Ubuntu 24.04.

## Stow packages

- `bash/` → `~/.bashrc.d/` (shell customizations)
- `bat/` → `~/.config/bat/`
- `claude/` → `~/.claude/settings.json`, `~/.claude/statusline-command.sh`, `~/.claude/skills/` (settings.json wires the agentbar hook on every Claude lifecycle event + the statusLine; `~/.claude/skills/` ships shared, dotfiles-owned skills like `vault-manager`, distinct from personal skills which live in a private branch; agent state for both the sidebar and the `M-;` session picker comes from that plugin's `@agent_*` pane options, not local hook scripts. Note: Claude Code does NOT load a user-level `~/.claude/settings.local.json` - hooks/settings there are silently ignored; anything that must take effect goes in `settings.json`)
- `dictate/` → `~/.local/bin/dictate` (toggle-key local Whisper dictation into tmux; opt-in. Has its own nested `CLAUDE.md` - read it before touching the script. Its deps are opt-in too: `./bootstrap.sh dictate-deps`)
- `ghostty/` → `~/.config/ghostty/` (Ghostty terminal config)
- `git/` → `~/.config/git/config` (delta pager, merge settings)
- `hunk/` → `~/.config/hunk/` (hunk diff viewer config, Solarized Light theme)
- `nvim/` → `~/.config/nvim/` (LazyVim config)
- `theme/` → `~/.local/bin/theme` (theme switcher; re-skins the terminal stack across the four flavors from `design/palette.toml`, writing per-tool files into `~/.config/theme/`)
- `tmux/` → `~/.tmux.conf`, `~/.gitmux.conf`, `~/.local/bin/` scripts (`tmux-gitlab.sh` GitLab status, session picker, resurrect guard, yank)
- `trace/` → `~/.local/bin/dotfiles-trace` (shared always-on trace log for the tmux/agent stack; see "Debugging" below)
- `yazi/` → `~/.config/yazi/` (yazi file manager config)

## Apps (built from source)

Buildable projects live under `apps/` - these are **not** stow packages and are never passed to `stow`. Each is self-contained with its own `Makefile` exposing a uniform `build` target, so `bootstrap.sh` builds any language the same way: `build_apps` loops over `apps/*/` running `make build`, and the toolchain gets a pinned `install_*` step (e.g. `install_go`). Add more apps by dropping a project with a `Makefile` under `apps/`.

- `apps/agentbar/` → Go tmux plugin (the Claude agent sidebar). Loaded by a `run-shell` line at the end of `tmux/.tmux.conf`, so it builds and runs straight from the repo. The Claude lifecycle hooks in `claude/.claude/settings.json` invoke its binary at `$HOME/dotfiles/apps/agentbar/bin/agentbar`. It has its own nested `CLAUDE.md` - read that before touching the code.

## Release furniture

- `Taskfile.yml` - the routine tasks; `task check` is the gate (see "Tasks" below)
- `.github/workflows/` - `ci.yml` on every push/PR, `release.yml` on a `v*` tag
- `cliff.toml` - git-cliff config: Conventional Commits to CHANGELOG.md and release notes
- `CHANGELOG.md` - Keep a Changelog; `0.1.0` hand-written, generated from `0.2.0` on

## Debugging (trace log)

**When something in the tmux/agent workflow misbehaves - a status-bar click didn't register, the sidebar shows the wrong agent state, a session switch felt slow, dictation went nowhere - look at the trace log first.** It is always on and records action _edges_ across the whole interactive stack:

- **Where:** `${XDG_STATE_HOME:-~/.local/state}/dotfiles/trace.log` (outside the repo, never committed). Size-capped at 1 MiB with one rotation (`trace.log.1`).
- **View:** `dotfiles-trace tail -f`, or `dotfiles-trace show --since 5m --src <tmux|agentbar|hook|sidebar|picker|dictate|resurrect|yank> --grep <pat>`. `dotfiles-trace path` prints the file.
- **Format:** logfmt - `ts=<iso ms> src=… evt=… pid=… k=v …`. The on-screen status clock is `%H:%M:%S`, so a screenshot anchors to a log window.
- **Reading the flaky-click case:** a `src=tmux evt=click range=…` line means tmux _received_ the click (so any failure is downstream - our bug); _no_ line for a click you made means the terminal dropped the event before tmux (the known Ghostty+tmux status-click bug, not fixable here).
- **Reading state drift:** `src=hook evt=event name=… prev=… new=… sid=…` is ground truth of what Claude told the sidebar (`via=cwd` means the pane was recovered by the cwd fallback - a resumed / `claude daemon run` session that fired the hook with no `$TMUX_PANE`); `src=hook evt=drop reason=no_pane cwd=… sid=…` flags a hook that arrived with no pane to land on. `agentbar doctor` (run `$HOME/dotfiles/apps/agentbar/bin/agentbar doctor`) rolls this into a per-pane health check - the one-command way to spot a stale sidebar.
- **Two writers, one format:** the `dotfiles-trace` CLI (`trace/`, used by all shell/tmux callers) and the Go `apps/agentbar/internal/trace` package (used by the sidebar + hook) - keep them in sync on timestamp, escaping, and rotation. **Log edges only, never hot loops** (mouse motion, ticks, status redraws, the dictate silence poll, fzf preview/list, statusline) - that keeps it free.
- **Toggles:** `tmux set -g @agentbar-trace-verbose on` adds the noisy sidebar events (mouse motion, ticks) for a live hunt (effect within ~1s, no restart). `DOTFILES_TRACE=0` disables entirely; for tmux `run-shell` children use `tmux set-environment -g DOTFILES_TRACE 0`.

## Vault template

`vault-template/` holds the boilerplate for the two notes vaults (`~/vaults/personal`, `~/vaults/work`). Like `apps/`, it is **not** a stow package and is never passed to `stow` - `bootstrap.sh` copies it into each vault as **real files** (via `seed_vault`), so the scaffolding gets committed into that vault's own private repo and stays portable and self-contained. `common/` is shared by both vaults (the folder skeleton, `Home.md`, note templates, the `.claude/` guardrail hooks + the `vault-check` integrity script, and the `.githooks/` pre-commit guard (integrity + secrets)); `personal/` and `work/` carry the vault-specific `CLAUDE.md` + `README.md`. Copies are seed-if-missing (`copy_if_absent`), so re-running bootstrap never clobbers live edits. Vault _content_ (the notes themselves) never lives here - this repo is public.

## Rules

- **Never commit personal info**: no names, emails, IP addresses, work-specific paths, or employer / product / project names
- **Audit before committing**: `task secrets` (gitleaks over the tree and the full history) must pass, and eyeball the
  diff for your name, employer, and project names - a scanner won't catch those
- **Only track customizations**: don't add stock Ubuntu defaults (prompt, bash-completion, color aliases) - those belong in the system `.bashrc`
- **Prefer `~/.local/bin`** for tool installations over system-wide installs
- **Keep it simple**: no unnecessary abstractions, no over-engineering

## Conventions

- Bash files in `.bashrc.d/` use `.bash` extension
- Only `00-path.bash` has a numeric prefix (must load first for PATH); all other files use plain names
- Each tool init file guards with `command -v tool &>/dev/null || return`
- Private/work-specific config goes in `~/.bashrc.d/local.bash` (not tracked)
- `bootstrap.sh` must be idempotent (safe to re-run)
- `bootstrap.sh` scaffolds two independent notes vaults - `~/vaults/personal` and `~/vaults/work` (`create_personal_vault` / `create_work_vault`, both calling `seed_vault` to copy `vault-template/` in as real files); any vault without a git remote is reported once at the very end (`print_vault_sync_hints`, optional) instead of mid-run, and the remote/identity are never created or stored here, keeping both out of this public repo
- Keep lists alphabetically sorted (stow packages, apt packages, pinned versions, bootstrap calls, docs)

## Deploy

When I say "deploy": **first commit and push, then make it live** on the running system.

1. **Commit & push** to `main` (run the secrets audit first).
2. **Make it live:**
   - **tmux** (`tmux/.tmux.conf`): `tmux source-file ~/.tmux.conf`. One server is shared by all sessions, so a single reload updates every existing session at once.
   - **Stowed scripts** (symlinks - `dictate/`, `bash/`, etc.): live the moment the repo file is saved; no step needed.
   - **New stow package or file**: `cd ~/dotfiles && stow <pkg>`, then reload the relevant tool.
   - **`apps/agentbar`** (Go): `task agentbar:build`, then reload tmux (`tmux source-file ~/.tmux.conf`). The sidebar is a long-lived process, so restarting it is a manual step you must list: **`prefix + e` twice**. Hook-path edits to the stowed `settings.json` take effect on the next agent lifecycle event.
3. Always list any steps I must run by hand - things that can't be scripted (re-login, `gsettings`/GNOME shortcut install, `systemctl --user …`, opening a fresh shell).

## Commits

[Conventional Commits](https://www.conventionalcommits.org/): `type(scope): summary`. The changelog and the release
notes are generated from these, so the type and scope are the machine-readable part - get them right.

- **Types**: `feat` · `fix` · `docs` · `refactor` · `perf` · `test` · `build` · `ci` · `chore`
- **Scope** is the area, matching a stow package, an app, or a repo concern: `agentbar`, `bash`, `bat`, `bootstrap`,
  `claude`, `design`, `dictate`, `ghostty`, `git`, `hunk`, `lint`, `nvim`, `release`, `task`, `theme`, `tmux`, `trace`,
  `vault`, `yazi`. Omit it only when a change genuinely spans everything.
- **Breaking = needs manual steps on the machine.** A `!` after the scope (`feat(tmux)!:`) or a `BREAKING CHANGE:`
  footer marks a release that can't just be pulled - a re-login, a re-stow, a GNOME shortcut, a systemd unit. It drives
  the MAJOR bump and renders as "needs manual steps" in the changelog.
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

SemVer, `v`-prefixed, starting at `v0.1.0` to match the sibling projects. `task release-notes` previews what the next
release would publish. The `0.1.0` entry in `CHANGELOG.md` is hand-written because the history predates the convention;
generation covers `0.2.0` on, which is why `task changelog` prepends rather than regenerating.

## Tasks

`Taskfile.yml` holds the routine work - run `task` for the list. `task check` is the gate CI runs; `task stow`,
`task fmt`, `task trace`, `task tmux-reload`, `task fresh` and the `agentbar:*` tasks cover the rest. Projects under
`apps/` keep their own build files and the `agentbar:*` tasks delegate to them.

## Formatting

- Markdown: `task fmt` (prettier, pinned in `Taskfile.yml` so CI and local agree). `task md` checks without writing.
- Shell: no formatter. `shfmt` is used only to find shell files by shebang for the lint gate - it reflows the compact
  `cmd; cmd` and `|| { … }` idioms this repo uses, so it is deliberately not run with `-w`.
- Lint gates on bugs, not style: `shellcheck -S warning` for shell, `ruff --select E9,F` for the Python in `dictate`.
  Deliberate idioms that a linter misreads carry a directive rather than being rewritten.
- Always use a plain hyphen (`-`), never em or en dashes
