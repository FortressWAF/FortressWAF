# PCI DSS Compliance

FortressWAF helps organizations achieve and maintain Payment Card Industry Data Security Standard (PCI DSS) compliance by providing the security controls required by the standard.

## PCI DSS 4.0 Overview

PCI DSS 4.0 includes 12 requirements organized into 6 goals:

| Goal | Requirements |
|------|--------------|
| **Goal 1: Network Security** | Req 1, 2 |
| **Goal 2: Data Protection** | Req 3, 4 |
| **Goal 3: Vulnerability Management** | Req 5, 6 |
| **Goal 4: Access Control** | Req 7, 8, 9 |
| **Goal 5: Monitoring & Testing** | Req 10, 11 |
| **Goal 6: Documentation** | Req 12 |

## FortressWAF Controls Mapping

### Requirement 1: Firewall Configuration

**Objective:** Protect cardholder data through network security controls.

| Sub-Requirement | FortressWAF Control | Evidence |
|----------------|---------------------|----------|
| 1.2.1 | WAF provides inline inspection of all traffic | Traffic inspection logs |
| 1.3.2 | WAF blocks unauthorized traffic | Block/allow rules |
| 1.3.3 | DMZ architecture supported | Deployment documentation |
| 1.4 | Anti-spoofing measures | IP reputation blocking |

### Requirement 2: Default Credentials

**Objective:** Eliminate default credentials and configurations.

| Sub-Requirement | FortressWAF Control | Evidence |
|----------------|---------------------|----------|
| 2.1 | Password change enforcement | Admin password policies |
| 2.2 | Configuration standards | Configuration documentation |
| 2.3 | Encryption for admin access | TLS enforcement |

**FortressWAF Implementation:**

```bash
# Enforce strong admin passwords
fortressctl settings update --admin-password-min-length 12 --password-complexity

# Disable default credentials
fortressctl users update admin --force-password-change
```

### Requirement 3: Protect Stored Cardholder Data

**Objective:** Protect stored cardholder data.

| Sub-Requirement | FortressWAF Control | Evidence |
|----------------|---------------------|----------|
| 3.2 | Data minimization in logs | PII masking in audit logs |
| 3.3 | Mask PAN when displayed | Response filtering |
| 3.4 | Encryption at rest | Database encryption configuration |

**FortressWAF Data Masking Configuration:**

```yaml
# Audit logging with PII masking
audit:
  enabled: true
  redact_fields:
    - credit_card
    - cvv
    - password
  # Automatic PAN detection and masking
  pan_masking:
    enabled: true
    mask_format: "XXXX-XXXX-XXXX-####"  # Last 4 digits visible
```

### Requirement 4: Encrypt Cardholder Data in Transit

**Objective:** Protect cardholder data during transmission over networks.

| Sub-Requirement | FortressWAF Control | Evidence |
|----------------|---------------------|----------|
| 4.1 | Strong TLS encryption | TLS 1.2+ enforcement |
| 4.2 | No unsecured PAN transmission | HTTPS enforcement |

**FortressWAF TLS Configuration:**

```yaml
server:
  tls:
    enabled: true
    min_version: "1.2"
    cipher_suites:
      - TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
      - TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
    ocsp_stapling: true

# Force HTTPS
rules:
  - name: Enforce HTTPS
    condition:
      request.scheme: equals "http"
    action:
      type: redirect
      status: 301
      redirect_url: "https://${request.host}${request.uri}"
```

### Requirement 6: Develop and Maintain Secure Systems

**Objective:** Establish processes to identify and remediate vulnerabilities.

| Sub-Requirement | FortressWAF Control | Evidence |
|----------------|---------------------|----------|
| 6.2 | OWASP Top 10 protection | SQLi, XSS, RCE rule sets |
| 6.3 | Code security review | Virtual patching |
| 6.4 | Automated attack detection | Real-time threat blocking |
| 6.5 | Injection/XSS prevention | Rule engine + ML detection |

**FortressWAF OWASP Protection:**

```yaml
rules:
  owasp_enabled: true

  # SQL Injection Protection
  sql_injection:
    enabled: true
    threshold: 0.75
    block_on_match: true

  # XSS Protection
  xss:
    enabled: true
    threshold: 0.8
    block_on_match: true

  # Command Injection Protection
  command_injection:
    enabled: true
    threshold: 0.7
    block_on_match: true
```

### Requirement 7: Restrict Access by Need-to-Know

**Objective:** Limit access to cardholder data to authorized personnel.

