local M = {}

local date_utils = require("task_manager.date")

-- Status map
local status_map = {
  [" "] = "todo",
  ["x"] = "done",
  ["/"] = "in_progress",
  ["-"] = "cancelled",
}

-- Generate a unique ID (simple random string)
function M.generate_id()
  local chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
  local id = "t:"
  for _ = 1, 6 do
    local r = math.random(1, #chars)
    id = id .. chars:sub(r, r)
  end
  return id
end

-- Parse a single line
-- Returns a task table if it's a task, or nil
function M.parse_line(line)
  -- Match the beginning of a task
  local prefix, status_char, rest = line:match("^(%s*[%-*]%s+)%[([ x/%-])%]%s+(.*)$")
  
  if not status_char then
    return nil
  end

  local task = {
    status = status_map[status_char] or "todo",
    tags = {},
    metadata = {},
    original_line = line,
    prefix_length = #prefix + 3, -- length of prefix + "[ ]"
  }

  local description_parts = {}
  
  -- We process words/tokens one by one
  for word in rest:gmatch("%S+") do
    if word == "|" then
      -- Skip standalone pipes added for formatting
    elseif word:match("^id:(.+)$") then
      task.id = word:match("^id:(.+)$")
    elseif word:match("^@(.+)$") then
      task.project = word:match("^@(.+)$")
    elseif word:match("^#(.+)$") then
      table.insert(task.tags, word:match("^#(.+)$"))
    elseif word:match("^tag:(.+)$") then
      table.insert(task.tags, word:match("^tag:(.+)$"))
    elseif word:match("^%+(.+)$") then
      task.priority = word:match("^%+(.+)$")
    elseif word:match("^(%w+):(.+)$") then
      local key, val = word:match("^(%w+):(.+)$")
      if key == "due" then
        task.due_date = date_utils.parse_relative(val)
      else
        task.metadata[key] = val
      end
    else
      table.insert(description_parts, word)
    end
  end

  task.description = table.concat(description_parts, " ")
  
  return task
end

function M.format_description(task)
  local parts = { task.description }
  
  if task.project then
    table.insert(parts, "@" .. task.project)
  end
  for _, tag in ipairs(task.tags) do
    table.insert(parts, "#" .. tag)
  end
  if task.priority then
    table.insert(parts, "+" .. task.priority)
  end
  if task.due_date then
    table.insert(parts, "due:" .. task.due_date)
  end
  -- Sort metadata keys for deterministic output
  local keys = {}
  for k in pairs(task.metadata) do
    table.insert(keys, k)
  end
  table.sort(keys)
  for _, k in ipairs(keys) do
    table.insert(parts, k .. ":" .. task.metadata[k])
  end
  if task.id then
    table.insert(parts, "id:" .. task.id)
  end
  
  return parts
end

-- Reconstruct a line from a task table (for writing back the ID)
function M.format_line(prefix, status, task)
  -- Map back status to char
  local reverse_status = {
    todo = " ",
    done = "x",
    in_progress = "/",
    cancelled = "-",
  }
  
  local status_char = reverse_status[status] or " "
  
  local parts = M.format_description(task)
  local line = prefix .. "[" .. status_char .. "] " .. table.remove(parts, 1)
  
  if #parts > 0 then
    line = line .. " | " .. table.concat(parts, " ")
  end
  
  return line
end

return M
