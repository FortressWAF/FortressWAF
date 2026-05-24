# SIEM Integration

FortressWAF exports security events to SIEM platforms for centralized monitoring, alerting, and compliance auditing.

## Architecture

```
Request --> FortressWAF --> Decision Log --> SIEM Buffer --> Batch Export --> SIEM Platform
                                                    |
                                              (Splunk / Elasticsearch)
```

Events are buffered in memory and exported in batches to avoid overwhelming your SIEM infrastructure.

## Configuration

```yaml
siem:
  enabled: true
  export_interval: 10s        # How often to flush events
  batch_size: 100             # Events per batch
  exporters:
    - type: splunk
      enabled: true
      url: "https://hec.splunk.example.com:8088/services/collector"
      token: "your-hec-token"
      index: "fortresswaf"
      verify_ssl: true
    - type: elasticsearch
      enabled: false
      url: "https://elastic.example.com:9200"
      username: "elastic"
      password: "your-password"
      index: "fortresswaf"
```

## Splunk (HEC)

### Configuration

```yaml
exporters:
  - type: splunk
    enabled: true
    url: "https://hec.splunk.example.com:8088/services/collector"
    token: "Splunk 00000000-0000-0000-0000-000000000000"
    index: "fortresswaf"
    verify_ssl: true
```

### Event Format

Events are sent as Splunk HEC JSON format:

```json
{
  "event": {
    "timestamp": "2026-05-24T12:00:00Z",
    "event_type": "waf_block",
    "host": "api.example.com",
    "src_ip": "203.0.113.1",
    "attack_type": "sqli",
    "rule_id": "SQLI-001",
    "threat_score": 95,
    "blocked": true
  },
  "host": "api.example.com",
  "index": "fortresswaf",
  "time": 1716541200
}
```

### Splunk Queries

```
# Top attack types
index="fortresswaf" | stats count by attack_type

# Top attacking IPs
index="fortresswaf" blocked=true | stats count by src_ip | sort -count

# Attack timeline
index="fortresswaf" | timechart count by attack_type

# Recent blocks
index="fortresswaf" blocked=true | table _time, src_ip, attack_type, rule_id, threat_score
```

### Alert Examples

```
# High severity alert
index="fortresswaf" threat_score>=90 | alert

# Brute force detection
index="fortresswaf" attack_type="credential_stuffing" | stats count by src_ip | where count > 10
```

## Elasticsearch

### Configuration

```yaml
exporters:
  - type: elasticsearch
    enabled: true
    url: "https://elastic.example.com:9200"
    username: "elastic"
    password: "your-password"
    index: "fortresswaf"
```

### Bulk Format

Events are sent using the Elasticsearch bulk API with daily index rotation (`fortresswaf-2026.05.24`).

### Kibana Visualizations

Create dashboards for:

- **Attack Map**: Geo-map of blocked requests by source IP
- **Threat Score Distribution**: Histogram of threat scores
- **Top Rules Triggered**: Bar chart of most frequent rule IDs
- **Real-time Attack Feed**: Table of recent blocks with details
- **Protocol Breakdown**: Pie chart of attack types

## CEF Format

For SIEMs supporting Common Event Format:

```
CEF:0|FortressWAF|FortressWAF|RCE-001|95|Very High
  src=203.0.113.1 dst=api.example.com
  requestMethod=POST request=/api/login
  attackType=sqli ruleId=SQLI-001
```

## Event Schema

| Field | Type | Description |
|-------|------|-------------|
| `timestamp` | ISO8601 | Event time |
| `event_type` | string | `waf_block`, `waf_monitor`, `rate_limit`, `bot_detect` |
| `host` | string | Target hostname |
| `src_ip` | string | Client IP address |
| `dst_ip` | string | Server IP address |
| `http_method` | string | HTTP method |
| `http_uri` | string | Request URI |
| `user_agent` | string | User-Agent header |
| `attack_type` | string | `sqli`, `xss`, `rce`, `bot`, `ddos`, etc. |
| `rule_id` | string | Triggered rule identifier |
| `threat_score` | float | 0-100 threat score |
| `blocked` | bool | Whether request was blocked |
| `country` | string | Source country (if GeoIP enabled) |
| `latency_ms` | float | Processing latency |

## Best Practices

- Set `export_interval` based on your event volume (10-30s typical)
- Adjust `batch_size` to match your SIEM's ingestion limits
- Use separate indexes for production vs staging environments
- Enable `verify_ssl` in production
- Rotate HEC tokens and Elasticsearch passwords regularly
- Monitor SIEM exporter errors in FortressWAF logs
