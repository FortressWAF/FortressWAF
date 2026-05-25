# Component Architecture

This document provides in-depth coverage of FortressWAF's internal components: the detection engine, configuration system, REST API, rate limiter, IP reputation, session manager, and SIEM exporter.

## Engine

**Package**: `internal/engine/`

The engine is the core of FortressWAF. It implements a pipeline of 18 inspectors that each analyze incoming requests independently.

### Inspector Interface

```go
type Inspector interface {
    Name() string
    Inspect(ctx *RequestContext) (*Decision, error)
}
```

Each inspector returns a `Decision` containing:
- **Action**: `block`, `allow`, `challenge`, `monitor`, `rate_limit`
- **RuleID**: Rule identifier string
- **Severity**: `critical`, `high`, `medium`, `low`, `info`
- **Score**: Threat score contribution (0–100)
- **Evidence**: Human-readable match description

### Pipeline Execution

```
Request → CAPTCHA → JWT → OAuth → mTLS → GraphQL → gRPC → SOAP → Bot → DDoS → SQLi → XSS → API Protect → RCE → Protocol → Upload → Credential → WebSocket → Response Inspect → Decision
```

Execution rules:
1. Inspectors run in fixed priority order
2. `nil` inspectors are skipped
3. A `block` decision short-circuits the pipeline
4. Non-block decisions accumulate into `ThreatScore`
5. After all inspectors run, `finalDecision()` applies thresholds

### Threat Scoring

| Score Range | Action | Description |
|-------------|--------|-------------|
| ≥ 90 | Block | High-confidence attack |
| ≥ 50 | Challenge | Suspicious — requires JS challenge |
| Any | RateLimit | If any inspector returned rate_limit |
| Else | Allow | Normal request |

### 18 Inspectors

| # | Inspector | File | Detection Method |
|---|-----------|------|------------------|
| 1 | CAPTCHA | `auth.go` | Token verification with reCAPTCHA/hCaptcha |
| 2 | JWT | `auth.go` | Token validation, JWKS caching, claims check |
| 3 | OAuth | `auth.go` | Token introspection (RFC 7662) |
| 4 | mTLS | `mtls.go` | Client certificate validation, CA chain, policy OID |
| 5 | GraphQL | `graphql.go` | Query depth, cost analysis, alias/batch limits |
| 6 | gRPC | `grpc.go` | Message size limits, per-service rate limiting |
| 7 | SOAP | `soap.go` | XML schema validation, nesting depth |
| 8 | Bot | `bot.go` | User-Agent matching, headless browser detection, JS challenge |
| 9 | DDoS | `ddos.go` | Slow loris, slow POST, cache busting, adaptive rate limits |
| 10 | SQLi | `sqli.go` | Tokenizer + 15 regex patterns, encoding bypass detection |
| 11 | XSS | `xss.go` | Reflected/stored/DOM patterns, event handler detection |
| 12 | API Protect | `api_protect.go` | OpenAPI schema enforcement, shadow API discovery |
| 13 | RCE | `rce.go` | Shell injection, SSTI, EL injection, deserialization, Log4Shell |
| 14 | Protocol | `protocol.go` | Verb tampering, header smuggling, malformed requests |
| 15 | Upload | `upload.go` | MIME validation, magic bytes, extension allow/block lists |
| 16 | Credential | `credential.go` | Brute force, credential stuffing, password spray detection |
| 17 | WebSocket | `websocket.go` | Frame type validation, rate limiting, origin check |
| 18 | Response Inspect | `middleware.go` | Response body analysis for data leakage |

### Concurrency

- Engine uses `sync.RWMutex` for thread-safe inspector updates
- Hot-reload via `UpdateInspector()` allows runtime inspector swaps
- `InspectRequest()` is safe for concurrent use

## Configuration System

**Package**: `internal/config/`

### Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  YAML File  │────►│   Load()     │────►│   Config    │
└─────────────┘     └──────────────┘     └─────────────┘
                           │                      │
                    ┌──────▼──────┐        ┌──────▼──────┐
                    │  Manager    │        │  Validate() │
                    │  (fsnotify) │        │             │
                    │  Hot-Reload │        │  Get()      │
                    └─────────────┘        └─────────────┘
