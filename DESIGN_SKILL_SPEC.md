# Per-Folder Documentation Skills — Specification

A staleness-resistant, language-native, per-folder documentation system: each folder's docs live in the **language-idiomatic doc file** (e.g. `doc.go` for Go), with sequence diagrams in a per-folder `diagrams.md`, and a `.design.yaml` sidecar holding machine-readable metadata. Maintained by per-language `/design-rebuild-<lang>` skills; freshness-checked by a language-agnostic `/design-drift-check`; collated into a top-level `MAP.md` by a language-agnostic `/design-map`.

## Goal

Capture the parts of a project that **do not survive rederivation by grep**: package purpose, the *why* behind each named entity, declared relationships between entities, and the bindings between entities and architectural rules. Make this artifact (i) per-folder so blast radius of any single update is small, (ii) **language-native** so readers (and LLMs) find it where they already look, (iii) machine-checkable via a structured sidecar so staleness is detectable, (iv) consumed by other skills (`/start_pr`, `/checkpoint`, plan mode) so being out of date has visible consequences.

## Non-goals

- Capturing every method or call edge. That rots fastest, adds least value, can be re-derived.
- Replacing `ARCHITECTURE.md` / `README.md` / `SUMMARY.md`. The per-folder artifacts are local; those remain top-level narrative.
- Replacing `CONSTRAINTS.md` / `CAPABILITIES.md`. Those stay, but reference entities from the per-folder sidecars.
- Generating from scratch every time. Rebuilds are full but rare; skip-if-fresh is the default.
- One artifact format across all languages. The whole point of the pivot is per-language idiomatic homes.

## Architectural principle: push variation to the boundary

Variation lives in *writing* documentation — Go's `doc.go` ≠ TypeScript's `README.md` + TSDoc ≠ Rust's `//!` ≠ Python's `__init__.py` docstring. Variation does NOT live in *reading* metadata. The `.design.yaml` schema is the same regardless of which language skill wrote it, so downstream tools stay language-agnostic.

| Skill | Language-specific? | Reads | Writes |
|---|---|---|---|
| `/design-rebuild-go` | yes | `*.go` source | `doc.go`, `diagrams.md`, `.design.yaml` |
| `/design-rebuild-ts` (future) | yes | `*.ts` / `*.tsx` | `README.md`, `diagrams.md`, `.design.yaml` |
| `/design-rebuild-<lang>` | yes | `*.<ext>` | language-native doc, `diagrams.md`, `.design.yaml` |
| `/design-drift-check` | **no** | `.design.yaml` | nothing (reports only) |
| `/design-map` | **no** | every `.design.yaml` in repo | `MAP.md` at repo root |

## Per-folder artifacts

Three files per qualifying folder. Each has one clearly-scoped job.

### 1. Language-native doc file (e.g. `doc.go`, `README.md`)

What the language ecosystem expects. Rendered natively by language tooling — `go doc`, `pkg.go.dev`, IDE hover, npm registry, etc. Readers and LLMs find it without knowing about this skill.

**Full skill ownership.** The doc file is rewritten from scratch on every rebuild — no markers, no header line, no "preserve human content" logic. The file's structure for Go is:

```go
// Package localauth provides form-based local username/password authentication.
//
// The package owns the HTTP layer for signup/login/verify/reset and the
// channel-linking helpers; storage interfaces are received via callback fields
// on LocalAuth, not embedded.
//
// ENTITIES
//
// LocalAuth — Central config-and-handler object holding all wiring.
// Deliberately a flat bag of optional fields so apps wire only what they use.
//
// LocalAuth.HandleSignup — Registration handler. Username reservation and
// verification-email failures are warned-not-fatal to avoid leaving
// half-created accounts.
//
// FLOWS
//
// See [diagrams.md](diagrams.md) for sequence diagrams of: signup, login,
// password reset.
package localauth
```

**Where each kind of documentation lives:**

| Kind of documentation | Where it lives | Owned by |
|---|---|---|
| Symbol-level godoc (types, funcs, methods) | The `.go` file containing the symbol | Human / agent (co-located with code) |
| Examples | `<pkg>_test.go` via `ExampleXxx` functions | Human / agent (idiomatic Go) |
| Package-level prose (intro + entity catalog + flow links) | `doc.go` | `/design-rebuild-go` skill (full file) |

The split exists because doc.go and source files have fundamentally different ownership boundaries. Source files contain code humans/agents wrote; the godoc comments next to each symbol live with the code's semantics and get reviewed alongside it in PRs. `doc.go` has no code — its only job is package-level prose — so the skill commandeering it cleanly is a net win.

