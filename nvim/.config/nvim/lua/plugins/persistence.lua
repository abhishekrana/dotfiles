return {
  {
    "folke/persistence.nvim",
    opts = { need = 0 },
    init = function()
      vim.api.nvim_create_autocmd("VimEnter", {
        group = vim.api.nvim_create_augroup("persistence_autoload", { clear = true }),
        nested = true,
        callback = function()
          if vim.fn.argc() > 0 or vim.g.started_with_stdin or vim.env.NVIM_NO_RESTORE then
            return
          end
          require("persistence").load()
        end,
      })
      vim.api.nvim_create_autocmd("StdinReadPre", {
        callback = function()
          vim.g.started_with_stdin = true
        end,
      })
    end,
  },
}
