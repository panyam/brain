# Stack Brain — Architecture

## Problem

A personal development stack of 16+ reusable components is consumed by 18+ projects. Without coordination:
- New projects require manually telling Claude which components to use
- Returning to old projects means manually updating stack versions
- No visibility into capability gaps or duplication across projects

## Solution: Three-Layer Architecture

### Layer 1: Self-Declaring Components (CAPABILITIES.md)

Each stack component declares what it provides in a `CAPABILITIES.md` file at its root. This is the **source of truth**. The file contains:
- Capability tags with descriptions
- Module path and location (arbitrary filesystem path)
- Stack dependencies (other components it depends on)
- Integration patterns
- Sub-components (embedded libraries like TS clients)

Version is NOT in CAPABILITIES.md — it's derived from git tags or package.json at query time.

### Layer 2: Brain (Coordination)

The brain at `~/newstack/brain/` aggregates and coordinates:
- **STACK_CATALOG.md** — generated index of all components (rebuilt by `stack-brain refresh`)
- **STACK_GAPS.md** — backlog of missing capabilities
- **CLAUDE.md** — rules for discovery, versioning, updates
- **Skills** — Claude Code slash commands for workflows
- **CLI** — `stack-brain` binary for deterministic operations

### Layer 3: Project Manifests (Stackfile.md)

Each project has a `Stackfile.md` that tracks which stack components it uses and at what version. This enables:
- Staleness detection (`stack-brain stale`)
- Version updates (`stack-brain update`)
- Audit baseline (`/stack-audit`)

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

- **Global CLAUDE.md** → thin pointer to brain (installed via `make setup`)
- **~/.claude/commands/** → skill files (installed via `make setup`)
- **~/.local/bin/stack-brain** → CLI binary (installed via `make setup`)
- **/checkpoint** → enhanced to sync stack artifacts alongside project docs
