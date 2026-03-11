local core = require("task_manager.core")

describe("TaskManager TaskAdd UI", function()
  it("should format string properly based on regex", function()
    local str1 = "- [ ] Finish feature"
    local prefix, status_char, desc = str1:match("^(%s*[%-*]%s+)%[([ x/%-])%]%s+(.*)$")
    assert.are.same("Finish feature", desc)

    local str2 = "- [ ] "
    local prefix2, status_char2, desc2 = str2:match("^(%s*[%-*]%s+)%[([ x/%-])%]%s+(.*)$")
    assert.are.same("", desc2)

    local str3 = "Write docs"
    local prefix3, status_char3, desc3 = str3:match("^(%s*[%-*]%s+)%[([ x/%-])%]%s+(.*)$")
    assert.is_nil(desc3)
    
    local str4 = "[ ] Just brackets"
    local alt_status, alt_desc = str4:match("^%s*%[([ x/%-])%]%s+(.*)$")
    assert.are.same("Just brackets", alt_desc)
    assert.are.same(" ", alt_status)
    
    local str5 = "- [ ] Test task which is urgent in work @work #urgent  "
    local p5, s5, d5 = str5:match("^(%s*[%-*]%s+)%[([ x/%-])%]%s+(.*)$")
    assert.are.same("Test task which is urgent in work @work #urgent  ", d5)
    assert.are.same("Test task which is urgent in work @work #urgent", d5:gsub("%s+$", ""))
  end)
end)
