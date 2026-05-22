Rebuild per-package Go documentation: writes `doc.go` (full skill ownership — rewritten from scratch each rebuild), `diagrams.md` (Mermaid sequence diagrams for multi-step flows), and `.design.yaml` (machine-readable metadata) for each Go package in the repo. Idempotent and skip-if-fresh by default; `--force` regenerates regardless.

See `~/newstack/brain/DESIGN_SKILL_SPEC.md` for the system-level spec — the `.design.yaml` schema, the "where each kind of doc lives" convention, and why responsibility is split across three files.

## When to use

- After a refactor touched one or more Go packages.
- When `/design-drift-check` reports a Go package as stale.
- Periodically (nightly via `/schedule`, weekly manually) as the baseline freshness defense.
- Before opening a multi-package PR so the reviewer guide can reference accurate entity names.

## Modes

- **Full rebuild** (no args, run at repo root): walk every qualifying Go package. **Skip-if-fresh by default** — packages whose `last_rebuilt` SHA is still in HEAD's history and have had no file changes since are left untouched.
- **Targeted** (`--folder <path>`): one Go package; skip-if-fresh still applies. Does not re-walk the depends_on chain; if a neighbor is also stale, run `/design-drift-check` and rebuild it separately.
- **`--force`**: regenerate every candidate package regardless of freshness. Use when prose came out shallow on a prior run, or you want a clean LLM pass.

After a successful run, the user should run `/design-map` to refresh top-level `MAP.md` — it's a separate skill so its concerns stay isolated.

## Which folders qualify

A folder is a Go package if:

- It contains at least one `*.go` file that is not `_test.go`.
- That file starts with a `package <name>` declaration.

Skip:

- `vendor/`, `.git/`, `node_modules/`, `dist/`, `build/`, `target/`, `__pycache__/`, `.next/`, `.nuxt/`.
- Generated code (heuristic: files with `// Code generated ... DO NOT EDIT.` at the top — skip the whole folder).
- `cmd/` and subfolders, `examples/`, `internal/cmd/` — usually pass-through containers; doc.go in these is low value.
- `package main` packages with no exported API. (If a `main` package has substantive structure worth documenting, prompt the user — borderline.)

If a folder is borderline, ask the user before generating artifacts. Less is more.

## Per-folder artifacts

Each qualifying package gets:

1. **`doc.go`** — package-level prose, **full skill ownership**. Rewritten from scratch on every rebuild — no markers, no header line, no "preserve human content" logic. Symbol-level godoc lives on its symbols in other source files; examples live in `*_test.go` as idiomatic Go `ExampleXxx` functions; this file is package-level prose only and is fully managed by this skill. First-run protection (Pass 0.5) prompts before overwriting a pre-existing `doc.go` in a folder that has no `.design.yaml` yet.

2. **`diagrams.md`** — only if multi-step flows worth diagramming exist. Mermaid `sequenceDiagram` blocks (or `flowchart` for decision trees). `###` headings serve as anchors so `doc.go` can link `[Signup flow](diagrams.md#signup)`. If no flows qualify, the file is not created.

3. **`.design.yaml`** — machine-readable metadata. Always written (the freshness anchor for `/design-drift-check`, the source for `/design-map`). Schema in `DESIGN_SKILL_SPEC.md`.

## Algorithm

### Pass 0 — orchestrator prep

```bash
head_sha=$(git rev-parse HEAD)
```

Every `.design.yaml` written in this rebuild carries the same `last_rebuilt: <head_sha>` so drift detection has a consistent anchor across the whole pass.

Also snapshot the **pre-rebuild** `depends_on` graph by reading the current `depends_on:` from each existing `.design.yaml`. Pass 2 uses this to decide which packages need their linking redone.

### Pass 0.5 — drift triage (skip-if-fresh)

Before dispatching any Pass 1 subagents, decide per folder whether it actually needs rebuilding. Same logic as `/design-drift-check`:

