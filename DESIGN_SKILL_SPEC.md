# Per-Folder Design Documentation — Specification

A staleness-resistant, **language-agnostic**, per-folder architecture-documentation system. Each folder gets one rich-markdown `DESIGN.md` (entities table, inline Mermaid flows, gotchas, dependency links) plus a machine-readable `.design.yaml` sidecar. A single `/design-rebuild` skill maintains both; `/design-drift-check` reports staleness; `/design-map` collates a top-level `MAP.md` from the sidecars.

## Goal

Capture the parts of a project that **do not survive rederivation by grep**: package purpose, the *why* behind each named entity, declared relationships between entities, and the bindings between entities and architectural rules. Make this artifact (i) per-folder so blast radius of any single update is small, (ii) **rendered richly on GitHub** (inline Mermaid, tables, structured headings, anchor links), (iii) machine-checkable via a structured sidecar so staleness is detectable, (iv) consumed by other skills (`/start_pr`, `/checkpoint`, plan mode) so being out of date has visible consequences.

## Non-goals

- Capturing every method or call edge. That rots fastest, adds least value, can be re-derived.
- Replacing `ARCHITECTURE.md` / `README.md` / `SUMMARY.md`. DESIGN.md is per-folder; those remain top-level narrative.
- Replacing `CONSTRAINTS.md` / `CAPABILITIES.md`. Those stay, but optionally reference entities from the per-folder sidecars.
- Generating from scratch every time. Rebuilds are full but rare; skip-if-fresh is the default.
- Managing language-native doc files (`doc.go`, `README.md`, `lib.rs`, `__init__.py` docstrings). Those are human/agent territory. The skill stays out of source files entirely.

## Architectural principle: one artifact, two surfaces

The skill owns **DESIGN.md** (for humans, LLMs, agents — rendered on GitHub with full markdown) and **`.design.yaml`** (for tooling — `/design-drift-check`, `/design-map`, `/start_pr`).

It does **not** own language-native doc files. Symbol-level documentation (godoc comments, JSDoc, Rust `///` doc comments, Python docstrings) lives on its symbols in the source files — co-located with the code it documents, reviewed alongside the code in PRs. The skill captures package-level *architecture* — purpose, entities, flows, gotchas, relationships — not API documentation.

## Per-folder artifacts

Two files per qualifying folder. Each has one clearly-scoped job.

### 1. `DESIGN.md` — rich-markdown architecture document

Fully skill-owned. Rewritten from scratch each rebuild. Renders natively on GitHub with inline Mermaid sequence diagrams, markdown tables, anchor-linked sections.

Structure:

```markdown
# <folder name>

<one-paragraph overview: what the package owns, what it doesn't, what's notable about its shape>

## Contents

- [Entities](#entities)
- [Flows](#flows)
  - [<Flow 1>](#<anchor>)
  - [<Flow 2>](#<anchor>)
- [Gotchas](#gotchas)
- [Depends on](#depends-on)

## Entities

| Entity | Kind | Role | Why |
|---|---|---|---|
| `LocalAuth` | struct | Central config-and-handler object. | Deliberately a flat bag of optional fields so apps wire only what they use. |
| `LocalAuth.HandleSignup` | method | Registration handler. | Username reservation and verification-email failures are warned-not-fatal to avoid leaving half-created accounts. |

## Flows

### Signup

\`\`\`mermaid
sequenceDiagram
    participant C as Client
    participant L as LocalAuth
    participant S as Stores
    C->>L: POST /signup
    L->>S: create user + identity + channel
    L-->>C: 200 (logged in or "verify email")
\`\`\`

### Login

\`\`\`mermaid
sequenceDiagram
    ...
\`\`\`

## Gotchas

- **Username login lacks the dummy-bcrypt timing defense** — `NewCredentialsValidatorWithUsername` does not run a dummy bcrypt on user-not-found, unlike the email/phone validator. Username-existence enumeration is possible via timing.
- **HandleLinkCredentials sniffs error strings** — the "already exists" → 409 mapping uses substring match on the error message from `LinkLocalCredentials`. Brittle coupling.

## Depends on

- [`core/`](../core/DESIGN.md) — `User`, `Identity`, `Channel`, `ChannelStore`, `IdentityStore`, ...
- [`utils/`](../utils/DESIGN.md) — `ComputeKid`, `JWK`, ...
```

