--- Find the nearest parent directory containing a .venv by walking up from the buffer.
--- Compatible with both lspconfig (fname string) and Neovim 0.11 native LSP (bufnr, callback).
local function find_venv_root(bufnr_or_fname, cb)
  local path
  if type(bufnr_or_fname) == "number" then
    path = vim.api.nvim_buf_get_name(bufnr_or_fname)
  else
    path = bufnr_or_fname
  end

  local root = vim.fs.root(path, { ".venv" })
    or vim.fs.root(path, { "pyproject.toml" })
    or vim.fs.root(path, { "setup.py", "setup.cfg", "pyrightconfig.json", ".git" })

  if cb then
    return cb(root)
  end
  return root
end

return {
  -- Use system ruff (via uvx) instead of Mason-installed ruff
  {
    "neovim/nvim-lspconfig",
    opts = {
      servers = {
        -- lang.python extra defaults to basedpyright; we use pyright below.
        basedpyright = { enabled = false },
        ruff = {
          cmd = { "uvx", "ruff", "server" },
          mason = false,
          root_dir = find_venv_root,
        },
        pyright = {
          root_dir = find_venv_root,
          -- Pyright auto-detects .venv in its root_dir, no settings needed.
          -- python.pythonPath is the only LSP setting that works (venv/venvPath
          -- are pyrightconfig.json-only). Set it as a fallback.
          before_init = function(_, config)
            local root = config.root_dir
            if not root then
              return
            end
            local venv_python = root .. "/.venv/bin/python"
            if vim.uv.fs_stat(venv_python) then
              config.settings = vim.tbl_deep_extend("force", config.settings or {}, {
                python = { pythonPath = venv_python },
              })
            end
          end,
        },
      },
    },
  },
  -- Show the active venv in statusline
  {
    "nvim-lualine/lualine.nvim",
    opts = function(_, opts)
      table.insert(opts.sections.lualine_x, {
        function()
          for _, client in ipairs(vim.lsp.get_clients({ bufnr = 0 })) do
            if client.name == "basedpyright" or client.name == "pyright" then
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
