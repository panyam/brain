# /design check

Check whether per-folder `.design.yaml` sidecars are stale relative to HEAD. Pure git,
no LLM judgment. Cheap enough to run on every PR.

Shared conventions (the sidecar schema, the freshness model, which folders qualify)
live in `SKILL.md`. Read that first if you have not.

See `~/newstack/brain/DESIGN_SKILL_SPEC.md` §"Drift check" for the design rationale.

## When to use

- From `/start_pr`: which folders touched by this PR have a stale `.design.yaml`?
- Standalone: spot-check before deciding whether to run a `/design rebuild`.
- Before `/checkpoint`: surface folders the next rebuild should hit.

## Modes

- `--folder <path>`: check one folder.
- `--touched`: derive folders from the current working-tree diff (`git diff --name-only` against `origin/main`, or `HEAD~1` if no upstream).
- No args: check every folder in the repo that has a `.design.yaml`.

## Algorithm

For each candidate folder `F`:

```bash
sidecar="$F/.design.yaml"

if [ ! -f "$sidecar" ]; then
  echo "$F: NO MODEL — no .design.yaml"
  continue
fi

# YAML extraction: prefer yq if available, fall back to grep.
if command -v yq >/dev/null 2>&1; then
  last=$(yq '.last_rebuilt' "$sidecar" 2>/dev/null)
else
  last=$(grep '^last_rebuilt:' "$sidecar" | head -1 | awk '{print $2}')
fi

if [ -z "$last" ] || [ "$last" = "null" ]; then
  echo "$F: NO ANCHOR — .design.yaml has no last_rebuilt SHA"
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
- `fresh (<sha>)` when `last_rebuilt` is HEAD's ancestor and nothing in the folder changed since.
- `DRIFT — ...` when something changed. A rebuild is recommended.
- `NO MODEL` when there is no `.design.yaml` (a pass-through folder, or one never rebuilt).
- `NO ANCHOR` when `.design.yaml` exists but carries no `last_rebuilt`. Treat it as drift and rerun `/design rebuild`.

Exit code `0` means every folder is fresh or has NO MODEL, and `1` means at least one DRIFT or NO ANCHOR turned up. Other skills can branch on it.

End with one practical line:

```
Drift detected in K folder(s). Run /design rebuild --folder <path> for targeted rebuilds, or /design rebuild for a full pass.
```

## Principles

- Git-only. No reading source files, no LLM calls. This mode is meant to be cheap enough to run on every PR.
- Report rather than act. The decision to rebuild belongs to the user, or to `/design rebuild` when it is explicitly invoked.
- NO MODEL is not an error. See the freshness model in `SKILL.md`.
