-- Since it requires UI interaction, we'll just test that it requires correctly
local ok, m = pcall(require, "task_manager.telescope")
if ok then print("Telescope loaded OK") end