Format rules:
- **One H1** with the folder name.
- **One opening paragraph** before the TOC.
- **`## Contents` TOC** auto-generated from the sections actually present. Kebab-case anchors. Nest one level deep for flows.
- **`## Entities`** is a markdown table; methods qualified as `Receiver.Method`; identifier names in backticks.
- **`## Flows`** has one `###` heading per flow, each containing a Mermaid `sequenceDiagram` (or `flowchart` for decision trees). Omit the section entirely if no flows worth diagramming.
- **`## Gotchas`** is a bulleted list, bolded short title + one-paragraph explanation per gotcha. Omit if there are none.
- **`## Depends on`** is filled by Pass 2 (cross-folder linking). The section is always present in the rendered file; Pass 1 leaves a placeholder, Pass 2 populates with sibling DESIGN.md links + the specific entities referenced.

### 2. `.design.yaml` — machine-readable sidecar

The contract for `/design-drift-check`, `/design-map`, and `/start_pr` cold-start. Schema:

```yaml
package: localauth                  # folder identity, typically the folder basename
purpose: Form-based local username/password auth — signup, login, verify, reset, channel linking.
language: go                        # informational only; used by /design-map's "Languages: ..." line
last_rebuilt: <commit SHA>          # the freshness anchor — set to git rev-parse HEAD at rebuild time
last_rebuilt_at: 2026-...           # ISO 8601 UTC, informational
entities:
  - name: LocalAuth
    kind: struct
    role: Central config-and-handler object.
    why: Deliberately a flat bag of optional fields so apps wire only what they use.
  - name: LocalAuth.HandleSignup
    kind: method
    role: Registration handler.
    why: Username reservation and verification-email failures are warned-not-fatal to avoid leaving half-created accounts.
depends_on:                         # filled by Pass 2; cross-folder references
  - path: ../core
    entities: [User, IdentityStore, ChannelStore, ...]
```

Notes:
- The `language:` field is informational. Drift-check and map don't branch on it; only `/design-map`'s stat line uses it.
- `last_rebuilt` is the only freshness signal. If it falls out of HEAD's history (force-push, squash), drift-check returns DRIFT and the next rebuild re-anchors.
- `entities[].name` is what `**Entities**:` lines in `CONSTRAINTS.md` / `CAPABILITIES.md` resolve against. Stability matters — renaming invalidates rule bindings.

## File layout

```
repo/
├── ARCHITECTURE.md          (existing; narrative)
├── README.md                (existing)
├── MAP.md                   ← auto-generated by /design-map
├── CONSTRAINTS.md           (top-level; cross-folder rules; qualified entity refs)
├── CAPABILITIES.md          (optional; top-level; cross-folder capabilities)
├── pkg-a/
│   ├── doc.go               (Go-native, hand-authored if at all — skill does not touch)
│   ├── DESIGN.md            ← rich architecture doc, skill-owned
│   ├── .design.yaml         ← machine-readable sidecar, skill-owned
│   ├── CONSTRAINTS.md       (optional; local rules; bare entity refs)
│   └── CAPABILITIES.md      (optional; local; bare entity refs)
├── pkg-b/
│   ├── DESIGN.md
│   └── .design.yaml
└── ...
```

Rule of thumb for constraint split:
- Constraint/capability entities all in one folder → that folder's file (bare refs).
- Entities span ≥2 folders → top-level file (qualified `folder/Entity` refs).
- Folder is a pass-through container (e.g., `cmd/`, `examples/`) → no DESIGN.md.

## `CONSTRAINTS.md` format

Per-folder and top-level CONSTRAINTS.md keep brain's existing prose format (`### Rule Name / **Rule** / **Why** / **Verify** / **Scope**`) and **optionally add an `**Entities**:` line** per rule binding to the folder's `.design.yaml` entities. Additive — existing files continue to work untouched.

```markdown
### Store mutations via CRUD only
**Rule**: store mutations only via Notebook CRUD methods — never reach into the store from outside the package
**Why**: the single-mutex invariant requires all writers to go through Notebook; direct access bypasses lock ordering
**Verify**: `! grep -rE 'notebook\.(store|cells)' --include='*.go' | grep -v 'notebook/'`
**Scope**: notebook package
**Entities**: Notebook, store
```

Top-level `CONSTRAINTS.md` entries use qualified refs (`folder/Entity`) for the same `**Entities**:` line since they cross folder boundaries.

## Algorithms

### Full rebuild (`/design-rebuild` invoked at repo root)

Three passes; each pass parallelizes across folders via subagents.

