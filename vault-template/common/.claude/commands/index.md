---
description: Regenerate per-folder index.md files from note frontmatter
---

Regenerate the `index.md` map for each PARA bucket in this vault from note frontmatter. Do NOT touch `Home.md`
(human-owned) or any note body.

Steps:

1. For each of `projects/`, `areas/`, `resources/` (recurse into `resources/` subfolders), find every `*.md` note.
   Exclude `index.md`, `templates/`, and `assets/`.
2. Read each note's YAML frontmatter. Use `title` (fall back to the filename) and `summary` (fall back to the first
   non-empty prose line, trimmed to one sentence).
3. Write `<folder>/index.md` (no frontmatter) as a bulleted list, most-recently-`updated` first. For `resources/`, group
   by subfolder with `##` headings. Line format: `* [Title](relative/path.md) - summary` Append ` (status)` for
   projects/areas whose `status` is not `active`.
4. Report a short summary: notes indexed per bucket, orphans (no inbound `[[links]]` and not in any index), and notes
   missing `title` or `summary`.

Idempotent. Show me the changes; do not commit.
