local sync = require("task_manager.sync")

local M = {}

M.config = {
  cmd = "task",
  directories = { vim.fn.expand("~/tasks") },
}

local db_watcher = nil

function M.setup(opts)
  if opts then
    M.config = vim.tbl_deep_extend("force", M.config, opts)
  end
  
  if vim.fn.executable(M.config.cmd) == 0 then
    vim.notify(string.format("TaskManager: '%s' binary not found. Please install the task manager TUI companion app and ensure it is in your PATH.", M.config.cmd), vim.log.levels.WARN)
  else
    -- Start db watcher to reload buffers/telescope if another instance modifies tasks
    M.start_watcher()
  end
end

function M.start_watcher()
  local db_path = vim.fn.expand("~/.local/share/nvim/task_manager.db")
  -- In case the user changed it in config.json, we'll try to read it via `task meta --json` ? 
  -- We don't have an endpoint for config yet. Assume default for now.
  local uv = vim.uv or vim.loop
  if not uv then return end

  -- Check if db exists
  if vim.fn.filereadable(db_path) == 0 then return end

  db_watcher = uv.new_fs_event()
  db_watcher:start(db_path, {}, function(err, filename, events)
    if err then return end
    vim.schedule(function()
      vim.api.nvim_exec_autocmds("User", { pattern = "TaskManagerUpdated" })
      -- We can optionally checktime here, but checktime checks file changes on disk, not DB changes.
      -- The DB watcher is great to trigger Telescope/LSP refresh.
    end)
  end)
end

function M.sync_current_buffer(bufnr)
  sync.sync_buffer(bufnr)
end

function M.index_tasks()
  for _, dir in ipairs(M.config.directories) do
    local expanded_dir = vim.fn.expand(dir)
    vim.fn.system({ M.config.cmd, "index", expanded_dir })
  end
  vim.notify("TaskManager: All tasks successfully indexed!", vim.log.levels.INFO)
end

function M.is_managed_file(file_path)
  for _, dir in ipairs(M.config.directories) do
    local expanded_dir = vim.fn.expand(dir)
    if file_path:find(expanded_dir, 1, true) == 1 then
      return true
    end
  end
  return false
end

function M.setup_lsp()
  local script_path = debug.getinfo(1, "S").source:sub(2)
  local plugin_root = script_path:match("(.*)/lua/task_manager/init%.lua")
  if not plugin_root then
    plugin_root = vim.fn.fnamemodify(script_path, ":h:h:h")
  end
  
  local cmd = { "nvim", "-l", plugin_root .. "/bin/task_manager_lsp.lua" }

  local lsp_group = vim.api.nvim_create_augroup("TaskManagerLSP", { clear = true })
  
  vim.api.nvim_create_autocmd("FileType", {
    group = lsp_group,
    pattern = { "markdown", "task_add" },
    callback = function(args)
      local filetype = vim.api.nvim_buf_get_option(args.buf, "filetype")
      local should_attach = false
      
      if filetype == "task_add" or vim.b[args.buf].is_task_manager_editor then
        should_attach = true
      else
        local file_path = vim.api.nvim_buf_get_name(args.buf)
        if M.is_managed_file(file_path) then
          should_attach = true
        end
        if vim.b[args.buf].is_task_manager_input then
          should_attach = true
        end
      end
      
      if should_attach then
        vim.lsp.start({
          name = "task-manager-lsp",
          cmd = cmd,
          root_dir = vim.fn.expand(M.config.directories[1]),
          settings = {
            task_manager = {
              cmd = M.config.cmd
            }
          }
        })
      end
    end,
  })
end

return M
