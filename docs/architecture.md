# FortressWAF Architecture

## Overview

FortressWAF operates as a **reverse proxy** that intercepts HTTP/HTTPS traffic, inspects requests through a multi-layered pipeline, and forwards legitimate traffic to upstream origin servers. The system is designed for **high throughput**, **low latency**, and **horizontal scalability**.

## High-Level Architecture

```
                              ┌──────────────────────────────────┐
                              │         Internet / Clients       │
                              └───────────────┬──────────────────┘
                                              │
                                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Global Load Balancer                               │
│                       (AWS ALB / GCP HTTP LB / HAProxy)                     │
└─────────────────────────────────────────────────────────────────────────────┘
                                              │
                    ┌─────────────────────────┼─────────────────────────┐
                    │                         │                         │
                    ▼                         ▼                         ▼
           ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
           │  FortressWAF     │     │  FortressWAF     │     │  FortressWAF     │
           │  Node 1          │     │  Node 2          │     │  Node N          │
           │  (Active)        │     │  (Active)        │     │  (Active)        │
           └────────┬────────┘     └────────┬────────┘     └────────┬────────┘
                    │                         │                         │
                    └─────────────────────────┼─────────────────────────┘
                                              │
                                              ▼
                              ┌──────────────────────────────┐
                              │    Backend Services          │
                              │  (App Servers / APIs / CDN)  │
                              └──────────────────────────────┘
```

## Proxy Node Internal Architecture

Each FortressWAF node contains the following components:

