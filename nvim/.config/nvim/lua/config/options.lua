-- Options are automatically loaded before lazy.nvim startup
-- Default options that are always set: https://github.com/LazyVim/LazyVim/blob/main/lua/lazyvim/config/options.lua
-- Add any additional options here
vim.o.background = "dark"

-- Migrated from VS Code settings
vim.o.colorcolumn = "120"          -- ruler at column 120
vim.o.tabstop = 4                  -- tab display width
vim.o.shiftwidth = 4               -- indent width
vim.o.softtabstop = 4              -- tab key width
vim.o.fixendofline = true          -- ensure final newline
vim.opt.clipboard = "unnamedplus"  -- yank to system clipboard
vim.o.relativenumber = false       -- absolute line numbers
vim.o.wrap = true                  -- wrap long lines
vim.o.linebreak = true             -- wrap at word boundaries, not mid-word


-- Mirror yank to the primary selection on Linux/Wayland (so Shift+Insert pastes).
-- macOS has no primary selection, so this is a no-op there.
if vim.fn.executable("wl-copy") == 1 then
  vim.api.nvim_create_autocmd("TextYankPost", {
    callback = function()
      vim.fn.system("wl-copy --primary", vim.fn.getreg('"'))
    end,
  })
end
