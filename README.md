# Stack Brain

Coordination layer for managing environments of repos — component discovery, version tracking, DAG-aware updates, architectural constraint enforcement, and agent-agnostic instruction file generation. Works with Claude Code, Cursor, Windsurf, and Copilot.

## Why This Exists

When working with AI coding agents across many projects and libraries, three problems keep coming up:

1. **Agents don't know what you've already built.** They reach for third-party dependencies or rewrite functionality that already exists in your stack.
2. **Architectural knowledge lives in your head.** You steer agents away from bad patterns in real-time, but that judgment is lost when the session ends. The next session makes the same mistakes.
3. **Configuration isn't portable.** Every new machine or session requires re-teaching the agent how you work.

Stack Brain solves these by making your stack discoverable, your architectural instincts enforceable, and your environment reproducible.

## Core Principles

1. **Reuse over reinvention** — use stack components before third-party deps, use third-party deps before building from scratch
2. **Self-declaring components** — each component declares what it provides; the catalog is just an index, not the source of truth
3. **Constraints are captured, not invented** — architectural rules come from real "that's wrong" corrections during sessions, not upfront design documents
4. **LLM does judgment, CLI does computation** — deterministic work (lookup, DAG, version compare) belongs in code, not the context window
5. **Lazy by default** — minimal token footprint; load only what's needed, when it's needed
6. **Agent pushback is a feature** — the agent should challenge you when a request conflicts with your own stated architectural rules
7. **Portable and idempotent** — `make setup` on any machine gets you to the same state; re-running is safe

## Quick Start

```bash
cd ~/newstack/brain
make setup
```

This installs everything:
- `~/.claude/CLAUDE.md` — global instructions with stack discovery rules, constraint enforcement, and pushback behavior
- `~/.claude/settings.json` — permissions, hooks, and plugin config (skipped if already exists)
- `~/.claude/scripts/` — hook scripts (SessionStart constraint checker, tab flash/reset)
- `~/.claude/commands/` — slash command skills
- `~/.local/bin/stack-brain` — CLI for deterministic operations

Re-run `make setup` after pulling updates. Run `make check` to verify everything is in place.

## How It Works

### Component Discovery

Each stack component has a `CAPABILITIES.md` declaring what it provides. The brain aggregates these into `STACK_CATALOG.md`. Before adding a third-party dependency, Claude checks the catalog:

```bash
stack-brain lookup "auth" "jwt"          # Search by keywords
stack-brain stale ~/projects/myapp       # Check for outdated deps
stack-brain dag                          # Show dependency tiers
```

### Architectural Constraints

Projects can have a `CONSTRAINTS.md` with enforceable rules — things like "no direct JWT parsing, use oneauth middleware" or "no try/catch that swallows errors". Each constraint has a verify pattern (grep, test command, or manual).

Constraints are:
- **Checked on session start** via a SessionStart hook ("This project has N constraints")
- **Validated by `/stack-audit`** as the highest-priority finding category
- **Maintained by `/checkpoint`** which captures corrections from the session
- **Enforced by the agent** which pushes back when a request would violate a constraint

Constraints follow a router pattern — project-level `CONSTRAINTS.md` is the entry point and can point to component-level constraints for rules that apply to all consumers of a component.

```
# Project CONSTRAINTS.md

### No Direct JWT Verification
**Rule**: Always go through oneauth middleware for auth
**Why**: Raw verification skips token rotation
**Verify**: `grep -rn 'jwt.Parse\|jwt.Verify' --include='*.go' | grep -v oneauth`
**Scope**: project-wide

### Connection Pooling
See oneauth/CONSTRAINTS.md: connection-pooling
```

### Environments

An **environment** is a named collection of repos you reason about together. Your personal stack is one environment; a work project with its own repos is another.

```bash
# Create an environment and add repos
stack-brain env create gvip
stack-brain env add ~/work/GVIP/* --env gvip

# Track external repos you don't control (pointer files only)
stack-brain env add ~/work/AVIP/AVIP_distribution --external --dep-type semantic --relationship "fork-of" --env gvip

# Detection is automatic — no config switching needed
cd ~/work/GVIP/svc && stack-brain lookup "pipeline"  # auto-detects env:gvip
```

Detection order: `STACK_ENV` env var > cwd membership > `--env` flag. Zero tokens, zero config files to edit.

External repos get thin pointer files (~80 tokens) in the env config — nothing written to the external repo. The pointer tells agents *where to look*, not *what's there*. `stale` checks commit drift on externals.

### Agent-Agnostic Emit

`stack-brain emit` compiles CONSTRAINTS.md + CAPABILITIES.md + env conventions into agent-native instruction files:

```bash
stack-brain emit newstack ~/projects/lilbattle           # specific repos
stack-brain emit newstack                                 # all repos in env
stack-brain emit newstack --target cursor                 # single agent
stack-brain emit newstack --dry-run                       # preview
```

| Agent | Output | Method |
|-------|--------|--------|
| Claude Code | CLAUDE.md | Marker injection (preserves hand-written content) |
| Cursor | .cursor/rules/stack-brain.mdc | Dedicated file |
| Windsurf | .windsurfrules | Marker injection |
| Copilot | .github/copilot-instructions.md | Marker injection |