**Why no markers.** An earlier iteration used `<!-- design:start --> ... <!-- design:end -->` HTML comments to scope the auto-section inside a shared doc.go. That leaked as visible literal text on pkgsite/pkg.go.dev (which doesn't filter HTML comments). Full file ownership eliminates the boundary entirely — nothing to mark, nothing to leak.

**First-run protection.** If a folder has an existing `doc.go` but no `.design.yaml` yet (i.e., this is the first time the skill has touched this folder), the skill prompts before overwriting. After the first successful rebuild, `.design.yaml` exists and signals "this folder is skill-managed" — subsequent runs overwrite freely.

### 2. `diagrams.md` (per-folder, optional)

Mermaid `sequenceDiagram` blocks (or `flowchart` for decision trees) for multi-step interactions: signup flows, request lifecycles, handshakes, state machines. `###` headings serve as anchors so the doc file can link `[Signup flow](diagrams.md#signup)`.

- Created only when at least one flow is worth diagramming. Trivial 1–2-step flows do not earn a diagram.
- Rendered natively on GitHub when the reader follows the link from the doc file.
- pkg.go.dev (and most language doc renderers) do not render Mermaid; that's why diagrams live in a separate file and are linked rather than inlined.

### 3. `.design.yaml` (per-folder, always)

Machine-readable sidecar. The contract for `/design-drift-check` and `/design-map`. Schema:

```yaml
package: localauth                  # package identity (typically the folder basename)
purpose: Form-based local username/password authentication...   # one sentence
language: go                        # which language family
doc_file: doc.go                    # which file holds the rendered package doc
diagrams_file: diagrams.md          # OMIT this key entirely if no flows
last_rebuilt: <commit SHA>          # the freshness anchor — set to git rev-parse HEAD at rebuild time
last_rebuilt_at: 2026-05-22T...     # ISO 8601 UTC, informational
entities:
  - name: LocalAuth
    kind: struct
    role: Central config-and-handler object holding all wiring.
    why: Deliberately a flat bag of optional fields so apps wire only what they use.
  - name: LocalAuth.HandleSignup
    kind: method
    role: Registration handler.
    why: Username reservation and verification-email failures are warned-not-fatal to avoid leaving half-created accounts.
depends_on:                         # filled by Pass 2; cross-folder references
  - path: ../core
    entities: [User, IdentityStore, TokenStore, ...]
```

Notes on the schema:

- `language:` is informational + used by `/design-map` to populate the "Languages" line at the top of MAP.md. Not branched on by drift-check or map otherwise.
- `doc_file:` is **what `/design-map` links to** for that folder. Resolution stays language-agnostic — the map skill doesn't need to know that `doc.go` is special; it just follows whatever path the sidecar declares.
- `last_rebuilt:` is the only freshness signal. If it falls out of HEAD's history (force-push, squash), drift-check returns DRIFT and the next rebuild re-anchors it.
- `entities[].name` is what `**Entities**:` lines in `CONSTRAINTS.md` / `CAPABILITIES.md` resolve against. Stability matters — renaming an entity invalidates rule bindings; rebuilds need to be paired with constraint updates.

## File layout

```
repo/
├── ARCHITECTURE.md          (existing; narrative)
├── README.md                (existing)
├── MAP.md                   ← auto-generated by /design-map
├── CONSTRAINTS.md           (top-level; cross-folder rules; qualified entity refs)
├── CAPABILITIES.md          (optional; top-level; cross-folder capabilities)
├── pkg-a/
│   ├── doc.go               ← language-native doc, full skill ownership
│   ├── diagrams.md          ← Mermaid (only if flows exist)
│   ├── .design.yaml         ← machine-readable sidecar
│   ├── CONSTRAINTS.md       (optional; local rules; bare entity refs)
│   └── CAPABILITIES.md      (optional; local; bare entity refs)
├── pkg-b/
│   ├── doc.go
│   └── .design.yaml         ← no diagrams.md — no flows worth diagramming
└── ...
```

Rule of thumb for constraint split:
- Constraint/capability entities all in one folder → that folder's file (bare refs).
- Entities span ≥2 folders → top-level file (qualified `folder/Entity` refs).
- Folder is a pass-through container (e.g., `cmd/`, `examples/`) → no doc artifacts.

## File formats

### `pkg/CONSTRAINTS.md` (per-folder; entity refs are bare)

Keeps brain's existing prose format (`### Rule Name / **Rule** / **Why** / **Verify** / **Scope**`) and **adds an optional `**Entities**:` line** per rule binding to the folder's `.design.yaml` entities. Additive — existing CONSTRAINTS.md files continue to work untouched.

```markdown
# Constraints — notebook

### Store mutations via CRUD only
**Rule**: store mutations only via Notebook CRUD methods — never reach into the store from outside the package
**Why**: the single-mutex invariant requires all writers to go through Notebook; direct access bypasses lock ordering
**Verify**: `! grep -rE 'notebook\.(store|cells)' --include='*.go' | grep -v 'notebook/'`
**Scope**: notebook package
**Entities**: Notebook, store
```

### Top-level `CONSTRAINTS.md` (entity refs are qualified)

Same prose format; `**Entities**:` entries use `folder/Entity` form since they cross folder boundaries.

```markdown
# Constraints

### Notebook events isolation
**Rule**: Notebook may emit via events.Emit but MUST NOT import internal event types directly
**Why**: keeps the events package's internal types unexposed; emit is the public contract
**Verify**: `! grep -E 'events\.(internal|event[A-Z])' notebook/*.go`
**Scope**: cross-folder (notebook → events)
**Entities**: notebook/Notebook, events/Emit
```

`CAPABILITIES.md` mirrors this structure with a positive framing — `**Provides**` and `**Used by**` lines in addition to the standard sections — and the same optional `**Entities**:` binding.

## Algorithms

### Full rebuild (`/design-rebuild-<lang>` invoked at repo root)

Three passes. Each pass parallelizes across folders via subagents to keep the orchestrator context clean.

```
orchestrator:
  folders = discover_folders(repo, language=<lang>)   # language-specific filter
  head_sha = git rev-parse HEAD
  prior_depends_on = snapshot every existing .design.yaml's depends_on field

  # Pass 0.5: drift triage
  rebuild_set, fresh_set = drift_triage(folders, --force?)

  # Pass 1: per-folder rebuild (parallel subagents)
  for folder in rebuild_set in parallel:
    spawn subagent that reads ONLY <folder>'s source
    subagent writes: <doc-file>, .design.yaml, optionally diagrams.md
    subagent rewrites <doc-file> from scratch (full ownership)

  # Pass 2: cross-folder linking (parallel subagents)
  pass_2_set = rebuild_set ∪ { f : prior_depends_on(f) ∩ rebuild_set ≠ ∅ }
  for folder in pass_2_set in parallel:
    spawn subagent that updates ONLY <folder>/.design.yaml's depends_on field

  # Pass 3: top-level constraint verification (orchestrator, no subagent)
  for file in [CONSTRAINTS.md, CAPABILITIES.md] at repo root:
    for rule in file:
      if rule has **Entities**: line:
        for ref in entities (qualified folder/Entity form):
          verify ref resolves against <folder>/.design.yaml's entities list
    report unresolved refs (do not auto-fix)
```

Subagent boundaries: each subagent gets one folder's files in its context. Orchestrator never reads folder code itself. Skip-if-fresh applies in Pass 1; Pass 2 narrows further based on `prior_depends_on`; Pass 3 is cheap and always runs.

### Targeted rebuild (`--folder <path>`)

Same passes, scoped to one folder. Does **not** re-walk the depends_on chain in the MVP — if a neighbor is also stale, run `/design-drift-check` and rebuild it separately. Future work may add chain-expansion.

### Drift check (`/design-drift-check`)

Pure git-only, language-agnostic. For each candidate folder F:

```bash
sidecar="$F/.design.yaml"
[ -f "$sidecar" ] || { echo "$F: NO MODEL"; continue; }

last=$(yq '.last_rebuilt' "$sidecar")    # or grep fallback
[ -z "$last" ] && { echo "$F: NO ANCHOR"; continue; }

git merge-base --is-ancestor "$last" HEAD || { echo "$F: DRIFT (last_rebuilt out of history)"; continue; }
[ -n "$(git diff --name-only "$last" HEAD -- "$F/")" ] && { echo "$F: DRIFT"; continue; }
echo "$F: fresh ($last)"
```

Exit code: 0 if all fresh / NO MODEL; 1 if any DRIFT or NO ANCHOR. Cheap enough to run on every PR.

### Map (`/design-map`)

1. Walk repo for every `.design.yaml`.
2. For each: parse `package`, `purpose`, `language`, `doc_file`, `diagrams_file`, `depends_on`.
3. Build dependency graph (nodes = folders, edges = internal `depends_on`).
4. Topologically sort for reading order; cycles get a dedicated section.
5. Render `MAP.md` at repo root with folder names linked to **`doc_file`** (relative), optional `· [diagrams]` link, "Languages: go (N), ts (M)" stat line, and a code-block dependency graph.

Always whole-repo; no partial maps. Idempotent — same sidecars in, byte-identical MAP.md out.

### Bootstrap (one-time enrichment of existing CONSTRAINTS/CAPABILITIES)

Existing `CONSTRAINTS.md` files use the prose format but have no `**Entities**:` line yet. Bootstrap **augments** them in place — no format conversion, just additive enrichment:

```
for each existing CONSTRAINTS.md or CAPABILITIES.md:
  for each rule heading (### Rule Name):
    candidates = grep matching entity names from sibling .design.yaml against the rule prose
    propose adding `**Entities**: <candidates>` under the existing **Verify** / **Scope** lines
    flag the proposal with `<!-- bootstrap_review -->` so the human knows to confirm
emit a report listing all proposed additions; do not auto-apply
```

After human review removes the marker comments and accepts/edits the entity list, the binding is explicit and self-policing thereafter. The rule body, format, and rest of the file are untouched.

## Migration from DESIGN.md (legacy single-file format)

Earlier iterations of this system used a single `DESIGN.md` per folder containing both human-readable body and YAML frontmatter. That format is **retired**. Migration:

1. Run the appropriate `/design-rebuild-<lang>` skill at the repo root with `--force`.
2. Skill writes the three new artifacts (`<doc-file>`, `diagrams.md`, `.design.yaml`) per qualifying folder. Existing `DESIGN.md` files are **not touched** by the skill.
3. User reviews `git diff`, then `git rm DESIGN.md` in each folder where the new artifacts now exist.
4. Run `/design-map` to refresh `MAP.md` against the new sidecar locations.

No automated migration script — the skill regenerates the content cleanly under the new format, and removing the legacy file is the user's deliberate act.

## Triggers

### (a) Scheduled remote rebuild

Use `/schedule` to register a nightly remote routine that runs the appropriate per-language rebuild and `/design-map`.

### (b) Manual on-demand

Direct skill invocation: `/design-rebuild-go [--folder <path>] [--force]` or any future `/design-rebuild-<lang>`.

### (c) Commit-count threshold (not in MVP)

Deferred. A git post-commit hook pinging the skill at every Nth commit. Add later if nightly proves too coarse.

## Integration

### `/start_pr` (Phase 2 — not in this PR)

1. Identify folders touched by the planned change.
2. Run `/design-drift-check` on those folders.
3. If drift: trigger targeted `/design-rebuild-<lang>` before continuing.
4. Load `.design.yaml` entities for touched folders into the plan context.
5. Plan and reviewer guide reference entities by name (e.g., "extends `Notebook.Stream` to honor a new `outputBuffered` variant").

### `/checkpoint`

Same drift check across all folders touched since the last checkpoint, not just one PR.

### Plan mode

Auto-load `.design.yaml` files for the folders the planning task touches. Plan output uses entity names from the sidecars.

## Open questions / Phase 2+

- **`/start_pr` integration** (Phase 2) — wire `/design-drift-check` into `/start_pr` so touched folders surface drift before planning.
- **`/schedule` integration** (Phase 2) — nightly routine for active repos.
- **Bootstrap mode** for existing `CONSTRAINTS.md` files (Phase 1.5).
- **`/design-rebuild-ts`** (Phase 1.5) — second language. Per-package `README.md` ownership model TBD: full ownership like Go's doc.go is cleanest, but a hand-written `README.md` is more common in TS than a hand-written `doc.go` is in Go, so a delineated convention may be needed. Decide during Phase 1.5 design. Links to `diagrams.md`; writes `.design.yaml` with `language: ts`.
- **Promote/demote auto-suggest** (Phase 3) — scan constraint files; surface rules whose entity refs all resolve to one folder ("consider demoting") or that grep-match across folders ("consider promoting").
- **pkg.go.dev / npm URL enrichment** in MAP.md — link to the public registry for each folder, in addition to the local doc file.
- **Polyglot folders** with multiple languages — MVP assumes one language per folder; cross-language folders deferred.

## What gets committed vs. derived

- Committed: per-folder `<doc-file>`, `diagrams.md`, `.design.yaml`, per-folder `CONSTRAINTS.md`/`CAPABILITIES.md`, top-level `CONSTRAINTS.md`/`CAPABILITIES.md`, top-level `MAP.md`.
- Not committed: any cache, hash file, or transient build output. `last_rebuilt` SHA is the only persistent freshness marker.

## MVP scope (this PR + immediate follow-ups)

1. **This PR**: `/design-rebuild-go` skill; `/design-drift-check` and `/design-map` read `.design.yaml`; retire legacy `/design-rebuild` skill and `DESIGN.md` format; spec revision.
2. **Phase 1.5**: `/design-rebuild-ts` skill; bootstrap mode for existing CONSTRAINTS.md.
3. **Phase 2**: `/start_pr` and `/schedule` integration.
4. **Phase 3**: promote/demote suggestions; pkg.go.dev URL enrichment; cross-repo references.
