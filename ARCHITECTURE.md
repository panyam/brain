# Stack Brain — Architecture

## Problem

A personal development stack of 16+ reusable components is consumed by 18+ projects. Without coordination:
- New projects require manually telling Claude which components to use
- Returning to old projects means manually updating stack versions
- No visibility into capability gaps or duplication across projects

## Solution: Layered Architecture

### Layer 1: Self-Declaring Components (CAPABILITIES.md)

Each stack component declares what it provides in a `CAPABILITIES.md` file at its root. This is the **source of truth**. The file contains:
- Capability tags with descriptions
- Module path and location (arbitrary filesystem path)
- Stack dependencies (other components it depends on)
- Integration patterns
- Sub-components (embedded libraries like TS clients)

Version is NOT in CAPABILITIES.md — it's derived from git tags or package.json at query time.

### Layer 2: Environments

An **environment** is a named collection of repos that are reasoned about together. It's a lens, not a container — repos are intrinsic (constraints and capabilities live in the repo), the environment just defines scope.

Each environment has:
- **Repo list** — which repos participate
- **Catalog** — auto-generated index of member repos' capabilities (via `stack-brain refresh`)
- **Conventions** — cross-cutting rules for this group of repos (router pattern: inline → pointers)
- **Gaps** — capability gaps identified in this environment
- **External repos** — thin pointer files for repos you track but don't control

The existing newstack setup is `env:newstack`. Work projects, open-source contributions, or any repo collection can be their own environment.

Detection is automatic: `STACK_ENV` env var > cwd membership > `--env` flag. Zero config switching.

### Layer 3: Brain (Coordination)

The brain at `~/newstack/brain/` provides the CLI, skills, and setup:
- **CLI** — `stack-brain` binary for deterministic operations
- **Skills** — Claude Code slash commands for workflows
- **CLAUDE.md** — rules for discovery, versioning, updates
- **STACK_GAPS.md** — backlog of missing capabilities (legacy; per-env gaps.md is preferred for new envs)

### Layer 4: Project Manifests (Stackfile.md)

Each project has a `Stackfile.md` that tracks which stack components it uses and at what version. This enables:
- Staleness detection (`stack-brain stale`)
- Version updates (`stack-brain update`)
- Audit baseline (`/stack-audit`)

### Layer 4: Architectural Constraints (CONSTRAINTS.md)

Enforceable rules about how code should be structured, captured from real "that's wrong" moments. Same router pattern as CAPABILITIES.md:
- **Project-level CONSTRAINTS.md** — entry point. Rules are inline or point to component constraints.
- **Component-level CONSTRAINTS.md** — optional. Rules that apply to all consumers of a component.
- Validated by `/stack-audit` as the highest-priority finding category.
- Updated by `/checkpoint` when architectural corrections happen during a session.
- Promotion path: project → component when the same rule appears in 2+ projects.

## Data Flow

```
Component CAPABILITIES.md ──► stack-brain refresh ──► STACK_CATALOG.md
                                                          │
Project go.mod ──► Stackfile.md ──► stack-brain stale ──► stale deps list
                                                          │
                                         LLM judgment ◄───┘
                                              │
                                    /stack-audit findings
                                    /stack-update migrations

Project CONSTRAINTS.md ──► (inline rules + component pointers)
       │                              │
       ▼                              ▼
  /stack-audit validation    Component CONSTRAINTS.md
       │
  /checkpoint capture ◄── mid-session architectural corrections
```

## Deterministic vs LLM Split

| Deterministic (CLI) | LLM Judgment (Skills) |
|---------------------|----------------------|
| Keyword lookup | Capability matching ("I need collab") |
| Version comparison | Drafting CAPABILITIES.md |
| DAG computation | Applying migration steps |
| Catalog refresh | Identifying capability gaps |
| go.mod updates | Auditing for convention drift |
| Migration file concatenation | Proposing code changes |
| Grep-based constraint checks | Judging "manual" constraints |
| | Suggesting new constraints from corrections |

## Dependency DAG

Components form a directed acyclic graph. Updates must follow topological order:

```
Tier 0 (leaves): goutils, gocurrent, tsutils, cachewarden, protoc-gen-*
Tier 1: templar, oneauth, servicekit, tlex, devloop
Tier 2: goapplib, s3gen, massrelay, Galore
Tier 3: Projects
```

## Token Efficiency

The system is designed for minimal context window impact:
- Global CLAUDE.md adds ~20 tokens (just a pointer)
- Discovery path: catalog (~950 tokens) + one CAPABILITIES.md (~300 tokens) = ~1,300 tokens
- CLI operations cost 0 LLM tokens (deterministic)
- Anti-pattern: reading all 16 CAPABILITIES.md files (~4,300 tokens) — prevented by CLI + CLAUDE.md rules

## Integration Points

- **~/.claude/CLAUDE.md** → global instructions including stack pointer and constraint rules (installed via `make setup`)
- **~/.claude/settings.json** → permissions, hooks (SessionStart constraint check), plugins (installed on fresh machines via `make setup`)
- **~/.claude/scripts/** → hook scripts including constraint checker (installed via `make setup`)
- **~/.claude/commands/** → skill files (installed via `make setup`)
- **~/.local/bin/stack-brain** → CLI binary (installed via `make setup`)
- **~/.config/stack-brain/envs/** → environment configs (created via `stack-brain env create`)
- **/checkpoint** → enhanced to sync stack artifacts and constraints alongside project docs

## Portability

All portable Claude Code configuration lives in `dotclaude/` — the canonical source for `~/.claude/` files. On a new machine:

```
git clone <brain-repo> ~/newstack/brain
cd ~/newstack/brain && make setup
```

This installs CLAUDE.md, settings.json, scripts, skills, and the CLI. Settings.json is not overwritten if it already exists.

## Agent-Agnostic Design (Future: Emit System)

The environment system is designed to support multiple AI agents, not just Claude Code. A planned `stack-brain emit` command will compile environment knowledge (constraints, conventions) into each agent's native instruction format:
- Claude Code → CLAUDE.md
- Cursor → .cursor/rules/
- Windsurf → .windsurfrules
- Copilot → .github/copilot-instructions.md

The `.config/stack-brain/` knowledge store is the single source of truth; agent-specific files are generated artifacts.
