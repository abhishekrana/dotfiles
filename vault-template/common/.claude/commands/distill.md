---
description: Distill a source or fleeting note into atomic permanent notes in resources/
argument-hint: [path to source note]
---

Distill $ARGUMENTS (a source, clipping, or fleeting note) into atomic permanent notes. Proposal-first.

1. Read the note and identify the distinct, durable ideas. One idea becomes one permanent note.
2. For each idea, propose a permanent note in the right `resources/<topic>/` folder:
   - a claim-titled filename (e.g. `histogram-diff-handles-code-moves.md`, not `diff.md`)
   - `type: permanent` frontmatter with a one-line `summary` and `tags`
   - `[[links]]` back to the source and to related notes you have verified exist (grep first)
3. Keep the original source note; add a link from it to each distilled note.

Show me the proposed notes before creating them. Extract, do not invent: never assert a claim the source does not
support, and mark anything you infer rather than extract. Do not commit.
