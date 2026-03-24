# Stack Constraints Check

Validate a project's code against its architectural constraints. Runs mechanical checks (grep patterns) and semantic review (LLM judgment) in a single pass. Designed to run in a parallel session without needing context from the current session.

## Steps

1. **Load constraints**: Read the project's CONSTRAINTS.md. If it doesn't exist, report "No constraints found" and stop. For any constraint that points to a component (`See {component}/CONSTRAINTS.md: {rule}`), read that component's file too.

2. **Mechanical pass** — for each constraint with a grep/command in the Verify field:
   - Run the verify command
   - Record matches as **definite violations** (the pattern was explicitly designed to catch these)
   - If the verify command finds nothing, mark as clean

3. **Semantic pass** — for each constraint (including those already checked mechanically):
   - Read the code files in the constraint's Scope
   - Look for violations that grep can't catch: wrong patterns, architectural drift, spirit-of-the-rule violations
   - For `verify: manual` constraints, this is the only check — give it extra attention
   - Classify findings as:
     - **Definite violation**: clearly breaks the stated rule
     - **Possible concern**: may violate the intent, needs human judgment
     - **Suggestion**: not a violation but the constraint could be stronger or the code could be cleaner

4. **Cross-constraint check**: Look for tensions between constraints or cases where fixing one violation would create another. Flag these explicitly.

5. **Report findings**:

   ```
   ## Constraint Violations

   ### {Constraint Name}
   **Status**: {X violations | clean}

   #### Definite Violations
   - {file:line}: {what's wrong} — violates: "{rule text}"

   #### Possible Concerns
   - {file:line}: {what looks off} — may conflict with intent: "{why text}"

   #### Suggestions
   - {observation about how the constraint could be strengthened or better verified}
   ```

   End with a summary:
   ```
   ## Summary
   - N constraints checked
   - N definite violations
   - N possible concerns
   - N constraints clean
   ```

6. **Do NOT auto-fix**: Present findings only. The user decides what to act on.

## Arguments
- $ARGUMENTS: Optional — a specific constraint name to check (e.g., "copyUnit" or "No Defensive Error Handling"). If omitted, checks all constraints.
