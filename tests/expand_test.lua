local vim = vim
local tm = { config = { inbox_file = { "~/tasks/inbox.md" } } }
local configured_inbox = tm.config.inbox_file
if type(configured_inbox) == "table" then
  configured_inbox = configured_inbox[1]
end
print(vim.fn.expand(configured_inbox))