| Sub-Requirement | FortressWAF Control | Evidence |
|----------------|---------------------|----------|
| 7.1 | Principle of least privilege | Role-based access control |
| 7.2 | User access rights | API key permissions |
| 7.3 | Role assignments | Admin role configuration |

**FortressWAF RBAC Configuration:**

```yaml
# Admin role definitions
admin:
  permissions:
    - sites:read
    - sites:write
    - rules:read
    - rules:write
    - certificates:manage
    - users:manage

  # API key permissions
api_key_permissions:
  read_only:
    - sites:read
    - stats:read
    - events:read
  full_access:
    - "*"
```

### Requirement 8: Authenticate Access

**Objective:** Authenticate all access to system components.

| Sub-Requirement | FortressWAF Control | Evidence |
|----------------|---------------------|----------|
| 8.1 | Unique user IDs | User management |
| 8.2 | MFA for admin access | MFA enforcement |
| 8.3 | MFA for all access | MFA for API and dashboard |
| 8.4 | Session timeout | Session management |
| 8.5 | API authentication | API key + JWT authentication |

**FortressWAF MFA Configuration:**

```yaml
# Require MFA for admin access
admin:
  mfa_required: true
  mfa_methods:
    - totp
    - hardware_key
    - sms  # Not recommended for high security

# Session configuration
session:
  timeout: 30m
  absolute_timeout: 8h
  idle_timeout: 15m
  max_concurrent: 1
```

### Requirement 10: Log and Monitor All Access

**Objective:** Track and monitor all access to network resources and cardholder data.

| Sub-Requirement | FortressWAF Control | Evidence |
|----------------|---------------------|----------|
| 10.1 | Audit trail | Event logging |
| 10.2 | Automatic audit trails | All actions logged |
| 10.3 | User identification | Audit logs with user ID |
| 10.4 | Time synchronization | Synchronized timestamps |
| 10.5 | Secure audit trails | Immutable logs |
| 10.6 | Log review | Analytics dashboard |
| 10.7 | Log retention | Configurable retention |

**FortressWAF Audit Configuration:**

```yaml
audit:
  enabled: true
  log_all_requests: false
  log_blocked_requests: true
  log_suspicious_requests: true
  log_request_body: false
  log_response_body: false
  retention_days: 365  # PCI DSS requirement: 1 year

  # Time synchronization
  timestamps:
    timezone: UTC
    format: ISO8601
    ntp_servers:
      - time.google.com
      - time.cloudflare.com

  # Immutable logging
  immutable:
    enabled: true
    destination: "s3://bucket/audit-logs/"
    encryption: AES256
```

### Requirement 11: Regular Testing

**Objective:** Regularly test security systems and processes.

| Sub-Requirement | FortressWAF Control | Evidence |
|----------------|---------------------|----------|
| 11.2 | Vulnerability scanning | Integration with vulnerability scanners |
| 11.3 | Penetration testing | WAF bypass testing |
| 11.4 | Intrusion detection | Real-time alerting |
| 11.5 | Change detection | File integrity monitoring |

**FortressWAF Virtual Patching:**

```bash
# Import vulnerability scan results
fortressctl virtual-patches import my-site \
  --scanner nessus \
  --file scan-report.nessus \
  --auto-enable critical
```

## Compliance Reporting

### Generate PCI DSS Compliance Report

```bash
# Generate comprehensive compliance report
fortressctl compliance report \
  --standard pci-dss \
  --site my-site \
  --period 90d \
  --format pdf \
  --output pci-compliance-report.pdf
```

### Report Contents

The compliance report includes:

1. **Executive Summary**
   - Compliance status
   - Risk assessment
   - Remediation recommendations

2. **Evidence by Requirement**
   - Requirement description
   - Control implementation
   - Supporting evidence
   - Pass/Fail status

3. **Audit Trail Samples**
   - Sample blocked attacks
   - Sample administrative actions
   - User access events

4. **Vulnerability Coverage**
   - Protected CVE list
   - Virtual patches active
   - Rule effectiveness metrics

### Automated Evidence Collection

```yaml
# Configure automated evidence collection
compliance:
  pci_dss:
    enabled: true
    collection:
      # Daily collection
      frequency: daily
      # Evidence types to collect
      evidence_types:
        - audit_logs
        - rule_statistics
        - blocked_attacks
        - configuration_snapshots
        - vulnerability_scans
      # Retention
      retention_days: 365
```

## Configuration Checklist

### Network Security

- [ ] WAF deployed in inline mode
- [ ] All traffic routes through WAF
- [ ] IP blocklists configured
- [ ] Rate limiting enabled
- [ ] DDoS protection enabled

### Encryption

