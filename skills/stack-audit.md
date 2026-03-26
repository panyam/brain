# Stack Audit

Audit a project for idiomatic use of stack components — find duplicated functionality, missed capabilities, and convention drift. This is the deep analysis complement to `stack-brain stale` (which only checks versions).

## Steps

1. **Identify stack components in use**: Run `stack-brain stale .` to get the list of stack components this project depends on (ignore staleness for now, focus on which components are present).

2. **For each component in use**, read its CAPABILITIES.md and CLAUDE.md from the location shown in the stale output. Focus on:
   - What capabilities it provides (the "Provides" section)
   - What conventions/patterns it expects (the "Conventions" section)
   - Any integration patterns in CLAUDE.md

3. **Check cross-cutting conventions**: Read the "Cross-Cutting Conventions" section of ~/newstack/brain/CLAUDE.md. For each convention that applies to this project type (e.g., "Go CLI Release" for Go projects with a main.go), check whether the project follows it. Report drift under Convention Drift.

4. **Check constraints**: If the project has a CONSTRAINTS.md, validate each constraint:
   - For constraints with a `Verify` that specifies a grep pattern or command, run it and check for violations
   - For constraints with `Verify: manual`, flag them as needing human review
   - For constraints that reference a component (e.g., "See oneauth/CONSTRAINTS.md: {rule}"), read and check those too
   - Report violations in a dedicated **Constraint Violations** section (before other findings, since these represent the user's explicit architectural decisions)

5. **Search the project for drift and duplication**: For each capability the component provides, grep the project codebase for patterns that suggest the project is:
   - **Duplicating**: Hand-rolling functionality the component already provides (e.g., writing custom auth middleware when oneauth has it, manual HTMX detection when goapplib has WithHtmx)
   - **Bypassing**: Using a third-party lib for something the stack covers (e.g., using a different JWT library when oneauth handles JWT)
   - **Using old patterns**: Using deprecated APIs or old conventions when newer ones exist in the current version
   - **Missing capabilities**: Not using a capability that would simplify the code (e.g., not using codec system when manually serializing WebSocket messages)

6. **Check for stack gaps**: Scan the project's go.mod/package.json for third-party dependencies. For each one, run `stack-brain lookup <keywords>` to check if a stack component covers that need. If it does, flag it as a potential replacement.

7. **Report findings**: Organize into categories (constraint violations first):

   ### Constraint Violations
   - **{rule name}**: {what was found} at {file:line} — violates: "{rule text}"

   ### Duplicated Functionality
   - {file:line}: {what it does} → already provided by {component}.{capability}

   ### Missed Capabilities
   - {component} provides {capability} which could replace {current approach in file:line}

   ### Convention Drift
   - {file:line}: uses {old pattern} → {component} convention is {new pattern}

   ### Third-Party Replacements
   - {third-party dep}: could be replaced by {stack component} ({capability})

   ### Stack Gaps (no component covers this)
   - {third-party dep}: {what it does} — candidate for new stack component?

8. **Prioritize**: Order findings by impact:
   - **Critical**: Constraint violations (explicit architectural rules broken)
   - **High**: Duplicated core functionality
   - **Medium**: Missed capabilities that would simplify code
   - **Low**: Convention drift that's cosmetic

9. **Do NOT auto-fix**: Present findings to the user. They decide what to address. This is an audit, not a migration.

## Arguments
- $ARGUMENTS: Optional — component name to audit against (e.g., "oneauth" to only check auth-related patterns). If omitted, audits against all components in use.
