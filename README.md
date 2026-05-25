# FortressWAF — Web Application Firewall

[![License](https://img.shields.io/badge/license-AGPL--3.0-blue)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)

**FortressWAF** is a Go-based Web Application Firewall with rule-based detection, rate limiting, IP reputation, and an optional ML sidecar. It proxies traffic to upstream services and can block or monitor requests based on configurable rules.

## Features

| Category | Details |
|----------|---------|
| **OWASP Detection** | SQLi, XSS, RCE, API protection rules (LFI, SSRF, XXE on roadmap) |
| **Rule Engine** | Pattern-based detection with YAML rules |
| **Rate Limiting** | Token bucket and leaky bucket per-IP and global |
| **IP Reputation** | Allow/block lists, TOR/VPN/proxy detection, datacenter IP ranges |
| **Dashboard** | Live metrics via WebSocket feed |
| **Virtual Patching** | Deploy WAF rules without modifying application code |
| **API Security** | JWT introspection, GraphQL depth limiting |
| **Multi-Tenancy** | Isolated workspaces per-site (Enterprise) |

## Quick Start

```bash
# Clone and run with Docker Compose
git clone https://github.com/FortressWAF/FortressWAF.git
cd FortressWAF
docker compose -f deploy/docker-compose.yml up -d

# Verify
curl http://localhost:8080/api/v1/health
```

## Architecture

```
Request -> TLS Term -> HTTP Parse -> Rate Limiter -> IP Reputation
                                |
                    Rule Engine Pipeline                     
           SQLi -> XSS -> RCE/LFI -> API Schema             
                                |                           
                        Decision Engine               
                     Allow / Block / Challenge              
                                |                           
                        Proxy Forwarder -> Origin            
```

## Performance

Performance varies by hardware, rule count, and configuration. Rough benchmarks on reference hardware (4 vCPU, 8GB RAM):

| Scenario | Throughput (approx) |
|----------|-------------------|
| No rules (passthrough) | ~85,000 req/s |
| Full rule set, no ML | ~45,000 req/s |
| With ML sidecar | lower (varies) |

## Documentation

| Document | Description |
|----------|-------------|
| [Getting Started](docs/getting-started.md) | Install and configure |
| [Architecture](docs/architecture.md) | System architecture |
| [Rule Language](docs/rule-language.md) | YAML-based rule DSL |
| [API Reference](docs/api-reference.md) | REST API documentation |
| [Deployment](docs/deployment.md) | Docker, K8s, cloud guides |
| [Compliance](docs/compliance.md) | Compliance reference docs |
| [Troubleshooting](docs/troubleshooting.md) | Common issues |

## Community vs Enterprise

| Feature | Community | Enterprise |
|---------|-----------|------------|
| Core WAF Engine | ✅ | ✅ |
| OWASP Top 10 Rules | ✅ | ✅ |
| Rate Limiting | ✅ | ✅ |
| Dashboard | ✅ | ✅ |
| REST API | ✅ | ✅ |
| IP Reputation | ✅ | ✅ |
| Bot Detection | ✅ | ✅ |
| ML Anomaly Detection | ❌ | ✅ |
| Multi-Tenancy | ❌ | ✅ |
| Compliance Documentation | ❌ | ✅ |
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

Contributions welcome. Submit PRs to [github.com/FortressWAF/FortressWAF](https://github.com/FortressWAF/FortressWAF/pulls).

## Security

See [SECURITY.md](SECURITY.md) for security policy and vulnerability reporting.

## License

- **Community Edition**: AGPL-3.0
- **Enterprise Edition**: Commercial license

Copyright © 2024-2026 FortressWAF.
