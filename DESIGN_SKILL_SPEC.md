# DESIGN.md Skill — Specification

A staleness-resistant project-understanding artifact, maintained by a skill that runs (a) on a schedule and (c) on demand, with cheap drift detection wired into `/start_pr`.

## Goal

Capture the parts of a project that **do not survive rederivation by grep**: component purpose, the *why* behind each named entity, declared relationships between entities, and the bindings between entities and architectural rules. Make this artifact (i) per-folder so blast radius of any single update is small, (ii) machine-checkable so staleness is detectable, (iii) consumed by other skills (`/start_pr`, `/checkpoint`, plan mode) so being out of date has visible consequences.

## Non-goals

- Capturing every method or call edge. That rots fastest, adds least value, can be re-derived.
- Replacing `ARCHITECTURE.md` / `README.md` / `SUMMARY.md`. DESIGN.md is per-folder; those remain top-level narrative.
- Replacing `CONSTRAINTS.md` / `CAPABILITIES.md`. Those stay, but reference entities from DESIGN.md.
- Generating from scratch every time. Rebuild is full but rare; in-between, edits are local.

## File layout

```
repo/
├── ARCHITECTURE.md          (existing; narrative)
├── CONSTRAINTS.md           (top-level; cross-folder rules; qualified entity refs)
├── CAPABILITIES.md          (optional; top-level; cross-folder capabilities)
├── pkg-a/
│   ├── DESIGN.md            ← per-folder artifact (new)
│   ├── CONSTRAINTS.md       (optional; local rules; bare entity refs)
│   └── CAPABILITIES.md      (optional; local; bare entity refs)
├── pkg-b/
│   └── DESIGN.md
└── ...
```

Rule of thumb for split:
- Constraint/capability entities all in one folder → that folder's file (bare refs).
- Entities span ≥2 folders → top-level file (qualified `folder/Entity` refs).
- Folder is a pass-through container (e.g., `cmd/`, `examples/`) → no DESIGN.md.

## File formats

### `pkg/DESIGN.md` (per-folder)

```markdown
---
package: demokit/notebook
purpose: Standalone cell-based TUI component — notebook owns a Bubble Tea program, cell store, and clipboard; CRUD safe from any goroutine
last_rebuilt: abc1234              # commit SHA at rebuild time
last_rebuilt_at: 2026-05-20T03:00Z # ISO timestamp (informational)
entities:
  - name: Notebook
    kind: struct
    role: Aggregate root; owns store, program, clipboard, rendezvous, keymap
    why: One coherent lifecycle (New → CRUD → Run → Stop) for embedded TUI use
  - name: store
    kind: struct (unexported)
    role: RWMutex-guarded cell list, cursor, header
    why: Single mutex covers everything mutated from >1 goroutine; viewport/width/height live on the model without a lock
  - name: Stream
    kind: method on Notebook
    role: Returns io.Writer over a cell's OutputBuffer; io.Discard if missing
    why: Callers never have to nil-check; mid-flight Remove silently drops chunks
  - name: outputBuffered
    kind: interface (unexported)
    role: Optional capability — cells that expose a streamable OutputBuffer
    why: Lets Stream stay polymorphic without forcing every cell to implement it
depends_on:
  - path: ../events
    entities: [Emit]                  # used for repaint nudges
  - path: github.com/charmbracelet/bubbletea
    entities: [Program, Model]        # external dep; documented for the reader
constraints_ref: ./CONSTRAINTS.md     # optional pointer
---

# notebook

Free-form prose. Use this section for narrative that doesn't fit cleanly into entity records — design history, surprising tradeoffs, things a reader should know before grepping. Keep it short; entities carry most of the weight.
```

### `pkg/CONSTRAINTS.md` (per-folder; entity refs are bare)

```markdown
---
constraints:
  - id: NB-C-001
    entities: [Notebook, store]
    rule: store mutations only via Notebook CRUD methods — never reach into store from outside the package
    verify: "! grep -rE 'notebook\\.(store|cells)' --include='*.go' | grep -v 'notebook/'"
  - id: NB-C-002
    entities: [Stream, outputBuffered]
    rule: Stream MUST return io.Discard (never nil) for missing cells or cells that don't implement outputBuffered
    verify: "grep -A5 'func.*Stream.*io.Writer' notebook/stream.go | grep -q io.Discard"
---

# Constraints — notebook

Prose elaboration if needed.
```

### Top-level `CONSTRAINTS.md` (entity refs are qualified)

```markdown
---
constraints:
  - id: TOP-C-001
    entities: [notebook/Notebook, events/Emit]
    rule: Notebook may emit via events.Emit but MUST NOT import internal event types directly
    verify: "! grep -E 'events\\.(internal|event[A-Z])' notebook/*.go"
---
```

`CAPABILITIES.md` mirrors this structure with `capabilities:` instead of `constraints:` and a positive phrasing (`role:` / `provides:`).

## Algorithms

### Full rebuild (scheduled or on-demand)

Three passes. Each pass parallelizes across folders via subagents to keep the orchestrator context clean.

