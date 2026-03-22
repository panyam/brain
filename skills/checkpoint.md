Sync learnings from this session into the project's checked-in documentation.

## Steps

1. **Discover project docs**: Find CLAUDE.md and any documentation files (.md) in the project root and one level deep. Note which ones exist — don't create new doc files unless explicitly needed.

2. **Read current state**: Read CLAUDE.md (if it exists), MEMORY.md (auto-memory), and all discovered project docs.

3. **Identify new learnings from this session**: Look at what was discovered, built, or changed. Focus on:
   - New conventions or patterns established
   - Gotchas and operational lessons (things that broke and why)
   - New env vars, commands, scripts, or configuration
   - Bug fixes that reveal something non-obvious
   - Architectural decisions made

4. **Update or create CLAUDE.md**:
   - If it exists, add new learnings and remove stale/wrong entries
   - If it doesn't exist, create it with the project's key conventions
   - Keep it concise — a quick-reference router, not a knowledge dump
   - Link to other project docs for detailed explanations (e.g., "See ARCHITECTURE.md for...")
   - Prioritize: build/test commands, checklists, env vars, gotchas
   - Structure sections to match what the project actually needs

5. **Update other project docs per the checkpoint procedure in global CLAUDE.md** (SUMMARY.md, NEXTSTEPS.md, ROADMAP.md, ARCHITECTURE.md, guides) — only the ones that exist or are clearly needed.

6. **Update stack artifacts** (if this project or component uses the stack):
   - If a **Stackfile.md** exists: verify component versions still match go.mod/package.json, update if drifted
   - If a **CAPABILITIES.md** exists (i.e., this IS a stack component): update it with any new capabilities, version bumps, or convention changes from this session
   - If a stack component's conventions or API changed: check if a migration note is needed in the component's migrations/ directory
   - If a new stack dependency was added during this session: add it to Stackfile.md
   - If a capability gap was identified: log it to ~/newstack/brain/STACK_GAPS.md

7. **Prune MEMORY.md**: Remove entries now captured in checked-in docs. MEMORY.md should only keep working notes too granular or ephemeral for project docs.

8. **Commit**: Stage all changed .md files and commit with message "Checkpoint: sync learnings to project docs"

## Principles
- CLAUDE.md should be useful to any contributor (human or AI) on day one
- Prefer linking to existing docs over duplicating content
- Gotchas and checklists are the highest-value content — if something bit us, it belongs here
- Keep MEMORY.md lean — scratch space, not permanent docs
- Don't create doc files that don't already exist unless there's a clear need
- Stack artifacts (Stackfile.md, CAPABILITIES.md) are part of the project's truth — keep them in sync
