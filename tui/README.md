# Task Manager TUI

This is the standalone Golang companion backend and Terminal UI (TUI) for `nvim-task-manager`.

It operates entirely statelessly, parsing your markdown lists, calculating task urgency scores, and interacting with the `task_manager.db` SQLite file locally on your disk.

## Installation

You can compile the binary manually:
```bash
go build -o task .
sudo mv task /usr/local/bin/
```

Or using Nix (with the included flake):
```bash
nix build
# The binary is compiled into ./result/bin/task
```

## Configuration

By default, the application uses Neovim standard paths. If you wish to configure it, create `~/.config/task-manager-tui/config.json`:

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

## TUI Usage

Simply run `task` in your terminal to open the UI. 

### Keybindings
- `j`/`k` or `Up`/`Down`: Navigate the list
- `space`: Toggle the current task as done/todo
- `a`: Append a new task to your inbox
- `e` or `enter`: Open the selected task directly in Neovim!
- `d` or `x`: Delete the current task entirely from your system
- `/`: Filter tasks (fuzzy search description, tags, projects)
- `q` or `esc`: Quit

The TUI automatically live-reloads if you edit your tasks inside Neovim concurrently!

### Pre-applied Filters
You can launch the TUI with filters pre-applied using CLI flags:
```bash
# Launch the TUI, immediately fuzzy-searching for "urgent"
task -f "urgent"

# Launch the TUI showing only tasks from the "work" project
task -p "work"

# Launch the TUI showing both open AND completed tasks
task -s "todo,in_progress,done"
```

## CLI Usage

The executable doubles as a powerful CLI designed to be orchestrated by Neovim.
```bash
# Output tasks as JSON
task list --status=todo --project=work --json

# Inject a task directly into your inbox
task add "Review pull requests @work #urgent due:today"

# Index an entire directory of markdown files to your SQLite DB
task index ~/pkm/

# Output LSP auto-completions
task meta --json
```