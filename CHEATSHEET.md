# Cheatsheet

Commands and keys to internalize for this setup. Sorted by frequency of use.

---

## Shell (Bash + Tools)

| Key / Command       | Action                                   |
| ------------------- | ---------------------------------------- |
| `cd <partial>`      | zoxide smart jump (learns from usage)    |
| `cdi <partial>`     | zoxide interactive (pick from matches)   |
| `Ctrl+R`            | fzf fuzzy search shell history           |
| `Ctrl+T`            | fzf insert file path (powered by fd)     |
| `Alt+C`             | fzf cd into directory (powered by fd)    |
| `gs` / `gd`         | git status / diff                        |
| `gdl`               | diff of the last commit (HEAD~1..HEAD)   |
| `gl` / `gp` / `gf`  | git log / push / fetch                   |
| `glog`              | git log graph (oneline, decorated)       |
| `gb`                | git branches sorted by recent use        |
| `gdd`               | open diffview in nvim (unstaged changes) |
| `gddm`              | diffview against main branch             |
| `gmr`               | diffview for merge request (vs main)     |
| `gw`                | live git diff in tmux split pane         |
| `lazygit`           | terminal git UI                          |
| `lazydocker`        | terminal docker UI                       |
| `bat <file>`        | cat with syntax highlighting             |
| `rg <pattern>`      | ripgrep — fast recursive search          |
| `fd <pattern>`      | fast find files (respects .gitignore)    |
| `fd -t d <pattern>` | find directories only                    |
| `vim`               | nvim, auto-restores session for cwd      |
| `vimc`              | nvim, skip session restore (clean start) |

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

| Key          | Action                        |
| ------------ | ----------------------------- |
| `<leader>d`  | go to definition              |
| `<leader>r`  | go to references              |
| `<leader>i`  | go to implementation          |
| `<leader>c`  | change word (without yanking) |
| `K`          | hover documentation           |
| `<leader>ca` | code action                   |
| `<leader>cr` | rename symbol                 |
| `<leader>cf` | format file/selection         |

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
| `<leader>l`  | open Lazy plugin manager   |
| `<leader>cm` | open Mason (LSP installer) |
| `<leader>xx` | diagnostics list (trouble) |

### Sessions (persistence.nvim)

Sessions are scoped per cwd and saved on quit. Bare `nvim` in a directory auto-restores buffers and the explorer (if it was open); `vim file` or `vimc` skips restore.

| Key          | Action                              |
| ------------ | ----------------------------------- |
| `<leader>qs` | restore session for cwd             |
| `<leader>ql` | restore last session (any cwd)      |
| `<leader>qd` | stop saving current session on quit |

### tmux Integration

| Key         | Action                            |
| ----------- | --------------------------------- |
| `<leader>p` | send last yank to right tmux pane |

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
| Search & replace across project | `<leader>sr` in nvim (grug-far)                         |
| Review branch changes           | `gddm` or `<leader>gg` then navigate                    |
| Run command in split            | `prefix \|` → run command → `prefix z` to zoom          |
| Copy terminal output            | `Alt+y` → navigate → `v` → select → `y`                 |
| Jump to recent directory        | `cd <partial-name>` (zoxide remembers)                  |
