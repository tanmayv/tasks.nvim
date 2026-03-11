-- Simulate being loaded from init.lua
local script_path = debug.getinfo(1, "S").source:sub(2)
print("script_path:", script_path)
local plugin_root = script_path:match("(.*)/tests/test_find_root%.lua")
print("plugin_root:", plugin_root)
