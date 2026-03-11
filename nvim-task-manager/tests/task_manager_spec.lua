local parser = require("task_manager.parser")
local db = require("task_manager.db")
local sync = require("task_manager.sync")
local tm = require("task_manager")

describe("TaskManager Parser", function()
  it("should parse a basic task", function()
    local task = parser.parse_line("- [ ] Buy milk")
    assert.are.same(task.status, "todo")
    assert.are.same(task.description, "Buy milk")
    assert.is_nil(task.project)
    assert.is_nil(task.priority)
    assert.are.same(#task.tags, 0)
    assert.is_nil(task.id)
  end)

  it("should parse metadata, project, and tags", function()
    local task = parser.parse_line("- [/] Write documentation @work #docs b:12345 cl:6789 due:2024-08-20 +high id:t:abc12")
    assert.are.same(task.status, "in_progress")
    assert.are.same(task.description, "Write documentation")
    assert.are.same(task.project, "work")
    assert.are.same(task.tags[1], "docs")
    assert.are.same(task.priority, "high")
    assert.are.same(task.due_date, "2024-08-20")
    assert.are.same(task.metadata["b"], "12345")
    assert.are.same(task.metadata["cl"], "6789")
    assert.are.same(task.id, "t:abc12")
  end)

  it("should parse tag: format", function()
    local task = parser.parse_line("- [x] Done task tag:cool")
    assert.are.same(task.status, "done")
    assert.are.same(task.tags[1], "cool")
  end)

  it("should generate random IDs", function()
    local id1 = parser.generate_id()
    local id2 = parser.generate_id()
    assert.is_not.same(id1, id2)
    assert.is_true(string.match(id1, "^t:%w+$") ~= nil)
  end)
end)

describe("TaskManager Database Integration", function()
  local test_db_path = "/tmp/task_manager_test.db"

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

  it("should sync buffer and generate ID", function()
    local bufnr = vim.api.nvim_create_buf(false, true)
    vim.api.nvim_buf_set_name(bufnr, "/tmp/task_tests/todo.md")
    
    local lines = {
      "# My Tasks",
      "- [ ] Implement feature @dev #test b:42"
    }
    
    vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, lines)
    
    -- Sync buffer
    sync.sync_buffer(bufnr)
    
    -- Check buffer lines updated with ID
    local new_lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
    assert.are.same(new_lines[1], "# My Tasks")
    assert.is_true(string.match(new_lines[2], "^%- %[ %] Implement feature %| @dev #test b:42 id:t:%w+$") ~= nil)
    
    -- Check database
    local tasks = db.db.tasks:get()
    assert.are.same(#tasks, 1)
    
    local task = tasks[1]
    assert.are.same(task.description, "Implement feature")
    assert.are.same(task.project, "dev")
    assert.are.same(task.status, "todo")
    
    local metadata = db.db.task_metadata:get({ where = { task_id = task.id } })
    assert.are.same(1, #metadata)
    assert.are.same("b", metadata[1].key)
    assert.are.same("42", metadata[1].value)
    
    local tags = db.db.task_tags:get({ where = { task_id = task.id } })
    assert.are.same(1, #tags)
    assert.are.same("test", tags[1].tag_name)
  end)

  it("should update existing task in DB when modified", function()
    local bufnr = vim.api.nvim_create_buf(false, true)
    vim.api.nvim_buf_set_name(bufnr, "/tmp/task_tests/todo2.md")
    
    local lines = {
      "- [ ] First step id:t:123"
    }
    vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, lines)
    
    -- Sync
    sync.sync_buffer(bufnr)
    
    -- Verify inserted
    local tasks = db.db.tasks:get()
    assert.are.same(#tasks, 1)
    assert.are.same(tasks[1].status, "todo")
    
    -- Modify line to Done
    lines = {
      "- [x] First step id:t:123"
    }
    vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, lines)
    
    -- Sync again
    sync.sync_buffer(bufnr)
    
    -- Verify updated
    tasks = db.db.tasks:get()
    assert.are.same(1, #tasks)
    assert.are.same("done", tasks[1].status)
  end)

  it("should delete tasks removed from buffer", function()
    local bufnr = vim.api.nvim_create_buf(false, true)
    vim.api.nvim_buf_set_name(bufnr, "/tmp/task_tests/todo_delete.md")
    
    local lines = {
      "- [ ] Keep me id:t:keep",
      "- [ ] Delete me id:t:del"
    }
    vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, lines)
    sync.sync_buffer(bufnr)
    
    local tasks = db.db.tasks:get({ where = { file_path = "/tmp/task_tests/todo_delete.md" } })
    assert.are.same(2, #tasks)
    
    -- Remove the second task
    lines = {
      "- [ ] Keep me id:t:keep"
    }
    vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, lines)
    sync.sync_buffer(bufnr)
    
    tasks = db.db.tasks:get({ where = { file_path = "/tmp/task_tests/todo_delete.md" } })
    assert.are.same(1, #tasks)
    assert.are.same("t:keep", tasks[1].id)
  end)

  it("should get tasks with filters", function()
    local bufnr = vim.api.nvim_create_buf(false, true)
    vim.api.nvim_buf_set_name(bufnr, "/tmp/task_tests/filters.md")
    
    local lines = {
      "- [ ] Task 1 @work #frontend id:t:t1",
      "- [/] Task 2 @work #backend #urgent id:t:t2",
      "- [x] Task 3 @home id:t:t3",
      "- [ ] Task 4 @home #frontend id:t:t4",
    }
    vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, lines)
    sync.sync_buffer(bufnr)
    
    -- Filter by project
    local work_tasks = db.get_tasks({ project = "work" })
    assert.are.same(2, #work_tasks)
    
    -- Filter by status
    local todo_tasks = db.get_tasks({ status = { "todo" } })
    assert.are.same(2, #todo_tasks)
    
    -- Filter by tag
    local frontend_tasks = db.get_tasks({ tags = { "frontend" } })
    assert.are.same(2, #frontend_tasks)
    
    -- Complex filter: work project, todo status, frontend tag
    local complex_tasks = db.get_tasks({ project = "work", status = { "todo" }, tags = { "frontend" } })
    assert.are.same(1, #complex_tasks)
    assert.are.same("t:t1", complex_tasks[1].id)
    assert.are.same("Task 1", complex_tasks[1].description)
    assert.are.same("work", complex_tasks[1].project)
    assert.are.same("frontend", complex_tasks[1].tags[1])
  end)
end)
