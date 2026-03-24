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
