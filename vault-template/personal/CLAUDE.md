# CLAUDE.md

## Overview

Personal knowledge vault - plain-markdown notes edited with Neovim (obsidian.nvim) and Obsidian-compatible. Synced to a
**private** GitHub repo.

**Personal content only.** Anything work- or company-related belongs in the separate work vault (`~/vaults/work`,
private GitLab), never here.

## Structure (PARA + capture)

- `inbox/` - new notes land here first (frictionless capture); file into PARA during the weekly review
- `dailies/` - one note per day (`YYYY-MM-DD.md`), the daily capture log
- `projects/` - active efforts with an end date
- `areas/` - ongoing responsibilities maintained indefinitely
- `resources/` - reference material + atomic permanent notes (nest by topic, e.g. `resources/git/`)
- `archive/` - finished projects and dormant notes
- `templates/` - note templates
- `assets/` - pasted images and attachments
- `Home.md` - Map of Content; the hand-curated front door (humans own this file)
- `.claude/` - guardrail hooks and `vault-check` (see Maintenance)

## PARA - the triage decision tree

Sort by **actionability, not topic**. When filing a note, walk this in order and stop at the first match:

1. Goal AND a defined end (deadline + outcome)? -> `projects/`
2. Ongoing responsibility with a standard to maintain, no end date? -> `areas/`
3. Reference material or a topic of interest, not tied to a current goal? -> `resources/` (nest by topic)
4. Finished or dormant? -> `archive/`
5. Cannot decide confidently? -> leave in `inbox/` and flag it. Never guess.

The same topic moves between buckets as its role changes: when a project finishes, archive it; when an area spawns a
concrete goal, make it a project. **Project vs Area is the key call - projects end, areas do not.** Keep PARA flat, nest
only inside `resources/`. No "false projects" (no-deadline dreams or hobbies belong in `resources/`); aim for ~10-15
active projects; never pre-create an empty folder.

## Frontmatter schema

Every filed note (not raw `inbox/` or `dailies/` captures) carries YAML frontmatter. Only `type` is required; the rest
are recommended and keep the vault greppable (`rg '^type: project' -l`, `rg '^status: active'`).

```yaml
---
type: permanent # note | project | area | resource | permanent | source | moc | daily
status: active # projects/areas only: active | someday | done | archived
title: Histogram diff handles code moves better
created: 2026-07-19
updated: 2026-07-19
tags: [git, diff]
aliases: []
---
```

## Conventions

- Capture-first: new notes are created in `inbox/` (`notes_subdir = "inbox"`), filed into PARA later
- Descriptive, claim-titled filenames for durable notes; timestamps only for `dailies/` and fleeting captures
- Tasks are plain `- [ ]` checkboxes inside the relevant project or daily note - not a central todo file; toggle with
  `<CR>` / `:Obsidian toggle_checkbox`
- Folders = where a note belongs, `#tags` = what it is about, `[[links]]` = how it connects
- Link liberally with `[[wiki-links]]`; a link to a not-yet-created note is fine - but never invent a link to a note you
  have not verified exists (grep first)
- Permanent notes are atomic (one idea) and titled as a claim, not a vague topic
- Rename notes via obsidian.nvim (`:Obsidian rename`) so backlinks update - do not `mv` by hand

## Maintenance

Notes are added and maintained by the global `vault-manager` skill: it searches first, then updates the right existing
note or creates a new atomic one, files it by the decision tree above, links related notes, and prunes duplicates -
always leaving changes uncommitted for you to review.

Integrity is deterministic, not LLM-judged: `.claude/vault-check.sh` flags broken frontmatter, empty notes, duplicate
names, and unresolved links. It runs as the vault's git pre-commit hook (a commit is blocked on hard errors;
`git commit --no-verify` overrides) and `vault-manager` runs it after its changes. The pre-commit hook also rejects
secrets, and a `.claude/` hook blocks file access outside this vault root.

## Rules

- **No work content**: no company names, work paths, credentials, or secrets - those live in the work vault
  (`~/vaults/work`)
- **Private**: this vault holds personal, private information. The GitHub repo must stay **private** at all times -
  never make it public, never push to a public remote, never share its contents externally
- **No secrets or credentials**: never store passwords, API keys, tokens, private keys, or sensitive identifiers
  (account / financial / ID numbers) in notes - even in a private repo. Keep them in a password manager and reference
  them, never paste the values

## Commits

- Do not add `Co-Authored-By` lines to commit messages
- Commits use the repo-local GitHub no-reply `user.email` - never change it to a real or work address

## Formatting

- Wrap Markdown prose at **<= 120 characters** per line
- Format Markdown with `npx prettier --write <file>.md` (the `.prettierrc` here sets the 120-char width)
- Always use a plain hyphen (`-`), never em or en dashes
