# Stack Update

Update a project's stack dependencies to latest versions, applying migrations. Can also be run from ~/newstack to cascade updates through inter-stack dependencies.

## Steps

1. **Read the brain**: Read ~/newstack/brain/STACK_CATALOG.md and ~/newstack/brain/CLAUDE.md for the dependency graph and update order.

2. **Determine scope**:
   - If run from a project dir (~/projects/*): update this project's stack deps
   - If run from ~/newstack or ~/newstack/{component}: update inter-stack deps, optionally cascade to projects
   - If $ARGUMENTS names a component: update everything downstream of that component

3. **Read the Stackfile**: Read the project's Stackfile.md (or go.mod/package.json if no Stackfile exists yet — create one).

4. **Build the update plan**: For each stack component used by the project:
   a. Read the component's CAPABILITIES.md for its current version
   b. Check the git HEAD in the component's worktree: `git -C {component-path} rev-parse --short HEAD`
   c. Compare against the version/ref pinned in Stackfile.md
   d. If stale, add to the update list

5. **Topological sort**: Order updates by the dependency tiers in STACK_CATALOG.md. Never update a component before its dependencies are updated.

6. **For each component to update (in order)**:
   a. Read the component's CAPABILITIES.md Migrations section
   b. If there are migration files between the old and new version, read them all in order (concatenate)
   c. Update the go.mod replace directive or package.json version
   d. Run `go mod tidy` (Go) or `npm install` (TS)
   e. Apply any breaking changes described in the migration docs
   f. Flag any convention changes to the user

7. **Update Stackfile.md**: Record new version, git ref, and date for each updated component.

8. **Summary**: Report what was updated, what migrations were applied, and anything needing manual attention.

## Arguments
- $ARGUMENTS: Optional scope — a component name to update downstream of, or "all" for everything
