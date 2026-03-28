# Stack Brain — Next Steps

## Immediate
- [ ] Constraint pointer resolution in emit — follow `See oneauth/CONSTRAINTS.md: rule-name` pointers
- [ ] Convention router resolution — `stack-brain query conventions` resolves `See X: Y` pointers
- [ ] Test `/stack-audit` on lilbattle and SDL — validate constraint checking + existing findings quality
- [ ] Seed CONSTRAINTS.md in SDL and excaliframe projects (lilbattle already done)

## Short Term
- [ ] Write CHANGELOG.md for components with recent breaking changes (oneauth, servicekit, goapplib)
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

## Deprioritized (revisit based on real usage)
- [ ] `stack-brain learn` — generate structured exploration prompt for new repos (existing `/stack-integrate` covers onboarding)
- [ ] `stack-brain checkpoint` — CLI version of /checkpoint (existing skill + manual emit covers the need)
- [ ] Team/org convention packs — shared constraint sets pulled from a central repo
- [ ] Web dashboard for stack health across all environments

## Done
- [x] Environment system (env create/list/add/remove/info, STACK_ENV + cwd detection)
- [x] Emit system (claude/cursor/windsurf/copilot, marker injection, explicit env arg)
- [x] Semantic deps in DAG (external_nodes, semantic_edges fields)
- [x] External repo stale checking (commit drift, commits_behind count)
- [x] Migrate newstack as env:newstack (--import-catalog from STACK_CATALOG.md)
