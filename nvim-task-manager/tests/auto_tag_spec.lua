local db = require("task_manager.db")
local sync = require("task_manager.sync")
local tm = require("task_manager")
local parser = require("task_manager.parser")

describe("TaskManager Auto Tagging", function()
  local test_db_path = "/tmp/task_manager_auto_tag.db"

  before_each(function()
    os.remove(test_db_path)
    tm.setup({
      db_path = test_db_path,
      directories = { "/tmp/task_tests" },
      auto_tags = {
        ["/daily/"] = { "daily" },
        ["/work/.*%.md$"] = { "work", "office" }
      }
    })
  end)

  after_each(function()
    db.close()
    os.remove(test_db_path)
  end)

  it("should auto-tag tasks matching directory pattern", function()
    local file_path = "/tmp/task_tests/daily/2024-01-01.md"
    local bufnr = vim.api.nvim_create_buf(false, true)
    vim.api.nvim_buf_set_name(bufnr, file_path)
    
    local lines = {
      "- [ ] Clean up inbox id:t:123"
    }
    
    vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, lines)
    sync.sync_buffer(bufnr)
    
    -- Verify DB content has the auto-tag
    local tasks = db.get_tasks({ tags = { "daily" } })
    assert.are.same(1, #tasks)
    assert.are.same("Clean up inbox", tasks[1].description)
  end)

  it("should support multiple auto-tags per pattern", function()
    local file_path = "/tmp/task_tests/work/project_x.md"
    local bufnr = vim.api.nvim_create_buf(false, true)
    vim.api.nvim_buf_set_name(bufnr, file_path)
    
    local lines = {
      "- [ ] Write report id:t:456"
    }
    
    vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, lines)
    sync.sync_buffer(bufnr)
    
    local work_tasks = db.get_tasks({ tags = { "work" } })
    assert.are.same(1, #work_tasks)
    
    local office_tasks = db.get_tasks({ tags = { "office" } })
    assert.are.same(1, #office_tasks)
    
    local both_tasks = db.get_tasks({ tags = { "work", "office" } })
    assert.are.same(1, #both_tasks)
  end)
  
  it("should not tag files that do not match", function()
    local file_path = "/tmp/task_tests/personal.md"
    local bufnr = vim.api.nvim_create_buf(false, true)
    vim.api.nvim_buf_set_name(bufnr, file_path)
    
    local lines = {
      "- [ ] Groceries id:t:789"
    }
    
    vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, lines)
    sync.sync_buffer(bufnr)
    
    local all_tasks = db.get_tasks()
    assert.are.same(1, #all_tasks)
    assert.are.same(0, #all_tasks[1].tags) -- Should have no tags
  end)
end)
