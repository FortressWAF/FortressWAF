# GDPR Compliance

The General Data Protection Regulation (GDPR) requires organizations to implement appropriate technical and organizational measures to protect personal data. FortressWAF provides features to help achieve and maintain GDPR compliance.

## GDPR Overview

| Principle | Description | FortressWAF Control |
|-----------|-------------|---------------------|
| Lawfulness, fairness, transparency | Process data lawfully | Consent tracking, data processing agreements |
| Purpose limitation | Collect for specified purposes | Data minimization in logging |
| Data minimization | Collect only necessary data | PII masking, log filtering |
| Accuracy | Keep data accurate | Data update capabilities |
| Storage limitation | Don't keep longer than needed | Configurable retention, automatic deletion |
| Integrity and confidentiality | Ensure security | WAF protection, encryption |
| Accountability | Demonstrate compliance | Audit logging, compliance reports |

## Data Processing Capabilities

### Personal Data Detection

FortressWAF can detect and protect personal data in requests and responses:

```yaml
pii_detection:
  enabled: true

  # Data types to detect
  data_types:
    - name: email
      patterns:
        - regex: "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}"
        - field_names: ["email", "email_address", "e-mail"]

    - name: phone
      patterns:
        - regex: "\\+?[0-9]{1,4}?[-.\\s]?\\(?[0-9]{1,3}?\\)?[-.\\s]?[0-9]{1,4}[-.\\s]?[0-9]{1,9}"
        - field_names: ["phone", "phone_number", "mobile", "tel"]

    - name: ssn
      patterns:
        - regex: "[0-9]{3}-[0-9]{2}-[0-9]{4}"
        - field_names: ["ssn", "social_security"]

    - name: ip_address
      patterns:
        - field_names: ["client_ip", "ip", "user_ip"]
      type: automatic

    - name: name
      patterns:
        - field_names: ["name", "first_name", "last_name", "full_name"]
```

### PII Masking in Logs

```yaml
audit:
  enabled: true
  retention_days: 90

  # Automatic PII masking
  pii_masking:
    enabled: true
    mask_format: "***MASKED***"

  # Specific field masking
  mask_fields:
    - email
    - phone_number
    - ssn
    - credit_card
    - password

  # Partial masking (show last 4 digits)
  partial_mask:
    enabled: true
    format: "****${last_4}"
    fields:
      - credit_card
      - phone
```

### Request/Response Filtering

```yaml
# Remove PII from requests before logging
request_filtering:
  remove_fields:
    - password
    - credit_card_number
    - cvv
    - ssn

# Remove PII from responses
response_filtering:
  enabled: true
  remove_patterns:
    - regex: "[0-9]{3}-[0-9]{2}-[0-9]{4}"  # SSN pattern
    - regex: "bearer\\s+[a-zA-Z0-9-._~]+/+/[a-zA-Z0-9-._~]+"  # JWT tokens
```

## Data Residency

### Geographic Data Storage

```yaml
data_residency:
  # Store data only in specified regions
  enabled: true
  allowed_regions:
    - EU  # European Union

  # Log storage location
  log_storage:
    region: EU
    datacenter: Frankfurt

  # Backup storage location
  backup_storage:
    region: EU
    datacenter: Dublin
```

### Data Transfer Controls

```yaml
data_transfer:
  # Restrict cross-border data transfers
  restricted: true

  # Allow transfers only to adequate countries
  adequate_countries:
    - EU
    - UK
    - CANADA
    - JAPAN

  # Require additional safeguards for other transfers
  additional_safeguards:
    - encryption_required: true
    - contract_required: true
    - dpia_required: true
```

## Right to Erasure

### Data Deletion Capabilities

FortressWAF supports the "right to be forgotten" by enabling deletion of personal data:

```bash
# Delete all data associated with a specific user
fortressctl data-subject erase \
  --email "user@example.com" \
  --type all \
  --include-logs \
  --include-sessions \
  --include-events

# Delete specific data types
fortressctl data-subject erase \
  --email "user@example.com" \
  --type logs \
  --before "2024-01-01T00:00:00Z"
```

### Deletion Workflow

```yaml
# Configure automated deletion
data_deletion:
  enabled: true

  # Automatic deletion after retention period
  retention_policies:
    - data_type: logs
      retention_days: 90
      action: delete

    - data_type: events
      retention_days: 90
      action: delete

    - data_type: sessions
      retention_days: 30
      action: delete

    - data_type: api_keys
      retention_days: 365
      action: anonymize
```

