local parser = require("task_manager.parser")
local task = parser.parse_line("- [ ] Finish auth @work #code due:tomorrow b:40")
local formatted = parser.format_line("- ", "todo", task)
print(formatted)
