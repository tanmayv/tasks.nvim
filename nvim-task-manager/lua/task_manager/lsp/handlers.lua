local M = {}
local db = require("task_manager.db")
local rpc = require("task_manager.lsp.rpc")

local _db_initialized = false

-- Fallback initialization
local function ensure_db()
  if not _db_initialized then
    local path = os.getenv("HOME") .. "/.local/share/nvim/task_manager.db"
    db.init(path)
    _db_initialized = true
  end
end

function M.update_config(config)
  if config and config.db_path then
    db.init(config.db_path)
    _db_initialized = true
  end
end

-- Get completions based on the cursor position
function M.completion(params, documents)
  ensure_db()
  local uri = params.textDocument.uri
  local position = params.position
  local text = documents[uri]
  
  if not text then return {} end

  -- Parse text into lines
  local lines = {}
  for line in text:gmatch("[^\r\n]+") do
    table.insert(lines, line)
  end

  local line = lines[position.line + 1]
  if not line then return {} end
  
  -- Check if it's a task line
  if not line:match("^%s*[%-*]%s+%[[ x/%-]%]") then
    return {}
  end

  -- Get string up to cursor to figure out what we are completing
  local prefix = line:sub(1, position.character)
  local trigger_word = prefix:match("(%S+)$")
  
  if not trigger_word then return {} end

  local items = {}

  -- Complete projects
  if trigger_word:match("^@(.*)$") then
    -- Note: using lua tables filtering here instead of raw sql due to sqlite.lua setup
    local all_tasks = db.db.tasks:get({ select = { "project" } }) or {}
    local projects_seen = {}
    for _, t in ipairs(all_tasks) do
      if t.project and not projects_seen[t.project] then
        projects_seen[t.project] = true
        table.insert(items, {
          label = t.project,
          kind = 21, -- Constant
          detail = "Project"
        })
      end
    end
  
  -- Complete tags
  elseif trigger_word:match("^#(.*)$") or trigger_word:match("^tag:(.*)$") then
    local all_tags = db.db.task_tags:get({ select = { "tag_name" } }) or {}
    local tags_seen = {}
    for _, t in ipairs(all_tags) do
      if t.tag_name and not tags_seen[t.tag_name] then
        tags_seen[t.tag_name] = true
        table.insert(items, {
          label = t.tag_name,
          kind = 21, -- Constant
          detail = "Tag"
        })
      end
    end
  
  -- Complete priorities
  elseif trigger_word:match("^%+(.*)$") then
    items = {
      { label = "high", kind = 12, detail = "Priority" },
      { label = "medium", kind = 12, detail = "Priority" },
      { label = "low", kind = 12, detail = "Priority" }
    }
  end

  return items
end

-- Parse lines for diagnostics (overdue or urgent hints)
function M.diagnostics(uri, text)
  ensure_db()
  if not text then return end

  local parser = require("task_manager.parser")
  local diagnostics = {}
  
  local line_num = 0
  for line in text:gmatch("([^\r\n]*)\r?\n?") do
    if line ~= "" then
      local task = parser.parse_line(line)
      if task and task.status ~= "done" and task.status ~= "cancelled" then
        
        -- Check urgency
        if task.priority == "high" or vim.tbl_contains(task.tags, "urgent") then
          table.insert(diagnostics, {
            range = {
              start = { line = line_num, character = 0 },
              ["end"] = { line = line_num, character = #line }
            },
            severity = 4, -- Hint
            source = "task-manager",
            message = "Urgent Task"
          })
        end

        -- Check due date
        if task.due_date then
          local today = os.date("!%Y-%m-%d")
          if task.due_date < today then
            table.insert(diagnostics, {
              range = {
                start = { line = line_num, character = 0 },
                ["end"] = { line = line_num, character = #line }
              },
              severity = 2, -- Warning
              source = "task-manager",
              message = "Task is OVERDUE! (Due: " .. task.due_date .. ")"
            })
          elseif task.due_date == today then
            table.insert(diagnostics, {
              range = {
                start = { line = line_num, character = 0 },
                ["end"] = { line = line_num, character = #line }
              },
              severity = 3, -- Information
              source = "task-manager",
              message = "Task is due TODAY."
            })
          end
        end
      end
    end
    line_num = line_num + 1
  end

  -- Send diagnostics payload
  rpc.send_notification("textDocument/publishDiagnostics", {
    uri = uri,
    diagnostics = diagnostics
  })
end

return M
