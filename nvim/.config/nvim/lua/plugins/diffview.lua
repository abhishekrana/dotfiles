return {
  "sindrets/diffview.nvim",
  cmd = { "DiffviewOpen", "DiffviewFileHistory", "DiffviewClose" },
  config = function()
    -- VS Code-like diff colors on Solarized Light
    -- Added lines: very subtle green tint
    vim.api.nvim_set_hl(0, "DiffAdd", { bg = "#dde8cc" })
    -- Removed lines: very subtle red tint
    vim.api.nvim_set_hl(0, "DiffDelete", { bg = "#f0ddd8", fg = "#b58900" })
    -- Changed lines: faint yellow tint
    vim.api.nvim_set_hl(0, "DiffChange", { bg = "#e8e2cc" })
    -- Changed text within a line: slightly stronger green
    vim.api.nvim_set_hl(0, "DiffText", { bg = "#ccdec0" })
    require("diffview").setup({
      diff_binaries = false,
      view = {
        default = {
          layout = "diff2_horizontal",
        },
        merge_tool = {
          layout = "diff3_horizontal",
        },
      },
    })
  end,
}
