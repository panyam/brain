# Stack Catalog Refresh

Regenerate ~/newstack/brain/STACK_CATALOG.md by reading all CAPABILITIES.md files across the stack.

## Steps

1. **Find all CAPABILITIES.md files**: Search for CAPABILITIES.md in:
   - ~/newstack/*/CAPABILITIES.md (flat repos)
   - ~/newstack/*/main/CAPABILITIES.md (main-branch worktrees)
   - ~/newstack/*/master/CAPABILITIES.md (master-branch worktrees)

2. **Parse each file**: Extract:
   - Component name (from # heading)
   - Version
   - Provides (capability tags)
   - Module path
   - Location
   - Stack dependencies
   - Status

3. **Rebuild the Capability Index table**: Sort by status (Mature → Active → Stable → Basic → Planning).

4. **Rebuild the Dependency Graph**: From the Stack Dependencies fields, construct the DAG.

5. **Rebuild the Topological Update Order**: Compute tiers from the DAG (leaves = Tier 0, then breadth-first).

6. **Rebuild the Capability Tags section**: Aggregate all capability tags for searchability.

7. **Write ~/newstack/brain/STACK_CATALOG.md**: Overwrite with the regenerated content. Keep the same format.

8. **Report**: List any changes (new components, removed components, version bumps, dependency changes).