```
orchestrator:
  folders = discover_folders(repo)  # exclude pass-through containers, vendor/, .git/
  head_sha = git rev-parse HEAD

  # Pass 1: per-folder DESIGN.md (parallel)
  for folder in folders in parallel:
    spawn subagent rebuild_folder(folder, head_sha)
    # subagent reads ONLY files in folder, writes folder/DESIGN.md frontmatter + prose
    # entities, purpose, role, why; depends_on left empty in pass 1

  # Pass 2: cross-folder linking (parallel)
  for folder in folders in parallel:
    spawn subagent link_folder(folder)
    # subagent reads its own DESIGN.md + grep results for imports/calls into sibling folders
    # writes depends_on block referencing actual entities in sibling DESIGN.mds

  # Pass 3: top-level verification (single pass; cheap)
  for file in [CONSTRAINTS.md, CAPABILITIES.md] at repo root:
    for entry in file.entries:
      for ref in entry.entities:
        verify ref resolves to a real entity in the named folder's DESIGN.md
    report unresolved refs

  # Pass 4 (optional): promote/demote suggestions
  scan all CONSTRAINTS/CAPABILITIES files; report:
    - top-level entries where all entity refs resolve to one folder ("consider demoting")
    - per-folder entries that grep matches in other folders too ("consider promoting")
```

Subagent boundaries: each subagent gets one folder's files in its context, returns the file contents to write. Orchestrator never reads folder code itself.

### Targeted rebuild (single folder)

Same three passes but scoped to one folder + its transitive depends_on chain. Used by `/start_pr` when drift is detected on touched folders.

### Drift check (start_pr hook)

```bash
# For each folder touched by the PR diff:
folder_design=$(folder)/DESIGN.md
[ -f "$folder_design" ] || { echo "no DESIGN.md in $folder"; continue; }

last=$(yq '.last_rebuilt' "$folder_design")

# Is the rebuild commit still in our history?
git merge-base --is-ancestor "$last" HEAD || drift=true

# Has anything in the folder changed since the rebuild?
[ -n "$(git diff --name-only $last HEAD -- $folder/)" ] && drift=true

if drift; then
  echo "drift in $folder — DESIGN.md last rebuilt at $last, behind HEAD"
fi
```

Policy when drift is detected:
- Touched folder (PR modifies files in it): trigger targeted rebuild subagent before generating the plan/reviewer guide.
- Untouched folder: warn-and-proceed; let the next scheduled rebuild handle it.

### Bootstrap (one-time migration of existing CONSTRAINTS/CAPABILITIES)

Existing CONSTRAINTS.md files have prose rules with no structured `entities:` field. Bootstrap:

```
for each existing CONSTRAINTS.md or CAPABILITIES.md:
  for each rule (parsed from current bullet/heading format):
    candidates = grep matching entity names from DESIGN.md against rule text
    write structured entry with entities: [candidates], rule: <prose>
    mark with `bootstrap_review: true` so the human knows to confirm
emit a report listing all bootstrap_review entries for human pass
```

After human review removes `bootstrap_review: true` flags, the binding is explicit and self-policing thereafter.

## Triggers

### (a) Scheduled remote rebuild

Use `/schedule` to register a nightly remote routine:
- Routine: `clone repo → run /design-rebuild → if changes, open PR or push to a branch`
- Cadence: nightly is fine for active repos; weekly for slow-moving ones.
- Requires the repo to be on GitHub (or another git host the remote agent can reach).

### (c) Manual on-demand

A skill invocation, e.g. `/design-rebuild [--folder <path>]`:
- No args: full rebuild.
- `--folder X`: targeted rebuild of X + its depends_on chain.
- Runs locally; uses subagents for per-folder work; reports a summary to the user.

### (b) Commit-count threshold

Not implemented in MVP. Can be added later via a git post-commit hook that pings the skill at every Nth commit. Skip unless nightly proves too coarse.

## Integration

### `/start_pr`

1. Identify folders touched by the planned change (or current diff).
2. Run drift check on those folders.
3. If drift: trigger targeted rebuild subagent before continuing.
4. Load `DESIGN.md` entities for touched folders into the plan context.
5. Plan and reviewer guide reference entities by name (e.g., "extends `Notebook.Stream` to honor a new `outputBuffered` variant") — no free-form prose for components that are already named in DESIGN.md.

### `/checkpoint`

Same drift check across all folders touched since the last checkpoint, not just one PR. Surfaces orphan files (touched but claimed by no DESIGN.md) and orphan symbols (newly exported, not modeled) for inclusion in the next rebuild.

### Plan mode

Auto-load `DESIGN.md` files for the folders the planning task touches. Plan output uses entity names. This makes plans more concrete and faster to write (entities are pre-named, relationships pre-mapped).

## Open questions / Phase 2

- **Promote/demote auto-suggest** (pass 4 above): include in MVP or defer? Recommend: include as a *report only*, never auto-rewrite.
- **Cross-repo references** (e.g., demokit's notebook used by another repo): out of scope for v1.
- **CAPABILITIES.md** at top level: leave to emerge; don't force-create.
- **Language coverage**: spec is language-agnostic but the pass-1 prompt for the subagent will need per-language hints (Go: exported symbols, interfaces; Python: module-level defs, classes; etc.). Start with Go.
- **Storage of `last_rebuilt`**: commit SHA in frontmatter is simplest. If the SHA falls out of history (force-push, squash), drift check returns "drift" — acceptable failure mode, just rebuild.

## What gets committed vs. derived

- Committed: per-folder `DESIGN.md`, per-folder `CONSTRAINTS.md`/`CAPABILITIES.md`, top-level `CONSTRAINTS.md`/`CAPABILITIES.md`.
- Not committed: any cache, hash file, or generated index. The frontmatter SHA is the only persistent freshness marker.

## MVP scope

1. The skill itself: `/design-rebuild` (manual trigger), full and targeted modes, subagent-based passes 1–3.
2. The drift-check subroutine (callable from any skill).
3. Bootstrap mode for existing CONSTRAINTS.md (Phase 1.5 if existing files need migration).
4. `/start_pr` hook wiring (Phase 2).
5. Schedule registration (Phase 2).
6. Promote/demote suggestion pass (Phase 3).
