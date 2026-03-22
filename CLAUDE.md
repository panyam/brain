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

## Component Conventions

Each stack component should have:
- **CAPABILITIES.md** — what it provides, version, integration, stack dependencies, location (read this to decide whether to use it)
- **CLAUDE.md** — how to work on it, build/test commands, coding conventions (read this when developing or integrating)
- **CHANGELOG.md** — version history (read this to understand what changed between versions)
- **migrations/** — actionable migration steps per version bump (read this when upgrading a project)

Components may contain **sub-components** (e.g., a TypeScript client embedded in a Go repo). These are declared as sub-entries in the parent's CAPABILITIES.md with their own location and module path.

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
