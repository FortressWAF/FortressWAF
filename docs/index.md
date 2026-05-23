---
hide:
  - navigation
  - footer
---

# FortressWAF Documentation

<p align="center">
  <img src="assets/logo.png" width="200" height="200" alt="FortressWAF Logo"/>
</p>

<h1 align="center">Enterprise Web Application Firewall</h1>

FortressWAF is a high-performance, container-native Web Application Firewall designed to protect modern web applications and APIs from OWASP Top 10 attacks, bot abuse, and advanced persistent threats. Built for scale and ease of use, FortressWAF deploys in minutes and provides real-time threat detection and mitigation.

## Why FortressWAF?

- **Real-time Protection**: Block SQL injection, XSS, command injection, and other OWASP Top 10 attacks in under 1ms latency overhead
- **Bot Management**: Advanced fingerprinting and behavioral analysis to identify and control bot traffic
- **ML-Powered**: Machine learning anomaly detection for zero-day attack identification
- **API Security**: Native support for REST, GraphQL, gRPC, and WebSocket APIs
- **Compliance Ready**: PCI DSS 4.0, GDPR, SOC 2 Type II compliant reporting
- **Cloud Native**: Kubernetes-native deployment with Helm charts and auto-scaling
- **High Performance**: < 1ms latency overhead at 100,000 RPS per instance

## Feature Overview

| Feature | Description |
|---------|-------------|
| **OWASP Protection** | Complete coverage for OWASP Top 10 including SQLi, XSS, LFI, RCE, XXE |
| **Bot Management** | Device fingerprinting, headless browser detection, tarpit mode |
| **DDoS Mitigation** | HTTP flood, Slowloris, Slow POST, cache-busting attacks |
| **API Security** | GraphQL validation, OpenAPI enforcement, shadow API discovery |
| **Virtual Patching** | One-click CVE patches, manual patch creation, patch testing |
| **Rate Limiting** | Token bucket, leaky bucket, sliding window algorithms |
| **ML Engine** | Isolation Forest, DistilBERT NLP, Random Forest, Gradient Boosting |
| **Compliance** | PCI DSS 4.0, GDPR, SOC 2 reporting and evidence collection |

## Architecture Overview

```
                                    ┌─────────────────────────────────────┐
                                    │          FortressWAF Cluster        │
                                    │                                     │
┌─────────┐     ┌─────────┐         │   ┌─────────────────────────────────┐│
│  User   │────▶│  Edge   │────────────▶│      Request Pipeline          ││
│ Traffic │     │  Router │         │   │  ┌─────────────────────────┐   ││
└─────────┘     └─────────┘         │   │  │  1. IP Reputation      │   ││
         ▲        │                 │   │  │  2. Rate Limiting      │   ││
         │        │                 │   │  │  3. Bot Detection      │   ││
         │        │                 │   │  │  4. Rule Engine        │   ││
         │        │                 │   │  │  5. ML Anomaly Score   │   ││
         │        │                 │   │  │  6. Decision & Action  │   ││
         │        │                 │   │  └─────────────────────────┘   ││
         │        │                 │   └─────────────────────────────────┘│
         │        │                 │              │                       │
         │        │                 │              ▼                       │
         │        │                 │   ┌─────────────────────────────────┐│
         │        │                 │   │       Backend Applications      ││
         │        │                 │   │   (Kubernetes / Docker / VMs)   ││
         │        │                 │   └─────────────────────────────────┘│
         │        │                 └─────────────────────────────────────┘
         │        │
         │   ┌────┴────┐
         │   │ Attack  │  (blocked)
         │   │  Log    │
         │   └─────────┘
         │
    ┌────┴────┐
    │ Blocked │
    │  Users  │
    └─────────┘
```

## Deployment Modes

FortressWAF supports multiple deployment architectures to fit your infrastructure:

| Mode | Description | Use Case |
|------|-------------|----------|
| **Inline** | Full inspection of all traffic | High-security environments |
| **Mirror (SPAN)** | Copy of traffic for analysis | Compliance logging, staging |
| **Hybrid** | Inline blocking + mirror analysis | Production with deep analysis |
| **Out-of-band** | API-based blocking decisions | Legacy applications |

## Edition Comparison

| Feature | Community | Enterprise |
|---------|-----------|------------|
| **Max RPS** | 1,000 | Unlimited |
| **Rule Engine** | Basic operators | Advanced DSL + ML |
| **Bot Detection** | Basic | Advanced fingerprinting |
| **ML Models** | None | All 4 model types |
| **API Endpoints** | 10 | Unlimited |
| **Support** | Community forum | 24/7 enterprise support |
| **SLA** | None | 99.99% uptime |
| **Compliance Reports** | Basic | Full audit suite |
| **Price** | Free | Contact sales |

## Quick Start

Get FortressWAF running in under 5 minutes with Docker Compose:

```bash
# Download and run the installer
curl -sSL https://install.fortresswaf.io | bash

# Or manually with Docker Compose
git clone https://github.com/fortresswaf/fortresswaf.git
cd fortresswaf/deploy/docker-compose
docker-compose up -d

# Access the dashboard
open https://localhost:8443
```

Default credentials: `admin` / `fortress-change-me-immediately`

## Next Steps

- **[Quick Start Guide](quickstart.md)** - Get up and running in 5 minutes
- **[Kubernetes Deployment](kubernetes.md)** - Deploy on Kubernetes with Helm
- **[Configuration Reference](configuration.md)** - Complete configuration guide
- **[Architecture Deep Dive](concepts/architecture.md)** - Understand how FortressWAF works
- **[Rule Engine](concepts/rules.md)** - Write custom rules with our DSL
- **[API Reference](api-reference.md)** - REST API documentation

## License

FortressWAF Community Edition is released under the Apache 2.0 License.

For Enterprise features and support, contact [sales@fortresswaf.io](mailto:sales@fortresswaf.io).
