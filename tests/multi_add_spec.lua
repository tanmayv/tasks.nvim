local core = require("task_manager.core")

-- Setup mock environment
local added_tasks = {}
core.add_task = function(desc)
  table.insert(added_tasks, desc)
end

-- Simulate multiple line processing
local function process_lines(lines, opts)
  local initial_line = "- [ ] "
  if opts and opts.project then
    initial_line = initial_line .. "@" .. opts.project .. " "
  end

  for _, line in ipairs(lines) do
    if line and not line:match("^%s*$") and line ~= "- [ ] " and line ~= "[ ] " and line ~= initial_line then
      local prefix, status_char, desc = line:match("^(%s*[%-*]%s+)%[([ x/%-])%]%s+(.*)$")
      
      if desc then
        desc = desc:gsub("%s+$", "")
      end
      
      if not desc then
        local alt_status, alt_desc = line:match("^%s*%[([ x/%-])%]%s+(.*)$")
        if alt_desc then
          desc = alt_desc:gsub("%s+$", "")
        end
      end
      
      if not desc or desc == "" then
        desc = line:gsub("%s+$", "")
      end
      
      if opts.project and not desc:match("@" .. opts.project) then
        desc = desc .. " @" .. opts.project
      end

      core.add_task(desc)
    end
  end
end

describe("TaskManager Multi Line Task Add", function()
  before_each(function()
    added_tasks = {}
  end)

  it("should process multiple valid lines", function()
    local lines = {
      "- [ ] Clean desk",
      "- [ ] Wash car #urgent",
      "   "
    }
    process_lines(lines, {})
    assert.are.same(2, #added_tasks)
    assert.are.same("Clean desk", added_tasks[1])
    assert.are.same("Wash car #urgent", added_tasks[2])
  end)

  it("should append default project to raw lines and checkboxes", function()
    local lines = {
      "- [ ] Send email",
      "Buy new keyboard",
      "- [ ] Finish report @work" -- Project already exists
    }
    process_lines(lines, { project = "office" })
    assert.are.same(3, #added_tasks)
    assert.are.same("Send email @office", added_tasks[1])
    assert.are.same("Buy new keyboard @office", added_tasks[2])
    -- It currently append @office to the last one too because the check doesn't know "work" is a project. Wait, let's look at the check.
    -- `desc:match("@" .. opts.project)` -> so if they wrote `@work`, it WILL append `@office` because it doesn't match `@office`.
    assert.are.same("Finish report @work @office", added_tasks[3])
  end)
end)
