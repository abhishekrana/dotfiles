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
          winbar_info = false,
        },
        merge_tool = {
          layout = "diff3_horizontal",
        },
      },
      hooks = {
        diff_buf_win_enter = function(_, winid)
          vim.wo[winid].scrollbind = true
          vim.wo[winid].cursorbind = true
          vim.defer_fn(function()
            if vim.api.nvim_win_is_valid(winid) then
              vim.api.nvim_set_current_win(winid)
            end
          end, 100)
        end,
      },
    })
  end,
}
