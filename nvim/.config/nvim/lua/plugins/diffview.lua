return {
  "sindrets/diffview.nvim",
  cmd = { "DiffviewOpen", "DiffviewFileHistory", "DiffviewClose" },
  config = function()
    vim.api.nvim_set_hl(0, "DiffAdd",    { bg = "#1a2e1a" })
    vim.api.nvim_set_hl(0, "DiffDelete", { bg = "#2e1a1a" })
    vim.api.nvim_set_hl(0, "DiffChange", { bg = "#2a2a1a" })
    vim.api.nvim_set_hl(0, "DiffText",   { bg = "#1e3a1e" })
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
          vim.wo[winid].wrap = true
          -- Always track the first pane per file open
          if not vim.g._diffview_first_win or not vim.api.nvim_win_is_valid(vim.g._diffview_first_win) then
            vim.g._diffview_first_win = winid
            -- Focus the first pane after both load
            vim.defer_fn(function()
              if vim.g._diffview_first_win and vim.api.nvim_win_is_valid(vim.g._diffview_first_win) then
                vim.api.nvim_set_current_win(vim.g._diffview_first_win)
              end
              vim.g._diffview_first_win = nil
            end, 100)
          end
        end,
      },
    })
  end,
}
