# Stack Constraints Overview

Scan all projects and components for CONSTRAINTS.md files and produce a portfolio-wide view of architectural rules. Session-independent — run this in a parallel terminal.

## Steps

1. **Scan for CONSTRAINTS.md files**: Search across ~/projects/, ~/work/, and ~/newstack/ for any CONSTRAINTS.md files. List all found files.

2. **Parse each file**: For each CONSTRAINTS.md, extract:
   - Constraint name (### heading)
   - Verify type (grep, test command, or manual)
   - Scope
   - Whether it's a pointer to a component constraint (`See ...`)
   - Whether it has a promotion comment (`<!-- Candidate for promotion ...`)

3. **Build the overview**:

   ```
   ## Portfolio Constraints Overview

   ### By Project
   | Project | Constraints | Grep-checkable | Manual | Promote candidates |
   |---------|------------|----------------|--------|-------------------|

   ### By Component (promoted)
   | Component | Constraints | Used by projects |
   |-----------|------------|-----------------|

   ### Duplicates (same rule in multiple projects)
   - "{rule name}" appears in: {project1}, {project2} → candidate for promotion to {component}

   ### Verify Coverage
   - Total constraints: N
   - Grep-checkable: N (automated)
   - Manual/philosophical: N (needs semantic review via /stack-constraints-check)
   - Promoted to component: N
   ```

4. **Identify hardening opportunities**: For each `verify: manual` constraint, assess whether any part could be made into a grep pattern, AST check, or test. Report these as:
   ```
   ### Hardening Candidates
   - {project}: "{constraint name}" — currently manual, could partially verify with: {suggested grep or test approach}
   ```

5. **Identify gaps**: Look for projects that use stack components but don't have constraints for common patterns (e.g., uses oneauth but has no auth constraint). Flag as:
   ```
   ### Missing Constraints
   - {project} uses {component} but has no constraint for {common pattern}
   ```

6. **Do NOT modify any files**: This is a read-only overview. Use `/stack-constraints-add` or `/stack-constraints-promote` to act on findings.

## Arguments
- $ARGUMENTS: Optional — "refresh" to also update ~/newstack/brain/CONSTRAINTS_CATALOG.md with the findings. If omitted, just prints the overview.
