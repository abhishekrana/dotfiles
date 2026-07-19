---
description: Propose a PARA destination and frontmatter for each note in inbox/
---

Triage `inbox/`. This is proposal-first: propose where each note goes and wait for my approval before moving anything.

For each note in `inbox/`:

1. Read it and summarize it in one line.
2. Apply the triage decision tree in CLAUDE.md (project / area / resource / archive). If you cannot classify it
   confidently, leave it in `inbox/` and flag it - never guess.
3. Propose: destination path, a `type`, a one-line `summary`, and 2-4 `tags`. Suggest `[[links]]` to existing notes only
   when you have verified the target exists (grep first; never invent a link).

Present all proposals as a table for me to approve or correct. After I approve:

- Move each note to its destination and add the frontmatter. Preserve backlinks - use `:Obsidian rename` semantics, not
  a bare `mv` that breaks links.
- Run `/index`.

Show the diff; do not commit.
