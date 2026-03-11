local core = require("task_manager.core")
local parser = require("task_manager.parser")
local tm = require("task_manager")
local db = require("task_manager.db")
local sync = require("task_manager.sync")

describe("TaskManager Editor Sync", function()
  local test_db_path = "/tmp/task_manager_edit_sync.db"
  local target_file = "/tmp/task_tests/edit_target.md"
  
  before_each(function()
    os.remove(test_db_path)
    os.remove(target_file)
    tm.setup({
      db_path = test_db_path,
      directories = { "/tmp/task_tests" },
      inbox_file = "/tmp/task_tests/inbox.md"
    })
    
    local target_buf = vim.api.nvim_create_buf(false, true)
    vim.api.nvim_buf_set_name(target_buf, target_file)
    
    -- Need exactly formatted lines with IDs
    local lines = {
      "- [ ] Keep me | id:t:111",
      "- [ ] Edit me | id:t:222",
      "- [ ] Delete me | id:t:333"
    }
    vim.api.nvim_buf_set_lines(target_buf, 0, -1, false, lines)
    -- Write to disk so it can be reliably loaded
    vim.api.nvim_buf_call(target_buf, function() vim.cmd('silent! w') end)
  end)
  
  after_each(function()
    db.close()
    os.remove(test_db_path)
  end)

  it("should process updates, deletions and additions from scratch buffer", function()
    -- Create the mock scratch buffer
    local edit_buf = vim.api.nvim_create_buf(false, true)
    vim.b[edit_buf].is_task_manager_editor = true
    
    -- Setup origin tracking correctly
    vim.b[edit_buf].task_origins = {
      ["t:111"] = { file_path = target_file, original_line = "- [ ] Keep me | id:t:111", id = "t:111" },
      ["t:222"] = { file_path = target_file, original_line = "- [ ] Edit me | id:t:222", id = "t:222" },
      ["t:333"] = { file_path = target_file, original_line = "- [ ] Delete me | id:t:333", id = "t:333" }
    }
    
    -- Simulate the user editing the buffer
    local new_lines = {
      "# Edit Mode",
      "- [ ] Keep me | id:t:111",            -- Unchanged
      "- [x] Edited task! | id:t:222",       -- Updated
      -- t:333 is deleted
      "- [ ] Brand new task @home"           -- Added
    }
    vim.api.nvim_buf_set_lines(edit_buf, 0, -1, false, new_lines)
    
    -- Execute the sync function
    core.apply_editor_changes(edit_buf)
    
    -- Verify target_file content
    local verify_buf = vim.fn.bufadd(target_file)
    local target_content = vim.api.nvim_buf_get_lines(verify_buf, 0, -1, false)
    
    assert.are.same(2, #target_content)
    assert.are.same("- [ ] Keep me | id:t:111", target_content[1])
    assert.are.same("- [x] Edited task! | id:t:222", target_content[2])
    
    -- Verify the new task was sent to inbox
    local inbox_buf = vim.fn.bufadd("/tmp/task_tests/inbox.md")
    local inbox_content = vim.api.nvim_buf_get_lines(inbox_buf, 0, -1, false)
    
    local found_new = false
    for _, line in ipairs(inbox_content) do
      -- The description extractor from task parser handles the checkboxes
      if line:match("Brand new task") and line:match("@home") then
        found_new = true
      end
    end
    assert.is_true(found_new)
  end)
end)
