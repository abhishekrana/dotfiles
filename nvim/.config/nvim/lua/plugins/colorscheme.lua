-- The `theme` switcher writes the active flavor to ~/.config/theme/nvim.lua
-- ({ colorscheme, background }); default to Solarized Light when it's absent.
local flavor = { colorscheme = "solarized", background = "light" }
do
  local ok, t = pcall(dofile, vim.fn.expand("~/.config/theme/nvim.lua"))
  if ok and type(t) == "table" and t.colorscheme then
    flavor = t
  end
end
vim.o.background = flavor.background

return {
  -- Solarized (one scheme for light + dark, driven by vim.o.background).
  {
    "maxmx03/solarized.nvim",
    lazy = false,
    priority = 1000,
    opts = {
      -- Tuned highlights only make sense on the light background.
      on_highlights = function()
        if vim.o.background ~= "light" then
          return {}
        end
        ---@type table<string, vim.api.keyset.highlight>
        return {
          Visual = { bg = "#eee8d5", fg = "#002b36" },
          Search = { bg = "#93A1A1", fg = "#fdf6e3" },
          IncSearch = { bg = "#B58900", fg = "#fdf6e3" },
          CurSearch = { bg = "#B58900", fg = "#fdf6e3" },
        }
      end,
    },
  },

  -- Catppuccin (latte + mocha).
  { "catppuccin/nvim", name = "catppuccin", lazy = false, priority = 1000 },

  -- Tell LazyVim which colorscheme this flavor uses.
  {
    "LazyVim/LazyVim",
    opts = { colorscheme = flavor.colorscheme },
  },
}
