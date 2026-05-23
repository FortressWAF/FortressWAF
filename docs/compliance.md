# Compliance Documentation

FortressWAF helps organizations meet security requirements for major compliance frameworks. This document maps FortressWAF capabilities to specific control requirements.

---

## PCI DSS v4.0

FortressWAF helps satisfy the following Payment Card Industry Data Security Standard requirements:

| Requirement | Control | How FortressWAF Helps |
|-------------|---------|----------------------|
| **6.4.2** | Automated technical solution for public-facing web applications | WAF protects all public-facing web apps and APIs from OWASP Top 10 attacks |
| **6.4.3** | Manage all changes to existing WAF rules | API-driven rule management with audit logging of all rule changes |
| **6.5.1** | Protect against injection attacks (SQLi, XSS, etc.) | Rule engine with 10,000+ pre-built rules covering SQLi, XSS, RCE, LFI, SSRF |
| **6.5.2** | Handle all web-based attacks | ML anomaly detection catches zero-day and novel attacks |
| **11.5.1.1** | Intrusion detection/prevention for web applications | Real-time attack detection and blocking with alerting |
| **12.3.1** | Security awareness training - attack simulation | Attack corpus with 200+ labeled payloads for testing |
| **12.8** | Service provider oversight | Audit trails, compliance reports, and SOC 2 Type II attestation available (Enterprise) |

### Enabling PCI DSS Mode

```yaml
# /etc/fortresswaf/config.yaml
compliance:
  mode: pci-dss
  audit_logging: true
  retention_days: 365
  encryption_at_rest: true

rules:
  profiles:
    - compliance/pci
    - owasp-top-10
```

### PCI DSS Reporting

```bash
fortressctl compliance report --framework pci-dss --period 90d
```

Generates a PDF report with:

- WAF coverage map against PCI DSS requirements
- Attack statistics and blocked threats
- Rule change audit log
- Uptime and availability metrics
- Incident response timeline

---

## GDPR

FortressWAF supports General Data Protection Regulation compliance:

| Article | Requirement | How FortressWAF Helps |
|---------|-------------|----------------------|
| **Art. 5(1)(c)** | Data minimization | Configurable logging - log only required fields, mask/truncate sensitive data |
| **Art. 5(1)(e)** | Storage limitation | Configurable log retention (default 90 days), automatic purging |
| **Art. 17** | Right to erasure | API endpoint to purge all data for a specific user/IP |
| **Art. 25** | Data protection by design | WAF sits between client and origin, reducing data exposure |
| **Art. 30** | Records of processing | Full audit logging of all configuration changes |
| **Art. 32** | Security of processing | Encryption in transit (TLS 1.3), at-rest encryption for logs, access controls |
| **Art. 33** | Data breach notification | Real-time alerting on security events |

### GDPR Configuration

```yaml
# /etc/fortresswaf/config.yaml
compliance:
  mode: gdpr
  data_protection:
    log_masking:
      enabled: true
      fields:
        - email
        - phone
        - ssn
        - credit_card
    ip_anonymization:
      enabled: true
      method: truncate  # truncate / hash / drop
      truncate_prefix: /24
    retention:
      access_logs: 90d      # Auto-purge after 90 days
      audit_logs: 365d
      attack_logs: 180d
    right_to_erasure:
      enabled: true
      purge_api: true
```

### Data Erasure API

```bash
# Request data erasure for a specific user
curl -X POST -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{"identifier": "user@example.com", "reason": "GDPR Art. 17 request", "request_id": "REQ-2024-001"}' \
  http://localhost:8080/api/v1/compliance/gdpr/erasure
```

---

## HIPAA

FortressWAF assists with Health Insurance Portability and Accountability Act compliance:

| Standard | Section | How FortressWAF Helps |
|----------|---------|----------------------|
| **Security Management** | 164.308(a)(1) | Risk analysis through attack visibility and reporting |
| **Security Awareness** | 164.308(a)(5) | Attack simulation with test corpus |
| **Access Control** | 164.312(a)(1) | Granular API access controls with token-based auth |
| **Audit Controls** | 164.312(b) | Comprehensive audit logging of all events |
| **Integrity Controls** | 164.312(c)(1) | Prevents data modification via injection attacks |
| **Transmission Security** | 164.312(e)(1) | TLS 1.2/1.3, mTLS support |
| **Contingency Plan** | 164.308(a)(7) | High availability with multi-node deployment |
| **Facility Access** | 164.310(a)(1) | Role-based access control (Enterprise) |

