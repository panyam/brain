# Stack Brain

This directory is the coordination layer for a personal development stack — a collection of reusable libraries and tools that projects should prefer over third-party dependencies.

Components can live anywhere on the filesystem. The brain doesn't assume a fixed root — each component's CAPABILITIES.md declares its own location. The catalog (STACK_CATALOG.md) is just an index of those locations.

## Memories

Read ~/newstack/brain/memories/ for persistent context about the stack system, user preferences, and past decisions. These are version-controlled and shared across all sessions.

## Core Principle

Projects should use stack components before reaching for third-party dependencies or building from scratch.

## stack-brain CLI

Use the `stack-brain` CLI for all deterministic stack operations. Do NOT grep or read CAPABILITIES.md files directly — the CLI is optimized for this and avoids wasting tokens.

```bash
# Search for components by keywords (multiple phrases supported)
stack-brain lookup "federated auth" signups wasm

# Check which deps are outdated in a project
stack-brain stale ~/projects/lilbattle

# Print the dependency DAG with topological tiers
stack-brain dag

# Show only components downstream of a specific component
stack-brain dag --downstream-of goutils

# Concatenate migration files between versions
stack-brain migrations servicekit 0.2.0 0.4.0

# Rebuild STACK_CATALOG.md from all CAPABILITIES.md files
stack-brain refresh
```

All commands output JSON (except migrations which outputs markdown). Use these instead of reading raw files.

## Environments

Environments let you group repos and reason about them together. The existing newstack setup is one environment; you can create others for work projects or any repo collection.

```bash
# Create an environment
stack-brain env create gvip

# Add repos (supports globs)
stack-brain env add ~/work/GVIP/*

# Add external repos you don't control (pointer files only, nothing written to repo)
stack-brain env add ~/work/AVIP/AVIP_distribution --external --dep-type semantic --relationship "fork-of"

# List environments
stack-brain env list

# Show active environment details
stack-brain env info

# Migrate existing newstack setup
stack-brain env create newstack --import-catalog ~/newstack/brain/STACK_CATALOG.md --brain-dir ~/newstack/brain
```

### Environment Detection (zero config switching)

Detection is automatic and deterministic (0 LLM tokens):
1. `STACK_ENV` env var — explicit override
2. cwd membership — if you're inside a registered repo, that env is active
3. `--env` flag on individual commands

All existing commands (lookup, stale, dag, refresh) are env-scoped when an environment is active. Without an active env, they fall back to legacy brain-dir behavior.

### External Repos

For repos you work with but don't control, use `--external`. This creates thin pointer files (~80 tokens) in the environment config — NOT in the external repo. The pointer tells the LLM *where to look*, not *what's there*. Actual source is read on demand.

External repos support semantic dependencies (fork-of, shares-api-contract) alongside hard module deps. Both show up in the DAG and are checked by `stale`.

### Environment Config Location

```
~/.config/stack-brain/envs/<name>/
  env.yaml          — repo list, metadata
  conventions.md    — cross-cutting rules for this group
  gaps.md           — capability gaps
  catalog.md        — auto-generated (via stack-brain refresh)
  external/         — pointer files for external repos
```

## Discovery Rule

Before adding any third-party dependency or building new infrastructure:

1. Run `stack-brain lookup <keywords>` with relevant terms
2. If a match is found → read that component's CAPABILITIES.md for integration details
3. If a component partially covers the need → use it and flag the gap to the user as a potential stack improvement
4. If nothing covers the need → flag it: "Your stack doesn't have a component for X — want to build one, or use a third-party lib for now?" and log it to ~/newstack/brain/STACK_GAPS.md

## Version Awareness

Each component's CAPABILITIES.md declares its current version. Each project's Stackfile.md pins which version it uses.

