# Stack Brain — Next Steps

## Immediate
- [ ] Test `/stack-audit` on lilbattle and SDL — validate findings quality
- [ ] Write CHANGELOG.md for components that have had recent breaking changes (oneauth, servicekit, goapplib)
- [ ] Create first migration docs (migrations/ directories) for components with version gaps
- [ ] Add sub-component entries to goapplib (tsappkit) and massrelay (ts client) CAPABILITIES.md

## Short Term
- [ ] Expose stack-brain as MCP server for stronger enforcement (LLM calls tools instead of bash)
- [ ] Add `stack-brain audit-deps` command — deterministic scan of go.mod for third-party deps that overlap with stack capabilities
- [ ] Improve `stack-brain update` to also update Stackfile.md after bumping versions
- [ ] Add `--format=text` output option to CLI for human-readable output alongside JSON

## Medium Term
- [ ] Convention versioning — track which conventions a project follows, not just which versions
- [ ] Auto-detect new stack components — scan for new CAPABILITIES.md files that aren't in the catalog
- [ ] Cross-project staleness dashboard — `stack-brain stale --all` across all projects
- [ ] Integration with devloop — auto-run stack-brain refresh when CAPABILITIES.md files change

## Ideas / Future
- [ ] Stack component scaffolding — `stack-brain new <name>` creates a new component with boilerplate (go.mod, CAPABILITIES.md, CLAUDE.md)
- [ ] Publish CAPABILITIES.md as part of go module metadata
- [ ] Web dashboard for stack health across all projects
