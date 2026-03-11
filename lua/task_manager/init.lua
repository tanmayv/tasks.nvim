local db = require("task_manager.db")
local sync = require("task_manager.sync")

local M = {}

M.config = {
  db_path = vim.fn.stdpath("data") .. "/task_manager.db",
  directories = { vim.fn.expand("~/tasks") },
  inbox_file = vim.fn.expand("~/tasks/inbox.md"),
  auto_tags = {}, -- e.g., { ["/daily/"] = { "daily" } }
}

function M.setup(opts)
  if opts then
    M.config = vim.tbl_deep_extend("force", M.config, opts)
  end
  
  -- Initialize db
  db.init(M.config.db_path)
end

function M.sync_current_buffer()
  sync.sync_buffer()
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
    -- Fallback if pattern matching fails
    plugin_root = vim.fn.fnamemodify(script_path, ":h:h:h")
  end
  
  -- Use nvim -l to execute the lua script, ensuring we have a Lua environment
  -- and access to neovim's standard library if needed
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
        -- Check if file is in our managed directories before attaching LSP
        local file_path = vim.api.nvim_buf_get_name(args.buf)
        if M.is_managed_file(file_path) then
          should_attach = true
        end
        
        -- Also attach if it's our special task input buffer just in case
        if vim.b[args.buf].is_task_manager_input then
          should_attach = true
        end
      end
      
      if should_attach then
        vim.lsp.start({
          name = "task-manager-lsp",
          cmd = cmd,
          root_dir = vim.fn.expand(M.config.directories[1]), -- Arbitrary root
          settings = {
            task_manager = {
              db_path = M.config.db_path
            }
          }
        })
      end
    end,
  })
end

return M
