return {
  {
    "Mofiqul/vscode.nvim",
    lazy = false,
    priority = 1000,
    config = function()
      require("vscode").setup({
        style = "dark",
        transparent = false,
        italic_comments = true,
      })
      require("vscode").load()
    end,
  },
  {
    "LazyVim/LazyVim",
    opts = {
      colorscheme = "vscode",
    },
  },
}
