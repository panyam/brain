# Stack Constraints Promote

Move a project-level constraint to a component's CONSTRAINTS.md so it applies to all consumers of that component.

## Steps

1. **Identify the constraint to promote**: Use $ARGUMENTS as the constraint name (or a substring). If not provided, read the current project's CONSTRAINTS.md and list all constraints that have a promotion comment (`<!-- Candidate for promotion to ...`). Ask the user which one to promote.

2. **Find duplicates across projects**: Search for CONSTRAINTS.md files across ~/projects/ and ~/work/ (and any other known project directories). For each one found, check if it contains a similar rule (same component, similar rule text). Report which projects have this constraint.

3. **Identify the target component**: From the constraint's scope or promotion comment, determine which stack component this belongs to. Verify the component exists by running `stack-brain lookup <component-name>`. Read the component's location from the catalog.

4. **Create or update component CONSTRAINTS.md**:
   - If the component already has a CONSTRAINTS.md, append the new rule
   - If not, create one from the template at ~/newstack/brain/templates/CONSTRAINTS.md
   - Adapt the verify pattern to work from the component consumer's perspective (the grep should check the consuming project, not the component itself)

5. **Update project CONSTRAINTS.md files**: For each project that had this constraint inline, replace the full constraint block with a pointer:
   ```markdown
   ### {Short Rule Name}
   See {component}/CONSTRAINTS.md: {rule name}
   ```

6. **Confirm with user**: Show the changes across all files before writing. The user approves or adjusts.

7. **Verify the pointer works**: Read the component's CONSTRAINTS.md from one of the updated project's perspective to confirm /stack-audit would find and check it.

8. **Report**: List what was promoted, where it now lives, and which projects were updated.

## Arguments
- $ARGUMENTS: Optional — the constraint name or substring to promote (e.g., "Auth Through OneAuth Only" or just "oneauth auth"). If omitted, lists promotion candidates from the current project.
