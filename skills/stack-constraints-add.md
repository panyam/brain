# Stack Constraints Add

Add a new architectural constraint to the current project's CONSTRAINTS.md.

## Steps

1. **Locate CONSTRAINTS.md**: Look for CONSTRAINTS.md in the project root (current directory or $ARGUMENTS path). If it doesn't exist, create one from the template at ~/newstack/brain/templates/CONSTRAINTS.md.

2. **Gather the constraint**: Check if the conversation contains a recent architectural correction (user said "no", "don't", "stop", or redirected an approach). If so, pre-fill the constraint from that context and confirm with the user. If no session context is available, ask the user to describe:
   - What must or must not happen (the rule)
   - Why (the incident or reasoning)
   - How to verify (grep pattern, test command, or "manual")
   - Scope (project-wide, specific dirs, or specific component usage)

3. **Check for duplicates**: Read the existing CONSTRAINTS.md and check if a similar rule already exists. If so, ask whether to update the existing rule or add a new one.

4. **Draft the constraint** in the standard format:
   ```markdown
   ### {Short Rule Name}
   **Rule**: {What must or must not happen}
   **Why**: {The incident or reasoning behind this rule}
   **Verify**: {grep pattern, test command, lint rule, or "manual"}
   **Scope**: {project-wide | specific files/dirs | specific component usage}
   ```

5. **Test the verify pattern**: If the verify field is a grep or command (not "manual"), run it against the codebase and report what it finds. This catches both existing violations and false positives in the pattern. Refine the pattern with the user if needed.

6. **Append to CONSTRAINTS.md**: Add the constraint under the `## Constraints` section.

7. **Check for promotion candidate**: If this rule is about how a specific stack component should be used (e.g., oneauth, goapplib), add a comment: `<!-- Candidate for promotion to {component}/CONSTRAINTS.md if seen in other projects -->`.

8. **Report**: Show the added constraint and any existing violations found during the verify test.

## Arguments
- $ARGUMENTS: Optional — a short description of the constraint (e.g., "no direct database queries from handlers"). If omitted, the skill will prompt interactively or derive from session context.
