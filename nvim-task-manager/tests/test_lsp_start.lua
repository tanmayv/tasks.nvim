local script_path = debug.getinfo(1, "S").source:sub(2)
print("script_path:", script_path)
local plugin_root = script_path:match("(.*)/tests/.*")
print("plugin_root:", plugin_root)
package.path = plugin_root .. "/lua/?.lua;" .. plugin_root .. "/lua/?/init.lua;" .. package.path
print("package.path:", package.path)
local ok, server = pcall(require, "task_manager.lsp.server")
print("OK:", ok)
if not ok then
  print("Err:", server)
end
