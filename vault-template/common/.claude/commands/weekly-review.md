---
description: Assist the weekly review - summarize the week, surface stale and orphan notes, propose archives
---

Run a weekly review of this vault. Proposal-first: report findings and propose changes; apply only what I approve, and
never commit.

1. Summarize the last 7 days of `dailies/`: key themes, decisions made, and `- [ ]` tasks still unchecked.
2. List `projects/` with `status: active` whose `updated` is older than 21 days - candidates to archive or revive.
3. List orphan notes: no inbound `[[links]]` and not referenced by any `index.md`.
4. List broken `[[links]]`: targets that do not resolve to a file.
5. Confirm `inbox/` is empty. If not, tell me to run `/inbox-triage`.
6. Propose a punch list: notes to archive, links to add, sources to `/distill`.

After I approve, apply the changes and run `/index`. Show the diff; do not commit.
