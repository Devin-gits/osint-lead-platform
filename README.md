# OSINT Lead Platform

A local-first control plane for permissioned lead enrichment and validation. It pairs a Go API with a Next.js operator console and keeps each module result, audit event, risk score, and pipeline run visible to the operator.

## What works today

Six modules, in-process async runs, deterministic risk scoring, and a local CRM-readiness policy are shipped. See the honest capability matrix in [Platform v1 status](docs/status/platform-v1.md), including which capabilities need optional Python tools or return structured errors when those tools are absent.

## Quick start

```bash
make demo
# UI:  http://localhost:3000/leads
# API: http://localhost:8080/healthz
# Stop: make demo-down
```

The demo binds only to loopback and uses the memory store. Do not run `npm run build` while the demo UI's `npm run dev` process uses the same `.next` directory.

## Local smoke gate

```bash
make smoke-api && make smoke-async && make smoke-platform
```

Run `make test-go` and `make test-ui` for the full repository quality gates. The console guide is in [ui/web-console/README.md](ui/web-console/README.md); API configuration is in [services/control-plane/README.md](services/control-plane/README.md); the operator path is in [docs/runbooks/local-dev-smoke.md](docs/runbooks/local-dev-smoke.md).

Every enrichment run must have a documented legal basis; `permission_ref` is mandatory for extraction and the default audit basis is GDPR Art.6(1)(f) legitimate interest.

## Pipeline

```text
Raw lead → enrichment and validation → deterministic risk score → explicit CRM-ready policy
```

## Repository structure

```text
osint-lead-platform/
├── docs/                 # Status, runbooks, architecture, compliance, research
├── modules/              # Module libraries
├── services/control-plane/ # Go API and in-process workers
├── ui/web-console/       # Next.js operator console
├── scripts/              # Demo and smoke helpers
└── Makefile              # Local demo, smoke, and quality targets
```

## Compliance

This platform processes personal data. Read [docs/compliance.md](docs/compliance.md) before using or extending it. Bulk breach signals, LinkedIn scraping, real CRM integration, and durable distributed queues are intentionally out of scope for v1.

## License

Code in this repository is MIT-licensed (see [LICENSE](LICENSE)). The license covers platform code only; it does not grant rights over personal data or override third-party tool licenses.
