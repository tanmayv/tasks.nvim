local db = require("task_manager.db")
local tm = require("task_manager")
local sync = require("task_manager.sync")

describe("TaskManager DB Get Tasks", function()
  local test_db_path = "/tmp/task_manager_db_get.db"

  before_each(function()
    os.remove(test_db_path)
    tm.setup({
      db_path = test_db_path,
      directories = { "/tmp/task_tests" }
    })
    
    local file_path = "/tmp/task_tests/get_tasks.md"
    local bufnr = vim.fn.bufnr(file_path)
    if bufnr == -1 then
      bufnr = vim.api.nvim_create_buf(false, true)
      vim.api.nvim_buf_set_name(bufnr, file_path)
    end
    
    local lines = {
      "- [ ] Task 1 @work #urgent",
      "- [ ] Task 2 @home #urgent",
      "- [x] Task 3 @work",
    }
    vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, lines)
    sync.sync_buffer(bufnr)
  end)

  after_each(function()
    db.close()
    os.remove(test_db_path)
  end)

  it("should handle string input for tags", function()
    local tasks = db.get_tasks({ tags = "urgent" })
    assert.are.same(2, #tasks)
  end)

  it("should handle string input for status", function()
    local tasks = db.get_tasks({ status = "done" })
    assert.are.same(1, #tasks)
  end)
end)
