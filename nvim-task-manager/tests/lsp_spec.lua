local server = require("task_manager.lsp.server")
local handlers = require("task_manager.lsp.handlers")

describe("TaskManager LSP Server", function()
  it("should format valid capabilities response", function()
    local res = server.handle_request("initialize", {}, 1)
    assert.are.same(1, res.capabilities.textDocumentSync)
    assert.is_not_nil(res.capabilities.completionProvider)
  end)
end)

local db = require("task_manager.db")
local tm = require("task_manager")
local sync = require("task_manager.sync")

describe("TaskManager LSP Handlers", function()
  local test_db_path = "/tmp/task_manager_lsp_test.db"

  before_each(function()
    os.remove(test_db_path)
    tm.setup({
      db_path = test_db_path,
      directories = { "/tmp/task_tests" }
    })
    
    -- Populate DB with some test data
    local file_path = "/tmp/task_tests/lsp_data.md"
    local bufnr = vim.fn.bufnr(file_path)
    if bufnr == -1 then
      bufnr = vim.api.nvim_create_buf(false, true)
      vim.api.nvim_buf_set_name(bufnr, file_path)
    end
    
    local lines = {
      "- [ ] Task 1 @work #frontend",
      "- [ ] Task 2 @home #backend",
      "- [ ] Task 3 @work #urgent"
    }
    vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, lines)
    sync.sync_buffer(bufnr)
    
    -- Tell LSP handler to use this test DB
    handlers.update_config({ db_path = test_db_path })
  end)

  after_each(function()
    db.close()
    os.remove(test_db_path)
  end)

  it("should provide project completions", function()
    local uri = "file:///tmp/task_tests/test.md"
    local documents = {
      [uri] = "- [ ] New task @"
    }
    
    local params = {
      textDocument = { uri = uri },
      position = { line = 0, character = 16 } -- After '@'
    }
    
    local completions = handlers.completion(params, documents)
    
    assert.is_not_nil(completions)
    assert.are.same(2, #completions) -- 'work' and 'home'
    
    local found_work = false
    local found_home = false
    for _, item in ipairs(completions) do
      if item.label == "work" then found_work = true end
      if item.label == "home" then found_home = true end
      assert.are.same(21, item.kind) -- Constant
    end
    
    assert.is_true(found_work)
    assert.is_true(found_home)
  end)

  it("should provide tag completions", function()
    local uri = "file:///tmp/task_tests/test.md"
    local documents = {
      [uri] = "- [ ] New task #"
    }
    
    local params = {
      textDocument = { uri = uri },
      position = { line = 0, character = 16 } -- After '#'
    }
    
    local completions = handlers.completion(params, documents)
    
    assert.is_not_nil(completions)
    assert.are.same(3, #completions) -- 'frontend', 'backend', 'urgent'
  end)
  
  it("should provide priority completions", function()
    local uri = "file:///tmp/task_tests/test.md"
    local documents = {
      [uri] = "- [ ] New task +"
    }
    
    local params = {
      textDocument = { uri = uri },
      position = { line = 0, character = 16 } -- After '+'
    }
    
    local completions = handlers.completion(params, documents)
    
    assert.is_not_nil(completions)
    assert.are.same(3, #completions)
    assert.are.same("high", completions[1].label)
  end)
end)

describe("TaskManager LSP Diagnostics", function()
  local rpc = require("task_manager.lsp.rpc")
  local original_send
  local sent_notifications = {}
  
  before_each(function()
    sent_notifications = {}
    original_send = rpc.send_notification
    
    -- Mock send_notification to capture payload
    rpc.send_notification = function(method, params)
      table.insert(sent_notifications, { method = method, params = params })
    end
  end)
  
  after_each(function()
    rpc.send_notification = original_send
  end)
  
  it("should generate hints for urgent tasks", function()
    local text = "- [ ] Fix critical bug #urgent\n- [ ] Another task +high"
    handlers.diagnostics("file:///test.md", text)
    
    assert.are.same(1, #sent_notifications)
    local notification = sent_notifications[1]
    
    assert.are.same("textDocument/publishDiagnostics", notification.method)
    local diags = notification.params.diagnostics
    
    assert.are.same(2, #diags)
    assert.are.same(4, diags[1].severity) -- Hint
    assert.are.same("Urgent Task", diags[1].message)
    assert.are.same(0, diags[1].range.start.line)
    
    assert.are.same(4, diags[2].severity)
    assert.are.same(1, diags[2].range.start.line)
  end)
  
  it("should generate warnings for overdue tasks", function()
    -- Create dates relative to today
    local today = os.date("!%Y-%m-%d")
    local yesterday_time = os.time() - (24 * 60 * 60)
    local yesterday = os.date("!%Y-%m-%d", yesterday_time)
    
    local text = string.format("- [ ] Overdue due:%s\n- [ ] Due today due:%s", yesterday, today)
    handlers.diagnostics("file:///test.md", text)
    
    assert.are.same(1, #sent_notifications)
    local diags = sent_notifications[1].params.diagnostics
    
    assert.are.same(2, #diags)
    
    -- Yesterday task
    assert.are.same(2, diags[1].severity) -- Warning
    assert.is_true(string.match(diags[1].message, "OVERDUE") ~= nil)
    
    -- Today task
    assert.are.same(3, diags[2].severity) -- Info
    assert.is_true(string.match(diags[2].message, "TODAY") ~= nil)
  end)
  
  it("should ignore completed or cancelled tasks", function()
    local text = "- [x] Done task #urgent\n- [-] Cancelled +high"
    handlers.diagnostics("file:///test.md", text)
    
    assert.are.same(1, #sent_notifications)
    local diags = sent_notifications[1].params.diagnostics
    assert.are.same(0, #diags) -- Should have no diagnostics
  end)
end)
