local s = "- [ ] Random task"
local prefix, status_char, desc = s:match("^(%s*[%-*]%s+)%[([ x/%-])%]%s+(.*)$")

print("Prefix:", prefix)
print("Status:", status_char)
print("Desc:", desc)
