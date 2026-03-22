# Stack Integrate

Onboard a new component (or sub-component) into the stack system. Can be run from any directory containing a Go module or npm package — not limited to ~/newstack.

## Steps

1. **Identify the component root**: Look for go.mod or package.json in the current directory (or $ARGUMENTS path if provided). This is the component root.

2. **Check for sub-components**: Scan for nested go.mod/package.json in subdirectories (e.g., `ts/`, `client/`, `web/`). These may be embedded sub-libraries that should be declared as sub-capabilities of the parent. Ask the user if any should be registered.

3. **Read the component's code and docs**: Read go.mod/package.json, any existing CLAUDE.md, README.md, SUMMARY.md, ARCHITECTURE.md. Understand what this component does.

4. **Identify stack dependencies**: Check go.mod for `github.com/panyam/*` imports or package.json for `@panyam/*` dependencies. Cross-reference against ~/newstack/brain/STACK_CATALOG.md to map them to known stack components.

5. **Draft CAPABILITIES.md**: Using the template at ~/newstack/brain/templates/CAPABILITIES.md, create a CAPABILITIES.md in the component root. Include:
   - Version (from go.mod module version, package.json version, or git tags)
   - Provides (capability tags with one-line descriptions)
   - Module path
   - Location (absolute path — no assumption about ~/newstack)
   - Stack dependencies
   - Integration snippet (go.mod require + replace, or npm install)
   - Status
   - Conventions
   - **Sub-components** (if any): for each embedded sub-library, add a sub-entry under Provides:
     ```
     - **sub-component-name** (module-or-package-name): description
       - location: {path to sub-component}
       - module: {sub-component module/package name}
     ```

6. **Confirm with user**: Show the drafted CAPABILITIES.md and ask the user to confirm or refine the capability descriptions and sub-component entries.

7. **Ensure CLAUDE.md exists**: If the component doesn't have a CLAUDE.md, draft one with build/test commands and coding conventions based on the codebase. Ask user to confirm.

8. **Update STACK_CATALOG.md**: Run `/stack-catalog-refresh` or manually add the new entry to ~/newstack/brain/STACK_CATALOG.md:
   - Add a row to the Capability Index table (and sub-rows for sub-components)
   - Add the component to the Dependency Graph
   - Update the Topological Update Order tiers

9. **Report**: Tell the user what was added, where it fits in the dependency graph, and list any sub-components registered.

## Arguments
- $ARGUMENTS: Optional — a path to the component directory, and/or a capability description (e.g., "/path/to/mylib for distributed task queue with retry semantics"). If omitted, uses current working directory.
