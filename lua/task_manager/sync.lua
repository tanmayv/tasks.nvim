local M = {}

function M.sync_buffer(bufnr)
  if not bufnr or bufnr == 0 then
    bufnr = vim.api.nvim_get_current_buf()
  end
  
  local file_path = vim.api.nvim_buf_get_name(bufnr)
  if file_path == "" then return end

  local tm = require("task_manager")
  
  -- The Go `task sync` modifies files on disk. If the buffer is modified but not saved,
  -- it's not ideal. However, this is usually called on BufWritePost, so the buffer is saved.
  -- We'll execute the background command to sync the file.
  
  -- Use jobstart to run it async
  vim.fn.jobstart({ tm.config.cmd, "sync", file_path }, {
    on_exit = function(_, code)
      if code == 0 then
        -- We might need to reload the buffer if the Go binary added an ID or modified formatting.
        -- checktime will reload if the file changed on disk
        vim.schedule(function()
          vim.cmd("checktime " .. bufnr)
        end)
      end
    end
  })
end

-- Index directory function (just a wrapper around task index)
function M.index_directory(dir_path)
  local tm = require("task_manager")
  vim.fn.system({ tm.config.cmd, "index", dir_path })
end

return M
