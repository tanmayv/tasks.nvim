# nvim-task-manager

A Neovim task management plugin powered by SQLite and Markdown. 

## Features
- Store tasks in markdown format.
- Automatically synchronizes tasks with an SQLite database on save.
- Parses metadata like `@project`, `#tag`, `tag:tagname`, `b:<bugid>`, `cl:<cl>`.
- Auto-generates unique `id:<id>` for each task upon first save.
- Clean updates and deletions when tasks are modified or removed from files.
- Provides `:TaskIndex` for onboarding a directory of markdown files.
- Provides `:Tasks` command to view, filter, and jump to tasks using `telescope.nvim`.

## Installation

### Complete Lazy.nvim Configuration Example
You can copy this directly into your `lua/plugins/task_manager.lua` file:

```lua
return {
  "tanmayv/nvim-task-manager",
  dependencies = {
    "nvim-telescope/telescope.nvim",
    "nvim-lua/plenary.nvim",
  },
  keys = {
    { "<leader>tn", "<cmd>TaskAdd<CR>", desc = "Add Task to Inbox" },
    { "<leader>tt", "<cmd>Tasks<CR>", desc = "View Open Tasks" },
    { "<leader>ta", function() require("task_manager.telescope").tasks({ status = { "todo", "in_progress", "done", "cancelled" } }) end, desc = "View All Tasks" },
    { "<leader>tu", function() require("task_manager.telescope").tasks({ tags = { "urgent" } }) end, desc = "View Urgent Tasks" },
    { "<leader>tw", function() require("task_manager.telescope").tasks({ project = "work" }) end, desc = "View Work Tasks" },
    { "<leader>tx", "<cmd>TaskToggle<CR>", mode = { "n", "v" }, desc = "Toggle Task Status" },
  },
  config = function()
    require("task_manager").setup({
      cmd = "task", -- Ensure the task companion binary is in your PATH
      directories = { "~/tasks" },
    })
    
    -- Optional: Initialize LSP for autocomplete & diagnostics
    require("task_manager").setup_lsp()
  end
}
```

### Pre-requisites (Companion TUI & Backend)
This Neovim plugin requires the `task` CLI backend to function. You must compile or install the Go companion tool.
```bash
cd tui
go build -o task .
sudo mv task /usr/local/bin/
```
Or if using Nix:
```bash
nix build
# Link the ./result/bin/task into your environment
```

Once installed, use `~/.config/task-manager-tui/config.json` to configure the database path, inbox file, and auto-tags!
```json
{
  "db_path": "~/.local/share/nvim/task_manager.db",
  "inbox_file": "~/tasks/inbox.md",
  "auto_tags": {
    "/daily/": ["daily"],
    "/work/": ["work"]
  }
}
```

The Go backend will automatically attach tags to tasks based on regex matches of their markdown file's path during synchronization.

## Usage
Simply write tasks in your configured directories as markdown lists:
```markdown
- [ ] Review PR | @work #code cl:1234
- [/] Fix auth bug | @work b:456 due:1d +high
```

On save, the plugin will append a unique `id` to the tasks and sync them to your SQLite database.

**Natural Language Dates:**
You can type natural language into the `due:` or `start:` tags and the plugin will automatically parse and convert it to `YYYY-MM-DD` standard format upon saving!
Supported formats:
- `due:today`
- `start:tomorrow`
- `due:3d` (3 days)
- `start:2w` (2 weeks)
- `due:1m` (1 month)

*Note:* If you set a `start:` date that is in the future, the task will receive a massive penalty to its score, effectively hiding it at the bottom of your Telescope list until the start date arrives!

### LSP Support
By calling `require("task_manager").setup_lsp()` in your config, the plugin runs a local LSP server.
1. **Autocomplete**: Auto-completes existing tags, projects, and priorities as you type `@`, `#`, or `+`!
2. **Diagnostics**: It provides inline Neovim Diagnostics (hints/warnings) for tasks that have a `#urgent` tag, `+high` priority, or a `due:date` that has elapsed!

### Capturing Tasks
You can capture tasks instantly to your configured `inbox_file` from anywhere in Neovim by running:
```vim
:TaskAdd
```
Or you can automatically pre-fill a project namespace by passing it as an argument:
```vim
:TaskAdd personal
```

This opens a beautiful, floating scratch window. 
- The window is treated as a native Markdown buffer, which means **the LSP server automatically attaches to it**!
- As you type your new task, you can use `@`, `#`, or `+` to autocomplete your existing projects and tags directly from your SQLite database.
- **Batch mode:** You can hit `<Enter>` to write as many tasks as you want on new lines!
- Hit `<C-s>` (Control-s) when done. It will parse all your lines, auto-sync them with unique IDs, apply the default project if provided, and drop them into your inbox file instantly!
- Hit `<Esc>` to cancel and close the prompt. (If you have typed anything, it will prompt you with a confirmation dialog so you don't lose your work!)

### Viewing Tasks
Run `:Tasks` to open the telescope UI and view all your *open* tasks (status `[ ]` or `[/]`). You can fuzzy search across description, tags, and projects. 

**Telescope Actions:**
- `<CR>` (Enter): Jumps instantly to the file and line containing that task.
- `<Tab>`: Select multiple tasks.
- `<C-x>` (Control + X): Instantly **toggles the selected task(s) between "Done" and "Todo"**.
  - Works on a single task, or any multiple tasks selected via `<Tab>`.
  - When marking as done, it automatically injects `done:YYYY-MM-DD` onto the task line.
  - When reverting to open, it automatically removes the `done:` metadata.
  - The UI will immediately refresh after the toggle!
- `<C-v>` (Control + V): **Exports the selected task(s) into a new Edit Buffer!**
  - A new vertical split window opens populated with your selected tasks.
  - You can edit their titles, delete lines to delete them from your system, or type brand-new tasks into this window.
  - When you run `:w` to save the buffer, the plugin calculates a diff and seamlessly syncs all your edits, deletions, and additions back to their original source files across your computer!

**Advanced UI Usage:**
You can create custom keymaps by passing filters to the `require("task_manager.telescope").tasks(opts)` function directly in your `init.lua`:

```lua
-- Add a new task
vim.keymap.set("n", "<leader>tn", "<cmd>TaskAdd<CR>")

-- Add a new task pre-filled with the 'work' project
vim.keymap.set("n", "<leader>twN", "<cmd>TaskAdd work<CR>")

-- View ALL tasks, including completed ones
vim.keymap.set("n", "<leader>ta", function()
  require("task_manager.telescope").tasks({ status = { "todo", "in_progress", "done", "cancelled" } })
end)

-- View ONLY @work tasks
vim.keymap.set("n", "<leader>tw", function()
  require("task_manager.telescope").tasks({ project = "work" })
end)

-- View ONLY #urgent tasks
vim.keymap.set("n", "<leader>tu", function()
  require("task_manager.telescope").tasks({ tags = { "urgent" } })
end)

-- View tasks that have EITHER the #urgent OR #frontend tag (OR logic)
vim.keymap.set("n", "<leader>to", function()
  require("task_manager.telescope").tasks({ 
    tags = { "urgent", "frontend" },
    match_any_tag = true 
  })
end)

-- Toggle task on the current line (or multiple in visual selection)
vim.keymap.set({ "n", "v" }, "<leader>tx", "<cmd>TaskToggle<CR>")
```
