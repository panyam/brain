# Stack Brain — Next Steps

## Immediate (Milestone 2: Emit System)
- [ ] Build `stack-brain emit` command — compile constraints + conventions into agent-native instruction files
- [ ] Claude Code target: generate CLAUDE.md with marker-based injection (`<!-- stack-brain:start/end -->`)
- [ ] Cursor target: generate `.cursor/rules/stack-brain.mdc` with MDC frontmatter
- [ ] Windsurf + Copilot targets: plain markdown output
- [ ] Template system for per-agent formatting

## Short Term (Milestones 3-4)
- [ ] `stack-brain learn` — generate structured exploration prompt for bootstrapping codebase understanding
- [ ] `stack-brain checkpoint` — CLI command to record session learnings to env knowledge store
- [ ] Per-env `conventions.md` with router pattern (inline → pointer when entries grow)
- [ ] Semantic dependency support in DAG and stale commands (fork-of, shares-api-contract edges)
- [ ] External repo stale checking — compare commit drift since last-checked

## Ongoing (Pre-Environment Work)
- [ ] Test `/stack-audit` on lilbattle and SDL — validate constraint checking + existing findings quality
- [ ] Seed CONSTRAINTS.md in SDL and excaliframe projects (lilbattle already done)
- [ ] Write CHANGELOG.md for components that have had recent breaking changes (oneauth, servicekit, goapplib)
- [ ] Create first migration docs (migrations/ directories) for components with version gaps
- [ ] Add sub-component entries to goapplib (tsappkit) and massrelay (ts client) CAPABILITIES.md
- [ ] Test `/stack-constraints-promote` end-to-end once 2+ projects share a constraint
- [ ] Add `stack-brain audit-deps` command — deterministic scan of go.mod for third-party deps that overlap with stack capabilities
- [ ] Improve `stack-brain update` to also update Stackfile.md after bumping versions

## Medium Term
- [ ] Convention versioning — track which conventions a project follows, not just which versions
- [ ] Cross-project staleness dashboard — `stack-brain stale --all` across all environments
- [ ] Integration with devloop — auto-run stack-brain refresh when CAPABILITIES.md files change
- [ ] CI/pre-commit integration for constraint verify patterns (enables fully unsupervised agents)
- [ ] MCP server (if emit proves insufficient for real-time agent interaction)

## Ideas / Future
- [ ] Stack component scaffolding — `stack-brain new <name>` creates a new component with boilerplate
- [ ] Team/org convention packs — shared constraint sets pulled from a central repo
- [ ] Web dashboard for stack health across all environments