- [ ] TLS 1.2+ enforced
- [ ] Weak ciphers disabled
- [ ] Self-signed certs not used in production
- [ ] Certificate monitoring enabled

### Access Control

- [ ] Admin accounts use strong passwords
- [ ] MFA enabled for all admin access
- [ ] API keys use appropriate permissions
- [ ] Sessions timeout configured
- [ ] Default admin password changed

### Logging and Monitoring

- [ ] Audit logging enabled
- [ ] Logs retained for 1 year
- [ ] Logs protected from modification
- [ ] Alerts configured for suspicious activity
- [ ] Regular log review process

### Vulnerability Protection

- [ ] OWASP Top 10 rules enabled
- [ ] SQL injection protection enabled
- [ ] XSS protection enabled
- [ ] Virtual patching configured
- [ ] Regular rule updates

## Audit Trail Requirements

PCI DSS requires specific audit trail information. FortressWAF captures:

```json
{
  "event": {
    "timestamp": "2024-01-15T12:30:00Z",
    "event_type": "attack.blocked",
    "user_identity": {
      "id": "user-uuid",
      "type": "api_key",
      "name": "CI/CD Key"
    },
    "client": {
      "ip": "1.2.3.4",
      "user_agent": "Mozilla/5.0...",
      "geo": {
        "country": "US",
        "city": "New York"
      }
    },
    "request": {
      "method": "POST",
      "path": "/api/checkout",
      "headers": {...},
      "body_hash": "sha256:..."
    },
    "response": {
      "status": 403,
      "size": 256
    },
    "decision": {
      "action": "block",
      "rule_id": "rule-uuid",
      "rule_name": "SQL Injection Block",
      "attack_type": "sql_injection"
    }
  }
}
```

## Continuous Compliance Monitoring

### Real-time Dashboard

```
┌─────────────────────────────────────────────────────────────────────┐
│                    PCI DSS COMPLIANCE STATUS                          │
├─────────────────────────────────────────────────────────────────────┤
│  Overall Status: ✓ COMPLIANT                                        │
│  Last Assessment: 2024-01-15 10:00 UTC                              │
│  Next Assessment: 2024-04-15 10:00 UTC                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Requirement      Status    Evidence     Last Check                  │
│  ──────────────────────────────────────────────────────────────     │
│  Req 1 (Firewall)    ✓ PASS    Logs        2024-01-15              │
│  Req 2 (Defaults)    ✓ PASS    Config      2024-01-15              │
│  Req 3 (Data Prot)   ✓ PASS    Audit      2024-01-15              │
│  Req 4 (Encryption)  ✓ PASS    Config      2024-01-15              │
│  Req 6 (Vuln Mgmt)   ✓ PASS    Rules       2024-01-15              │
│  Req 7 (Access)      ✓ PASS    RBAC        2024-01-15              │
│  Req 8 (Auth)        ✓ PASS    MFA         2024-01-15              │
│  Req 10 (Logging)    ✓ PASS    Audit       2024-01-15              │
│  Req 11 (Testing)    ✓ PASS    Reports     2024-01-15              │
│                                                                      │
├─────────────────────────────────────────────────────────────────────┤
│  ATTACKS BLOCKED (Last 24h):                                        │
│  ──────────────────────────────────────────────────────────────     │
│  SQL Injection:     1,234    ████████████████████                  │
│  XSS:                 567    ████████████                            │
│  Command Inj:          89    ██                                       │
│  DDoS:                 45    █                                        │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Automated Alerts

```yaml
alerts:
  pci_dss:
    - name: compliance_threshold_breach
      condition: compliance_score < 0.95
      severity: critical
      channels: [email, slack, pagerduty]

    - name: audit_log_gap
      condition: audit_log_gap_hours > 1
      severity: high
      channels: [email]

    - name: certificate_expiring
      condition: cert_expiry_days < 30
      severity: warning
      channels: [slack]
```

## Integration with QSA

### Qualified Security Assessor (QSA) Access

```bash
# Generate read-only access for QSA
fortressctl users create \
  --name "QSA Audit Account" \
  --email qsa@example.com \
  --role viewer \
  --expires 2024-06-30

# Generate compliance evidence package
fortressctl compliance export \
  --site my-site \
  --period 90d \
  --include-evidence true \
  --password-protect \
  --output qsa-evidence-package.zip
```

### Custom Reporting Periods

```bash
# Generate report for specific assessment period
fortressctl compliance report \
  --site my-site \
  --start 2023-10-01 \
  --end 2023-12-31 \
  --include-evidence \
  --format pdf
```