```bash
for folder in candidate_folders; do
  sidecar="$folder/.design.yaml"

  if [ ! -f "$sidecar" ]; then mark_rebuild "$folder"; continue; fi
  if [ "$FORCE" = "1" ]; then mark_rebuild "$folder"; continue; fi

  if command -v yq >/dev/null; then
    last=$(yq '.last_rebuilt' "$sidecar" 2>/dev/null)
  else
    last=$(grep '^last_rebuilt:' "$sidecar" | head -1 | awk '{print $2}')
  fi
  if [ -z "$last" ] || [ "$last" = "null" ]; then mark_rebuild "$folder"; continue; fi
  if ! git merge-base --is-ancestor "$last" HEAD; then mark_rebuild "$folder"; continue; fi
  if [ -n "$(git diff --name-only "$last" HEAD -- "$folder/")" ]; then mark_rebuild "$folder"; continue; fi

  mark_fresh "$folder"
done
```

Output: `rebuild_set` (proceed to Pass 1) and `fresh_set` (left untouched).

**First-run protection** (runs as the final step of Pass 0.5, only over `rebuild_set`):

```bash
for folder in rebuild_set; do
  sidecar="$folder/.design.yaml"
  existing_doc="$folder/doc.go"

  # If a doc.go already exists but the folder has no .design.yaml yet, it might
  # be a hand-written file from before adoption — or a marker-era file from
  # before this bug fix. Either way, prompt before overwriting.
  if [ -f "$existing_doc" ] && [ ! -f "$sidecar" ]; then
    echo "FIRST RUN: $folder/doc.go exists but $folder/.design.yaml does not."
    echo "  Overwriting will replace the file's contents entirely."
    echo "  (Skill assumes pre-existing doc.go was either marker-era from prior runs"
    echo "   or hand-written. After this run, .design.yaml signals 'managed' and"
    echo "   subsequent rebuilds overwrite confidently without prompting.)"
    read -p "  Overwrite $folder/doc.go? (yes / skip) " answer
    [ "$answer" = "skip" ] && remove_from_rebuild_set "$folder" && add_to_skip_set "$folder"
  fi
done
```

After the first successful rebuild, `.design.yaml` is present and signals "this folder is skill-managed" — subsequent runs skip the prompt and overwrite `doc.go` freely.

### Pass 1 — per-package generation (parallel subagents)

For each candidate Go package in `rebuild_set`, spawn one `Agent` with `subagent_type=general-purpose` and the following prompt:

