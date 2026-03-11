local M = {}
local sqlite = require("sqlite.db")

M.db_path = vim.fn.stdpath("data") .. "/task_manager.db"
M.db = nil

function M.init(db_path)
  if db_path then
    M.db_path = db_path
  end

  M.db = sqlite({
    uri = M.db_path,
    tasks = {
      id = { type = "text", primary = true, required = true },
      description = { "text" },
      status = { "text" },
      project = { "text" },
      priority = { "text" },
      due_date = { "text" },
      file_path = { "text" },
      line_number = { "integer" },
      created_at = { "integer" },
      updated_at = { "integer" }
    },
    task_metadata = {
      id = true,
      task_id = { type = "text", reference = "tasks.id", on_delete = "cascade" },
      key = { "text" },
      value = { "text" }
    },
    task_tags = {
      id = true,
      task_id = { type = "text", reference = "tasks.id", on_delete = "cascade" },
      tag_name = { "text" }
    }
  })
end

function M.upsert_task(task, file_path, line_number)
  local now = os.time()
  
  -- Check if task exists
  local existing = M.db.tasks:where({ id = task.id })
  
  if existing and existing.id then
    -- Update
    M.db.tasks:update({
      where = { id = task.id },
      set = {
        description = task.description,
        status = task.status,
        project = task.project,
        priority = task.priority,
        due_date = task.due_date,
        file_path = file_path,
        line_number = line_number,
        updated_at = now
      }
    })
  else
    -- Insert
    M.db.tasks:insert({
      id = task.id,
      description = task.description,
      status = task.status,
      project = task.project,
      priority = task.priority,
      due_date = task.due_date,
      file_path = file_path,
      line_number = line_number,
      created_at = now,
      updated_at = now
    })
  end

  -- Delete existing metadata and tags for this task
  M.db.task_metadata:remove({ task_id = task.id })
  M.db.task_tags:remove({ task_id = task.id })

  -- Insert new metadata
  if task.metadata then
    for k, v in pairs(task.metadata) do
      M.db.task_metadata:insert({
        task_id = task.id,
        key = k,
        value = v
      })
    end
  end

  -- Insert new tags
  if task.tags then
    for _, tag in ipairs(task.tags) do
      M.db.task_tags:insert({
        task_id = task.id,
        tag_name = tag
      })
    end
  end
end

-- Fetch tasks based on filters
function M.get_tasks(opts)
  opts = opts or {}
  
  -- First, get all tasks that match the simple criteria (status, project)
  local where = {}
  
  if opts.status then
    if type(opts.status) == "string" then
      opts.status = { opts.status }
    end
    where.status = opts.status
  end
  
  if opts.project then
    where.project = opts.project
  end
  
  local query = {}
  if next(where) ~= nil then
    query.where = where
  end
  
  local tasks = M.db.tasks:get(query)
  if not tasks then return {} end
  
  -- Normalize tags filter to always be a table
  if opts.tags and type(opts.tags) == "string" then
    opts.tags = { opts.tags }
  end
  
  -- Now filter by tags in Lua to keep things simple and robust with sqlite.lua's limitations
  local filtered_tasks = {}
  
  for _, task in ipairs(tasks) do
    -- Get tags
    task.tags = {}
    local tags_res = M.db.task_tags:get({ where = { task_id = task.id } })
    if tags_res then
      for _, tag_row in ipairs(tags_res) do
        table.insert(task.tags, tag_row.tag_name)
      end
    end
    
    -- Check if it has all required tags
    local has_all_tags = true
    if opts.tags and #opts.tags > 0 then
      for _, required_tag in ipairs(opts.tags) do
        local found = false
        for _, tag in ipairs(task.tags) do
          if tag == required_tag then
            found = true
            break
          end
        end
        if not found then
          has_all_tags = false
          break
        end
      end
    end
    
    if has_all_tags then
      -- Get metadata
      task.metadata = {}
      local metadata_res = M.db.task_metadata:get({ where = { task_id = task.id } })
      if metadata_res then
        for _, meta_row in ipairs(metadata_res) do
          task.metadata[meta_row.key] = meta_row.value
        end
      end
      
      table.insert(filtered_tasks, task)
    end
  end
  
  -- Calculate scores
  local now = os.time()
  local today = os.date("!%Y-%m-%d", now)
  
  for _, task in ipairs(filtered_tasks) do
    local score = 0
    
    -- Base score: age (older tasks get higher score, 1 point per day old)
    if task.created_at then
      local age_days = math.floor((now - task.created_at) / (24 * 60 * 60))
      score = score + math.max(0, age_days)
    end
    
    -- Priority score
    if task.priority == "high" then
      score = score + 50
    elseif task.priority == "medium" then
      score = score + 20
    end
    
    -- Tag score
    for _, tag in ipairs(task.tags) do
      if tag == "urgent" then
        score = score + 100
      end
    end
    
    -- Due Date Score (The most critical factor)
    if task.due_date then
      if task.due_date < today then
        -- Overdue: Massive score, increases by 10 points for every day overdue
        -- Approximate date difference since Lua os.time parsing is tricky without full dates
        -- We'll do a simple string comparison hack or parse it
        local y, m, d = task.due_date:match("(%d%d%d%d)%-(%d%d)%-(%d%d)")
        if y and m and d then
          local due_time = os.time({year=y, month=m, day=d})
          local days_overdue = math.floor((now - due_time) / (24 * 60 * 60))
          score = score + 200 + (days_overdue * 10)
        else
          score = score + 200
        end
      elseif task.due_date == today then
        -- Due today
        score = score + 150
      else
        -- Due in the future:
        local y, m, d = task.due_date:match("(%d%d%d%d)%-(%d%d)%-(%d%d)")
        if y and m and d then
          local due_time = os.time({year=y, month=m, day=d})
          local days_until = math.floor((due_time - now) / (24 * 60 * 60))
          
          -- Max 100 points for being due tomorrow, decaying to 0 over 14 days
          if days_until <= 14 then
            local future_score = 100 - (days_until * 7)
            score = score + math.max(0, future_score)
          end
        end
      end
    end
    
    task.score = score
  end
  
  -- Sort by score descending
  table.sort(filtered_tasks, function(a, b)
    if a.score ~= b.score then
      return a.score > b.score
    end
    -- Fallback to description alphabetical if scores tie
    return a.description < b.description
  end)
  
  return filtered_tasks
end

-- Close db connection
function M.close()
  if M.db then
    M.db:close()
  end
end

return M
