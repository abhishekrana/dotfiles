---
description: Report orphans, broken links, and backlink suggestions (proposal-only)
---

Audit link health across this vault. Report only; propose fixes and apply nothing without my approval.

1. Broken `[[links]]`: wiki-links whose target file does not exist. List each with its source note.
2. Orphans: notes with no inbound links and not listed in any `index.md`.
3. Backlink suggestions: for each orphan, suggest 1-3 existing notes it could link to or from (grep to verify the
   targets exist - never invent one).
4. Duplicate or near-duplicate titles.

Present a report. If I approve fixes, apply them and run `/index`. Do not commit.