> Read every `*.go` file in `<folder>` excluding `_test.go`. Do NOT follow imports to sibling folders or descend into examples/test fixtures. Identify:
>
> - **Package name** from the `package <name>` declaration.
> - **Purpose** — one sentence: what the package owns, what it deliberately doesn't, what's notable about its shape.
> - **Entities** worth naming — exported structs, interfaces, types, key functions and methods. For each:
>   - `name` (e.g. `LocalAuth`, `LocalAuth.HandleSignup`)
>   - `kind` (`struct`, `interface`, `type`, `func`, `method`, `const`, etc.)
>   - `role` — one line: what it is in this package
>   - `why` — one line: design rationale, constraint enforced, gotcha. Avoid restating what the code does; favour the non-obvious why.
> - **Flows** — multi-step interactions worth diagramming (signup, request lifecycle, handshake, state machine). For each, identify participants and message order.
>
> Write three files into `<folder>`:
>
> ---
>
> **(1) `.design.yaml`** — machine-readable metadata, always written:
>
> ```yaml
> package: <name>
> purpose: <one-sentence purpose>
> language: go
> doc_file: doc.go
> diagrams_file: diagrams.md       # OMIT this key entirely if no flows
> last_rebuilt: <head_sha>
> last_rebuilt_at: <ISO 8601 UTC>
> entities:
>   - name: <name>
>     kind: <kind>
>     role: <one line>
>     why: <one line>
> depends_on: []                    # Pass 2 fills this; leave empty here
> ```
>
> ---
>
> **(2) `doc.go`** — package documentation, **full skill ownership**. Write from scratch using this exact structure (no markers, no header line, no "generated by" comment):
>
> ```go
> // Package <name> <one-sentence purpose>.
> //
> // <one paragraph: what the package owns, what it doesn't, what's notable about
> // its shape>
> //
> // ENTITIES
> //
> // <EntityName> — <role>. <why>.
> //
> // <EntityName.Method> — <role>. <why>.
> //
> // <...more entities>
> //
> // FLOWS
> //
> // See [diagrams.md](diagrams.md) for sequence diagrams of: <comma-separated flow names>.
> package <name>
> ```
>
> Format rules:
>
> - **Each line a Go comment** (`// `-prefixed). The whole comment block must be immediately adjacent to `package <name>` (no blank line between them) so godoc treats it as the package comment.
> - **ALL-CAPS section heading** (`ENTITIES`, `FLOWS`) on its own line surrounded by blank-comment lines — godoc renders these as headings on pkg.go.dev.
> - **Entity entries**: `EntityName — role. why.` Each entity in its own paragraph (separated by `//` blank-comment line). Methods qualified with the receiver type (`LocalAuth.HandleSignup`).
> - **Wrap lines at ~80 columns.** gofmt does not touch comments but readability matters.
> - **Omit the FLOWS section entirely** if `diagrams.md` was not written for this folder.
> - **NO markdown tables** (pkg.go.dev does not render them).
> - **NO Mermaid in doc.go** (lives in diagrams.md; doc.go only links).
> - **NO HTML comments** (`<!-- ... -->` renders as literal text on pkgsite — the bug this design eliminates). Never emit any HTML-comment syntax in this file.
> - **NO "generated by" header line.** doc.go is recognized as skill-managed by the presence of `.design.yaml` in the same folder, not by an in-file marker. First-run protection in Pass 0.5 handles the adoption case.
>
> ---
>
> **(3) `diagrams.md`** — only if flows were identified:
>
> ````markdown
> # <package> diagrams
>
> ### Signup
>
> ```mermaid
> sequenceDiagram
>     participant C as Client
>     participant L as LocalAuth
>     participant S as Stores
>     C->>L: POST /signup
>     L->>S: create user + identity + channel
>     ...
> ```
>
> ### Login
>
> ```mermaid
> sequenceDiagram
>     ...
> ```
> ````
>
> Use `###` headings; the anchor is the lowercased text with `-` for spaces (e.g. `#signup`, `#password-reset`). Use `sequenceDiagram` for temporal sequences; reserve `flowchart` for decision trees. Skip trivial 1–2-step flows — diagrams.md is for genuinely multi-step interactions.
>
> ---
>
> Do not commit. Do not read files outside `<folder>`. Do not modify any file outside `<folder>`.

Spawn all per-package subagents in a single message so they run concurrently. The orchestrator never reads the package's Go code.

### Pass 2 — cross-package linking (parallel subagents)

Pass 2 only re-runs for packages whose `depends_on:` may have shifted:

- Every package in `rebuild_set`, AND
- Every package whose **pre-rebuild** `depends_on:` referenced any package in `rebuild_set`.

Packages that are fresh AND whose dependencies are all fresh are skipped — their `depends_on:` is already current.

For each affected package, spawn one `Agent` (`subagent_type=general-purpose`) with this prompt:

> Read `<folder>/.design.yaml`. Read `<repo>/go.mod` to find the module path. Grep `<folder>/*.go` (excluding `_test.go`) for imports under the module path that target sibling packages in this repo. For each sibling used:
>
> 1. Read that sibling's `.design.yaml` to enumerate its declared entities.
> 2. Scan `<folder>/*.go` for actual references to those entities (`<pkg>.EntityName` form, where `<pkg>` is the imported alias).
>
> Update `<folder>/.design.yaml`'s `depends_on:` block to:
>
> ```yaml
> depends_on:
>   - path: <relative path to sibling, e.g. ../core>
>     entities: [Entity1, Entity2, ...]
> ```
>
> Touch ONLY the `depends_on:` field of `.design.yaml`. Do not modify `doc.go` or `diagrams.md`. Do not commit. Do not read files outside `<folder>` except the sibling `.design.yaml` files and `<repo>/go.mod`.