### Deletion Certificate

When data is deleted, generate a certificate:

```bash
fortressctl data-subject certificate \
  --email "user@example.com" \
  --request-id "req-12345" \
  --output deletion-certificate.pdf
```

## Data Processing Agreements

### DPA Configuration

```yaml
data_processing_agreement:
  enabled: true

  # DPA reference
  dpa_id: "DPA-2024-001"
  dpa_version: "2.0"
  effective_date: "2024-01-01"
  expiry_date: "2026-12-31"

  # Processor obligations
  obligations:
    - Process only on documented instructions
    - Ensure confidentiality of personnel
    - Implement appropriate security measures
    - Enable data subject rights
    - Assist with data protection impact assessments
    - Delete or return data at end of agreement
```

### Sub-Processor Management

```yaml
sub_processors:
  enabled: true

  # List of approved sub-processors
  approved:
    - name: Amazon Web Services
      purpose: Cloud infrastructure
      country: Ireland
      adequacy: EU

    - name: Datadog
      purpose: Monitoring
      country: EU
      adequacy: EU
```

## Consent Management

### Consent Tracking

```yaml
consent:
  enabled: true

  # Track consent for data processing
  tracking:
    - purpose: security_logging
      description: "Security logging for threat detection"
      required: true
      legal_basis: legitimate_interest

    - purpose: analytics
      description: "Usage analytics"
      required: false
      legal_basis: consent

    - purpose: marketing
      description: "Marketing communications"
      required: false
      legal_basis: consent

  # Store consent records
  storage:
    backend: postgresql
    encryption: AES256
    retention_years: 7
```

### Cookie Consent

```yaml
# Cookie consent for web applications
cookie_consent:
  enabled: true

  # Required cookies
  necessary:
    - name: fw_session
      purpose: authentication
      duration: 24h

    - name: fw_security
      purpose: security
      duration: 1h

  # Optional cookies (require consent)
  optional:
    - name: fw_analytics
      purpose: analytics
      default: declined

    - name: fw_tracking
      purpose: marketing
      default: declined
```

## Breach Notification

### Automated Breach Detection

```yaml
breach_detection:
  enabled: true

  # PII data breach triggers
  triggers:
    - type: unauthorized_access
      description: "Unauthorized access to personal data"
      indicators:
        - failed_login_threshold: 10
        - unusual_access_patterns: true

    - type: data_exfiltration
      description: "Large-scale data access"
      indicators:
        - bulk_request_threshold: 1000
        - unusual_time_access: true

    - type: system_compromise
      description: "Potential system breach"
      indicators:
        - malware_detected: true
        - unusual_process: true
```

### Breach Notification Workflow

```yaml
breach_notification:
  enabled: true

  # Notification timeline (GDPR: 72 hours)
  timeline:
    internal_notification: immediate
    supervisory_authority: 72h
    data_subjects: 72h

  # Notification channels
  channels:
    - type: email
      recipients:
        - dpo@company.com
        - security@company.com
    - type: slack
      channel: "#gdpr-breach"
    - type: pagerduty
      severity: critical

  # Include in notification
  notification_content:
    - nature_of_breach
    - categories_affected
    - approximate_number_affected
    - likely_consequences
    - measures_taken
```

## Data Protection Impact Assessment

### DPIA Support

```bash
# Generate DPIA-relevant documentation
fortressctl compliance dpiadocs \
  --site my-site \
  --output dpiadocs.pdf
```

### Documentation Includes

1. **System Description**
   - Data processing activities
   - Data flows
   - Technology used

2. **Necessity Assessment**
   - Purpose specification
   - Data minimization justification
   - Legitimate interest assessment

3. **Risk Assessment**
   - Identified risks
   - Impact evaluation
   - Mitigation measures

4. **Consultation记录**
   - Stakeholder input
   - DPO opinion
   - Security team review

## Compliance Reporting

### GDPR Compliance Report

```bash
# Generate comprehensive GDPR report
fortressctl compliance report \
  --standard gdpr \
  --site my-site \
  --period 90d \
  --include:
    - data_processing_activities
    - consent_records
    - data_subject_requests
    - breach_incidents
    - risk_assessment
  --format pdf \
  --output gdpr-compliance-report.pdf
```

### Report Contents

