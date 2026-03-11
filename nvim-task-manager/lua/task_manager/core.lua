local parser = require("task_manager.parser")

local M = {}

function M.toggle_done(file_path, line_number)
  local bufnr = vim.fn.bufadd(file_path)
  if not vim.api.nvim_buf_is_loaded(bufnr) then
    vim.fn.bufload(bufnr)
  end

  local lines = vim.api.nvim_buf_get_lines(bufnr, line_number - 1, line_number, false)
  if not lines or #lines == 0 then return false end
  
  local line = lines[1]
  local task = parser.parse_line(line)
  
  if not task then return false end
  
  -- Toggle logic
  if task.status == "done" then
    task.status = "todo"
    task.metadata["done"] = nil
  else
    task.status = "done"
    -- Use UTC date for consistency
    task.metadata["done"] = os.date("!%Y-%m-%d")
  end
  
  -- Reconstruct line
  local prefix = line:sub(1, task.prefix_length - 3)
  local new_line = parser.format_line(prefix, task.status, task)
  
  -- Update buffer
  vim.api.nvim_buf_set_lines(bufnr, line_number - 1, line_number, false, { new_line })
  
  -- Check if we can write the file (in tests we use non-file buffers)
  if vim.api.nvim_buf_get_option(bufnr, "buftype") == "" then
    vim.api.nvim_buf_call(bufnr, function()
      vim.cmd('silent! write')
    end)
  end
  
  return true
end

function M.add_task(description)
  if not description or description == "" then return false end

  local tm = require("task_manager")
  local configured_inbox = tm.config.inbox_file
  
  -- Safeguard against accidental table/list configuration
  if type(configured_inbox) == "table" then
    configured_inbox = configured_inbox[1]
  end
  
  -- Expand might return a list if someone passed wildcards accidentally, take first
  local expanded = vim.fn.expand(configured_inbox)
  if type(expanded) == "table" then
    expanded = expanded[1]
  end
  
  local inbox_path = expanded
  
  -- Create parent directory if it doesn't exist
  local dir = vim.fn.fnamemodify(inbox_path, ":h")
  if vim.fn.isdirectory(dir) == 0 then
    vim.fn.mkdir(dir, "p")
  end
  
  -- Open or create the buffer
  local bufnr = vim.fn.bufadd(inbox_path)
  if not vim.api.nvim_buf_is_loaded(bufnr) then
    vim.fn.bufload(bufnr)
  end
  
  -- Add the new task line
  local new_task_line = "- [ ] " .. description
  local line_count = vim.api.nvim_buf_line_count(bufnr)
  
  -- Append to the end
  vim.api.nvim_buf_set_lines(bufnr, line_count, line_count, false, { new_task_line })
  
  -- Sync the buffer to parse metadata, add ID, and save to DB
  require("task_manager.sync").sync_buffer(bufnr)
  
  -- Save the file
  if vim.api.nvim_buf_get_option(bufnr, "buftype") == "" then
    vim.api.nvim_buf_call(bufnr, function()
      vim.cmd('silent! write')
    end)
  end
  
  vim.notify("Added task to " .. vim.fn.fnamemodify(inbox_path, ":t"), vim.log.levels.INFO)
  return true
end

function M.open_task_input(opts)
  opts = opts or {}
  
  -- Create a scratch buffer
  local bufnr = vim.api.nvim_create_buf(false, true)
  
  -- Set flags so LSP knows this is our input buffer
  vim.api.nvim_buf_set_name(bufnr, "task_manager_input_" .. bufnr)
  vim.b[bufnr].is_task_manager_input = true
  
  -- Pre-fill with a checkbox
  local initial_line = "- [ ] "
  if opts.project then
    initial_line = initial_line .. "@" .. opts.project .. " "
  end
  
  vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, { initial_line })
  
  -- Calculate centered window position
  local width = math.floor(vim.o.columns * 0.6)
  if width < 50 then width = 50 end
  if width > 120 then width = 120 end
  
  local height = 10 -- Give it some room so they can add multiple lines
  local row = math.floor((vim.o.lines - height) / 2) - 5
  local col = math.floor((vim.o.columns - width) / 2)
  
  local win_opts = {
    relative = "editor",
    width = width,
    height = height,
    row = row,
    col = col,
    style = "minimal",
    border = "rounded",
    title = " Add New Tasks (Ctrl-S to save, Esc to cancel) ",
    title_pos = "center"
  }
  
  -- Open the floating window
  local win = vim.api.nvim_open_win(bufnr, true, win_opts)
  
  -- Set options for the buffer/window
  vim.api.nvim_buf_set_option(bufnr, "filetype", "task_add")
  vim.api.nvim_buf_set_option(bufnr, "bufhidden", "wipe")
  
  -- Use syntax from markdown to keep the nice highlighting
  vim.cmd("setlocal syntax=markdown")
  
  -- Automatically prefix new lines with the checkbox + project context when hitting enter
  vim.cmd("setlocal formatoptions+=r formatoptions+=o")
  vim.api.nvim_buf_set_option(bufnr, "comments", "b:- [ ],b:-")
  
  -- Set mappings
  local map_opts = { noremap = true, silent = true, buffer = bufnr }
  
  -- Ctrl-S: Save tasks
  vim.keymap.set({ "n", "i" }, "<C-s>", function()
    -- Get ALL lines
    local lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
    local added_count = 0
    
    for _, line in ipairs(lines) do
      if line and not line:match("^%s*$") and line ~= "- [ ] " and line ~= "[ ] " and line ~= initial_line then
        -- Remove the prefix we added if they kept it, or handle it organically
        -- NOTE: using the exact same robust regex from parser.lua
        local prefix, status_char, desc = line:match("^(%s*[%-*]%s+)%[([ x/%-])%]%s+(.*)$")
        
        -- Trim any extra whitespace they left trailing at the end when pressing enter
        if desc then
          desc = desc:gsub("%s+$", "")
        end
        
        if not desc then
          -- Fallback: maybe they just typed `[ ] task` without the `- `
          local alt_status, alt_desc = line:match("^%s*%[([ x/%-])%]%s+(.*)$")
          if alt_desc then
            desc = alt_desc:gsub("%s+$", "")
          end
        end
        
        -- If user didn't write a valid checkbox format but just typed raw text, use the whole line
        if not desc or desc == "" then
          desc = line:gsub("%s+$", "")
        end
        
        -- Ensure default project is injected if they typed raw lines without it
        if opts.project and not desc:match("@" .. opts.project) then
          desc = desc .. " @" .. opts.project
        end

        M.add_task(desc)
        added_count = added_count + 1
      end
    end
    
    if added_count == 0 then
      vim.notify("Task creation cancelled (empty tasks)", vim.log.levels.WARN)
    end
    
    -- Close window
    if vim.api.nvim_win_is_valid(win) then
      vim.api.nvim_win_close(win, true)
    end
  end, map_opts)
  
  -- Escape: Cancel
  vim.keymap.set({ "n", "i" }, "<Esc>", function()
    if not vim.api.nvim_win_is_valid(win) then return end
    
    -- Check if buffer has been modified beyond the initial setup
    local lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
    local has_content = false
    
    for _, line in ipairs(lines) do
      if line and not line:match("^%s*$") and line ~= "- [ ] " and line ~= "[ ] " and line ~= initial_line then
        has_content = true
        break
      end
    end
    
    if has_content then
      local choice = vim.fn.confirm("You have unsaved tasks. Discard them?", "&Yes\n&No", 2)
      if choice == 1 then
        vim.api.nvim_win_close(win, true)
        vim.notify("Task creation cancelled", vim.log.levels.INFO)
      end
    else
      vim.api.nvim_win_close(win, true)
      vim.notify("Task creation cancelled", vim.log.levels.INFO)
    end
  end, map_opts)
  
  -- Enter insert mode and go to end of line
  vim.cmd("startinsert!")
