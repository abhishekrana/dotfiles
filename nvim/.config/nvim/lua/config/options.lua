-- Options are automatically loaded before lazy.nvim startup
-- Default options that are always set: https://github.com/LazyVim/LazyVim/blob/main/lua/lazyvim/config/options.lua
-- Add any additional options here
vim.o.background = "light"

-- Migrated from VS Code settings
vim.o.colorcolumn = "120"          -- ruler at column 120
vim.o.tabstop = 4                  -- tab display width
vim.o.shiftwidth = 4               -- indent width
vim.o.softtabstop = 4              -- tab key width
vim.o.fixendofline = true          -- ensure final newline
vim.opt.clipboard = "unnamedplus"  -- yank to system clipboard


-- Copy to both clipboard and primary selection (for Shift+Insert)
vim.api.nvim_create_autocmd("TextYankPost", {
  callback = function()
    local text = vim.fn.getreg('"')
    vim.fn.system("wl-copy --primary", text)
  end,
})
