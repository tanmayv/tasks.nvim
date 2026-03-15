local M = {}
local rpc = require("task_manager.lsp.rpc")
local parser = require("task_manager.parser")

local _cmd = "task"

function M.update_config(config)
  if config and config.cmd then
    _cmd = config.cmd
  end
end

-- Get completions based on the cursor position
function M.completion(params, documents)
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

  -- Fetch meta if needed
  local meta = nil
  if trigger_word:match("^@(.*)$") or trigger_word:match("^#(.*)$") or trigger_word:match("^tag:(.*)$") then
    local handle = io.popen(_cmd .. " meta --json 2>/dev/null")
    if handle then
      local result = handle:read("*a")
      handle:close()
      local ok, decoded = pcall(vim.fn.json_decode, result)
      if ok and decoded then
        meta = decoded
      end
    end
  end

  -- Complete projects
  if trigger_word:match("^@(.*)$") then
    if meta and meta.projects then
      for _, p in ipairs(meta.projects) do
        table.insert(items, {
          label = p,
          kind = 21, -- Constant
          detail = "Project"
        })
      end
    end
  
  -- Complete tags
  elseif trigger_word:match("^#(.*)$") or trigger_word:match("^tag:(.*)$") then
    if meta and meta.tags then
      for _, t in ipairs(meta.tags) do
        table.insert(items, {
          label = t,
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
  if not text then return end

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
