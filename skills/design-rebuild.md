Rebuild per-folder DESIGN.md artifacts in a repo. Manual trigger; run at the repo root for a full rebuild, or with `--folder <path>` for a single folder.

See `~/newstack/brain/DESIGN_SKILL_SPEC.md` for the system-level spec (file formats, why per-folder, additive constraint binding). This skill is the executor.

## When to use

- After a refactor touched more than one folder.
- When `/design-drift-check` reports a folder as stale.
- Periodically (nightly via `/schedule`, weekly manually) as the baseline freshness defense.
- Before opening a multi-folder PR so the reviewer guide can reference accurate entity names.

## Modes

- **Full rebuild** (no args, run at repo root): rebuild every folder that should have a DESIGN.md.
- **Targeted** (`--folder <path>`): rebuild just that folder. MVP scope — does not also re-walk the depends_on chain; the neighbor DESIGN.mds are read as-is. If a neighbor is also stale, run `/design-drift-check` to find out and rebuild it separately.

## Which folders get a DESIGN.md

Include a folder if it has more than ~3 source files and at least one entity worth naming (struct, interface, exported function with a non-trivial role). Skip:
- Pass-through containers where each subfolder is its own concern (e.g. `cmd/`, `examples/`).
- Generated code, `vendor/`, `node_modules/`, `.git/`, `dist/`, `build/`.
- Folders that are just data or fixtures.

If a folder is borderline, ask the user before generating one for it. Better to have fewer real DESIGN.mds than many shallow ones.

## Algorithm

### Pass 0 — orchestrator prep

```bash
head_sha=$(git rev-parse HEAD)
```

Every DESIGN.md written in this rebuild carries the same `last_rebuilt: <head_sha>` so drift detection has a consistent anchor across the whole pass.

### Pass 1 — per-folder DESIGN.md generation (parallel subagents)

For each candidate folder, spawn one `Agent` with `subagent_type=general-purpose` and a prompt of roughly:

> Read every source file in `<folder>` (do NOT follow imports to sibling folders or descend into examples/test fixtures). Identify the package's purpose in one short sentence. List the entities worth naming — structs, interfaces, exported types, key methods. For each entity, give:
> - `name` and `kind`
> - `role` — one line: what it is in the package
> - `why` — one line: the design rationale, the constraint it enforces, or the gotcha it documents. Avoid restating what the code does; favour the non-obvious why.
>
> Write `<folder>/DESIGN.md` with YAML frontmatter (`package`, `purpose`, `last_rebuilt: <head_sha>`, `entities:` list) and a short prose body for anything that doesn't fit in entity records. Leave `depends_on:` empty — Pass 2 fills it. Do not commit. Do not read files outside `<folder>`.

Spawn all per-folder subagents in a single message so they run concurrently. The orchestrator never reads the folder's code itself — keeps the main context clean and the rebuild parallelizable.

### Pass 2 — cross-folder linking (parallel subagents)

After every Pass 1 subagent returns, spawn one `Agent` (`subagent_type=general-purpose`) per folder with a prompt of:

> Read `<folder>/DESIGN.md`. Grep `<folder>` for imports and call sites that target sibling folders in the repo (e.g. `<repo>/events`, `<repo>/state`). For each sibling used, read that sibling's `DESIGN.md` to look up the entities being referenced. Update `<folder>/DESIGN.md`'s `depends_on:` block with one entry per sibling folder listing the entities used. Touch only the `depends_on:` field. Do not commit.

Again, parallel; orchestrator coordinates only.

### Pass 3 — top-level constraint verification (single orchestrator pass)

Walk repo-root `CONSTRAINTS.md` and `CAPABILITIES.md` if they exist. For each rule that has an `**Entities**:` line (the additive binding):

- Parse the comma-separated qualified refs (`folder/Entity` form).
- For each ref, read the named folder's `DESIGN.md` and verify the entity name exists in its `entities:` list.
- Collect unresolved refs.

Report unresolved refs to the user. Do NOT auto-fix — the rule may need updating, or the entity may have been renamed and the human has to decide.

Rules with no `**Entities**:` line are fine; the additive format is opt-in. Do not insert `**Entities**:` lines automatically — the human declares those deliberately, and the bootstrap flow that infers them is deferred.

## Output

End the skill with a structured summary, then a one-line nudge about review:

```
Rebuilt: N folders
  - path/to/folder-a (12 entities)
  - path/to/folder-b (5 entities)
Skipped: M folders
  - path/to/cmd (pass-through container)
  - path/to/vendor (excluded)
Cross-folder relationships recorded: K
  - folder-a → events (Emit)
  - folder-a → state (State)
Top-level refs verified: P pass, Q fail
  FAIL: CONSTRAINTS.md "Notebook events isolation" — events/InternalType not found in events/DESIGN.md

Review with `git diff` and commit (or `git restore`) per folder as you see fit. This skill never commits.
```

## Principles

- One folder per subagent. Orchestrator never reads folder code. Subagents must not read across folder boundaries during Pass 1 — pollutes the model with assumptions.
- All DESIGN.mds in a single rebuild share one `last_rebuilt` SHA. Drift detection depends on this consistency.
- Never auto-commit. Never auto-branch. Never auto-add `**Entities**:` lines to existing CONSTRAINTS.md / CAPABILITIES.md rules — humans declare those bindings.
- The why beats the what. Entity roles and rationale beat method lists. If a fact is obvious from the code, don't repeat it.
- A folder that doesn't earn a DESIGN.md (too small, too mechanical, too pass-through) shouldn't have one. Less is more.
- When in doubt about whether a folder qualifies, ask the user before writing the file.
