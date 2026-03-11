local M = {}

function M.ensure_buffer_loaded(file_path)
  local bufnr = vim.fn.bufadd(file_path)
  if not vim.api.nvim_buf_is_loaded(bufnr) then
    vim.fn.bufload(bufnr)
  end
  return bufnr
end

function M.save_buffer(bufnr)
  if vim.api.nvim_buf_get_option(bufnr, "buftype") == "" then
    vim.api.nvim_buf_call(bufnr, function()
      vim.cmd('silent! write')
    end)
  end
end

return M