Edit CONSTRAINTS.md (single source of truth), re-run `emit`, all agents get updated. Idempotent.

### Version Updates

Components form a DAG. Updates cascade in topological order:

```
Tier 0 (leaves): goutils, gocurrent, tsutils, cachewarden, protoc-gen-*
Tier 1:          templar, oneauth, servicekit, tlex, devloop
Tier 2:          goapplib, s3gen, massrelay, Galore
Tier 3:          Projects
```

## CLI

### Discovery & Versioning

| Command | Purpose |
|---------|---------|
| `stack-brain lookup "auth" "jwt"` | Search components by keywords |
| `stack-brain stale <project-dir>` | Check which deps are outdated (includes external repo drift) |
| `stack-brain dag` | Print dependency tiers (includes semantic edges) |
| `stack-brain dag --downstream-of goutils` | Scoped subgraph |
| `stack-brain migrations <comp> <from> <to>` | Concatenate migration docs |
| `stack-brain update <project-dir>` | Bump stale deps in go.mod |
| `stack-brain refresh` | Rebuild STACK_CATALOG.md |

### Environments

| Command | Purpose |
|---------|---------|
| `stack-brain env create <name>` | Create a new environment |
| `stack-brain env create <name> --import-catalog <path>` | Migrate from existing STACK_CATALOG.md |
| `stack-brain env add [paths...]` | Add repos (supports globs) |
| `stack-brain env add --external <path>` | Track external repo (pointer only) |
| `stack-brain env list` | List all environments |
| `stack-brain env info` | Show active environment details |
| `stack-brain env remove <path>` | Remove repo from environment |

### Emit (Agent-Agnostic)

| Command | Purpose |
|---------|---------|
| `stack-brain emit <env> [repos...]` | Generate instruction files for all agents |
| `stack-brain emit <env> --target cursor` | Single agent target |
| `stack-brain emit <env> --dry-run` | Preview without writing |

All commands output JSON (except migrations which outputs markdown). Discovery commands are automatically scoped to the active environment when one is detected.

## Skills (Slash Commands)

| Skill | Purpose |
|-------|---------|
| `/stack-integrate` | Onboard a new component into the stack |
| `/stack-update` | DAG-aware cascading version updates |
| `/stack-audit` | Find drift, duplication, missed capabilities, and constraint violations |
| `/stack-constraints-add` | Capture a new architectural constraint |
| `/stack-constraints-promote` | Move a project constraint to component level |
| `/stack-catalog-refresh` | Rebuild the catalog from CAPABILITIES.md files |
| `/checkpoint` | Sync session learnings to project docs, stack artifacts, and constraints |

## Directory Structure

```
~/newstack/brain/
├── CLAUDE.md                  ← Rules: discovery, versioning, constraints, emit, environments
├── STACK_CATALOG.md           ← Auto-generated component index
├── STACK_GAPS.md              ← Backlog of missing capabilities
├── cmd/stack-brain/           ← Go CLI source
│   ├── internal/env/          ← Environment config package
│   └── internal/emit/         ← Emit template engine
├── skills/                    ← Canonical skill files (installed to ~/.claude/commands/)
├── templates/
│   ├── CAPABILITIES.md        ← Template for new components
│   ├── CONSTRAINTS.md         ← Template for project/component constraints
│   └── Stackfile.md           ← Template for project manifests
├── dotclaude/                 ← Portable ~/.claude/ configuration
│   ├── CLAUDE.md              ← Global instructions (canonical source)
│   ├── settings.json          ← Permissions, hooks, plugins
│   └── scripts/               ← Hook scripts
├── memories/                  ← Persistent context across sessions
├── Makefile                   ← make setup / make check / make refresh
└── .gitignore

~/.config/stack-brain/          ← Environment configs (per-user, not in repo)
└── envs/
    └── <name>/
        ├── env.yaml            ← Repo list, metadata
        ├── conventions.md      ← Cross-cutting rules for this group
        ├── gaps.md             ← Capability gaps
        └── external/           ← Pointer files for external repos
```

## Per-Component Files

Each stack component should have:
- **CAPABILITIES.md** — what it provides (source of truth for the catalog)
- **CONSTRAINTS.md** — optional; architectural rules for consumers
- **CLAUDE.md** — how to develop/integrate with it
- **CHANGELOG.md** — version history
- **migrations/** — per-version migration steps

## Per-Project Files

- **Stackfile.md** — which stack components the project uses and at what version
- **CONSTRAINTS.md** — architectural rules enforced during audit and checkpoint

## Design Details

- **Version comes from git tags / package.json** — not manually maintained in CAPABILITIES.md
- **Components can live anywhere** — no hardcoded root; location is declared per-component
- **Sub-components supported** — embedded libs (e.g., TS client in a Go repo) declared as sub-entries in parent's CAPABILITIES.md
- **Token budget**: global CLAUDE.md ~20 tokens (always loaded), catalog ~950 tokens (on discovery), single CAPABILITIES.md ~300 tokens (on match)

## Requirements

- Go 1.24+ (for CLI)
- Git
- Any AI coding agent: Claude Code, Cursor, Windsurf, or Copilot (emit generates instruction files for all of them)