```

### Key Features

- **YAML-based** with full struct mapping via `gopkg.in/yaml.v3`
- **Environment variable expansion**: `${VAR:-default}` syntax
- **Hot-reload** via `fsnotify` file watcher
- **Default values** provided by `DefaultConfig()`
- **Validation** via `Config.Validate()` ensuring required fields

### Config Structure (30+ sections)

```go
type Config struct {
    Sites        []SiteConfig
    Rules        []RuleConfig
    TLS          TLSConfig
    Admin        AdminConfig
    ML           MLConfig
    Redis        RedisConfig
    DB           DBConfig
    JWT          JWTConfig
    OAuth        OAuthConfig
    GraphQL      GraphQLConfig
    MTLS         MTLSConfig
    WebSocket    WebSocketConfig
    SIEM         SIEMConfig
    RewriteRules []RewriteRuleConfig
    SQLI         FeatureConfig
    XSS          FeatureConfig
    RCE          FeatureConfig
    DDoS         FeatureConfig
    Protocol     FeatureConfig
    Bot          FeatureConfig
    APIProtect   FeatureConfig
    Upload       FeatureConfig
    Credential   CredentialConfig
    CAPTCHA      CAPTCHAConfig
    SOAP         SOAPConfig
    GRPC         GRPCConfig
    Prometheus   PrometheusConfig
    // ... and more
}
```

### Hot-Reload

The `Manager` watches the config file directory for changes. On write events, it triggers `Reload()` and notifies registered callbacks via `OnChange()`. This allows:

- Adding/removing sites without restart
- Updating rule configurations
- Toggling feature flags

## REST API

**Package**: `internal/api/`

### Server Architecture

```
Admin Server (:8443)
├── /health              Health check
├── /metrics             Prometheus metrics
├── /ready               Readiness probe
├── /live                Liveness probe
├── /api/v1/health       Authenticated health
├── /api/v1/status       System status
├── /api/v1/config       Get current config
├── /api/v1/reload       Force config reload
├── /api/v1/sites        List/Manage sites
└── /api/v1/rules        List/Manage rules
```

### Authentication

All `/api/v1/*` endpoints require a Bearer token from the configured `admin.api_keys`.

### Handlers

- **handleHealth**: Returns `{"status":"ok"}`
- **handleMetrics**: Prometheus format output via `promhttp.Handler`
- **handleReady**: Verifies config manager is responding
- **handleStatus**: Returns version, uptime, request stats
- **handleGetConfig**: Returns sanitized config (secrets masked)
- **handleReload**: Triggers config hot-reload
- **handleListSites**: Returns configured sites
- **handleListRules**: Returns configured rules

## Rate Limiter

**Package**: `internal/ratelimit/`

### Algorithms

| Algorithm | Description | Use Case |
|-----------|-------------|----------|
| **Fixed Window** | Count per fixed time interval | Simple per-IP limits |
| **Sliding Window** | Count per rolling time window | Smooth rate limiting |
| **Token Bucket** | Burst allowance with refill | API rate limits |
| **Leaky Bucket** | Constant processing rate | Queue management |

### Granularity Levels

- Per IP address
- Per user (via JWT claims)
- Per session
- Per API key
- Per endpoint
- Per geo region

### Implementation

- In-memory counters with optional Redis backend
- Background cleanup goroutine for stale entries
- Priority queue with double-burst for priority keys

## IP Reputation

**Package**: `internal/reputation/`

### Features

- **TOR detection**: Known TOR exit node IPs
- **Proxy/VPN detection**: Commercial proxy and VPN provider ranges
- **ASN filtering**: Allow/block by autonomous system number
- **CIDR matching**: Custom allowlist and blocklist CIDR ranges
- **GeoIP integration**: Country-based allow/block via `internal/geo/`

### Data Sources

- Checks are performed against embedded CIDR lists
- No external API calls (all data is built-in)
- Lists are loaded at startup and are static

### Performance

- CIDR matching uses binary search on sorted ranges
- Typical lookup time: ~10μs

## Session Manager

**Package**: `internal/session/`

### Features

- Cookie-based session management
- Configurable TTL
- Optional Redis backend for distributed deployments
- Session data stored as signed cookies or Redis key-value pairs

### Session Flow

```
Request → Session Middleware → Parse Cookie → Load Session → Attach to Context
```

### Storage Backends

| Backend | Persistence | Cluster Support |
|---------|-------------|-----------------|
| Memory | Volatile | No |
| Redis | Persistent | Yes |

## SIEM Exporter

**Package**: `internal/siem/`

### Architecture

```
Engine Events → SIEM Manager → Batch Buffer → Exporters
                                              ├── Elasticsearch
                                              ├── Splunk (HTTP Event Collector)
                                              ├── Syslog (RFC 5424)
                                              ├── JSON File
                                              └── Webhook (Slack, Teams, etc.)
```

### Event Types

- **Request Events**: Per-request inspection results
- **Alert Events**: High-severity matches requiring attention
- **Audit Events**: Configuration changes, admin actions

### Configuration

```yaml
siem:
  enabled: true
  export_interval: 10s
  batch_size: 100
  exporters:
    - type: elasticsearch
      url: http://elasticsearch:9200
      index: fortresswaf-events
    - type: slack
      url: https://hooks.slack.com/services/xxx
```

### Batching

Events are batched per exporter and flushed on interval or when batch size is reached. Failed exports are retried with exponential backoff.
