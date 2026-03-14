-- Autocmds are automatically loaded on the VeryLazy event
-- Default autocmds that are always set: https://github.com/LazyVim/LazyVim/blob/main/lua/lazyvim/config/autocmds.lua
--
-- Add any additional autocmds here
-- with `vim.api.nvim_create_autocmd`
--
-- Or remove existing autocmds by their group name (which is prefixed with `lazyvim_` for the defaults)
-- e.g. vim.api.nvim_del_augroup_by_name("lazyvim_wrap_spell")

-- Clean markdown: no spell check, no conceal, no link underlines
vim.api.nvim_create_autocmd("FileType", {
  pattern = "markdown",
  callback = function()
    vim.opt_local.spell = false
    vim.opt_local.conceallevel = 0
    vim.api.nvim_set_hl(0, "@markup.link.label.markdown_inline", { underline = false })
    vim.api.nvim_set_hl(0, "@label.markdown_inline", { underline = false })
  end,
})

-- Auto-open Snacks explorer on startup but keep cursor on the file
vim.api.nvim_create_autocmd("User", {
  pattern = "VeryLazy",
  once = true,
  callback = function()
    Snacks.explorer({ cwd = LazyVim.root() })
    vim.defer_fn(function()
      vim.cmd("wincmd l")
    end, 100)
  end,
})
