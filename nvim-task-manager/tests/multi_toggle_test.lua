local core = require("task_manager.core")
local parser = require("task_manager.parser")
local tm = require("task_manager")
tm.setup({ directories = { "/tmp/task_tests" }, db_path = "/tmp/test_multi.db" })

local bufnr = vim.api.nvim_create_buf(false, true)
vim.api.nvim_buf_set_name(bufnr, "/tmp/task_tests/multi_toggle.md")
vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, {
  "- [ ] Task 1",
  "- [ ] Task 2",
  "- [x] Task 3"
})

core.toggle_done("/tmp/task_tests/multi_toggle.md", 1)
core.toggle_done("/tmp/task_tests/multi_toggle.md", 2)
core.toggle_done("/tmp/task_tests/multi_toggle.md", 3)

local lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
for i, line in ipairs(lines) do
  print(i .. ": " .. line)
end
