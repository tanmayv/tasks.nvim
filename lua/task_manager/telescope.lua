local M = {}

function M.tasks(opts)
  opts = opts or {}
  
  local has_telescope, pickers = pcall(require, "telescope.pickers")
  if not has_telescope then
    vim.notify("Telescope.nvim is required for this feature.", vim.log.levels.ERROR)
    return
  end
  
  local finders = require("telescope.finders")
  local conf = require("telescope.config").values
  local entry_display = require("telescope.pickers.entry_display")
  local actions = require("telescope.actions")
  local action_state = require("telescope.actions.state")
  
  local db = require("task_manager.db")
  
  -- Default to open tasks if status is not explicitly requested
  if not opts.status then
    opts.status = { "todo", "in_progress" }
  end
  
  local tasks = db.get_tasks(opts)
  if not tasks or #tasks == 0 then
    vim.notify("No tasks found matching criteria.", vim.log.levels.INFO)
    return
  end

  local displayer = entry_display.create({
    separator = " ",
    items = {
      { width = 6 }, -- Score
      { width = 4 }, -- Status
      { width = 12 }, -- Project
      { width = 15 }, -- Tags
      { width = 20 }, -- File Context
      { remaining = true }, -- Description
    },
  })

  local function make_display(entry)
    local task = entry.value
    
    local score_str = string.format("[%d]", task.score or 0)
    local hl_score = "TelescopeResultsNumber"
    if task.score and task.score > 100 then
      hl_score = "DiagnosticError" -- Highlight urgent tasks in red
    elseif task.score and task.score > 50 then
      hl_score = "DiagnosticWarn"  -- Highlight medium-urgent in yellow
    end
    
    local status_map = {
      todo = "[ ]",
      in_progress = "[/]",
      done = "[x]",
      cancelled = "[-]"
    }
    
    local status_str = status_map[task.status] or "[?]"
    local hl_status = "TelescopeResultsIdentifier"
    if task.status == "done" then hl_status = "TelescopeResultsComment" end
    
    local project_str = task.project and ("@" .. task.project) or ""
    local tags_str = ""
    if task.tags and #task.tags > 0 then
      tags_str = "#" .. table.concat(task.tags, " #")
    end
    
    -- Extract filename from path for concise display
    local filename = vim.fn.fnamemodify(task.file_path, ":t")
    local context_str = string.format("(%s:%d)", filename, task.line_number)

    return displayer({
      { score_str, hl_score },
      { status_str, hl_status },
      { project_str, "TelescopeResultsConstant" },
      { tags_str, "TelescopeResultsSpecialComment" },
      { context_str, "TelescopeResultsComment" },
      task.description,
    })
  end

  pickers.new({}, {
    prompt_title = "Tasks",
    finder = finders.new_table({
      results = tasks,
      entry_maker = function(task)
        -- Combine fields for fuzzy searching
        local search_str = task.description
        if task.project then search_str = search_str .. " @" .. task.project end
        if task.tags and #task.tags > 0 then
          search_str = search_str .. " #" .. table.concat(task.tags, " #")
        end
        
        return {
          value = task,
          display = make_display,
          ordinal = search_str,
          filename = task.file_path,
          lnum = task.line_number,
          col = 1,
        }
      end,
    }),
    sorter = conf.generic_sorter({}),
    previewer = conf.grep_previewer({}),
    layout_strategy = "vertical",
    layout_config = {
      width = 0.95,
      height = 0.95,
      preview_height = 0.4,
    },
    attach_mappings = function(prompt_bufnr, map)
      actions.select_default:replace(function()
        actions.close(prompt_bufnr)
        local selection = action_state.get_selected_entry()
        if selection then
          vim.cmd("edit " .. vim.fn.fnameescape(selection.filename))
          vim.api.nvim_win_set_cursor(0, { selection.lnum, 0 })
          -- Optional: center screen on line
          vim.cmd("normal! zz")
        end
      end)
      
      -- Add mapping to toggle task done state
      local function toggle_task()
        local current_picker = action_state.get_current_picker(prompt_bufnr)
        local multi_selections = current_picker:get_multi_selection()
        local selections_to_toggle = {}

        if not vim.tbl_isempty(multi_selections) then
          selections_to_toggle = multi_selections
        else
          local selection = action_state.get_selected_entry()
          if selection then
            table.insert(selections_to_toggle, selection)
          end
        end

        if vim.tbl_isempty(selections_to_toggle) then
          return
        end
        
        local core = require("task_manager.core")
        local toggled_count = 0
        local failed_count = 0

        for _, selection in ipairs(selections_to_toggle) do
          local success = core.toggle_done(selection.filename, selection.lnum)
          if success then
            toggled_count = toggled_count + 1
          else
            failed_count = failed_count + 1
          end
        end
        
        if toggled_count > 0 then
          -- Re-fetch tasks based on original options
          local updated_tasks = db.get_tasks(opts)
          
          current_picker:refresh(
            finders.new_table({
              results = updated_tasks,
              entry_maker = current_picker.finder.entry_maker,
            }),
            { reset_prompt = false }
          )
          
          if toggled_count == 1 then
            vim.notify("Task state toggled!", vim.log.levels.INFO)
          else
            vim.notify(toggled_count .. " tasks toggled successfully!", vim.log.levels.INFO)
          end
        end

        if failed_count > 0 then
          vim.notify("Failed to toggle " .. failed_count .. " tasks.", vim.log.levels.WARN)
        end
      end

      local function copy_tasks()
        local current_picker = action_state.get_current_picker(prompt_bufnr)
        local multi_selections = current_picker:get_multi_selection()
        local selections_to_copy = {}

        if not vim.tbl_isempty(multi_selections) then
          selections_to_copy = multi_selections
        else
          local selection = action_state.get_selected_entry()
          if selection then
            table.insert(selections_to_copy, selection)
          end
        end

        if vim.tbl_isempty(selections_to_copy) then
          return
        end

        actions.close(prompt_bufnr)

        -- Open a vertical split scratch buffer
        vim.cmd("vnew")
        local bufnr = vim.api.nvim_get_current_buf()
        
        vim.api.nvim_buf_set_option(bufnr, "buftype", "acwrite") -- Allow saving to trigger BufWriteCmd
        vim.api.nvim_buf_set_option(bufnr, "bufhidden", "wipe")
        vim.api.nvim_buf_set_option(bufnr, "filetype", "markdown")
        vim.api.nvim_buf_set_name(bufnr, "task_manager_edit_" .. bufnr)
        
        -- Mark as an editor buffer
        vim.b[bufnr].is_task_manager_editor = true
        
        local parser = require("task_manager.parser")
        local lines = { "# Edit Tasks (Save with :w to apply changes)", "" }
        local origins = {}
        
        local current_line = 3 -- 1-based index for lines array, tracking where tasks start
        
        for _, selection in ipairs(selections_to_copy) do
          local task = selection.value
          local line = parser.format_line("- ", task.status, task)
          table.insert(lines, line)
          
          -- Save origin tracking information
          origins[task.id] = {
            file_path = task.file_path,
            original_line = task.original_line,
            id = task.id
          }
          current_line = current_line + 1
        end
        
        vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, lines)
        vim.b[bufnr].task_origins = origins
        
        -- Give user instructions
        vim.notify("Opened " .. #selections_to_copy .. " tasks for editing. Save buffer to apply changes.", vim.log.levels.INFO)
      end

      map("i", "<C-x>", toggle_task)
      map("n", "<C-x>", toggle_task)
      
      map("i", "<C-v>", copy_tasks)
      map("n", "<C-v>", copy_tasks)
      
      return true
    end,
  }):find()
end

return M
