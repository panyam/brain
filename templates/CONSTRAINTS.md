# Constraints

> Architectural rules for this project. Validated by `/stack-audit`.
> Component-level constraints (if any) are in each component's CONSTRAINTS.md and checked automatically.

## Stack-Wide Constraints

### SHA-Pinned CI and Goreleaser Releases
**Rule**: All Go binaries must use goreleaser for cross-platform releases. GitHub Actions must be pinned by commit SHA (not version tags). Release workflows must run tests before building.
**Why**: Version-tag-based Actions are vulnerable to supply chain attacks (e.g. tj-actions/changed-files, March 2025) where attackers replace a tag to exfiltrate secrets. SHA pinning is immutable.
**Verify**: `grep -rn 'uses:' .github/workflows/ | grep -v '#' | grep '@v'` — should return nothing (all actions pinned by SHA with version in comment).
**Scope**: All `.github/workflows/*.yml` files.

## Project-Specific Constraints

<!-- Add project-specific constraints below in this format: -->
<!-- ### {Short Rule Name} -->
<!-- **Rule**: {What must or must not happen} -->
<!-- **Why**: {The incident, design decision, or reasoning behind this rule} -->
<!-- **Verify**: {How to check — grep pattern, test command, lint rule, or "manual"} -->
<!-- **Scope**: {project-wide | specific files/dirs | specific component usage} -->

<!-- To promote a constraint to a component: move it to that component's CONSTRAINTS.md -->
<!-- and replace it here with: "See {component}/CONSTRAINTS.md: {rule name}" -->