```
orchestrator:
  folders = discover_folders(repo)              # exclude pass-through containers, vendor/, .git/, etc.
  head_sha = git rev-parse HEAD
  prior_depends_on = snapshot every existing .design.yaml's depends_on field

  # Pass 0.5: drift triage + first-run protection
  rebuild_set, fresh_set = drift_triage(folders, --force?)
  for folder in rebuild_set:
    if DESIGN.md exists but .design.yaml does NOT: prompt; skip if user declines

  # Pass 1: per-folder rebuild (parallel subagents)
  for folder in rebuild_set in parallel:
    spawn subagent reading ONLY <folder>'s source
    subagent writes: DESIGN.md (full structure: H1 + intro + Contents + Entities + optional Flows + optional Gotchas + Depends on placeholder), .design.yaml

  # Pass 2: cross-folder linking (parallel subagents)
  pass_2_set = rebuild_set ∪ { f : prior_depends_on(f) ∩ rebuild_set ≠ ∅ }
  for folder in pass_2_set in parallel:
    spawn subagent that updates <folder>/.design.yaml's depends_on field
    AND updates <folder>/DESIGN.md's `## Depends on` section with sibling links

  # Pass 3: top-level constraint verification (orchestrator, no subagent)
  for file in [CONSTRAINTS.md, CAPABILITIES.md] at repo root:
    for rule in file:
      if rule has **Entities**: line:
        for ref in entities (qualified folder/Entity form):
          verify ref resolves against <folder>/.design.yaml's entities list
    report unresolved refs (do not auto-fix)
```

Skip-if-fresh applies in Pass 1; Pass 2 narrows further based on `prior_depends_on`; Pass 3 is cheap and always runs.

### Targeted rebuild (`--folder <path>`)

Same passes, scoped to one folder. Does **not** re-walk the depends_on chain in the MVP — if a neighbor is also stale, run `/design-drift-check` and rebuild it separately.

### Drift check (`/design-drift-check`)

Pure git-only, language-agnostic. For each candidate folder F:

```bash
sidecar="$F/.design.yaml"
[ -f "$sidecar" ] || { echo "$F: NO MODEL"; continue; }

last=$(yq '.last_rebuilt' "$sidecar")
[ -z "$last" ] && { echo "$F: NO ANCHOR"; continue; }