end

function M.apply_editor_changes(bufnr)
  local origins = vim.b[bufnr].task_origins or {}
  local lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
  
  local current_tasks = {}
  local parsed_current = {}
  
  -- 1. Parse current buffer to find what tasks still exist and what new ones were added
  for _, line in ipairs(lines) do
    if not line:match("^%s*$") and not line:match("^#") then
      local task = parser.parse_line(line)
      if task then
        if task.id then
          current_tasks[task.id] = { line = line, task = task }
        else
          -- This is a newly added task line without an ID
          -- M.add_task appends `- [ ]` automatically, so we just want to pass the raw description
          local desc = task.description
          if task.project then desc = desc .. " @" .. task.project end
          for _, tag in ipairs(task.tags) do desc = desc .. " #" .. tag end
          if task.priority then desc = desc .. " +" .. task.priority end
          if task.due_date then desc = desc .. " due:" .. task.due_date end
          
          M.add_task(desc)
        end
      end
    end
  end

  -- 2. Figure out deletions and updates grouped by file
  local file_changes = {}
  
  for orig_id, origin in pairs(origins) do
    if not file_changes[origin.file_path] then
      file_changes[origin.file_path] = { deletes = {}, updates = {} }
    end
    
    local current = current_tasks[orig_id]
    
    if not current then
      -- Task was deleted from the editor buffer
      table.insert(file_changes[origin.file_path].deletes, origin)
    else
      -- Task exists, check if modified.
      -- To simplify, we'll just check if the raw line changed.
      -- We must re-format the parsed task to standardize it before comparing, 
      -- or just compare the literal line strings if we trust the user.
      -- Let's compare raw line strings. The user's new string is `current.line`.
      if origin.original_line ~= current.line then
        table.insert(file_changes[origin.file_path].updates, {
          origin = origin,
          new_line = current.line
        })
      end
    end
  end

  -- 3. Apply changes to files
  for file_path, changes in pairs(file_changes) do
    if #changes.deletes > 0 or #changes.updates > 0 then
      -- Load the buffer for this file
      local target_buf = vim.fn.bufadd(file_path)
      if not vim.api.nvim_buf_is_loaded(target_buf) then
        vim.fn.bufload(target_buf)
      end
      
      local target_lines = vim.api.nvim_buf_get_lines(target_buf, 0, -1, false)
      
      -- We must find lines by exact string match because line numbers might have shifted
      -- since they opened the editor buffer.
      
      -- Apply updates
      for _, update in ipairs(changes.updates) do
        for i, t_line in ipairs(target_lines) do
          if t_line == update.origin.original_line then
            target_lines[i] = update.new_line
            break
          end
        end
      end
      
      -- Apply deletes (mark them as nil first to not mess up iteration indices)
      for _, del in ipairs(changes.deletes) do
        for i, t_line in ipairs(target_lines) do
          if t_line == del.original_line then
            target_lines[i] = false -- Mark for deletion
            break
          end
        end
      end
      
      -- Rebuild final lines array
      local final_lines = {}
      for _, l in ipairs(target_lines) do
        if l ~= false then
          table.insert(final_lines, l)
        end
      end
      
      -- Write back to buffer
      vim.api.nvim_buf_set_lines(target_buf, 0, -1, false, final_lines)
      
      -- Save to trigger sync
      if vim.api.nvim_buf_get_option(target_buf, "buftype") == "" then
        vim.api.nvim_buf_call(target_buf, function()
          vim.cmd('silent! write')
        end)
      end
    end
  end
  
  vim.notify("Task changes successfully synced!", vim.log.levels.INFO)
end

return M
