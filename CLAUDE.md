# CLAUDE.md

## Project overview

Personal dotfiles managed with GNU Stow on Ubuntu 24.04.

## Stow packages

`stow <pkg>` from the repo root links a package into `$HOME`. Only customizations are tracked - never stock Ubuntu
defaults. Per-package pitfalls load from `.claude/rules/` when you touch those files; read the nested `CLAUDE.md` in
`dictate/` before changing it.

| Package    | Links to                                               |
| ---------- | ------------------------------------------------------ |
| `bash/`    | `~/.bashrc.d/`                                         |
| `bat/`     | `~/.config/bat/`                                       |
| `claude/`  | `~/.claude/` settings, the status lines and their hook |
| `clip/`    | `~/.local/bin/clip` - the one clipboard path           |
| `dictate/` | `~/.local/bin/dictate` - local Whisper dictation       |
| `ghostty/` | `~/.config/ghostty/`                                   |
| `git/`     | `~/.config/git/config`                                 |
| `herdr/`   | `~/.local/bin/herdr-forge` - forge line, link chooser  |
| `hunk/`    | `~/.config/hunk/` - diff viewer                        |
| `leaf/`    | `~/.config/leaf/` - markdown previewer                 |
| `nvim/`    | `~/.config/nvim/` - LazyVim                            |
| `theme/`   | `~/.local/bin/theme` - re-skins the whole stack        |
| `tmux/`    | `~/.tmux.conf` and `~/.local/bin/` scripts             |
| `trace/`   | `~/.local/bin/dotfiles-trace`                          |
| `yazi/`    | `~/.config/yazi/`                                      |

The terminal stack is one design: every flavor comes from `design/palette.toml` via the `theme` switcher, so no tool
here carries its own colours.

## Apps (built from source)

Buildable projects live under `apps/` - these are **not** stow packages and are never passed to `stow`. Each carries its
own `Makefile` with a uniform `build` target, so `bootstrap.sh` builds any language the same way and the toolchain gets
a pinned `install_*` step. Add one by dropping a project with a `Makefile` under `apps/`.

- `apps/agentbar/` → **two binaries from one Go module**, each with its own nested `CLAUDE.md` - read the relevant one
  before touching the code. `bin/agentbar` is the tmux sidebar, loaded by a `run-shell` line at the end of
  `tmux/.tmux.conf` and invoked by the Claude lifecycle hooks in `claude/.claude/settings.json` at
  `$HOME/dotfiles/apps/agentbar/bin/agentbar`. `bin/workdesk` is the GitLab work inbox (`cmd/workdesk/CLAUDE.md`),
  toggled by `Alt+n` and the `≡ workdesk` chip - both through `tmux-workdesk.sh` so the two cannot drift, by absolute
  path so the binding never depends on tmux's PATH. Separate commands, not subcommands: agentbar runs on every lifecycle
  event and must not carry a forge client, so a GitLab failure can never be a sidebar failure.
- `apps/folio/` → **a markdown reader that reads like a page**, in Rust (ratatui, comrak). One binary, `bin/folio`,
  linked into `~/.local/bin` by `bootstrap.sh`. The look is a TOML style file (`styles/github.toml` first) naming
  palette roles, so every flavor in `design/palette.toml` works and the `theme` switcher drives it via `FOLIO_THEME`.
  `--inline` renders to stdout for previews. Rust is pinned twice by necessity - `RUST_VERSION` in `install.sh` and
  `rust-toolchain.toml` - and `task conf` checks they agree. It has its own nested `CLAUDE.md` and `DESIGN.md` - read
  both before touching it. leaf stays the previewer until folio reaches parity.

## Installing software

`install.sh` is the only thing in this repo that downloads a tool, and it holds every version pin. It doubles as a CLI
so CI installs with the same code a machine does:

```sh
./install.sh                 # list the steps
./install.sh all             # every tool (bootstrap.sh calls this)
./install.sh gate-tools      # just what `task check` needs (CI calls this)
./install.sh install_tmux    # one step by name
```

