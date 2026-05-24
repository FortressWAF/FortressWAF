# FortressWAF — Enterprise Web Application Firewall

[![CI](https://github.com/FortressWAF/FortressWAF/actions/workflows/ci.yml/badge.svg)](https://github.com/FortressWAF/FortressWAF/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![OWASP](https://img.shields.io/badge/OWASP-Top%2010-red)](https://owasp.org/www-project-top-ten/)
[![Stars](https://img.shields.io/github/stars/FortressWAF/FortressWAF?style=social)](https://github.com/FortressWAF/FortressWAF)
[![Forks](https://img.shields.io/github/forks/FortressWAF/FortressWAF?style=social)](https://github.com/FortressWAF/FortressWAF)

**FortressWAF** is a production-grade, enterprise-ready Web Application Firewall that protects web applications and APIs from OWASP Top 10 threats, zero-day attacks, DDoS, and bot abuse. Built in Go with a machine learning sidecar, it provides defense-in-depth with sub-millisecond latency.

## Features

| Category | Details |
|----------|---------|
| **OWASP Top 10** | Full coverage: SQLi, XSS, RCE, LFI, SSRF, XXE, broken auth, misconfiguration |
| **ML-Powered Detection** | Anomaly detection engine with ensemble models (Random Forest + XGBoost + Transformer) |
| **Layer 7 DDoS Protection** | Sliding window rate limiting, token bucket, per-route limits, burst control |
| **Bot Management** | 500+ signatures, JS challenges, CAPTCHA integration, fingerprinting |
| **API Security** | Schema validation, JWT introspection, GraphQL depth limiting, OWASP API Top 10 |
| **IP Reputation** | Real-time threat intel feeds, geo-blocking, TOR/VPN/proxy detection |
| **Real-time Dashboard** | Live metrics, attack visualization, top targets, latency heatmaps, WebSocket feed |
| **Virtual Patching** | Deploy WAF rules instantly without modifying application code |
| **Multi-Tenancy** | Isolated workspaces, custom rulesets, per-tenant analytics (Enterprise) |
| **Compliance Ready** | PCI DSS, GDPR, HIPAA, SOC 2, ISO 27001, FIPS 140-2 |

## Quick Start

```bash
# One-line install (Linux)
curl -sSL https://install.fortresswaf.io | bash

# Or clone and run with Docker Compose
git clone https://github.com/zulfff/FortressWAF.git
cd FortressWAF
docker compose -f deploy/docker-compose.yml up -d

# Verify
curl http://localhost:8080/api/v1/health
```

## Architecture

```
                    ┌─────────────────────────────────────┐
                    │         Internet / Clients           │
                    └──────────────┬──────────────────────┘
                                   │
                    ┌──────────────▼──────────────────────┐
                    │        Global Load Balancer          │
                    └──────────────┬──────────────────────┘
                                   │
              ┌────────────────────┼────────────────────┐
              │                    │                    │
              ▼                    ▼                    ▼
     ┌────────────────┐  ┌────────────────┐  ┌────────────────┐
     │ FortressWAF     │  │ FortressWAF     │  │ FortressWAF     │
     │ Node 1          │  │ Node 2          │  │ Node N          │
     │ (Active)        │  │ (Active)        │  │ (Active)        │
     └────────┬────────┘  └────────┬────────┘  └────────┬────────┘
              │                    │                    │
              └────────────────────┼────────────────────┘
                                   │
                    ┌──────────────▼──────────────────────┐
                    │        Backend Services             │
                    │   (App Servers / APIs / CDN)        │
                    └─────────────────────────────────────┘
```

**Inside each node:**

```
Request ──► TLS Term ──► HTTP Parse ──► Rate Limiter ──► IP Reputation
                                                              │
                    ┌─────────────────────────────────────────┘
                    ▼
           Rule Engine Pipeline
     ┌──────────┬──────────┬──────────┬──────────┐
     │ SQLi     │ XSS      │ RCE/LFI  │ API      │
     │ Detector │ Detector │ Detector │ Schema   │
     └──────────┴──────────┴──────────┴──────────┘
                              │
                    ┌─────────▼─────────┐
                    │ ML Inference       │
                    │ (optional sidecar) │
                    └─────────┬─────────┘
                              │
                    ┌─────────▼─────────┐
                    │ Decision Engine    │
                    │ Allow/Block/Challenge│
                    └─────────┬─────────┘
                              │
                    ┌─────────▼─────────┐
                    │ Proxy Forwarder   │──► Origin
                    └───────────────────┘
```

## Performance

| Metric | Without ML | With ML |
|--------|-----------|---------|
| Throughput | 85,000 req/s | 45,000 req/s |
| P99 Latency | 3.8ms | 12ms |
| Memory | 150MB | 1.2GB |

## Documentation

Full documentation is available at [https://docs.fortresswaf.io](https://docs.fortresswaf.io).

| Document | Description |
|----------|-------------|
| [Getting Started](docs/getting-started.md) | Install and configure in 5 minutes |
| [Architecture](docs/architecture.md) | System architecture deep dive |
| [Rule Language](docs/rule-language.md) | YAML-based rule DSL reference |
| [API Reference](docs/api-reference.md) | Full REST API documentation |
| [Deployment](docs/deployment.md) | Docker, K8s, bare metal, cloud guides |
| [Compliance](docs/compliance.md) | PCI DSS, GDPR, HIPAA, SOC 2 |
| [Troubleshooting](docs/troubleshooting.md) | Common issues and solutions |

## Community vs Enterprise

| Feature | Community | Enterprise |
|---------|-----------|------------|
| Core WAF Engine | ✅ | ✅ |
| OWASP Top 10 Rules | ✅ | ✅ |
| Rate Limiting | ✅ | ✅ |
| Dashboard | ✅ | ✅ |
| REST API | ✅ | ✅ |
| 5,000+ Rule Corpus | ✅ | ✅ |
| IP Reputation | ✅ | ✅ |
| Bot Detection | ✅ | ✅ |
| ML Anomaly Detection | ❌ | ✅ |
| Multi-Tenancy | ❌ | ✅ |
| Compliance Modes (PCI/GDPR/HIPAA) | ❌ | ✅ |
| FIPS 140-2 | ❌ | ✅ |
| SLA Support (24/7) | ❌ | ✅ |
| SSO/SAML | ❌ | ✅ |
| Audit Logs | ❌ | ✅ |
| Dedicated Support | ❌ | ✅ |
| Custom Rules Priority | ❌ | ✅ |

## Quick Reference

```bash
# CLI
fortressctl config validate          # Validate configuration
fortressctl rules list               # List all rules
fortressctl rules apply --profile    # Apply rule profile
fortressctl rules test               # Test a rule against a payload
fortressctl compliance report        # Generate compliance report

# API
curl http://localhost:8080/api/v1/health                        # Health check
curl -H "Authorization: Bearer $TOKEN" /api/v1/metrics          # Metrics
curl -H "Authorization: Bearer $TOKEN" /api/v1/events           # Security events
curl -H "Authorization: Bearer $TOKEN" /api/v1/rules            # List rules
curl -X POST -H "Authorization: Bearer $TOKEN" /api/v1/rules    # Add rule
```

## Contributing

Contributions are welcome! Please read our guidelines and submit PRs to [github.com/zulfff/FortressWAF](https://github.com/zulfff/FortressWAF/pulls).

## Security

Please see [SECURITY.md](SECURITY.md) for our security policy and vulnerability reporting process.

## License

- **Community Edition**: Apache 2.0 - Free for production use
- **Enterprise Edition**: Commercial license with additional features

Copyright © 2024-2025 FortressWAF. Open-core security for the modern web.

---

<p align="center">
  <strong>Built to compete with Cloudflare. Priced for the mid-market.</strong>
</p>
