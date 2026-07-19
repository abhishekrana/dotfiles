# Personal Notes Vault

A plain-markdown personal knowledge vault, edited in Neovim via
[obsidian.nvim](https://github.com/obsidian-nvim/obsidian.nvim) (Obsidian-compatible). **Private** - synced to a private
GitHub repo.

## Structure - the PARA method

PARA organizes notes by **how actionable they are right now, not by topic**. That single idea is the whole method:

| Folder       | What it holds                      | Test                                       |
| ------------ | ---------------------------------- | ------------------------------------------ |
| `projects/`  | a goal with **an end**             | "Does it finish?" - ship a feature, a trip |
| `areas/`     | a **standard to maintain**, no end | "Keep it up forever?" - health, dotfiles   |
| `resources/` | **reference** / topics of interest | "Want it later?" - git notes, recipes      |
| `archive/`   | **done or dormant** items          | "Inactive?"                                |

Supporting folders (the mechanics, not PARA):

- `inbox/` - where new notes land; filed into PARA later
- `dailies/` - one note per day (`YYYY-MM-DD.md`)
- `templates/` - note templates
- `assets/` - pasted images and attachments
- `Home.md` - Map of Content; the hand-curated index / front door

Folders answer _where a note belongs_; `#tags` answer _what it is about_; `[[links]]` answer _how it connects_ - use all
three rather than forcing everything into folders.

## Frontmatter

Every filed note carries YAML frontmatter (`type` required; `title`, `status`, `tags`, `created`, `updated`
recommended) - it keeps the vault greppable. Full schema and the filing decision tree live in [`CLAUDE.md`](CLAUDE.md).

## Tasks

Tasks live **with their context**, not in one big list: a project's to-dos go in its project note, quick captures in
today's daily note. Use plain `- [ ]` checkboxes (toggle with `<CR>` in obsidian.nvim).

## Workflow

1. **Capture** everything into `inbox/` or today's daily note - zero decisions, zero friction.
2. **Link liberally** with `[[wiki-links]]`; linking to a note that does not exist yet is fine.
3. **File and maintain**: the `vault-manager` skill files captures into PARA, updates existing notes, links, and prunes.
4. **Permanent notes**: durable, reusable ideas become atomic notes in `resources/`, titled as a claim.

## Agent layer

This vault is set up for Claude Code. The global `vault-manager` skill adds and maintains notes (search-first: update,
create, file, link, prune), and `.claude/vault-check.sh` enforces integrity deterministically - run by the skill and by
the git pre-commit hook, which also blocks secrets. See [`CLAUDE.md`](CLAUDE.md) for the details.

## Privacy

Personal and private. The repo stays **private**; no work/company content (that belongs in the separate work vault,
`~/vaults/work`) and no secrets or credentials.
