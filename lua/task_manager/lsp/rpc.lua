local M = {}

function M.read_message()
  -- Read headers
  local content_length = 0
  while true do
    local line = io.read("*l")
    if not line then return nil end -- EOF
    if line == "" or line == "\r" then break end -- End of headers

    local cl_str = line:match("Content%-Length: (%d+)")
    if cl_str then
      content_length = tonumber(cl_str)
    end
  end

  if content_length == 0 then return nil end

  -- Read body
  local body = io.read(content_length)
  if not body then return nil end

  local ok, decoded = pcall(vim.json.decode, body)
  if not ok then return nil end
  return decoded
end

function M.write_response(id, result)
  local response = {
    jsonrpc = "2.0",
    id = id,
    result = result
  }
  
  local ok, encoded = pcall(vim.json.encode, response)
  if not ok then return end
  
  io.write(string.format("Content-Length: %d\r\n\r\n%s", #encoded, encoded))
  io.flush()
end

function M.send_notification(method, params)
  local notification = {
    jsonrpc = "2.0",
    method = method,
    params = params
  }
  
  local ok, encoded = pcall(vim.json.encode, notification)
  if not ok then return end
  
  io.write(string.format("Content-Length: %d\r\n\r\n%s", #encoded, encoded))
  io.flush()
end

return M
