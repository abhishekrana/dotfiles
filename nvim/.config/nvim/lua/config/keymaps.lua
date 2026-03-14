-- Keymaps are automatically loaded on the VeryLazy event
-- Default keymaps that are always set: https://github.com/LazyVim/LazyVim/blob/main/lua/lazyvim/config/keymaps.lua
-- Add any additional keymaps here

local map = vim.keymap.set

-- =============================================================================
-- Migrated from VS Code vim settings
-- =============================================================================

-- ff to escape and save (insert + normal mode)
map("i", "ff", "<Esc><cmd>w<CR>", { desc = "Escape and save" })
map("n", "ff", "<cmd>w<CR>", { desc = "Save file" })

-- H/L to beginning/end of line (like ^ and $)
map("n", "H", "^", { desc = "Beginning of line" })
map("n", "L", "$", { desc = "End of line" })
map("v", "H", "^", { desc = "Beginning of line" })
map("v", "L", "$", { desc = "End of line" })

-- <leader>c to change inner word without yanking
map("n", "<leader>c", '"_ciw', { desc = "Change word (no yank)" })

-- LSP shortcuts (matching VS Code bindings)
map("n", "<leader>d", vim.lsp.buf.definition, { desc = "Go to definition" })
map("n", "<leader>r", vim.lsp.buf.references, { desc = "Go to references" })
map("n", "<leader>i", vim.lsp.buf.implementation, { desc = "Go to implementation" })

-- Quit all (<leader>q)
map("n", "<leader>q", "<cmd>qa<CR>", { desc = "Quit all" })

-- Buffer navigation (<leader>j/k)
map("n", "<leader>j", "<cmd>bprevious<CR>", { desc = "Previous buffer" })
map("n", "<leader>k", "<cmd>bnext<CR>", { desc = "Next buffer" })

-- Jump to buffer by number (<leader>1-9)
map("n", "<leader>1", "<cmd>BufferLineGoToBuffer 1<CR>", { desc = "Go to buffer 1" })
map("n", "<leader>2", "<cmd>BufferLineGoToBuffer 2<CR>", { desc = "Go to buffer 2" })
map("n", "<leader>3", "<cmd>BufferLineGoToBuffer 3<CR>", { desc = "Go to buffer 3" })
map("n", "<leader>4", "<cmd>BufferLineGoToBuffer 4<CR>", { desc = "Go to buffer 4" })
map("n", "<leader>5", "<cmd>BufferLineGoToBuffer 5<CR>", { desc = "Go to buffer 5" })
map("n", "<leader>6", "<cmd>BufferLineGoToBuffer 6<CR>", { desc = "Go to buffer 6" })
map("n", "<leader>7", "<cmd>BufferLineGoToBuffer 7<CR>", { desc = "Go to buffer 7" })
map("n", "<leader>8", "<cmd>BufferLineGoToBuffer 8<CR>", { desc = "Go to buffer 8" })
map("n", "<leader>9", "<cmd>BufferLineGoToBuffer 9<CR>", { desc = "Go to buffer 9" })

-- Ctrl-u/d scroll 10 lines instead of half-page
map("n", "<C-u>", "10k", { desc = "Scroll up 10 lines" })
map("n", "<C-d>", "10j", { desc = "Scroll down 10 lines" })

-- Swap jump list (Ctrl-i and Ctrl-o swapped, matching your VS Code habit)
map("n", "<C-i>", "<C-o>", { desc = "Jump back" })
map("n", "<C-o>", "<C-i>", { desc = "Jump forward" })

-- Send last yank to right tmux pane
map("n", "<leader>p", function()
  vim.fn.system('tmux send-keys -t "{right}" ' .. vim.fn.shellescape(vim.fn.getreg('"')) .. ' Enter')
end, { desc = "Send yank to right pane" })