```markdown
# GDPR Compliance Report
Period: 2024-01-01 to 2024-03-31
Generated: 2024-04-01

## 1. Data Processing Activities
| Activity | Data Types | Legal Basis | Volume |
|----------|------------|-------------|--------|
| Security Logging | IP, Email, User-Agent | Legitimate Interest | 15M events |
| Session Management | Session ID, IP | Contract | 500K sessions |
| Analytics | Aggregated Stats | Consent | 10K records |

## 2. Data Subject Rights
| Right | Requests Received | Requests Fulfilled | Avg Response Time |
|-------|-------------------|--------------------|--------------------|
| Access | 45 | 45 | 15 days |
| Erasure | 12 | 12 | 20 days |
| Rectification | 3 | 3 | 10 days |

## 3. Consent Management
| Consent Type | Total Consents | Withdrawn | Active |
|--------------|----------------|-----------|--------|
| Analytics | 1,000 | 150 | 850 |
| Marketing | 500 | 100 | 400 |

## 4. Data Breaches
| Date | Type | Records Affected | Notification Sent |
|------|------|-------------------|-------------------|
| None | - | - | - |

## 5. Risk Assessment
| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Unauthorized Access | Low | High | Encryption, MFA |
| Data Loss | Low | High | Backups, Replication |
```

## Configuration Checklist

### Data Minimization

- [ ] PII masking enabled in logs
- [ ] Unnecessary data fields removed
- [ ] Log retention configured
- [ ] Automatic deletion enabled

### Data Subject Rights

- [ ] Right to access configured
- [ ] Right to erasure configured
- [ ] Right to rectification configured
- [ ] Deletion workflow documented

### Consent Management

- [ ] Cookie consent enabled
- [ ] Consent records stored
- [ ] Withdrawal mechanism working
- [ ] Consent audit trail maintained

### Security Measures

- [ ] Encryption at rest enabled
- [ ] Encryption in transit enabled
- [ ] Access controls configured
- [ ] Audit logging enabled
- [ ] Breach detection configured

### Documentation

- [ ] DPA in place
- [ ] Records of processing activities maintained
- [ ] DPIA completed
- [ ] Compliance reports generated

## Data Flow Mapping

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        PERSONAL DATA FLOW                                 │
└─────────────────────────────────────────────────────────────────────────┘

  User Request
       │
       ▼
┌─────────────────┐
│    FortressWAF   │
│                 │
│  ┌───────────┐  │
│  │ PII       │  │  1. Detect PII
│  │ Detection │──┼────────────────────────────────┐
│  └───────────┘  │                                │
│       │         │                                ▼
│       ▼         │                    ┌───────────────────┐
│  ┌───────────┐  │                    │    Audit Logs     │
│  │ PII       │  │  2. Mask PII       │                   │
│  │ Masking   │──┼───────────────────▶│ - IP (masked)     │
│  └───────────┘  │                    │ - Email (masked)   │
│       │         │                    │ - Session ID       │
│       ▼         │                    └───────────────────┘
│  ┌───────────┐  │
│  │ Forward   │  │                    ┌───────────────────┐
│  │ Request   │──┼───────────────────▶│   Database        │
│  └───────────┘  │                    │                   │
│       │         │                    │ - Encrypted       │
│       ▼         │                    │ - Regional        │
│  ┌───────────┐  │                    └───────────────────┘
│  │  Backend  │  │
│  │  Response │  │
│  └───────────┘  │
│       │         │
│       ▼         │
│  ┌───────────┐  │                    ┌───────────────────┐
│  │ Response  │  │  3. Filter        │   Compliance      │
│  │ Filtering │──┼───────────────────▶│   Dashboard       │
│  └───────────┘  │                    │                   │
│                 │                    │ - Reports         │
└─────────────────┘                    │ - Metrics         │
                                        └───────────────────┘
```

## Integration with Data Protection Authorities

### Supervisory Authority Notification

```yaml
# Configure authority details
supervisory_authority:
  name: "Information Commissioner's Office"
  country: "UK"
  contact: dpo@company.com

  # Automatic notification settings
  notification:
    enabled: true
    template: "gdpr_breach_notification.txt"
    method: secure_email
```

### International Data Transfer

```yaml
# Transfer Impact Assessment
transfers:
  enabled: true

  assessments:
    - destination: "United States"
      framework: "EU-US Data Privacy Framework"
      status: adequate
      assessment_date: "2024-01-15"

    - destination: "India"
      framework: "Standard Contractual Clauses"
      status: pending
      assessment_date: "2024-03-01"
```
