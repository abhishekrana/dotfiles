# Cheatsheet

Commands and keys to internalize for this setup. Sorted by frequency of use.

---

## Shell (Bash + Tools)

| Key / Command       | Action                                    |
| ------------------- | ----------------------------------------- |
| `cd <partial>`      | zoxide smart jump (learns from usage)     |
| `cdi <partial>`     | zoxide interactive (pick from matches)    |
| `Ctrl+R`            | fzf fuzzy search shell history            |
| `Ctrl+T`            | fzf insert file path (bat preview)        |
| `Alt+C`             | fzf cd into directory (powered by fd)     |
| `rfv [query]`       | live ripgrep + fzf, opens nvim at line    |
| `gs`                | git status                                |
| `gd` / `gds`        | hunk: review working tree / latest commit |
| `gdw`               | hunk working-tree review, auto-reload     |
| `gl` / `gp` / `gf`  | git log / push / fetch                    |
| `gb`                | git branches sorted by recent use         |
| `gbr`               | remote branches + tip author (last 500)   |
| `gdm`               | hunk: review merge request (vs main)      |
| `gw`                | live git diff in tmux split pane          |
| `gwta`              | add worktree: <dir> <branch>              |
| `gwts`              | fzf-switch worktree (cd into pick)        |
| `gwtm`              | ff worktree's own branch to origin/main   |
| `gwtls`             | list worktrees                            |
| `gwtrm`             | remove a worktree (not rm -rf)            |
| `gwt`               | git worktree (raw passthrough)            |
| `lazygit`           | terminal git UI                           |
| `lazydocker`        | terminal docker UI                        |
| `bat <file>`        | cat with syntax highlighting (paged)      |
| `rg <pattern>`      | ripgrep — fast recursive search           |
| `fd <pattern>`      | fast find files (respects .gitignore)     |
| `fd -t d <pattern>` | find directories only                     |
| `vim`               | nvim (clean start, no session restore)    |
| `vimr`              | nvim, restore saved session for cwd       |

---

### fzf (inside a picker)

| Key      | Action                                                                     |
| -------- | -------------------------------------------------------------------------- |
| `Ctrl+/` | toggle preview pane (in `Ctrl+T` and `rfv`)                                |
| `Ctrl+Y` | copy to clipboard and exit: file contents in `Ctrl+T`, command in `Ctrl+R` |
| `Enter`  | accept selection (in `rfv`: opens nvim at file + line)                     |

---

### hunk (diff viewer)

delta is git's pager, so raw `git diff` / `git show` / `git log` open in delta.
The `gd` / `gds` / `gdw` / `gdm` aliases launch hunk's native interactive
reviewer directly. delta also colors `git add -p` staging and `git blame`.

| Key / Command | Action                                        |
| ------------- | --------------------------------------------- |
| `gd` / `gds`  | native review of working tree / latest commit |
| `gdw`         | working-tree review, auto-reload on change    |
| `gdm`         | review merge request (branch vs main)         |
| `t`           | theme picker inside hunk (persists)           |

---

## Tmux (prefix = Ctrl+b)

### Sessions & Windows

| Key               | Action                              |
| ----------------- | ----------------------------------- |
| `prefix d`        | detach session                      |
| `prefix c`        | new window (inherits cwd)           |
| `prefix x`        | kill pane (no confirmation)         |
| `prefix X`        | kill session                        |
| `prefix ,`        | rename window                       |
| `prefix $`        | rename session                      |
| `Alt+1..9`        | switch to window 1-9 (no prefix!)   |
| `Alt+j` / `Alt+k` | previous / next window (no prefix!) |
| `prefix P`        | move window left                    |
| `prefix N`        | move window right                   |

### Panes

| Key                     | Action                                  |
| ----------------------- | --------------------------------------- |
| `prefix \|`             | split horizontal                        |
| `prefix -`              | split vertical                          |
| `prefix h/j/k/l`        | navigate panes (vim-style)              |
| `Alt+m`                 | cycle panes (no prefix!)                |
| `prefix H/J/K/L`        | resize pane (repeatable)                |
| `prefix >` / `prefix <` | swap pane down / up                     |
| `prefix =`              | tile all panes evenly                   |
| `prefix z`              | toggle pane zoom (fullscreen)           |
| `prefix S`              | toggle synchronized typing to all panes |

### Copy Mode

| Key        | Action                       |
| ---------- | ---------------------------- |
| `Alt+y`    | enter copy mode (no prefix!) |
| `prefix [` | enter copy mode (vi keys)    |
| `v`        | begin selection              |
| `Ctrl+v`   | rectangle selection          |
| `y`        | yank to clipboard + primary  |
| `q`        | exit copy mode               |

### Other

| Key        | Action              |
| ---------- | ------------------- |
| `prefix r` | reload tmux config  |
| `prefix I` | install TPM plugins |

---

## Neovim (LazyVim)

`<leader>` = `Space`

### Essential Motions

| Key                 | Action                                |
| ------------------- | ------------------------------------- |
| `H` / `L`           | beginning / end of line (custom)      |
| `Ctrl+u` / `Ctrl+d` | scroll up / down 10 lines (custom)    |
| `Ctrl+o` / `Ctrl+i` | jump forward / back (swapped)         |
| `ff` (insert)       | escape + save                         |
| `ff` (normal)       | save file                             |
| `s`                 | flash.nvim — jump to any visible text |

