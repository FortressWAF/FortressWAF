# FortressWAF Architecture

## Overview

FortressWAF operates as a **reverse proxy** that intercepts HTTP/HTTPS traffic, inspects requests through a pipeline, and forwards them to upstream origin servers.

## Proxy Node Pipeline

```
Request → TLS Term → HTTP Parse → Rate Limiter → IP Reputation → Rule Engine → Decision → Proxy Forwarder → Origin
```

### Stage 1: TLS Termination

- TLS 1.2 minimum, TLS 1.3 supported
- mTLS support for API-to-API communication
- No automatic certificate management; provide certificates via config

### Stage 2: HTTP Parsing

- HTTP/1.1 support
- URL normalization and path canonicalization
- Body parsing for `application/x-www-form-urlencoded` and `multipart/form-data`
- Header size and count limits enforced
- HTTP/2 supported via TLS configuration; HTTP/3 not currently supported

### Stage 3: Rate Limiting

- Token bucket and leaky bucket algorithms
- Per-IP and per-route limits
- Global rate limits
- Returns 429 with `Retry-After` header

### Stage 4: IP Reputation

- TOR exit node detection
- Known proxy/VPN detection via range lists
- Datacenter IP detection
- Custom allow/block lists
- Reputation scoring (0-100)
- No real-time threat feed integration

### Stage 5: Rule Engine

Evaluates requests against a configurable rule set. Each rule specifies:

- Conditions (pattern match, regex, prefix/suffix, CIDR)
- Operators (AND, OR, NOT)
- Actions (block, allow, challenge, log, rate-limit)
- Severity levels (critical, high, medium, low, info)

Available rule phases:
- **Phase 1**: Request headers (User-Agent, Referer, Authorization, Cookies)
- **Phase 2**: Request path and query parameters
- **Phase 3**: Request body

Response inspection is implemented. The `ResponseInspector` middleware wraps the HTTP response writer and captures response bodies for analysis when enabled.

### Stage 6: ML Inference (Enterprise, optional)

The ML sidecar runs as a separate process and provides:

- **Anomaly Scoring**: Scores requests 0.0-1.0 based on deviation from learned baseline
- **Feature Extraction**: Request structure and character distribution features
- **Model**: Single model (not ensemble). Model type depends on configuration.
- **Threshold**: Configurable (default 0.7)
- Training data is static; no continuous learning

### Stage 7: Decision Engine

```
1. If allowlist match → ALLOW
2. If blocklist match → BLOCK (403)
3. If rate limit exceeded → BLOCK (429)
4. If ML score > threshold (configured to block) → BLOCK (403)
5. If rule matches with action=block → BLOCK (403)
6. If rule matches with action=challenge → CHALLENGE
7. Otherwise → ALLOW (forward to upstream)
```

### Stage 8: Proxy Forwarder

- Connection pooling with keep-alive
- Forwards requests to configured upstream
- Response streaming
- No circuit breaker or retry logic

## Data Storage

```
FortressWAF ──► Local filesystem (config files, YAML)
             ──► Redis (rate limit counters, session state, optional)
             ──► Splunk / HTTP endpoint / Slack (SIEM export, optional)
```

No built-in database (PostgreSQL, S3, etc.) is required. Config is file-based.

## Dashboard

```
Proxy Node ──WebSocket──► Dashboard (Go + React)
                           - Live metrics
                           - Attack visualization
```

## Deployment Modes

### Reverse Proxy Mode (Default)

```
Client → FortressWAF → Origin Server
```

FortressWAF sits in front of your application. All traffic passes through it.

This is the only currently supported deployment mode. Transparent bridge, sidecar, and API gateway modes are not implemented.

## Performance

Performance varies significantly by hardware, rule count, and configuration. The numbers below are rough estimates from development environments and should not be taken as guaranteed.

| Component | Approximate latency |
|-----------|-------------------|
| TLS termination | ~50μs |
| HTTP parsing | ~20μs |
| Rate limiting (local) | ~10μs |
| IP reputation | ~10μs |
| Rule engine (100 rules) | ~100μs |
| Rule engine (10K rules) | ~2ms |
| ML inference | ~5ms |
