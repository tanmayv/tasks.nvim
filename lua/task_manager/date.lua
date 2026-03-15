local M = {}

-- Parses natural language strings into YYYY-MM-DD
function M.parse_relative(date_str)
  if not date_str then return nil end
  
  -- If it's already YYYY-MM-DD, just return it
  if date_str:match("^%d%d%d%d%-%d%d%-%d%d$") then
    return date_str
  end

  local now = os.time()
  
  if date_str:lower() == "today" then
    return os.date("!%Y-%m-%d", now)
  elseif date_str:lower() == "tomorrow" then
    return os.date("!%Y-%m-%d", now + (24 * 60 * 60))
  end
  
  -- Handle numbers with 'd', 'w', 'm' suffix
  local num, unit = date_str:match("^(%d+)([dwm])$")
  if num and unit then
    num = tonumber(num)
    local offset = 0
    if unit == "d" then
      offset = num * 24 * 60 * 60
    elseif unit == "w" then
      offset = num * 7 * 24 * 60 * 60
    elseif unit == "m" then
      -- Approximation of a month as 30 days for simple date math
      offset = num * 30 * 24 * 60 * 60
    end
    return os.date("!%Y-%m-%d", now + offset)
  end
  
  -- Return original if unrecognized
  return date_str
end

return M