`bootstrap.sh` sources it and adds the machine wiring: stow, the `.bashrc` patch, the vaults, the resurrect timer and
`apps/` builds. It takes no arguments - run a single step through `install.sh`. Steps needing the stowed configs in
place run after `stow_packages`.

- **tmux is pinned and built from source.** Ubuntu 24.04 ships 3.4, which the sidebar's e2e suite fails on.
- **herdr is a second writer to the stowed `claude/.claude/settings.json`**: `install_herdr` runs
  `herdr integration install claude`, which adds a `SessionStart` entry there and writes a hook and a generated skill
  under `~/.claude/`. Both are herdr-managed and deliberately untracked, so an integration update shows up as a diff in
  that tracked file. `install_herdr_dictate` follows the pinned ref, upgrading a plugin installed from a different one,
  but leaves a plugin linked from a working tree alone - a checkout is never replaced by the release.

## Release furniture

- `.github/workflows/` - `ci.yml` on every push/PR, `release.yml` on a `v*` tag
- `cliff.toml` - git-cliff config: Conventional Commits to CHANGELOG.md and release notes

## Debugging

**Read the trace log first** when anything in the tmux/agent workflow misbehaves: `dotfiles-trace tail -f`, or
`dotfiles-trace show --since 5m --src <src> --grep <pat>`. It is always on, records action edges across the whole
interactive stack, and lives at `${XDG_STATE_HOME:-~/.local/state}/dotfiles/trace.log` (never committed, 1 MiB with one
rotation). `.claude/rules/trace.md` has the per-symptom guide and the rules for adding a trace point.

## Vault template

`vault-template/` holds the boilerplate for the notes vault (`~/vaults/work`). Like `apps/`, it is **not** a stow
package - `bootstrap.sh` copies it into the vault as **real files**, so the scaffolding is committed into that vault's
own private repo. `common/` carries the skeleton, templates, `.claude/` guardrail hooks + `vault-check` and the
`.githooks/` pre-commit guard; `work/` carries its `CLAUDE.md` + `README.md`. Copies are seed-if-missing, so re-running
bootstrap never clobbers live edits. Vault _content_ never lives here - this repo is public.

## Rules

- **Never commit personal info**: no names, emails, IP addresses, work-specific paths, or employer / product / project
  names. This includes anything read out of a work forge - **no ticket or MR numbers, branch names, CODEOWNERS paths, or
  queue statistics** (counts of open MRs, ages, approval numbers). Those are findings about the employer's codebase, not
  facts about these dotfiles; they belong in the work vault. Tool docs describe behaviour, never the data it returned.
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
- `bootstrap.sh` scaffolds the notes vault (see "Vault template"); a vault with no git remote is reported once at the
  end, and the remote/identity are never created or stored here
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
  `claude`, `clip`, `design`, `dictate`, `folio`, `ghostty`, `git`, `herdr`, `hunk`, `install`, `leaf`, `lint`, `nvim`,
  `release`, `task`, `theme`, `tmux`, `trace`, `vault`, `workdesk`, `yazi`. Omit it only when a change genuinely spans
  everything.
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
needing manual steps bumps the MINOR, everything else the PATCH.

- **A tag with no published Release is not a release.** Check the newest tag has one before tagging; if not, fix the
  failure and ask whether to re-tag or bump - never tag over it.
- **Green `task check` does not mean it will publish.** `release.yml` also runs `test/bootstrap-fresh.sh`, which the
  gate does not: run `task fresh` before tagging.

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
- **120 columns**, enforced per language: `task width` (shell, Go, Rust), `ruff` E501 (Python), prettier `printWidth`
  (markdown, yaml, json). Markdown table rows and long inline-code spans can exceed it - prettier will not break an
  unbreakable token.
- Always use a plain hyphen (`-`), never em or en dashes
