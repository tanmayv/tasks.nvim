-- This is a manual test script to verify the UI logic
local core = require("task_manager.core")
local tm = require("task_manager")
tm.setup({ directories = { "/tmp" } })
core.open_task_input()
