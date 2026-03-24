#!/bin/bash
# Reset iTerm2 tab color when user resumes interaction

# Kill the flash process if running
if [[ -f /tmp/claude-tab-flash.pid ]]; then
  kill "$(cat /tmp/claude-tab-flash.pid)" 2>/dev/null
  rm -f /tmp/claude-tab-flash.pid
fi

# Reset tab color to default
printf '\e]6;1;bg;*;default\a' > /dev/tty
