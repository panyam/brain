---
name: User Preferences for Stack Work
description: How the user prefers to work with the stack system — conventions, style, decision patterns
type: feedback
---

- Keep things thin and lazy — minimize token usage, don't load files greedily
- Deterministic operations belong in scripts/CLI, not LLM reasoning
- Prefer deriving data from source (git tags, go.mod) over manually maintained fields
- Components should self-declare (CAPABILITIES.md) — central catalog is just an index
- No hardcoded paths — components can live anywhere
- Skills should be portable — canonical copies in brain/skills/, installed via make setup
- make setup should be idempotent and handle updates (no separate make update needed)
- Audit before update — understand what needs to change before bumping versions
- Don't auto-fix during audits — present findings, let user decide
- Sub-components (embedded TS clients etc.) should be declared in parent's CAPABILITIES.md
- The user uses bare repos + worktrees (via `wt`/worktrunk tool)
- The user values the LLM coordinating and driving, not explaining processes
