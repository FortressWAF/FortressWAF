# FortressWAF

Web application firewalls have been around for decades, but most of them were designed when applications were simpler. Monolithic codebases, predictable request patterns, and a clear network perimeter. That world is gone. Applications today are distributed, APIs are public, and attackers have moved past simple signature-based exploits.

We built FortressWAF because the tooling hadn't caught up. Existing solutions were either expensive enterprise suites that required dedicated ops teams, or open source projects that covered one or two attack vectors and left the rest to chance. Neither approach fits the way modern teams ship software.

FortressWAF started as an internal project to solve a specific problem: we needed a security layer that could inspect traffic across HTTP, WebSocket, and GraphQL — without requiring separate rule sets for each protocol. The project grew as we added detection modules for the attacks we actually encountered in production, not the ones that looked good on a compliance checklist.

## Why FortressWAF

Most WAFs evaluate requests against static signatures and make a binary decision: pass or block. This works for known attacks but falls apart when applications evolve. Custom endpoints trigger false positives. Legitimate traffic with unusual parameters gets blocked. Teams spend more time tuning rules than building features.

FortressWAF uses a layered inspection pipeline. Each module scores a request independently, and the engine makes a contextual decision based on cumulative risk. A request that exceeds the rate limit but passes all other checks might get challenged instead of blocked. A request from a known bad IP that also contains SQL-like patterns gets blocked immediately. The modules cross-validate each other, which reduces false positives without sacrificing coverage.

The architecture is straightforward: deploy as a reverse proxy, an API gateway, or a sidecar. Configuration is YAML, changes apply at runtime, and there is no separate management server to maintain.

## What It Does

- **Request inspection** — 20+ detection modules in a configurable pipeline
- **Authentication** — JWT validation, OAuth 2.0 introspection, mTLS, API key management
- **Threat detection** — SQL injection, cross-site scripting, remote code execution, protocol anomalies
- **Traffic management** — DDoS protection, rate limiting (token bucket, leaky bucket, sliding window, fixed window), session tracking, bot detection
- **Content security** — File upload validation, credential leak prevention, GraphQL depth analysis
- **Reputation** — IP intelligence feeds, Tor/proxy/VPN detection, ASN-based filtering
- **Observability** — SIEM-compatible event export, structured logging, live metrics
- **Management** — REST API and CLI for runtime configuration, hot-reload without restart

## Project Structure

```
cmd/
  proxy/         — WAF proxy server entry point
  ctl/           — Command-line administration tool
internal/
  engine/        — Core inspection pipeline and all detection modules
  api/           — Management REST API server
  config/        — YAML configuration loading and live-reload
  reputation/    — IP reputation engine with threat feed integration
  ratelimit/     — Rate limiter supporting four algorithms
  session/       — Session management and tracking
  siem/          — SIEM event aggregation and export
dashboard/       — Web-based monitoring dashboard (Vue.js)
ml-engine/       — Machine learning anomaly detection (Python)
```

## Deployment

Run as a single Go binary behind your load balancer or as a sidecar alongside application services. Configuration is YAML, loaded at startup, and watched for changes at runtime.

```
docker run -v /path/to/config.yaml:/etc/fortresswaf/config.yaml -p 8080:8080 fortresswaf/proxy
```

## Developer

Built and maintained by [@zulfff](https://github.com/zulfff).

## License

Dual-license. See [LICENSE](LICENSE) for details.
