# Stack Gaps

> Capabilities needed by projects but not yet covered by a stack component.
> Each entry is a candidate for a new component in ~/newstack.

<!--
When adding a gap, use this format:

## {Capability Name}
- **Needed by**: {project(s)}
- **Date identified**: {YYYY-MM-DD}
- **Stopgap used**: {third-party lib or inline code, if any}
- **Notes**: {any context}
- **Status**: Open | In Progress | Resolved (→ {component})
-->

*No gaps recorded yet.*

## Hybrid Store — Server-Rendered Templates + Client-Side Data Loading
- **Needed by:** Lucid Capture (Go stack dashboard needs to render template server-side but load project list from IndexedDB client-side)
- **Existing pattern:** Excaliframe does this — templates rendered by Go, data fetched client-side via JS
- **What's needed:** A reusable primitive in GoAppLib/tsappkit that:
  - Server renders the page shell (template with placeholders)
  - Client JS discovers `data-component` mount points
  - Component loads data from a configurable source (IndexedDB, REST API, or both)
  - Renders into the mount point using jsx-dom or DOM manipulation
- **Stopgap:** Inline `<script>` in templates that reads IndexedDB directly
- **Date:** 2026-03-24

## TUI / interactive terminal UI (no stack component)
Identified 2026-07-17 while building mcpkit agentchat's playground (epic 983, PR 987).
The stack catalog has no terminal-UI / interactive-CLI component. agentchat needs an
editable cursor-traversable input, up/down command history, scrollback, and a slash-command
tab-palette. Adopted the Charm stack (bubbletea + bubbles + lipgloss) in cmd/agentchat — the
de-facto Go TUI standard, no in-stack alternative exists. If a second stack project needs a
TUI, consider promoting a shared Charm-based helper.
Extended 2026-07-21 (issue 1063 B1/B2, glamour-at-commit): added `github.com/charmbracelet/glamour`
to cmd/agentchat for rendering finished assistant markdown to terminal ANSI. Same Charm stack;
fold into any shared Charm-based helper if one is promoted.

## pgvector Go client (semantic vector store)
Identified 2026-07-20 while shipping mcpkit's durable semantic MemoryStore (issue 1019, PR 1047).
Adopted `github.com/pgvector/pgvector-go` for `gormstore.SemanticMemoryStore` (Postgres + pgvector
ANN retrieval behind `agent.MemoryStore`). The pgvector *image* (`pgvector/pgvector`) is already
used in `docker/backends`; this adds the Go client for encoding vectors + the `<=>` cosine-distance
query. Not yet in STACK_CATALOG.md — add via /stack-catalog-refresh. If a second project needs
vector search, this is the seam to promote.

## Request-time markdown -> HTML render (no lightweight stack component)
Identified 2026-07-31 while building diffpp's PR description drawer (VW-018, PR 123).
The stack's markdown capability lives in `s3gen` (github.com/panyam/s3gen), a full
static-site generator (build pipeline + template rendering). For a single request-time
string render (render one PR body to sanitized HTML per request), s3gen is too heavy.
Adopted `github.com/yuin/goldmark` (the GFM lib an SSG wraps) + `github.com/microcosm-cc/bluemonday`
(sanitizer, since the body is external content) directly in `internal/prdesc`. If a second
project needs request-time markdown->HTML, consider extracting a thin `Markdown(string) -> safeHTML`
helper (goldmark + bluemonday) as a shared stack component, or exposing one from s3gen.

## SSE client reading (servicekit has it; inference-plane reimplemented a subset)
- **Needed by**: inference-plane (`cmd/iplane/cmd/load_session.go`, the load generator's OpenAI streaming parser)
- **Date identified**: 2026-08-26
- **Stopgap used**: none needed. This is the reverse of the usual gap: `servicekit/http/sse_reader.go`
  already ships `SSEEventReader`, a client-side WHATWG-spec reader (`event` / `data` / `id` / `retry`,
  comments, multi-line data concatenation). iplane hand-rolled a `bufio.Scanner` that keeps only lines
  matching the literal prefix `"data: "`.
- **Notes**: the hand-rolled version diverges from spec in ways that happen not to bite against vLLM:
  `data:` with no space is dropped, multi-line `data` fields are not concatenated, and a frame over
  1 MB errors the scan. Discovered while fixing measurement accounting, when the question "is this a
  pushdown to servicekit?" turned out to have the answer "no, it is already there".
  The split if adopted: framing and stream termination to servicekit; the `[DONE]` sentinel (an OpenAI
  convention absent from the SSE spec), the two OpenAI frame shapes, and TTFT/ITL timing stay in iplane.
  Tracked as inference-plane issue 450.
- **Status**: Open. Worth a note in STACK_CATALOG.md that servicekit covers SSE on both sides, since
  the read side was not obvious enough to prevent one reimplementation.

## servicekit RequestLogger stripped http.Flusher (fixed, v0.1.5)
- **Needed by**: inference-plane (streaming proxy through `iplane serve`)
- **Date identified**: 2026-08-24
- **Stopgap used**: none; fixed upstream and released as servicekit v0.1.5.
- **Notes**: `statusRecorder` embedded `http.ResponseWriter` to capture the status code, and that
  interface carries only Header/Write/WriteHeader, so the wrap silently removed `http.Flusher`.
  Any handler behind `RequestLogger` that type-asserts for a flusher stops streaming, with identical
  status and bytes and only timing as the observable. Measured downstream: 0.001ms between tokens
  through the middleware against 9.055ms direct. `statusRecorder` now implements `Unwrap` (for
  `http.ResponseController`) and `Flush`.
- **Status**: Resolved (→ servicekit v0.1.5). Logged because the failure mode generalises: any
  middleware in the stack that wraps a ResponseWriter should be checked for the same interface loss.
