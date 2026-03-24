#!/bin/bash
# Flash iTerm2 tab color when Claude needs attention
# Uses iTerm2 proprietary escape sequences

# Kill any existing flash process for this terminal
if [[ -f /tmp/claude-tab-flash.pid ]]; then
  kill "$(cat /tmp/claude-tab-flash.pid)" 2>/dev/null
  rm -f /tmp/claude-tab-flash.pid
fi

# Start flashing in background with a 5-minute auto-timeout
# Terminates via: reset-tab.sh (on next prompt), or self-timeout, or SIGTERM
(
  SECONDS=0
  MAX_SECONDS=30

  while (( SECONDS < MAX_SECONDS )); do
    # Bright orange
    printf '\e]6;1;bg;red;brightness;255\a\e]6;1;bg;green;brightness;140\a\e]6;1;bg;blue;brightness;0\a' > /dev/tty
    sleep 0.6
    # Dim amber
    printf '\e]6;1;bg;red;brightness;140\a\e]6;1;bg;green;brightness;70\a\e]6;1;bg;blue;brightness;0\a' > /dev/tty
    sleep 0.6
  done

  # Timed out — reset tab color and clean up
  printf '\e]6;1;bg;*;default\a' > /dev/tty
  rm -f /tmp/claude-tab-flash.pid
) &

echo $! > /tmp/claude-tab-flash.pid