### Buffers & Windows

| Key                       | Action                      |
| ------------------------- | --------------------------- |
| `<leader>1..9`            | jump to buffer 1-9          |
| `<leader>j` / `<leader>k` | previous / next buffer      |
| `<leader>bd`              | close buffer                |
| `<leader>qq`              | quit all                    |
| `<leader>h` / `<leader>l` | move to left / right window |
| `Ctrl+h/j/k/l`            | navigate splits             |

### File Navigation

| Key               | Action                        |
| ----------------- | ----------------------------- |
| `<leader><space>` | find files (snacks picker)    |
| `<leader>/`       | live grep (search in project) |
| `<leader>fr`      | recent files                  |
| `<leader>fb`      | open buffers                  |
| `<leader>-`       | open yazi at current file     |
| `<leader>cw`      | open yazi at cwd              |
| `<leader>e`       | toggle file explorer (snacks) |

### LSP & Code

| Key          | Action                |
| ------------ | --------------------- |
| `<leader>d`  | go to definition      |
| `<leader>r`  | go to references      |
| `<leader>i`  | go to implementation  |
| `K`          | hover documentation   |
| `<leader>ca` | code action           |
| `<leader>cr` | rename symbol         |
| `<leader>cf` | format file/selection |

### Git (in Neovim)

| Key                      | Action                   |
| ------------------------ | ------------------------ |
| `<leader>gg`             | open lazygit             |
| `<leader>gB`             | git blame line           |
| `]h` / `[h`              | next / previous git hunk |
| `:DiffviewOpen`          | side-by-side diff        |
| `:DiffviewFileHistory %` | current file history     |

### Search & Replace

| Key                   | Action                                           |
| --------------------- | ------------------------------------------------ |
| `<leader>sr`          | search & replace (grug-far)                      |
| `*`                   | search word under cursor                         |
| `n` / `N`             | next / previous match                            |
| `:cfdo %s/old/new/g`  | replace across all quickfix files                |
| `:cfdo %s/old/new/gc` | replace across all quickfix files (confirm each) |

### UI & Info

| Key          | Action                     |
| ------------ | -------------------------- |
| `:Lazy`      | open Lazy plugin manager   |
| `<leader>cm` | open Mason (LSP installer) |
| `<leader>xx` | diagnostics list (trouble) |

### Sessions (persistence.nvim)

Sessions are scoped per cwd and saved on quit. Use `vimr` to restore buffers and the explorer (if it was open); plain `vim`/`nvim` always starts clean.

| Key          | Action                              |
| ------------ | ----------------------------------- |
| `<leader>qs` | restore session for cwd             |
| `<leader>ql` | restore last session (any cwd)      |
| `<leader>qd` | stop saving current session on quit |

### tmux Integration

| Key         | Action                            |
| ----------- | --------------------------------- |
| `<leader>p` | send last yank to right tmux pane |

### Notes (Obsidian vault in `~/vaults/personal`)

Markdown vault with `[[wiki-links]]`, daily notes and tags (obsidian.nvim).
Images render inline in the buffer (image.nvim) and paste from the clipboard
(img-clip.nvim). Keys below work in markdown buffers.

| Key          | Action                                       |
| ------------ | -------------------------------------------- |
| `<leader>on` | new note                                     |
| `<leader>oo` | quick-switch note                            |
| `<leader>os` | search notes (grep)                          |
| `<leader>ot` | today's daily note                           |
| `<leader>oy` | yesterday's daily note                       |
| `<leader>od` | list daily notes                             |
| `<leader>ob` | backlinks to this note                       |
| `<leader>ol` | links in this note                           |
| `<leader>oT` | search tags                                  |
| `<leader>or` | rename note (updates links)                  |
| `<leader>op` | paste image from clipboard into note         |
| `<CR>`       | smart action (follow link / toggle checkbox) |
| `]o` / `[o`  | next / previous link in note                 |
| `gf`         | follow `[[wiki-link]]` under cursor          |

---

## Ghostty

| Key               | Action                   |
| ----------------- | ------------------------ |
| `Alt+h` / `Alt+l` | previous / next tab      |
| `Ctrl+Shift+,`    | reload config            |
| `Ctrl+Shift+C`    | copy to clipboard        |
| `Ctrl+Shift+V`    | paste from clipboard     |
| select text       | auto-copies to clipboard |

---

## Workflow Combos

| Scenario                        | Keys                                                    |
| ------------------------------- | ------------------------------------------------------- |
| Quick file edit                 | `cd proj` → `vim .` → `<leader><space>` → type filename |
| Jump to code by content         | `rfv parseConfig` → Enter (opens nvim at the line)      |
| Search & replace across project | `<leader>sr` in nvim (grug-far)                         |
| Review branch changes           | `gdm` or `<leader>gg` then navigate                     |
| Run command in split            | `prefix \|` → run command → `prefix z` to zoom          |
| Copy terminal output            | `Alt+y` → navigate → `v` → select → `y`                 |
| Jump to recent directory        | `cd <partial-name>` (zoxide remembers)                  |
