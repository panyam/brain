Check whether per-folder DESIGN.md files are stale relative to HEAD. Pure git, no LLM judgment. Use standalone, or as a cheap subroutine from `/start_pr` / `/checkpoint`.

See `~/newstack/brain/DESIGN_SKILL_SPEC.md` §"Drift check" for the design rationale.

## When to use

- From `/start_pr`: which folders touched by this PR have a stale DESIGN.md?
- Standalone: spot-check before deciding whether to run `/design-rebuild`.
- Before `/checkpoint`: surface folders the next rebuild should hit.

## Modes

- `--folder <path>`: check one folder.
- `--touched`: derive folders from the current working-tree diff (`git diff --name-only` against `origin/main`, or `HEAD~1` if no upstream).
- No args: check every folder in the repo that has a DESIGN.md.

## Algorithm

For each candidate folder `F`:

```bash
design="$F/DESIGN.md"

if [ ! -f "$design" ]; then
  echo "$F: NO MODEL — no DESIGN.md"
  continue
fi

# Frontmatter extraction: prefer yq if available, fall back to sed.
if command -v yq >/dev/null 2>&1; then
  last=$(yq '.last_rebuilt' "$design" 2>/dev/null)
else
  last=$(sed -n '/^---$/,/^---$/p' "$design" | grep '^last_rebuilt:' | head -1 | awk '{print $2}')
fi

if [ -z "$last" ] || [ "$last" = "null" ]; then
  echo "$F: NO ANCHOR — DESIGN.md has no last_rebuilt SHA"
  continue
fi

if ! git merge-base --is-ancestor "$last" HEAD 2>/dev/null; then
  echo "$F: DRIFT — last_rebuilt $last not in HEAD history (force-pushed, squashed, or rewritten)"
  continue
fi

changes=$(git diff --name-only "$last" HEAD -- "$F/")
if [ -n "$changes" ]; then
  n=$(echo "$changes" | wc -l | tr -d ' ')
  echo "$F: DRIFT — $n file(s) changed since $last"
else
  echo "$F: fresh ($last)"
fi
```

## Output

One line per folder. Suggested status vocabulary:
- `fresh (<sha>)` — last_rebuilt is HEAD's ancestor and nothing in the folder changed since.
- `DRIFT — ...` — something changed; rebuild is recommended.
- `NO MODEL` — no DESIGN.md (might be a pass-through folder, or never rebuilt).
- `NO ANCHOR` — DESIGN.md exists but frontmatter has no `last_rebuilt`. Treat as drift; rerun `/design-rebuild`.

Exit code: `0` if all folders are fresh or have NO MODEL; `1` if any DRIFT or NO ANCHOR was detected. Other skills can `if /design-drift-check --touched; then ...; fi`.

End with one practical line:

```
Drift detected in K folder(s). Run /design-rebuild --folder <path> for targeted rebuilds, or /design-rebuild for a full pass.
```

## Principles

- Git-only. No reading source files, no LLM calls. This skill is meant to be cheap enough to run on every PR.
- Report; do not act. The decision to rebuild belongs to the user (or to `/design-rebuild` when explicitly invoked).
- Frontmatter SHA is the only freshness signal. If it falls out of history (force-push, squash), report DRIFT and let `/design-rebuild` re-anchor it.
- NO MODEL is not an error. A folder may legitimately not have a DESIGN.md (pass-through, fixtures, generated). Don't pressure-create them.
