---
name: Stack Brain System
description: Environment-based codebase governance — component discovery, version tracking, DAG-aware updates, constraint enforcement, and agent-agnostic portability
type: project
---

Stack brain manages **environments** — named collections of repos reasoned about together. The personal stack (16+ components in ~/newstack consumed by 18+ projects) is `env:newstack`, but any repo collection can be its own environment.

**Key concepts:**
- **Environment** — a lens over repos: defines scope for lookup/dag/stale/audit. Detected automatically via STACK_ENV or cwd membership.
- **Intrinsic to repos** — CAPABILITIES.md, CONSTRAINTS.md, CLAUDE.md live in each repo, unchanged across environments.
- **External repos** — repos you track but don't control get thin pointer files (~80 tokens) in the env config, not in the repo itself.
- **Semantic deps** — DAG edges can be "hard" (module import) or "semantic" (fork-of, shares-api-contract).

**CLI tool (stack-brain):**
- `lookup`, `stale`, `dag`, `refresh`, `update`, `migrations` — all env-scoped when active, legacy fallback otherwise
- `env create/list/add/remove/info` — environment management
- `env add --external` — track external repos with pointer files
- `env create --import-catalog` — migrate from existing STACK_CATALOG.md

**Environment config:** `~/.config/stack-brain/envs/<name>/` with env.yaml, conventions.md, gaps.md, catalog.md, external/

**Design principles:**
- Environments are lenses, not containers — repos are intrinsic
- STACK_ENV + cwd auto-detection — zero config switching, zero LLM tokens
- External repos get pointers, not summaries — route to source, never stale
- Third-party lib docs via context7 — nothing stored
- Conventions use router pattern — inline → pointer when they grow
- CLI-first, MCP later if needed
- `emit <env> [repos...]` — compile constraints + conventions into agent instruction files (CLAUDE.md, .cursorrules, .windsurfrules, copilot). Requires explicit env arg (write ops should not be implicit). Marker-based injection preserves hand-written content. Idempotent.
- Repo name inference from CAPABILITIES.md H1 heading (handles worktree dirs like main/master)
- `stale` now reports external repo commit drift with commits_behind count
- `dag` now includes semantic_edges and external_nodes fields
