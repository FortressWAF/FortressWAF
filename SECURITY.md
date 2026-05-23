# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 1.x (latest) | ✅ Full support, including security patches |
| 0.x | ❌ End of life, upgrade recommended |

We recommend all users run the latest stable release. Security patches are backported to the latest minor version only.

## Reporting a Vulnerability

We take the security of FortressWAF seriously. If you believe you have found a security vulnerability, please report it to us as described below.

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via email to:

**security@fortresswaf.io**

You should receive a response within 24 hours. If for some reason you do not, please follow up via email to ensure we received your original message.

### PGP Encryption

For sensitive information, please encrypt your report using our PGP key:

```
-----BEGIN PGP PUBLIC KEY BLOCK-----

mQINBGWo8vMBEADJ7j6V5Q1z0yFk3K5G8c3x7X9v0b2n4k5p6q7r8s9t0a1b2c3d
4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6
...
=ABCD
-----END PGP PUBLIC KEY BLOCK-----
```

Fingerprint: `A1B2 C3D4 E5F6 7890 ABCD EF12 3456 7890 ABCD EF12`

Key ID: `0xABCDEF1234567890`

Download from: `https://keys.fortresswaf.io/security.asc`

### What to Include

To help us triage and fix the issue quickly, please include:

- Type of issue (e.g., buffer overflow, SQL injection, XSS, etc.)
- Full paths of source file(s) related to the manifestation of the issue
- The location of the affected source code (tag/branch/commit or direct URL)
- Any special configuration required to reproduce the issue
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit it

### What to Expect

1. **Acknowledgment**: We will acknowledge receipt within 24 hours
2. **Triage**: We will triage the issue within 72 hours
3. **Fix**: We will develop and test a fix, typically within 7-14 days for critical issues
4. **Release**: We will release a security patch and notify you
5. **Disclosure**: We will coordinate public disclosure timing

## Responsible Disclosure Policy

We request a **90-day embargo period** from the date of the initial report. During this period:

- We will work on a fix and release
- You may monitor our progress
- No public disclosure of the vulnerability

After the embargo period (or after a fix is released, whichever is sooner), we encourage:

- Publishing a write-up of the vulnerability
- Credit in our security acknowledgments page
- CVE assignment (we will help with this process)

## Bug Bounty Program

We offer a bug bounty program for security researchers who responsibly disclose vulnerabilities.

### Scope

- FortressWAF core proxy (Go)
- ML engine (Python)
- Dashboard (React/TypeScript)
- REST API
- Configuration parsing
- Authentication/authorization mechanisms

### Out of Scope

- Clickjacking on pages with no sensitive actions
- CSRF on forms with no sensitive actions
- Missing HTTP security headers (without demonstrated impact)
- Rate limiting bypass on non-authenticated endpoints
- Social engineering attacks
- Physical attacks
- Denial of service attacks
- Third-party dependencies (report to the dependency maintainer)

### Rewards

| Severity | Reward |
|----------|--------|
| Critical (CVSS 9.0-10.0) | $5,000 |
| High (CVSS 7.0-8.9) | $2,000 |
| Medium (CVSS 4.0-6.9) | $500 |
| Low (CVSS 1.0-3.9) | $100 |

### Eligibility

- First reporter of a unique vulnerability
- Must follow responsible disclosure policy
- Must not violate any laws
- Must not access or modify data not owned by you
- Must not perform denial of service testing
- Must not use automated scanning tools without permission

## Security.txt

```
-----BEGIN SECURITY.TXT-----
Contact: mailto:security@fortresswaf.io
Expires: 2025-12-31T23:59:59Z
Encryption: https://keys.fortresswaf.io/security.asc
Acknowledgments: https://fortresswaf.io/security/hall-of-fame
Preferred-Languages: en
Policy: https://fortresswaf.io/security/policy
Canonical: https://fortresswaf.io/.well-known/security.txt
-----END SECURITY.TXT-----
```

## Security Hall of Fame

We thank the following researchers for their contributions to FortressWAF security:

- *Your name could be here!*

[Submit a report](mailto:security@fortresswaf.io) to join the hall of fame.

## Additional Security Measures

### Supply Chain Security

- All releases are signed with Cosign and our PGP key
- Docker images are signed and verified
- Go module checksums are verified
- npm packages are verified with integrity hashes
- Python packages are pinned with hashes in requirements.txt

### Runtime Security

- FortressWAF runs as a non-root user by default
- All secrets are read from files or environment variables, never passed as CLI args
- TLS certificates are automatically rotated
- Database connections use TLS with certificate verification
- Redis connections use AUTH and TLS
- Admin API requires authentication on all endpoints

### Dependency Scanning

We use automated dependency scanning:

- Go: `govulncheck` in CI
- Python: `pip-audit` in CI  
- Node: `npm audit` in CI
- Containers: Trivy scan in CI
- Daily: Dependabot alerts

## Questions

For questions about this security policy, please contact security@fortresswaf.io.
