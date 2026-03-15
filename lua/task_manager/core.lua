local parser = require("task_manager.parser")
local utils = require("task_manager.utils")

local M = {}

function M.toggle_done(file_path, line_number)
  local bufnr = vim.fn.bufnr(file_path)
  if bufnr ~= -1 and vim.api.nvim_buf_get_option(bufnr, 'modified') then
    -- Save buffer first so CLI operates on latest state
    vim.cmd("silent! write " .. file_path)
  end

  local lines = vim.api.nvim_buf_get_lines(bufnr, line_number - 1, line_number, false)
  if not lines or #lines == 0 then return false end
  
  local line = lines[1]
  local task = parser.parse_line(line)
  if not task or not task.id then return false end
  
  local tm = require("task_manager")
  local job_id = vim.fn.jobstart({ tm.config.cmd, "toggle", task.id, file_path }, {
    on_exit = function(_, code)
      if code == 0 then
        vim.schedule(function()
          if bufnr ~= -1 then
            vim.cmd("checktime " .. bufnr)
          end
        end)
      end
    end
  })
  
  return job_id > 0
end

function M.add_task(description)
  if not description or description == "" then return false end

  local tm = require("task_manager")
  vim.fn.jobstart({ tm.config.cmd, "add", description }, {
    on_exit = function(_, code)
      if code == 0 then
        vim.schedule(function()
          vim.notify("Task added via CLI", vim.log.levels.INFO)
          vim.cmd("checktime")
        end)
      else
        vim.schedule(function()
          vim.notify("Failed to add task via CLI", vim.log.levels.ERROR)
        end)
      end
    end
  })
  
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
  
  -- Write edited lines to temp file
  local edited_lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
  local edited_temp = vim.fn.tempname()
  vim.fn.writefile(edited_lines, edited_temp)
  
  -- Write origins to temp JSON file
  local origins_temp = vim.fn.tempname()
  local json_origins = vim.fn.json_encode(origins)
  vim.fn.writefile({json_origins}, origins_temp)
  
  local tm = require("task_manager")
  vim.fn.jobstart({ tm.config.cmd, "bulk-update", "--edited-file", edited_temp, "--origins", origins_temp }, {
    on_exit = function(_, code)
      vim.schedule(function()
        -- Cleanup temp files
        os.remove(edited_temp)
        os.remove(origins_temp)
        
        if code == 0 then
          vim.notify("Task changes successfully synced!", vim.log.levels.INFO)
          -- Reload all buffers that might have been modified
          vim.cmd("checktime")
        else
          vim.notify("Failed to apply bulk update", vim.log.levels.ERROR)
        end
      end)
    end
  })
end

return M
