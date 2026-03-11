local s = "- [ ] Test task which is urgent in work @work #urgent  "
local function clean(line)
  local prefix, status_char, desc = line:match("^(%s*[%-*]%s+)%[([ x/%-])%]%s+(.*)$")
  if desc then
    return desc:gsub("%s+$", "")
  end
  return line:gsub("%s+$", "")
end

print("Cleaned: '" .. clean(s) .. "'")