```
┌─────────────────────────────────────────────────────────────────────┐
│                         FortressWAF Proxy Node                       │
│                                                                     │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌──────────┐ │
│  │ TLS     │  │ HTTP    │  │ Request │  │ Rate    │  │ IP       │ │
│  │ Term.   │─▶│ Parser  │─▶│ Normal- │─▶│ Limiter │─▶│ Rep.     │ │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘  └────┬─────┘ │
│                                                            │       │
│  ┌─────────────────────────────────────────────────────────▼─────┐ │
│  │                    Rule Engine Pipeline                        │ │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐         │ │
│  │  │ SQLi     │ │ XSS      │ │ RCE/LFI  │ │ API      │         │ │
│  │  │ Detector │→│ Detector  │→│ Detector  │→│ Schema   │ ...     │ │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘         │ │
│  └──────────────────────────────────────────────────────┬────────┘ │
│                                                         │         │
│  ┌──────────────────────────────────────────────────────▼────────┐ │
│  │                ML Inference Sidecar (optional)                 │ │
│  │  ┌──────────┐    ┌──────────┐    ┌──────────┐                │ │
│  │  │ Feature  │───▶│ Model    │───▶│ Anomaly  │                │ │
│  │  │ Extractor│    │ Scorer   │    │ Decision │                │ │
│  │  └──────────┘    └──────────┘    └──────────┘                │ │
│  └──────────────────────────────────────────────────────┬────────┘ │
│                                                         │         │
│  ┌──────────────────────────────────────────────────────▼────────┐ │
│  │                    Decision Engine                             │ │
│  │            (Allow / Block / Challenge / Rate Limit)            │ │
│  └──────────────────────────────────────────────────────┬────────┘ │
│                                                         │         │
│  ┌──────────────────────────────────────────────────────▼────────┐ │
│  │                  Proxy Forwarder                               │ │
│  │  ┌──────────┐    ┌──────────┐    ┌──────────┐                │ │
│  │  │ Load     │───▶│ Connection───▶│ Response │                │ │
│  │  │ Balancer │    │ Pool     │    │ Filter   │                │ │
│  │  └──────────┘    └──────────┘    └──────────┘                │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │              Sidecar Processes                                 │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐     │  │
│  │  │ Metrics  │  │ Logging  │  │ Dashboard│  │ API      │     │  │
│  │  │ Collector│  │ Pipeline │  │ Server   │  │ Gateway  │     │  │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘     │  │
│  └──────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

## Request Inspection Pipeline

### Stage 1: TLS Termination

- TLS 1.3 preferred, TLS 1.2 minimum
- Automatic certificate management via ACME/LetsEncrypt
- Mutual TLS (mTLS) support for API-to-API communication
- Session resumption and OCSP stapling

### Stage 2: HTTP Parsing & Normalization

- Full HTTP/1.1, HTTP/2, HTTP/3 (QUIC) support
- Request normalization (URL decoding, path canonicalization, case normalization)
- Body parsing for `application/x-www-form-urlencoded`, `multipart/form-data`, JSON, XML
- Protocol validation and anomaly detection (HTTP request smuggling, splitting)
- Header size and count limits enforced

### Stage 3: Rate Limiting

- **Sliding Window**: Tracks requests per IP/route in a sliding time window
- **Token Bucket**: Burst control with configurable refill rate
- **Per-route limits**: Different limits for `/api/login` vs `/static/*`
- **Global limits**: Total requests across all routes
- **Backpressure**: Returns 429 with `Retry-After` header

### Stage 4: IP Reputation

- Real-time threat intelligence feed integration
- Geo-IP blocking
- TOR exit node detection
- Known proxy/VPN detection
- Custom allow/block lists
- Reputation scoring (0-100, lower = worse)

### Stage 5: Rule Engine

The rule engine evaluates requests against a multi-layered rule set:

- **Phase 1**: Request headers (User-Agent, Referer, Authorization, Cookies)
- **Phase 2**: Request path and query parameters
- **Phase 3**: Request body content
- **Phase 4**: Response headers and body (for data leakage prevention)

Each rule can specify:

- Conditions (pattern matching, regex, exact match, prefix/suffix)
- Operators (AND, OR, NOT)
- Actions (block, allow, challenge, log, rate-limit)
- Severity levels (critical, high, medium, low, info)

### Stage 6: ML Inference (Enterprise)

The ML sidecar runs as a separate container/process and provides:

- **Anomaly Scoring**: Requests are scored 0.0-1.0 based on deviation from learned baseline
- **Feature Extraction**: 200+ features including request structure, character distributions, n-gram analysis, header patterns
- **Model Types**: Ensemble of Random Forest, XGBoost, and a lightweight Transformer
- **Threshold**: Configurable (default 0.7). Requests above threshold are blocked or challenged
- **Continuous Learning**: Models are retrained periodically on production traffic

### Stage 7: Decision Engine

Combines outputs from all previous stages and makes a final decision:

```
Decision Logic:
1. If ANY rule matches with action=block → BLOCK (403)
2. If ML score > threshold and config says block → BLOCK (403)
3. If IP reputation score < threshold → BLOCK (403)
4. If rate limit exceeded → BLOCK (429)
5. If IP in allowlist → ALLOW (skip all checks)
6. If rule matches with action=challenge → CHALLENGE (JS challenge/CAPTCHA)
7. Otherwise → ALLOW (forward to upstream)
```

### Stage 8: Proxy Forwarder

- Connection pooling with keep-alive
- Load balancing across multiple upstreams
- Circuit breaker pattern (fail fast when upstream is unhealthy)
- Retry with exponential backoff
- Response buffering and streaming
- Response header/body inspection (DLP)

## Data Flow

```
Client ──HTTPS──► FortressWAF ──HTTP──► Origin
                     │
                     ├──► Redis (session state, rate limits)
                     │
                     ├──► PostgreSQL (config, rules, audit logs)
                     │
                     └──► S3/MinIO (access logs, attack recordings)
```

### Logging Pipeline

```
Proxy ──► Log Buffer ──► Log Processor ──► Elasticsearch ──► Kibana
                │
                └──► S3 Archive (90-day retention)
```

### Metrics Pipeline

```
Proxy ──► Prometheus ──► Grafana Dashboard
                │
                └──► Alertmanager (PagerDuty, Slack, Email)
```

## Deployment Modes

### Reverse Proxy Mode (Default)

```
Client ➔ FortressWAF ➔ Origin Server
```

FortressWAF sits directly in front of your application. All traffic passes through it.

### Transparent Bridge Mode

```
Client ➔ [FortressWAF (transparent)] ➔ Origin Server
```

FortressWAF operates as a network bridge without changing IP addresses. Requires network-level configuration.

### Sidecar Mode

```
Client ➔ Application ➔ FortressWAF (sidecar) ➔ External APIs
```

FortressWAF runs as a sidecar next to each application instance. Ideal for service mesh deployments.

### API Gateway Mode

```
Client ➔ FortressWAF (API Gateway) ➔ Microservices
     │                                    │
     └── Auth, Rate Limit, Routing ───────┘
```

FortressWAF acts as a full API gateway with routing, authentication, and rate limiting.

## Component Diagrams

### ML Engine Sidecar

```
┌──────────────┐    gRPC     ┌──────────────────────┐
│  Proxy Node  │◄───────────►│  ML Inference Sidecar │
│  (Go)        │             │  (Python/FastAPI)     │
└──────────────┘             │                        │
                              │  ┌──────────────────┐ │
                              │  │ Feature Extractor│ │
                              │  └────────┬─────────┘ │
                              │           │           │
                              │  ┌────────▼─────────┐ │
                              │  │ Model Ensemble   │ │
                              │  │ ┌──┐ ┌──┐ ┌──┐ │ │
                              │  │ │RF│ │GB│ │NN│ │ │
                              │  │ └──┘ └──┘ └──┘ │ │
                              │  └────────┬─────────┘ │
                              │           │           │
                              │  ┌────────▼─────────┐ │
                              │  │ Score Aggregator │ │
                              │  └──────────────────┘ │
                              └────────────────────────┘
```

### Dashboard Architecture

```
┌──────────────┐    WebSocket   ┌──────────────────────┐
│  Proxy Node  │───────────────►│  Dashboard Server    │
│              │ (real-time     │  (Go + React)        │
│              │  metrics)      │                        │
└──────────────┘                │  ┌──────────────────┐ │
                                │  │ WebSocket Hub    │ │
                                │  └────────┬─────────┘ │
                                │           │           │
                                │  ┌────────▼─────────┐ │
                                │  │ React Frontend   │ │
                                │  │ (Charts/Alerts)  │ │
                                │  └──────────────────┘ │
                                └────────────────────────┘
```

## Security Boundaries

| Boundary | Control |
|----------|---------|
| Internet → WAF | TLS, rate limiting, IP reputation, WAF rules |
| WAF → Origin | Internal network only, mTLS optional, source IP restriction |
| WAF → ML Sidecar | gRPC over localhost or mTLS, request payload privacy |
| WAF → Redis | AUTH, TLS, network isolation |
| WAF → PostgreSQL | TLS, certificate auth, least privilege role |
| Admin API | Bearer token, IP allowlist, audit logging |

## Performance Characteristics

| Component | Latency | Throughput (per core) |
|-----------|---------|----------------------|
| TLS termination | ~50μs | 50,000 req/s |
| HTTP parsing | ~20μs | 75,000 req/s |
| Rate limiting (Redis) | ~1ms | 30,000 req/s |
| IP reputation (local) | ~10μs | 100,000 req/s |
| Rule engine (100 rules) | ~100μs | 40,000 req/s |
| Rule engine (10K rules) | ~2ms | 8,000 req/s |
| ML inference | ~5ms | 2,000 req/s |
| Total pipeline (no ML) | ~1.5ms | 25,000 req/s |
| Total pipeline (with ML) | ~8ms | 1,500 req/s |
