# OSINT Lead Platform — v1 status

This is the shipped `main`-branch truth for local operators. “Available” means the control plane exposes the module; it does not mean every optional third-party executable is installed or every network-dependent check will be green.

## Capability matrix

| Capability | Status | Needs | Notes |
|---|---|---|---|
| `email-validate` | available without optional tools | email; DNS/network for MX signal | Syntax, disposable, role, and provider checks work without a key. SMTP probing is disabled, so `deliverable` can honestly be `unknown`. |
| `phone-validate` | available offline | phone | libphonenumber parsing is local. Optional numverify carrier lookup is not required for the demo. |
| `domain-intel` | available with network-dependent checks | domain; network; optional `theHarvester` on `PATH` | Go DNS/TLS/HTTP/WHOIS checks run directly. Missing `theHarvester` yields a structured sub-result rather than a fake success. |
| `social-footprint` | available with optional Python tooling | email; Maigret Python wrapper for useful live results | Without the wrapper/Python it returns structured `unknown` or `skipped`; it is not a fully green offline demo path. |
| `extraction` | available with optional Python tooling | public URL, `permission_ref`, Crawl4AI Python environment | Missing Crawl4AI returns a structured `error`; missing `permission_ref` returns `skipped`. Optional Firecrawl is not required. |
| `company-enrich` | available locally | domain, company, or URL; `permission_ref` | `company-enrich/local` is deterministic and works without a key. The optional DiscoLike adapter needs `DISCOLIKE_API_KEY`. |
| async runs | shipped | `CONTROL_PLANE_WORKERS` optional; default `2` | `POST /api/leads/{id}/run` and `POST /api/pipelines/run` return `202` with `run_id`; poll `GET /api/runs/{id}`. The queue is in-process and jobs are lost on restart. |
| CRM-ready policy | local policy only | validated, permissioned lead | Promotion/demotion/export are local policy and a JSON export stub. There is no real CRM or HubSpot connector. |
| `risk_score` v2 | shipped | module results and lead fields | Deterministic 0–100 score with `low`, `medium`, `high`, or `unknown` level. |

## Demo path

```bash
make demo
# UI:  http://localhost:3000/leads
# API: http://localhost:8080/healthz
# Stop: make demo-down
```

`make demo` starts the memory-backed control plane and Next.js development UI on `127.0.0.1` only. It refuses to overwrite services already using ports 8080 or 3000. Logs and process IDs are stored under `${TMPDIR:-/tmp}/osint-lead-platform-demo`.

Do not run `npm run build` while the demo UI's `npm run dev` process uses the same `.next` directory; Next may emit `ENOENT` `app-build-manifest` 500 errors.

## Local quality and smoke gates

```bash
make test-go
make test-ui
make smoke-api && make smoke-async && make smoke-platform
```

`smoke-platform` accepts a structured extraction error when Crawl4AI is not installed. Use `make install-extraction-venv` and `EXTRACTION_CRAWL4AI_PYTHON="$PWD/modules/extraction/.venv/bin/python" make demo-api-ok` only when an extraction `ok` path is required.

## Operator behavior

- Lead and bulk module submissions are asynchronous. A `202` response contains a job reference, **not** the updated lead body.
- Lead detail polls an active UUID-shaped `active_run_id`; the `/leads` bulk banner polls its run and refreshes the list on terminal status.
- `/runs` and `/runs/{id}` poll while visible runs are `queued` or `running`.
- The default audit legal basis is GDPR Art.6(1)(f) legitimate interest. Extraction requires a `permission_ref`.

## Deliberate v1 limits

- No durable queue, multi-instance worker coordination, real auth/SSO, real CRM connector, LinkedIn scraping, or bulk breach/leak signals.
- Social and extraction are honest about missing optional tools instead of manufacturing positive results.
- The memory store is appropriate for the local demo only; data disappears when the API restarts.
