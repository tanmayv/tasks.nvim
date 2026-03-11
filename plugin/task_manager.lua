-- Setup autocmds and commands

local group = vim.api.nvim_create_augroup("TaskManagerSync", { clear = true })

-- Set up custom syntax highlighting groups
vim.api.nvim_create_autocmd("ColorScheme", {
  group = group,
  callback = function()
    -- Link our custom groups to standard syntax groups, or define custom colors
    vim.api.nvim_set_hl(0, "TaskManagerProject", { link = "Special" })
    vim.api.nvim_set_hl(0, "TaskManagerTag", { link = "Keyword" })
    vim.api.nvim_set_hl(0, "TaskManagerPriority", { link = "WarningMsg" })
    vim.api.nvim_set_hl(0, "TaskManagerDueDate", { link = "DiagnosticInfo" })
    vim.api.nvim_set_hl(0, "TaskManagerId", { link = "Comment" })
    vim.api.nvim_set_hl(0, "TaskManagerMetadata", { link = "Identifier" })
    vim.api.nvim_set_hl(0, "TaskManagerPipe", { link = "NonText" })
  end
})

-- Force an initial load of colors immediately
vim.cmd("doautocmd ColorScheme")

vim.api.nvim_create_autocmd("BufEnter", {
  group = group,
  pattern = "*.md",
  callback = function()
    -- Only apply syntax highlighting to actual task files
    local tm = require("task_manager")
    local file_path = vim.api.nvim_buf_get_name(0)
    local in_dir = false
    
    for _, dir in ipairs(tm.config.directories) do
      if file_path:find(vim.fn.expand(dir), 1, true) == 1 then
        in_dir = true
        break
      end
    end
    
    if in_dir or vim.b.is_task_manager_input or vim.b.is_task_manager_editor then
      -- Apply regex based syntax matches
      vim.cmd([[
        syntax match TaskManagerProject /@[a-zA-Z0-9_-]\+/
        syntax match TaskManagerTag /#[a-zA-Z0-9_-]\+/
        syntax match TaskManagerPriority /+[a-zA-Z0-9_-]\+/
        syntax match TaskManagerDueDate /due:[a-zA-Z0-9_-]\+/
        syntax match TaskManagerId /id:t:[a-zA-Z0-9_-]\+/
        syntax match TaskManagerMetadata /\<[bcl|done]\+:[a-zA-Z0-9_-]\+/
        syntax match TaskManagerPipe /|/
      ]])
    end
  end
})

vim.api.nvim_create_autocmd("BufWritePre", {
  group = group,
  pattern = "*.md",
  callback = function(args)
    local tm = require("task_manager")
    
    -- Fast check if buffer is in one of the configured directories
    local file_path = vim.api.nvim_buf_get_name(args.buf)
    
    local in_dir = false
    for _, dir in ipairs(tm.config.directories) do
      local expanded_dir = vim.fn.expand(dir)
      if file_path:find(expanded_dir, 1, true) == 1 then
        in_dir = true
        break
      end
    end
    
    if in_dir then
      tm.sync_current_buffer()
    end
  end
})

vim.api.nvim_create_autocmd("BufWriteCmd", {
  group = group,
  pattern = "task_manager_edit_*",
  callback = function(args)
    if vim.b[args.buf].is_task_manager_editor then
      require("task_manager.core").apply_editor_changes(args.buf)
      vim.api.nvim_buf_set_option(args.buf, "modified", false)
    end
  end
})

vim.api.nvim_create_user_command("TaskIndex", function()
  require("task_manager").index_tasks()
end, { desc = "Index all task markdown files into the database" })

vim.api.nvim_create_user_command("TaskSync", function()
  require("task_manager").sync_current_buffer()
end, { desc = "Sync the current buffer to the database" })

vim.api.nvim_create_user_command("Tasks", function()
  require("task_manager.telescope").tasks()
end, { desc = "View open tasks using Telescope" })

vim.api.nvim_create_user_command("TaskAdd", function(opts)
  local args = {}
  if opts.args and opts.args ~= "" then
    args.project = opts.args
  end
  require("task_manager.core").open_task_input(args)
end, { 
  desc = "Quickly add a new task to your inbox file",
  nargs = "?"
})

vim.api.nvim_create_user_command("TaskToggle", function(opts)
  local file_path = vim.api.nvim_buf_get_name(0)
  
  -- Handle visual range if provided
  local start_line = opts.line1
  local end_line = opts.line2
  
  local core = require("task_manager.core")
  local toggled_count = 0
  
  for line_number = start_line, end_line do
    local success = core.toggle_done(file_path, line_number)
    if success then
      toggled_count = toggled_count + 1
    end
  end
  
  if toggled_count > 0 then
    if start_line == end_line then
      vim.notify("Task toggled successfully", vim.log.levels.INFO)
    else
      vim.notify(toggled_count .. " tasks toggled successfully", vim.log.levels.INFO)
    end
  else
    vim.notify("No valid tasks found in selection", vim.log.levels.WARN)
  end
end, { 
  desc = "Toggle the completion state of the task on the current line or selection",
  range = true 
})
