---
name: vault-manager
description:
  Manages the user's plain-markdown notes vault (~/vaults/work or ~/vaults/personal) - adds, updates, files, links, and
  prunes notes, keeping it coherent. Use only when the user explicitly asks to put something in their vault or notes,
  e.g. "save this to my vault" or "add this to my notes". Searches the vault first, follows the target vault's CLAUDE.md
  rules, records source provenance, and never commits.
---

# Vault manager

Create, update, file, reorganize, link, and prune notes in the user's notes vault so the knowledge base stays coherent.
Make the smallest change that satisfies the request; nothing is committed, so the human reviews every change in
`git diff`.

## Steps

1. **Pick the vault.** `~/vaults/work` for work/company content, `~/vaults/personal` for personal. Never mix; if unsure,
   ask.
2. **Read the vault's rules.** Read `<vault>/CLAUDE.md` for its folder layout, filing decision tree, and frontmatter
   schema, and follow them.
3. **Search first.** Before writing, moving, or deleting anything, `rg -i` the vault (filenames, titles, tags, body) to
   see what already exists. This is what keeps the base coherent.
4. **Do the right operation(s):**
   - **Add:** update the existing note if one covers the topic (extend/correct, bump `updated:`); otherwise create a new
     atomic, claim-titled note filed by the decision tree (confident -> `resources/<topic>/`, else `inbox/`).
   - **Correct:** when new information contradicts a note, fix it in place - never leave two versions.
   - **Reorganize:** move a note to the right bucket when its actionability changes. Rename via the vault's rename
     convention so `[[links]]` stay intact - never break backlinks.
   - **Link:** add `[[links]]` to closely related notes you verified exist - never invent a link.
   - **Prune:** merge near-duplicates; delete truly redundant or dead notes. Deletion is destructive - state exactly
     what you will remove and why first, and prefer merging into a better note over deleting outright.
5. **Keep notes atomic** (one idea each; split rather than bolt on) and **stamp provenance** using the vault's schema
   (`source: <repo>@<branch>`, `updated:`).
6. **Verify.** Run `<vault>/.claude/vault-check.sh` and resolve what it flags - fix every `ERROR`, and address warnings
   (unresolved links, missing `type`) where sensible - before finishing.
7. **Do not commit.** Leave every change uncommitted for the human to review in `git diff` and commit on their schedule.

## Rules

- Search before you write, move, or delete - prefer updating / merging / linking over creating a near-duplicate.
- Make the smallest coherent change, and report exactly what changed (created / updated / moved / linked / deleted).
- Record only what the session established; mark anything inferred rather than extracted.
- One idea per note; never edit a note another agent may be writing.
