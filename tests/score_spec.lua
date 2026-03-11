local db = require("task_manager.db")
local tm = require("task_manager")
local sync = require("task_manager.sync")

describe("TaskManager Scoring", function()
  local test_db_path = "/tmp/task_manager_score.db"

  before_each(function()
    os.remove(test_db_path)
    tm.setup({
      db_path = test_db_path,
      directories = { "/tmp/task_tests" }
    })
    
    local file_path = "/tmp/task_tests/scoring.md"
    local bufnr = vim.api.nvim_create_buf(false, true)
    vim.api.nvim_buf_set_name(bufnr, file_path)
    
    -- Setup relative dates
    local today = os.date("!%Y-%m-%d")
    local yesterday = os.date("!%Y-%m-%d", os.time() - 86400)
    local next_week = os.date("!%Y-%m-%d", os.time() + (86400 * 7))
    
    local lines = {
      "- [ ] Boring task",
      "- [ ] High priority task +high",
      "- [ ] Urgent task #urgent",
      string.format("- [ ] Overdue task due:%s", yesterday),
      string.format("- [ ] Due today task due:%s", today),
      string.format("- [ ] Next week task due:%s", next_week),
      string.format("- [ ] Future start task start:%s", next_week),
    }
    vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, lines)
    sync.sync_buffer(bufnr)
  end)

  after_each(function()
    db.close()
    os.remove(test_db_path)
  end)

  it("should sort tasks by score correctly", function()
    local tasks = db.get_tasks()
    
    -- Let's check the ordering:
    -- 1. Overdue should be highest (> 200)
    -- 2. Due today (150)
    -- 3. Urgent (100)
    -- 4. High priority (50)
    -- 5. Next week (~51) -- Wait, next week is 7 days. Score = 100 - (7*7) = 51.
    -- 6. Boring task (0)
    -- 7. Future start task (< -900)
    
    assert.is_true(tasks[1].description:match("Overdue") ~= nil)
    assert.is_true(tasks[2].description:match("today") ~= nil)
    assert.is_true(tasks[3].description:match("Urgent") ~= nil)
    assert.is_true(tasks[4].description:match("Next week") ~= nil)
    assert.is_true(tasks[5].description:match("High priority") ~= nil)
    assert.is_true(tasks[6].description:match("Boring") ~= nil)
    assert.is_true(tasks[7].description:match("Future start") ~= nil)
    
    assert.is_true(tasks[1].score > 200)
    assert.are.same(150, tasks[2].score)
    assert.are.same(100, tasks[3].score)
    assert.is_true(tasks[7].score <= -900)
  end)
end)
