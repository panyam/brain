# Security Audit

Run a comprehensive security audit on the current project. Combines automated tool scanning with AI-assisted threat model review to find vulnerabilities, gaps in test coverage, and hardening opportunities.

## When to Use

- Before a release or deployment
- After adding new dependencies
- After adding new HTTP endpoints or auth flows
- Quarterly review
- When the user asks to check security posture

## Steps

### Phase 1: Automated Scanning

Run the project's `make audit` if it exists, otherwise run each tool individually:

```bash
# Check if make audit exists
grep -q '^audit:' Makefile && make audit

# Or run individually:
govulncheck ./...                    # Known CVEs in dependencies
gosec -quiet -severity=high ./...    # Code-level anti-patterns
gitleaks detect --source . -v        # Accidentally committed secrets
```

If any tool isn't installed, note it as a gap and continue.

For each finding:
- **Critical** (govulncheck with symbol match): fix immediately, upgrade the dependency
- **High** (gosec): evaluate if it's a real issue or false positive — check the code context
- **Informational** (gosec medium/low): note for review, don't block

### Phase 2: Dependency Audit

1. Read `go.mod` (or `package.json` for Node projects)
2. Check for:
   - Any new transitive dependencies since last audit
   - Dependencies with known maintainer compromise history
   - Pinned vs floating versions
   - For GitHub Actions: are they pinned by SHA? (CVE-2025-30066)
3. Run `stack-brain stale .` to check if stack components are up to date

### Phase 3: Threat Model Review

For each category below, read the relevant code and assess:

**Authentication Entry Points**
- List all HTTP endpoints that handle auth (login, signup, token exchange, OAuth callback)
- For each: what untrusted input reaches it? What validates that input?
- Check: rate limiting configured? CSRF protection? Body size limits?
- Check: timing oracle prevention? (bcrypt dummy hash for non-existent users)

**JWT / Token Security**
- Are tokens signed with adequate key sizes? (RSA ≥ 2048, HS256 key ≥ 32 bytes)
- Is the `alg` header validated against the expected algorithm? (prevents CVE-2015-9235)
- Is `aud` (audience) validated? `iss` (issuer)?
- Do tokens include `jti` for revocation support?
- Is there a token blacklist? If not, what's the max exposure window?
- Are refresh tokens rotated on use? Is reuse detected? Does reuse revoke the family?

**Key Management**
- Where are signing keys stored? Encrypted at rest?
- Are HS256 secrets ever exposed in JWKS? (must be asymmetric-only)
- Is there key rotation support? Grace period for old keys?
- Are `kid` (Key ID) headers in all JWTs?

**Input Validation**
- Are all file paths sanitized? (path traversal: `../`, null bytes, absolute paths)
- Are request body sizes limited? (DoS via oversized JSON)
- Are all database queries parameterized? (SQL injection)
- Is user-generated content HTML-escaped? (XSS)

**HTTP Security Headers**
- HSTS (Strict-Transport-Security)
- X-Content-Type-Options: nosniff
- X-Frame-Options: DENY (clickjacking)
- Content-Security-Policy
- Referrer-Policy

**OAuth / PKCE**
- Is PKCE enabled for all OAuth flows? (required by OAuth 2.1)
- Is the state cookie short-lived? (< 10 minutes, not 30 days)
- Is the OAuth callback validated against the registered redirect URI?

**File Permissions** (for FS-backed stores)
- Directories: 0700 (owner-only)
- Data files: 0600 (owner read/write only)
- Key files: never world-readable

### Phase 4: Test Coverage Check

For each security property identified in Phase 3:
1. Search for a corresponding test (`grep -r "// See:" --include="*_test.go"`)
2. Check that the test has a `// See:` link to the relevant RFC, CVE, CWE, or OWASP reference
3. Flag any security property without a test as a **gap**

Convention: security tests should include reference links:
```go
// See: https://nvd.nist.gov/vuln/detail/CVE-2015-9235
// See: https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.3
// See: https://cwe.mitre.org/data/definitions/284.html
```

### Phase 5: Report

Output a structured report:

```markdown
## Security Audit Report — {project name}
Date: {date}

### Critical Findings
(Fix immediately — known vulnerabilities with active exploits)

### High Findings
(Fix soon — real security issues)

### Gaps
(Security properties without test coverage)

### Recommendations
(Hardening opportunities, not blocking)

### Dependencies
(Outdated, newly added, or risky transitive deps)

### Pass
(What's working well — acknowledge good practices)
```

For each finding, include:
- What the issue is
- Why it matters (attack scenario)
- How to fix it
- Reference (CVE/CWE/RFC/OWASP link)
- Suggested test to prove the fix

### Phase 6: File Issues

For Critical and High findings:
- Create GitHub issues with the `security` label
- Include the attack scenario and fix recommendation
- Reference the audit report

## Supported Project Types

- **Go**: `govulncheck`, `gosec`, `gitleaks`, `go test -race`
- **Node**: `npm audit`, `eslint-plugin-security` (if available)
- **Python**: `safety check`, `bandit` (if available)
- **All**: `gitleaks` for secret scanning

## References

- OWASP Top 10: https://owasp.org/www-project-top-ten/
- OWASP JWT Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html
- CWE/SANS Top 25: https://cwe.mitre.org/top25/
- RFC 7519 (JWT): https://datatracker.ietf.org/doc/html/rfc7519
- RFC 7636 (PKCE): https://datatracker.ietf.org/doc/html/rfc7636
- RFC 6797 (HSTS): https://datatracker.ietf.org/doc/html/rfc6797
