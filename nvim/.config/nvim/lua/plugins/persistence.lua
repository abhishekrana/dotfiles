-- Restore the persistence.nvim session for the cwd when launched via `vimr`
-- (which sets NVIM_RESTORE=1). Plain `nvim` / `vim` start fresh so aliases
-- that pass `-c "<command>"` aren't shadowed by a restored session.
-- :mksession can't capture snacks.explorer windows, so we track its
-- open/closed state in a sidecar marker file and reopen it after load.

local function explorer_open()
  for _, win in ipairs(vim.api.nvim_list_wins()) do
    if vim.bo[vim.api.nvim_win_get_buf(win)].filetype:match("^snacks_") then
      return true
    end
  end
  return false
end

-- One marker per cwd, alongside the session files.
local function marker_path()
  local dir = vim.fn.stdpath("state") .. "/sessions"
  vim.fn.mkdir(dir, "p")
  return dir .. "/explorer_" .. vim.fn.getcwd():gsub("[/:]", "%%")
end

-- Restore only when `vimr` set NVIM_RESTORE=1 and we're at a bare prompt.
local function should_restore()
  return vim.env.NVIM_RESTORE == "1" and vim.fn.argc() == 0 and not vim.g.started_with_stdin
end

return {
  {
    "folke/persistence.nvim",
    opts = { need = 0 },
    init = function()
      local group = vim.api.nvim_create_augroup("persistence_autoload", { clear = true })

      vim.api.nvim_create_autocmd("StdinReadPre", {
        group = group,
        callback = function()
          vim.g.started_with_stdin = true
        end,
      })

      -- Persistence emits this just before :mksession runs.
      vim.api.nvim_create_autocmd("User", {
        group = group,
        pattern = "PersistenceSavePre",
        callback = function()
          if explorer_open() then
            vim.fn.writefile({ "1" }, marker_path())
          else
            vim.fn.delete(marker_path())
          end
        end,
      })

      vim.api.nvim_create_autocmd("VimEnter", {
        group = group,
        nested = true,
        callback = function()
          if not should_restore() then
            return
          end
          local want_explorer = vim.fn.filereadable(marker_path()) == 1
          require("persistence").load()
          if not want_explorer then
            return
          end
          -- Defer so the picker opens after session restore settles;
          -- enter/focus = false keeps the cursor in the file buffer.
          vim.schedule(function()
            local ok, picker = pcall(require, "snacks.picker")
            if ok then
              pcall(picker.pick, "explorer", { enter = false, focus = false })
            end
          end)
        end,
      })
    end,
  },
}
