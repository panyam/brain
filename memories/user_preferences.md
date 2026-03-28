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
- Evolve existing tools rather than building new ones (not N+1) — validated when choosing to add environments to stack-brain rather than creating a separate "codebrain" tool
- YAGNI on plugin frameworks — don't build abstraction until there are 2+ concrete use cases (validated: no Plugin interface, just stack subcommands)
- Pull model over push for external repo tracking — the consumer asks "what changed upstream?" rather than trying to notify upstream contributors
- CLI-first, MCP later — emit handles cross-agent delivery without the complexity of a server
- Global config switching is an anti-pattern — prefer env var + cwd auto-detection (zero tokens, zero config management)
- Write operations (emit) should require explicit params, not implicit detection — "be explicit about what you're stamping into repos"
- User prefers to run onboarding commands themselves (e.g., `/stack-integrate`) rather than having the agent create files directly
- Deprioritize features that duplicate existing workflows — `stack-brain checkpoint` CLI was deprioritized because `/checkpoint` skill + manual emit already covers the need
