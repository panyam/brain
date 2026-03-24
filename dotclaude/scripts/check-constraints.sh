#!/bin/bash
# SessionStart hook: surface CONSTRAINTS.md if present in project root

CONSTRAINTS_FILE="CONSTRAINTS.md"

if [ -f "$CONSTRAINTS_FILE" ]; then
  count=$(grep -c '^### ' "$CONSTRAINTS_FILE" 2>/dev/null)
  count=${count:-0}
  echo "⚠ This project has ${count} architectural constraints in CONSTRAINTS.md — read before making structural changes."

  # Also check for component-level constraint pointers
  pointers=$(grep -c 'See .*/CONSTRAINTS.md' "$CONSTRAINTS_FILE" 2>/dev/null)
  pointers=${pointers:-0}
  if [ "$pointers" -gt 0 ]; then
    echo "  (${pointers} route to component-level constraints)"
  fi
fi
