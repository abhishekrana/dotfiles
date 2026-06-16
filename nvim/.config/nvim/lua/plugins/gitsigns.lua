return {
  "lewis6991/gitsigns.nvim",
  opts = {
    current_line_blame = true,
    current_line_blame_opts = {
      virt_text = true,
      virt_text_pos = "eol",
      delay = 300,
      ignore_whitespace = false,
    },
    current_line_blame_formatter = "  <author>, <author_time:%R> · <summary>",
  },
  keys = {
    { "<leader>gb", "<cmd>Gitsigns toggle_current_line_blame<cr>", desc = "Toggle inline git blame" },
  },
}
