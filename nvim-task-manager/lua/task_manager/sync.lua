local parser = require("task_manager.parser")
local db = require("task_manager.db")

local M = {}

function M.sync_buffer(bufnr)
  if not bufnr or bufnr == 0 then
    bufnr = vim.api.nvim_get_current_buf()
  end
  
  local file_path = vim.api.nvim_buf_get_name(bufnr)
  local lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
  local changes = {}
  local current_ids = {}
  
  for i, line in ipairs(lines) do
    local task = parser.parse_line(line)
    if task then
      local prefix = line:sub(1, task.prefix_length - 3)
      if not task.id then
        task.id = parser.generate_id()
      end
      
      -- Format the line to standardize | position
      local new_line = parser.format_line(prefix, task.status, task)
      if line ~= new_line then
        table.insert(changes, { line_number = i, text = new_line })
      end
      
      current_ids[task.id] = true
      
      -- Apply auto-tags based on filepath
      local tm = require("task_manager")
      if tm.config.auto_tags then
        for pattern, tags_to_add in pairs(tm.config.auto_tags) do
          if file_path:match(pattern) then
            for _, tag in ipairs(tags_to_add) do
              if not vim.tbl_contains(task.tags, tag) then
                table.insert(task.tags, tag)
              end
            end
          end
        end
      end
      
      -- Upsert to DB
      if db.db then
        db.upsert_task(task, file_path, i)
      end
    end
  end
  
  -- Apply changes to buffer
  for _, change in ipairs(changes) do
    vim.api.nvim_buf_set_lines(bufnr, change.line_number - 1, change.line_number, false, { change.text })
  end

  -- Delete missing tasks
  if db.db then
    local db_tasks = db.db.tasks:get({ where = { file_path = file_path } })
    if db_tasks then
      for _, db_task in ipairs(db_tasks) do
        if not current_ids[db_task.id] then
          db.db.tasks:remove({ id = db_task.id })
        end
      end
    end
  end
end

-- Scan a directory for markdown files and sync them
function M.index_directory(dir_path)
  -- Simple find command
  local handle = io.popen('find "' .. dir_path .. '" -name "*.md"')
  if not handle then return end
  
  local result = handle:read("*a")
  handle:close()
  
  for file in string.gmatch(result, "[^\n]+") do
    local bufnr = vim.fn.bufadd(file)
    vim.fn.bufload(bufnr)
    M.sync_buffer(bufnr)
    
    -- Save the buffer if changes were made
    if vim.api.nvim_buf_get_option(bufnr, 'modified') then
      vim.api.nvim_buf_call(bufnr, function()
        vim.cmd('silent write')
      end)
    end
  end
  
  print("Indexing complete.")
end

return M
