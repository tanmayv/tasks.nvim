#!/usr/bin/env -S nvim -l

-- A simple executable script to start the Lua LSP server
-- We need to ensure the lua path can find our plugin files

local script_path = debug.getinfo(1, "S").source:sub(2)
local plugin_root = script_path:match("(.*)/bin/.*")
if not plugin_root then
  plugin_root = script_path:match("(.*)/.*") -- fallback if not in bin
end
if not plugin_root or plugin_root == "" then
  plugin_root = "."
end

-- Add the plugin's lua directory to the package path so it can require task_manager files
package.path = plugin_root .. "/lua/?.lua;" .. plugin_root .. "/lua/?/init.lua;" .. package.path
package.cpath = package.cpath .. ";" .. plugin_root .. "/lua/?.so"

-- Make sure we also add lazy.nvim paths if neovim is booting this up
local runtime_path = vim.o.runtimepath
-- Sometimes 'sqlite' is installed via lazy and we need its path:
for path in runtime_path:gmatch("([^,]+)") do
  package.path = package.path .. ";" .. path .. "/lua/?.lua;" .. path .. "/lua/?/init.lua"
end

-- Attempt to start the server
local ok, server = pcall(require, "task_manager.lsp.server")
if not ok then
  io.stderr:write("Failed to load LSP server: " .. tostring(server) .. "\n")
  os.exit(1)
end

server.start()
