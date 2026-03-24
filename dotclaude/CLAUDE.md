## Stack System
- Before adding third-party dependencies or building infrastructure from scratch, consult ~/newstack/brain/STACK_CATALOG.md
- For full stack rules, conventions, and update procedures, see ~/newstack/brain/CLAUDE.md

## Project Understanding
- Make sure to read ARCHITECTURE.md, SUMMARY.md, ROADMAP.md and NEXTSTEPS.md to understand the project, style, decisions and how we are thinking about the architecture and use that understanding when proposing changes. Some of these files may not exist.

## Architectural Constraints
- Projects may have a CONSTRAINTS.md at their root containing enforceable architectural rules
- Before completing work, validate against CONSTRAINTS.md if it exists
- When the user corrects an architectural direction mid-session, ask: "Should this become a constraint?"
- Constraints with a `Verify` field that specifies a grep/command should be checkable automatically
- To promote a recurring project constraint to a component: move it to that component's CONSTRAINTS.md and replace the project entry with a pointer
- `/stack-audit` checks constraints as its highest-priority finding category
- **Push back when a request would violate a constraint.** If the user asks you to do something that conflicts with a rule in CONSTRAINTS.md, flag it before proceeding:
  - Quote the specific constraint by name
  - Explain the conflict
  - Ask whether to proceed anyway (and if so, whether the constraint should be updated or removed)
  - Do NOT silently comply — the whole point of constraints is that they survive the user forgetting why the rule exists
- **Push back on architectural smell even without a constraint.** If you see a pattern that contradicts ARCHITECTURE.md or introduces coupling/complexity that the project's design was avoiding, say so. The user values being challenged when there's good reason. If the user agrees it was a bad direction, suggest capturing it as a constraint.

## Checkpoint
- Use the `/checkpoint` skill — it handles all doc updates, stack artifact sync, and memory pruning