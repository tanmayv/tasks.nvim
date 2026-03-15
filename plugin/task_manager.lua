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

local function apply_syntax(bufnr)
  -- Only apply syntax highlighting to actual task files
  local tm = require("task_manager")
  local file_path = vim.api.nvim_buf_get_name(bufnr)
  local in_dir = tm.is_managed_file(file_path)
  
  if in_dir or vim.b[bufnr].is_task_manager_input or vim.b[bufnr].is_task_manager_editor then
    vim.api.nvim_buf_call(bufnr, function()
      -- Apply regex based syntax matches bound to the specific buffer window
      vim.cmd([[
        syntax match TaskManagerProject /@[a-zA-Z0-9_-]\+/ containedin=ALL
        syntax match TaskManagerTag /#[a-zA-Z0-9_-]\+/ containedin=ALL
        syntax match TaskManagerPriority /+[a-zA-Z0-9_-]\+/ containedin=ALL
        syntax match TaskManagerDueDate /\<\(due\|start\):[a-zA-Z0-9_-]\+/ containedin=ALL
        syntax match TaskManagerId /id:t:[a-zA-Z0-9_-]\+/ containedin=ALL
        syntax match TaskManagerMetadata /\<[bcl|done]\+:[a-zA-Z0-9_-]\+/ containedin=ALL
        syntax match TaskManagerPipe /|/ containedin=ALL
      ]])
    end)
  end
end

vim.api.nvim_create_autocmd({"BufEnter", "FileType"}, {
  group = group,
  pattern = {"*.md", "markdown"},
  callback = function(args)
    apply_syntax(args.buf)
  end
})

-- Apply syntax to any already opened buffers (handles lazy-loading scenarios)
for _, bufnr in ipairs(vim.api.nvim_list_bufs()) do
  if vim.api.nvim_buf_is_valid(bufnr) and vim.api.nvim_buf_is_loaded(bufnr) then
    local file_path = vim.api.nvim_buf_get_name(bufnr)
    if file_path:match("%.md$") then
      apply_syntax(bufnr)
    end
  end
end

vim.api.nvim_create_autocmd("BufWritePost", {
  group = group,
  pattern = "*.md",
  callback = function(args)
    local tm = require("task_manager")
    
    -- Fast check if buffer is in one of the configured directories
    local file_path = vim.api.nvim_buf_get_name(args.buf)
    
    local in_dir = tm.is_managed_file(file_path)
    
    if in_dir then
      tm.sync_current_buffer(args.buf)
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