git merge-base --is-ancestor "$last" HEAD || { echo "$F: DRIFT (out of history)"; continue; }
[ -n "$(git diff --name-only "$last" HEAD -- "$F/")" ] && { echo "$F: DRIFT"; continue; }
echo "$F: fresh ($last)"
```

Exit code: 0 if all fresh / NO MODEL; 1 if any DRIFT or NO ANCHOR.

### Map (`/design-map`)

1. Walk repo for every `.design.yaml`.
2. For each: parse `package`, `purpose`, `language`, `depends_on`.
3. Build dependency graph (nodes = folders, edges = internal `depends_on`).
4. Topologically sort for reading order; cycles get a dedicated section.
5. Render `MAP.md` at repo root with folder names linked to **`<folder>/DESIGN.md`** (relative), a "Languages: ..." stat line, and a code-block dependency graph.

Always whole-repo; idempotent.

### Bootstrap (one-time enrichment of existing CONSTRAINTS/CAPABILITIES)

Existing `CONSTRAINTS.md` files use the prose format but have no `**Entities**:` line yet. Bootstrap **augments** them in place — no format conversion:

```
for each existing CONSTRAINTS.md or CAPABILITIES.md:
  for each rule heading (### Rule Name):
    candidates = grep matching entity names from sibling .design.yaml against the rule prose
    propose adding `**Entities**: <candidates>` under the existing **Verify** / **Scope** lines
    flag the proposal with `<!-- bootstrap_review -->` so the human knows to confirm
emit a report; do not auto-apply
```

After human review, the binding is explicit and self-policing thereafter.

## Migration from prior formats

### From single-file `DESIGN.md` (the original format)

If your folder already has a hand-written `DESIGN.md`, the skill prompts before overwriting on first run (first-run protection in Pass 0.5). Choose `skip` to preserve, or `yes` to let the skill take over.

### From the per-language `doc.go` + `diagrams.md` + `.design.yaml` format (retired)

If your repo was migrated under the retired `/design-rebuild-go` skill, clean up via:

1. `find . -name doc.go -exec grep -l 'ENTITIES\|<!-- design' {} \;` to identify skill-generated `doc.go` files. `git rm` them (the skill no longer manages doc.go).
2. `find . -name diagrams.md` — delete those (their content moves inline into the new `DESIGN.md`).
3. Run `/design-rebuild --force` at the repo root — writes fresh `DESIGN.md` per folder with diagrams inline; refreshes `.design.yaml`.
4. Run `/design-map` to refresh `MAP.md` against the new DESIGN.md links.

No automated migration script — the skill regenerates content cleanly under the new format, and removing legacy files is the user's deliberate act.

## Triggers

### (a) Scheduled remote rebuild

Use `/schedule` to register a nightly remote routine that runs `/design-rebuild` and `/design-map`.

### (b) Manual on-demand

Direct skill invocation: `/design-rebuild [--folder <path>] [--force]`.

### (c) Commit-count threshold (not in MVP)

Deferred. A git post-commit hook pinging the skill at every Nth commit. Add later if nightly proves too coarse.

## Integration

### `/start_pr` (Phase 2 — not yet wired)

1. Identify folders touched by the planned change.
2. Run `/design-drift-check` on those folders.
3. If drift: suggest `/design-rebuild` before continuing.
4. Load `.design.yaml` entities for touched folders into the plan context.
5. Plan and reviewer guide reference entities by name (e.g., "extends `Notebook.Stream` to honor a new `outputBuffered` variant").
6. Reviewer's guide auto-includes touched folders' `DESIGN.md` as the entry-point read.

### `/checkpoint` (Phase 2 — not yet wired)

Drift check across all folders touched since the last checkpoint; surface stale folders as a "consider running /design-rebuild" nudge. Don't auto-rebuild.

### Plan mode

Auto-load `.design.yaml` files for the folders the planning task touches. Plan output uses entity names from the sidecars.

## Open questions / Phase 2+

- **`/start_pr` + `/checkpoint` wiring** (Phase 2) — designed; not yet implemented.
- **`/schedule` integration** (Phase 2) — nightly routine for active repos.
- **Bootstrap mode** for existing `CONSTRAINTS.md` files (Phase 1.5).
- **Promote/demote auto-suggest** (Phase 3) — scan constraint files; surface rules whose entity refs all resolve to one folder ("consider demoting") or that grep-match across folders ("consider promoting").
- **pkg.go.dev / npm URL enrichment** in `MAP.md` — link to the public registry for each folder.
- **Splitting big DESIGN.md files** — if a folder's DESIGN.md becomes unwieldy (>2000 lines), the user can hand-split into `<folder>/DESIGN.md` (overview) + `<folder>/designs/<topic>.md` (deep dives). Not baked into the skill until evidence shows it's needed.
- **`/design-audit-symbols`** — passive checker for missing/shallow godoc on exported symbols (Phase 3, only if symbol-doc drift becomes a real problem).

## What gets committed vs. derived

- Committed: per-folder `DESIGN.md`, `.design.yaml`, per-folder `CONSTRAINTS.md`/`CAPABILITIES.md`, top-level `CONSTRAINTS.md`/`CAPABILITIES.md`, top-level `MAP.md`.
- Not committed: any cache, hash file, or transient build output. `last_rebuilt` SHA is the only persistent freshness marker.

## Design history (why this is the third iteration)

This spec has been through three pivots; each was a real experiment that produced useful evidence:

1. **Original (PR #1, #2): single `DESIGN.md` per folder** — language-agnostic, YAML frontmatter, rich markdown body. Worked well structurally; the format reconciliation with existing CONSTRAINTS.md was the headline open question.

2. **Per-language with marker injection (PR #3): doc.go + diagrams.md + .design.yaml** — pivoted to write Go's idiomatic doc home (doc.go) with skill content scoped by `<!-- design:start --> ... <!-- design:end -->` markers, plus a separate diagrams.md for Mermaid. Goal: integrate with pkg.go.dev / `go doc` natively. **Bug:** pkgsite renders HTML comments as visible literal text. PR #4 patched the markers but left the broader question open.

3. **Full doc.go ownership (PR #4): doc.go with no markers, rewritten whole** — eliminated the marker bug but produced flat, hard-to-consume output. pkg.go.dev's rendering surface can't carry markdown tables, can't render Mermaid, and links to relative files are unreliable. The rich content we want to produce is fundamentally incompatible with Go-doc rendering.

4. **This pivot (current): unified `DESIGN.md` per folder, language-agnostic** — back to single rich-markdown per folder, with inline Mermaid (no separate diagrams.md), a `## Contents` TOC for navigation, and explicit non-ownership of language-native doc files. GitHub-rendered markdown is the universal rendering surface; symbol-level docs stay in source where the language ecosystem (godoc, JSDoc, etc.) handles them naturally.

The earlier work wasn't wasted — it was the test that disproved the per-language hypothesis. Without seeing flat doc.go on pkgsite, the constraints wouldn't have been clear.

## MVP scope (this PR + immediate follow-ups)

1. **This PR**: `/design-rebuild` skill (resurrected, unified); `/design-drift-check` + `/design-map` minor updates; retire `/design-rebuild-go`; spec rewrite.
2. **Immediate next (separate PR in oneauth)**: migrate oneauth — delete the marker-era + doc.go-era files, run new `/design-rebuild --force`, regenerate `MAP.md`.
3. **Phase 1.5**: bootstrap mode for existing CONSTRAINTS.md.
4. **Phase 2**: `/start_pr` and `/checkpoint` integration; `/schedule` registration.
5. **Phase 3**: promote/demote suggestions; pkg.go.dev URL enrichment; splitting big DESIGN.md (only if needed); `/design-audit-symbols` (only if needed).
