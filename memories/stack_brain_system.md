---
name: Stack Brain System
description: Personal dev stack management system — coordination layer for component discovery, version tracking, DAG-aware updates, and gap reporting
type: project
---

A "stack brain" system manages a personal development stack (16+ components, mostly in ~/newstack) consumed by 18+ projects (mostly in ~/projects).

**Key artifacts:**
- ~/newstack/brain/ — coordination layer (STACK_CATALOG.md, STACK_GAPS.md, CLAUDE.md, skills/, cmd/stack-brain/)
- CAPABILITIES.md — per-component self-declaration of what it provides, dependencies, location
- Stackfile.md — per-project record of which stack components are used and at what version
- CHANGELOG.md + migrations/ — per-component version history and migration steps (future)

**CLI tool (stack-brain):**
- `lookup` — multi-phrase keyword search across capabilities
- `stale` — compare project versions against component HEAD tags
- `dag` — topological sort of dependency graph
- `update` — bump stale deps in go.mod
- `migrations` — concatenate migration files between versions
- `refresh` — rebuild STACK_CATALOG.md from all CAPABILITIES.md files

**Skills:**
- `/stack-integrate` — onboard new components from any path
- `/stack-update` — DAG-aware cascading version updates
- `/stack-audit` — find duplication, drift, missed capabilities
- `/stack-catalog-refresh` — regenerate catalog
- `/checkpoint` — enhanced to sync stack artifacts

**Design principles:**
- CAPABILITIES.md is source of truth, catalog is generated index
- Version derived from git tags/package.json, not manually maintained
- Components can live anywhere (no hardcoded root)
- LLM does judgment, CLI does computation
- Lazy loading: ~1,300 tokens for a discovery decision