When updating a project to a newer stack version:
1. Run `stack-brain stale <project-dir>` to see what's outdated
2. Run `stack-brain dag` to determine update order
3. For each stale component, run `stack-brain migrations <component> <from> <to>` to get migration steps
4. Apply migration steps — the component docs tell you what to do, you coordinate and execute
5. Update the project's Stackfile.md with new version, ref, and date

## DAG-Aware Updates

Stack components depend on each other. Always update in topological order (leaves first). Run `stack-brain dag` to get the current tiers. Use `stack-brain dag --downstream-of <component>` to scope updates.

## Gap Reporting

When you identify a capability gap (something needed that no stack component covers), log it to ~/newstack/brain/STACK_GAPS.md with: what was needed, which project needed it, what stopgap was used, and the date.

## Architectural Constraints (CONSTRAINTS.md)

Projects and components can have a CONSTRAINTS.md declaring enforceable architectural rules. This follows the same router pattern as CAPABILITIES.md — project-level is the entry point, and it can either define rules inline or point to component-level constraints.

### How it works

1. **Project-level CONSTRAINTS.md** — the file you read first. Contains project-specific rules and pointers to component constraints.
2. **Component-level CONSTRAINTS.md** — optional. When a constraint applies to all users of a component, it lives here instead of being duplicated across projects.
3. **Router pattern** — a project constraint can delegate: `See oneauth/CONSTRAINTS.md: no-direct-jwt`. The project file is always the entry point; component files are reached through it.

### Constraint format

```markdown
### {Short Rule Name}
**Rule**: {What must or must not happen}
**Why**: {The incident or reasoning behind this rule}
**Verify**: {grep pattern, test command, lint rule, or "manual"}
**Scope**: {project-wide | specific files/dirs | specific component usage}
```

### When to create constraints

- When the user corrects an architectural direction mid-session → ask: "Should this become a constraint?"
- During `/checkpoint` → review any corrections from the session, update CONSTRAINTS.md
- During `/stack-audit` → validate all constraints (highest priority finding category)

### Promotion path

A constraint starts in a project. If the same rule appears in 2+ projects, promote it to the component's CONSTRAINTS.md and replace project entries with pointers.

## Component Conventions

Each stack component should have:
- **CAPABILITIES.md** — what it provides, version, integration, stack dependencies, location (read this to decide whether to use it)
- **CONSTRAINTS.md** — optional; architectural rules for how this component must be used (read this during audit and before completing work)
- **CLAUDE.md** — how to work on it, build/test commands, coding conventions (read this when developing or integrating)
- **CHANGELOG.md** — version history (read this to understand what changed between versions)
- **migrations/** — actionable migration steps per version bump (read this when upgrading a project)

Components may contain **sub-components** (e.g., a TypeScript client embedded in a Go repo). These are declared as sub-entries in the parent's CAPABILITIES.md with their own location and module path.

## Cross-Cutting Conventions

These apply to all stack projects, not just a single component. Skills like `/stack-audit` and `/stack-integrate` should check these.

### Go CLI Release
- GoReleaser v2 for binary releases (use Context7 for current syntax if needed)
- Version via ldflags: `-X {module}/cmd.Version={{.Version}}`
- Platforms: linux/darwin/windows × amd64/arm64, `CGO_ENABLED=0`, `-trimpath`
- CI: GitHub Actions — test on push/PR to main, release on `v*` tag push

### Go Module Hygiene
- Pre-push hook must run `make test` and fail if `go.mod` has uncommented `replace.*locallinks` (local dev symlinks must not leak to remote)
- `go.mod` replace directives for local dev should be commented out before push

## Integration Pattern

Go projects use replace directives for local development, pointing to wherever the component lives:
```go
replace github.com/panyam/{name} => {component-location}
```

Some projects use a `locallinks/` symlink directory to abstract the paths. Either pattern works — the component's CAPABILITIES.md documents which to use.

## Setup

```bash
cd ~/newstack/brain && make setup
```

This installs the CLI, skills, and global CLAUDE.md pointer. Re-run after updates.
