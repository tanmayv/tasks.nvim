local rpc = require("task_manager.lsp.rpc")
local handlers = require("task_manager.lsp.handlers")

local M = {}

M.capabilities = {
  textDocumentSync = 1, -- Full sync for simplicity
  completionProvider = {
    triggerCharacters = { "@", "#", ":" },
    resolveProvider = false,
  },
  diagnosticProvider = {
    interFileDependencies = false,
    workspaceDiagnostics = false,
  }
}

-- Simple state manager for open files
local documents = {}

function M.handle_request(method, params, id)
  if method == "initialize" then
    return {
      capabilities = M.capabilities,
      serverInfo = {
        name = "task-manager-lsp",
        version = "0.1.0"
      }
    }
  elseif method == "shutdown" then
    return nil
  elseif method == "textDocument/completion" then
    return handlers.completion(params, documents)
  end
  return nil
end

function M.handle_notification(method, params)
  if method == "initialized" then
    -- Ready
  elseif method == "textDocument/didOpen" then
    documents[params.textDocument.uri] = params.textDocument.text
    handlers.diagnostics(params.textDocument.uri, params.textDocument.text)
  elseif method == "textDocument/didChange" then
    documents[params.textDocument.uri] = params.contentChanges[1].text
    handlers.diagnostics(params.textDocument.uri, params.contentChanges[1].text)
  elseif method == "textDocument/didClose" then
    documents[params.textDocument.uri] = nil
  elseif method == "workspace/didChangeConfiguration" then
    -- Handle config updates (e.g. db path)
    if params.settings and params.settings.task_manager then
      handlers.update_config(params.settings.task_manager)
    end
  end
end

function M.start()
  while true do
    local msg = rpc.read_message()
    if not msg then break end -- EOF or error
    
    if msg.id then
      -- It's a Request
      local result = M.handle_request(msg.method, msg.params, msg.id)
      -- Some requests like shutdown don't strictly need a result but we must respond
      if msg.method ~= "exit" then
        rpc.write_response(msg.id, result)
      end
    else
      -- It's a Notification
      if msg.method == "exit" then
        os.exit(0)
      else
        M.handle_notification(msg.method, msg.params)
      end
    end
  end
end

return M
