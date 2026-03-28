# Stack Brain — Summary

A coordination layer for managing a personal development stack across multiple projects. The brain teaches Claude Code which stack components exist, when to use them, and how to keep projects up to date.

## What It Does

1. **Discovery**: Before adding third-party deps, Claude checks the stack catalog for existing components that cover the need
2. **Version tracking**: Each project's `Stackfile.md` pins which stack components it uses and at what version
3. **DAG-aware updates**: Components depend on each other; updates cascade in topological order (leaves first)
4. **Auditing**: `/stack-audit` compares project code against component capabilities to find duplication, drift, and missed features
5. **Constraint enforcement**: Projects and components can declare architectural rules in `CONSTRAINTS.md`, validated by audit and enforced by the agent (including pushback when requests conflict)
6. **Gap reporting**: When no component covers a need, it's logged to `STACK_GAPS.md` as a candidate for a new component
7. **Portability**: All Claude Code configuration (CLAUDE.md, settings, hooks, scripts) lives in `dotclaude/` and installs via `make setup`

## Architecture

```
~/.claude/CLAUDE.md          ← 2-line pointer to brain (always loaded, ~20 tokens)
     │
     ▼
~/newstack/brain/
├── CLAUDE.md                ← rules: discovery, versioning, DAG updates, gap reporting
├── STACK_CATALOG.md         ← auto-generated index of all components (via stack-brain refresh)
├── STACK_GAPS.md            ← backlog of missing capabilities
├── Makefile                 ← make setup / make check / make refresh
├── cmd/stack-brain/         ← Go CLI for deterministic operations
├── skills/                  ← Claude Code slash commands (canonical copies)
│   ├── stack-integrate.md   ← /stack-integrate: onboard new components
│   ├── stack-update.md      ← /stack-update: DAG-aware version cascade
│   ├── stack-audit.md       ← /stack-audit: find drift, duplication, missed capabilities + constraint violations
│   ├── stack-constraints-add.md ← /stack-constraints-add: capture a new architectural constraint
│   ├── stack-constraints-promote.md ← /stack-constraints-promote: move project constraint to component level
│   ├── stack-catalog-refresh.md ← /stack-catalog-refresh: rebuild catalog
│   └── checkpoint.md        ← /checkpoint: enhanced with stack + constraints awareness
├── dotclaude/               ← portable ~/.claude/ configuration (canonical source)
│   ├── CLAUDE.md            ← global instructions
│   ├── settings.json        ← permissions, hooks, plugins
│   └── scripts/             ← hook scripts (constraint checker, tab flash/reset)
├── templates/
│   ├── CAPABILITIES.md      ← template for new component declarations
│   ├── CONSTRAINTS.md       ← template for architectural rules
│   └── Stackfile.md         ← template for new project manifests
└── .gitignore
```

## Per-Component Files (in each stack component's repo)

- **CAPABILITIES.md** — self-declaration: what it provides, module, location, stack deps, integration, conventions
- **CLAUDE.md** — how to develop/integrate with the component
- **CHANGELOG.md** — version history (future)
- **migrations/** — per-version migration steps (future)

## Per-Project Files

- **Stackfile.md** — which stack components the project uses and at what version
- **CONSTRAINTS.md** — architectural rules enforced during audit and checkpoint (routes to component constraints when applicable)

## CLI Tool: stack-brain

Handles all deterministic operations so the LLM doesn't waste tokens scanning files:

| Command | Purpose |
|---------|---------|
| `stack-brain lookup "auth" "jwt"` | Search components by keywords |
| `stack-brain stale <project-dir>` | Check which deps are outdated |
| `stack-brain dag` | Print dependency tiers |
| `stack-brain dag --downstream-of goutils` | Scoped subgraph |
| `stack-brain migrations <comp> <from> <to>` | Concatenate migration docs |
| `stack-brain update <project-dir>` | Bump stale deps in go.mod |
| `stack-brain refresh` | Rebuild STACK_CATALOG.md |
| `stack-brain env create <name>` | Create a new environment |
| `stack-brain env add [paths...]` | Add repos to active environment |
| `stack-brain env add --external <path>` | Track external repo (pointer only) |
| `stack-brain env list` | List all environments |
| `stack-brain env info` | Show active environment details |
| `stack-brain env remove <path>` | Remove repo from environment |
| `stack-brain emit <env> [repos...]` | Generate agent instruction files (CLAUDE.md, .cursorrules, etc.) |
| `stack-brain emit <env> --target cursor` | Emit for a specific agent |
| `stack-brain emit <env> --dry-run` | Preview without writing |

All existing commands (lookup, stale, dag, refresh) are automatically scoped to the active environment when one is detected.

## Design Decisions

- **CAPABILITIES.md is the source of truth**, not the catalog — catalog is a generated index
- **Version comes from git tags / package.json**, not from CAPABILITIES.md — one less thing to keep in sync
- **Components can live anywhere** — no hardcoded ~/newstack root; location is declared per-component
- **Sub-components supported** — embedded libs (e.g., TS client in a Go repo) declared as sub-entries in parent's CAPABILITIES.md
- **LLM does judgment, CLI does computation** — lookup/stale/dag/refresh are deterministic; drafting/auditing/migrating need LLM reasoning
- **Lazy loading by design** — global CLAUDE.md is a thin pointer (~20 tokens), catalog is read only during discovery (~950 tokens), individual CAPABILITIES.md only for matched components (~300 tokens)
- **Skills are portable** — canonical copies live in brain/skills/, installed to ~/.claude/commands/ via `make setup`
- **Constraints are captured, not invented** — they come from real corrections during sessions, not upfront design
- **Agent pushback is a feature** — the agent should challenge requests that violate constraints or architectural intent

## Token Budget

| Layer | ~Tokens | When |
|-------|---------|------|
| Global CLAUDE.md pointer | ~20 | Every session |
| Brain CLAUDE.md | ~800 | On discovery |
| STACK_CATALOG.md | ~950 | On discovery (or via CLI for 0 tokens) |
| Single CAPABILITIES.md | ~300 | For matched component |
| All CAPABILITIES.md | ~4,300 | Never (anti-pattern) |