### HIPAA Configuration

```yaml
# /etc/fortresswaf/config.yaml
compliance:
  mode: hipaa
  hipaa:
    phi_detection: true       # Detect PHI in request/response bodies
    phi_fields:                # Protected Health Information patterns
      - ssn
      - mrn
      - dob
      - diagnosis_code
      - patient_id
    log_phi_access: true       # Log any access to endpoints containing PHI
    brute_force_protection:
      enabled: true
      max_attempts: 5
      window: 300s
    session_timeout: 900       # 15 minute idle timeout
    encryption:
      tls_min_version: "1.2"
      ciphers:
        - TLS_AES_256_GCM_SHA384
        - TLS_CHACHA20_POLY1305_SHA256
```

---

## SOC 2

FortressWAF supports SOC 2 Trust Services Criteria:

| Category | Criteria | How FortressWAF Helps |
|----------|----------|----------------------|
| **Security** | CC6.1 - Logical and physical access | API authentication, IP allowlisting, rate limiting |
| **Security** | CC6.6 - Prevent/detect malware | ML anomaly detection, rule engine blocks malicious payloads |
| **Security** | CC7.1 - Detect security events | Real-time detection, alerting, dashboard |
| **Security** | CC7.2 - Respond to incidents | Incident investigation through detailed attack logs |
| **Availability** | A1.1 - Maintain operations | High availability deployment, health checks, circuit breaker |
| **Availability** | A1.2 - Monitor availability | Prometheus metrics, uptime monitoring, alerting |
| **Confidentiality** | C1.1 - Protect confidential info | Data loss prevention (DLP) through response inspection |
| **Processing Integrity** | PI1.1 - Complete/accurate processing | Request validation, schema enforcement |

---

## ISO 27001

| Annex A Control | Name | How FortressWAF Helps |
|-----------------|------|----------------------|
| **A.8.7** | Protection against malware | ML-based malware detection in uploaded files |
| **A.8.16** | Monitoring activities | Comprehensive logging and metrics |
| **A.8.20** | Networks security | Network-level attack prevention |
| **A.8.24** | Use of cryptography | TLS termination with strong ciphers |
| **A.8.25** | Secure development | Virtual patching without code changes |
| **A.8.28** | Information security testing | Built-in attack corpus for testing |
| **A.8.29** | Security testing in acceptance | CI/CD integration for automated security testing |

---

## FIPS 140-2 (Enterprise)

FortressWAF Enterprise supports FIPS 140-2 validated cryptographic modules:

```yaml
# /etc/fortresswaf/config.yaml
compliance:
  mode: fips-140-2
  fips:
    enabled: true
    crypto_module: /usr/lib64/libfortress-fips.so
    tls_min_version: "1.2"
    allowed_ciphers:
      - TLS_AES_256_GCM_SHA384
      - TLS_AES_128_GCM_SHA256
    key_derivation: kbkdf
    rng: drbg_ctr_aes256
```

---

## Compliance Automation

### Generate Compliance Reports

```bash
# Generate all compliance reports
fortressctl compliance report --all --period 90d --output ./reports/

# Generate specific framework report
fortressctl compliance report --framework pci-dss --period 365d

# Schedule daily compliance checks
fortressctl compliance schedule --interval daily --notify admin@example.com
```

### Audit Log Export

```bash
# Export audit logs for compliance review
fortressctl audit export \
  --since "2024-01-01" \
  --until "2024-03-15" \
  --format csv \
  --output audit-export-q1-2024.csv
```

### Compliance Dashboard

The FortressWAF dashboard includes a compliance module (Enterprise) showing:

- Compliance score by framework
- Control coverage map
- Recent audit events
- Open compliance gaps
- Remediation recommendations

---

## Regional Compliance

| Region | Regulation | Supported |
|--------|-----------|-----------|
| EU | GDPR | ✅ |
| US (Healthcare) | HIPAA | ✅ |
| US (Payments) | PCI DSS | ✅ |
| Global | SOC 2 | ✅ |
| Global | ISO 27001 | ✅ |
| US (Federal) | FIPS 140-2 | Enterprise |
| China | PIPL | Enterprise |
| Brazil | LGPD | Enterprise |
| Japan | APPI | Enterprise |
| Australia | Privacy Act | Enterprise |