Parallel; orchestrator coordinates only.

### Pass 3 — top-level constraint verification (single orchestrator pass)

Walk repo-root `CONSTRAINTS.md` and `CAPABILITIES.md` if they exist. For each rule with an `**Entities**:` line (the additive binding):

- Parse the comma-separated qualified refs (`<folder>/Entity` form).
- For each ref, read the named folder's `.design.yaml` and verify the entity name exists in its `entities:` list.

Report unresolved refs. Do NOT auto-fix — the rule may need updating, or the entity may have been renamed, and the human has to decide.

Rules with no `**Entities**:` line are fine; the additive binding is opt-in.

## Output

End the skill with a structured summary, then nudges about MAP.md and review:

```
Rebuilt:    N packages (Pass 1 ran — three files written)
  - path/to/pkg-a (12 entities, 2 flows)
  - path/to/pkg-b (5 entities, 0 flows)
Fresh:      F packages (skip-if-fresh — left untouched)
  - path/to/foo
Ineligible: M folders (not Go, pass-through, or excluded)
  - path/to/cmd      (pass-through)
  - path/to/vendor   (excluded)
Cross-package linking updated: K packages (Pass 2)
Top-level refs verified: P pass, Q fail
  FAIL: CONSTRAINTS.md "Notebook events isolation" — events/InternalType not in events/.design.yaml

Files per package: doc.go, .design.yaml, optionally diagrams.md.

If any package was rebuilt, MAP.md may now be stale — run /design-map.

Review with `git diff` and commit (or `git restore`) per folder. This skill never commits.

If migrating from the old single-file DESIGN.md format: run `git rm DESIGN.md` in each
folder where the new artifacts now exist. This skill never auto-deletes legacy files.
```

## Principles

- One package per subagent. Orchestrator never reads `.go` code. Subagents must not read across folder boundaries during Pass 1.
- All `.design.yaml` files in a single rebuild share one `last_rebuilt` SHA. Drift detection depends on the consistency.
- **Re-runs are no-ops on unchanged packages.** Skip-if-fresh is the default; `--force` regenerates. Critical for nightly schedules.
- **`doc.go` is fully skill-owned.** Rewritten from scratch each rebuild. No markers, no header line, no preservation logic. Symbol-level godoc lives on its symbols in other source files; package examples in `*_test.go` via `ExampleXxx`; package-level prose in `doc.go`. First-run protection (Pass 0.5) prompts before overwriting a pre-existing `doc.go` in a folder that has no `.design.yaml` yet; after that, `.design.yaml` is the "managed" signal and overwrites are silent.
- **No HTML comments anywhere in `doc.go`.** `<!-- ... -->` renders as literal text on pkgsite/pkg.go.dev — that's the bug this design eliminates. Never emit HTML-comment syntax in `doc.go`.
- Never auto-commit. Never auto-branch. **Never auto-delete legacy `DESIGN.md` files** during migration — user reviews and `git rm` themselves.
- Never auto-add `**Entities**:` lines to existing `CONSTRAINTS.md` / `CAPABILITIES.md` rules — humans declare bindings.
- This skill does not generate MAP.md. That's `/design-map`'s job; this skill only nudges the user to run it when rebuilds happened.
- **No Mermaid in `doc.go`.** Doesn't render on pkg.go.dev; Mermaid lives in `diagrams.md`.
- **No markdown tables in `doc.go`.** pkg.go.dev doesn't render them; use flat lists with ALL-CAPS section headings.
- The why beats the what. Entity descriptions favour design rationale over restating code.
- A folder that doesn't earn the three artifacts (too small, too mechanical, no real entities, pass-through) shouldn't get them. Less is more.
- When in doubt about whether a folder qualifies, ask the user before writing.
