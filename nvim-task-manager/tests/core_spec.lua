local core = require("task_manager.core")
local parser = require("task_manager.parser")
local sync = require("task_manager.sync")
local db = require("task_manager.db")
local tm = require("task_manager")

describe("TaskManager Core Actions", function()
  local test_db_path = "/tmp/task_manager_core_test.db"

  before_each(function()
    os.remove(test_db_path)
    tm.setup({
      db_path = test_db_path,
      directories = { "/tmp/task_tests" }
    })
  end)

  after_each(function()
    db.close()
    os.remove(test_db_path)
  end)

  it("should toggle task done status and metadata", function()
    local file_path = "/tmp/task_tests/core_toggle.md"
    local bufnr = vim.api.nvim_create_buf(false, true)
    vim.api.nvim_buf_set_name(bufnr, file_path)
    
    local lines = {
      "- [ ] Implement feature @dev #test id:t:123"
    }
    
    vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, lines)
    
    -- Sync initially to populate DB
    sync.sync_buffer(bufnr)
    
    -- Toggle to DONE
    local success = core.toggle_done(file_path, 1)
    assert.is_true(success)
    
    -- Verify buffer content
    local new_lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
    local task = parser.parse_line(new_lines[1])
    assert.are.same("done", task.status)
    assert.is_not_nil(task.metadata["done"])
    assert.are.same(os.date("!%Y-%m-%d"), task.metadata["done"])
    
    -- Sync again to ensure DB is updated
    sync.sync_buffer(bufnr)
    
    -- Verify DB content
    local db_tasks = db.get_tasks()
    assert.are.same(1, #db_tasks)
    assert.are.same("done", db_tasks[1].status)
    assert.are.same(os.date("!%Y-%m-%d"), db_tasks[1].metadata["done"])
    
    -- Toggle back to TODO
    success = core.toggle_done(file_path, 1)
    assert.is_true(success)
    
    -- Verify buffer content again
    new_lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
    task = parser.parse_line(new_lines[1])
    assert.are.same("todo", task.status)
    assert.is_nil(task.metadata["done"])
    
    -- Sync again
    sync.sync_buffer(bufnr)
    
    -- Verify DB content again
    db_tasks = db.get_tasks()
    assert.are.same(1, #db_tasks)
    assert.are.same("todo", db_tasks[1].status)
    assert.is_nil(db_tasks[1].metadata["done"])
  end)

  it("should add task to inbox and sync", function()
    local test_inbox = "/tmp/task_tests/test_inbox.md"
    os.remove(test_inbox) -- Ensure clean state
    
    tm.setup({
      db_path = test_db_path,
      directories = { "/tmp/task_tests" },
      inbox_file = test_inbox
    })
    
    local success = core.add_task("Buy groceries @home #errands")
    assert.is_true(success)
    
    -- Verify buffer content
    local bufnr = vim.fn.bufadd(test_inbox)
    local lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
    
    -- Note: buffer append adds to end of file
    -- A newly created file has 1 empty line, so task should be at index 2
    local task_line = lines[2]
    assert.is_not_nil(task_line)
    
    local task = parser.parse_line(task_line)
    assert.are.same("todo", task.status)
    assert.are.same("Buy groceries", task.description)
    assert.are.same("home", task.project)
    assert.are.same("errands", task.tags[1])
    assert.is_not_nil(task.id) -- Sync should have generated an ID
    
    -- Verify DB content
    local db_tasks = db.get_tasks()
    assert.are.same(1, #db_tasks)
    assert.are.same(task.id, db_tasks[1].id)
    assert.are.same("todo", db_tasks[1].status)
    assert.are.same("home", db_tasks[1].project)
  end)
end)
