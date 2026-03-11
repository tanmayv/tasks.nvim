#!/bin/bash
export VIMRUNTIME=$(nvim --print-appinfo | grep -o '\"runtime_dir\": \"[^\"]*' | cut -d '"' -f 4)

nvim --headless \
  -c "set rtp+=.tests/plenary.nvim" \
  -c "set rtp+=.tests/sqlite.lua" \
  -c "set rtp+=." \
  -c "PlenaryBustedDirectory tests { minimal_init = 'tests/minimal_init.lua' }" \
  -c "qa!"
