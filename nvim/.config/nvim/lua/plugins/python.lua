return {
  -- Use system ruff (via uvx) instead of Mason-installed ruff
  {
    "neovim/nvim-lspconfig",
    opts = {
      servers = {
        ruff = {
          cmd = { "uvx", "ruff", "server" },
          mason = false,
        },
      },
    },
  },
  -- Auto-detect and activate venvs for LSP servers
  { "jglasovic/venv-lsp.nvim", opts = {} },
  -- Show the venv that pyright/basedpyright is actually using in statusline
  {
    "nvim-lualine/lualine.nvim",
    opts = function(_, opts)
      table.insert(opts.sections.lualine_x, {
        function()
          for _, client in ipairs(vim.lsp.get_clients({ bufnr = 0 })) do
            if client.name == "basedpyright" or client.name == "pyright" then
              local python_path = vim.tbl_get(client, "config", "settings", "python", "pythonPath")
              if python_path then
                -- Extract service name: .../data-hub/.venv/bin/python -> data-hub
                local service = python_path:match("([^/]+)/.venv/")
                if service then
                  return service
                end
              end
              -- Fallback: use LSP root_dir
              local root = client.config.root_dir
              if root then
                return root:match("([^/]+)$")
              end
            end
          end
          return ""
        end,
        cond = function()
          return vim.bo.filetype == "python"
        end,
        icon = "",
      })
    end,
  },
}
