-- Obsidian-style markdown notes in ~/vaults/work - with [[wiki-links]], backlinks,
-- tags, daily notes and templates. Inline image rendering is handled by image.nvim
-- (see image.lua); clipboard paste by img-clip.
return {
  {
    "obsidian-nvim/obsidian.nvim",
    version = "*",
    ft = "markdown",
    dependencies = { "nvim-lua/plenary.nvim" },
    opts = {
      -- Use the new `:Obsidian <subcommand>` form (keymaps below already do).
      legacy_commands = false,
      -- Keep raw markdown (matches the `conceallevel = 0` markdown autocmd in
      -- config/autocmds.lua). Obsidian's conceal-based UI would fight that, so
      -- leave it off. Image rendering (image.nvim) is independent of this.
      ui = { enable = false },
      workspaces = {
        { name = "vault", path = "~/vaults/work" },
      },
      -- The vault has no inbox: a note is created where it belongs. A note typed here
      -- is a durable claim more often than anything else.
      notes_subdir = "knowledge",
      new_notes_location = "notes_subdir",
      daily_notes = {
        folder = "log",
        date_format = "%Y-%m-%d",
        template = "daily.md",
      },
      templates = {
        folder = "templates",
      },
      attachments = {
        folder = "assets",
      },
      -- Stamp a `type` on every note's frontmatter so the vault stays greppable
      -- (`rg '^type:'`) and legible to Claude Code. New captures default to
      -- `type: note`; obsidian's own id/aliases/tags and any manually-set fields
      -- (e.g. the daily template's `type: daily`) are preserved. /inbox-triage
      -- refines the type (project/area/resource/...) when it files the note.
      frontmatter = {
        func = function(note)
          local out = { id = note.id, aliases = note.aliases, tags = note.tags }
          if note.metadata ~= nil and not vim.tbl_isempty(note.metadata) then
            for k, v in pairs(note.metadata) do
              out[k] = v
            end
          end
          out.type = out.type or "note"
          return out
        end,
      },
      -- Completion is served by obsidian.nvim's built-in LSP (obsidian-ls), which
      -- LazyVim's blink.cmp surfaces as an LSP source; the old `completion.blink`
      -- flag is gone. `min_chars` still gates the ref/tag/new-note sources.
      completion = {
        min_chars = 2,
      },
    },
    config = function(_, opts)
      -- Disable obsidian.nvim's in-process LSP (obsidian-ls). On v3.16.5 it force-enables
      -- a recursive `**/*.md` file watcher (a capability Neovim disables on Linux for
      -- performance) that walks the vault incl. `.git/` and hangs nvim on open/idle/quit.
      -- We don't need a markdown language server: follow-link (<CR>/gf), backlinks, links,
      -- tags, search, rename, daily notes and templates all run through obsidian's own
      -- commands, which call the handlers directly + ripgrep. Only in-buffer [[link]]/#tag
      -- autocompletion and LSP folding are lost. There is no config flag, so no-op the LSP
      -- start function before setup registers its autocmd.
      local ok, lsp = pcall(require, "obsidian.lsp")
      if ok then
        lsp.start = function() return nil end
      end
      require("obsidian").setup(opts)
    end,
    keys = {
      { "<leader>o", "", desc = "+obsidian/notes", ft = "markdown" },
      { "<leader>on", "<cmd>Obsidian new<cr>", desc = "New note" },
      { "<leader>oo", "<cmd>Obsidian quick_switch<cr>", desc = "Quick switch note" },
      { "<leader>os", "<cmd>Obsidian search<cr>", desc = "Search notes (grep)" },
      { "<leader>ot", "<cmd>Obsidian today<cr>", desc = "Today's daily note" },
      { "<leader>oy", "<cmd>Obsidian yesterday<cr>", desc = "Yesterday's daily note" },
      { "<leader>od", "<cmd>Obsidian dailies<cr>", desc = "List daily notes" },
      { "<leader>ob", "<cmd>Obsidian backlinks<cr>", desc = "Backlinks", ft = "markdown" },
      { "<leader>ol", "<cmd>Obsidian links<cr>", desc = "Links in note", ft = "markdown" },
      { "<leader>oT", "<cmd>Obsidian tags<cr>", desc = "Search tags", ft = "markdown" },
      { "<leader>or", "<cmd>Obsidian rename<cr>", desc = "Rename note & update links", ft = "markdown" },
      -- Toggle-checkbox and follow-link are covered by obsidian's built-in
      -- buffer keys: <CR> (smart action) and `gf`. No custom maps needed.
    },
  },

  -- Paste images from the clipboard straight into a note (Wayland-aware via
  -- wl-paste). Saves the file under `assets/` next to the note and inserts the
  -- markdown link at the cursor.
  {
    "HakonHarnes/img-clip.nvim",
    event = "VeryLazy",
    opts = {
      default = {
        dir_path = "assets",
        relative_to_current_file = true,
        use_absolute_path = false,
        file_name = "%Y-%m-%d-%H-%M-%S",
        prompt_for_file_name = false,
      },
      filetypes = {
        markdown = {
          url_encode_path = true,
          template = "![$CURSOR]($FILE_PATH)",
        },
      },
    },
    keys = {
      { "<leader>op", "<cmd>PasteImage<cr>", desc = "Paste image from clipboard", ft = "markdown" },
    },
  },
}
