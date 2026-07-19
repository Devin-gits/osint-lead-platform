# OSINT Lead Platform

A platform to **enrich and validate leads generated from ads and business websites**, where explicit permission to process that data has been obtained. Built collaboratively by human + AI agents, with each stage gated by research and review before any code lands.

## Status: v1 control plane and UI live (five modules available)

Five modules (`email-validate`, `phone-validate`, `domain-intel`, `social-footprint`, `extraction`) are wired into the Go control-plane and consumed by the Next.js web console. `company-enrich` is planned. For the current shipped state, see [`docs/status/platform-v1.md`](docs/status/platform-v1.md).

Module implementation PRs still target `modules/<name>/` and must follow the module contract in `docs/architecture.md` / `docs/codemap/01-module-contract.md`.

## Pipeline

```
Ad / Website  →  Raw lead (name, email, phone, company, domain, url)
             →  Enrichment   (fill missing fields)
             →  Validation   (is it real, deliverable, low-risk?)
             →  CRM-ready record
```

| Stage | Module | Status | Candidate tools |
|---|---|---|---|
| Ingest | `modules/domain-intel` | available | web-check, theHarvester |
| Validate | `modules/email-validate` | available | AfterShip email-verifier |
| Validate | `modules/phone-validate` | available | libphonenumber, optional numverify |
| Validate | `modules/social-footprint` | available | Maigret (curated platform list) |
| Enrich | `modules/extraction` | available | Crawl4AI (default), optional Firecrawl |
| Enrich | `modules/company-enrich` | planned | discolike-cli, local-enrichment-tool, waterfall pattern + paid APIs |

Full rationale for every candidate tool: [`docs/research/osint-tooling-research.md`](docs/research/osint-tooling-research.md).

## Repo structure

```
osint-lead-platform/
├── README.md
├── docs/
│   ├── research/            # Tool surveys and deep-dive notes
│   ├── status/              # Shipped-state docs
│   │   └── platform-v1.md
│   ├── compliance.md        # GDPR / ToS risk notes — read before building any module
│   ├── architecture.md      # Pipeline & module contracts
│   └── frontend/            # UI planning and API-contract docs
├── modules/                  # Module libraries (one folder per capability)
├── services/
│   └── control-plane/        # Go HTTP API that wires modules together
├── ui/
│   └── web-console/          # Next.js 15 control-plane UI
├── evaluations/              # Stage 1 output: one scorecard per tool evaluated
│   └── TEMPLATE.md
├── .github/
│   ├── PULL_REQUEST_TEMPLATE.md
│   ├── ISSUE_TEMPLATE/tool-evaluation.md
│   └── workflows/            # ui.yml + control-plane.yml
├── scripts/                  # Local smoke and demo helpers
│   └── smoke-extraction.sh
├── Makefile                  # Convenience targets: demo-api, demo-ui, smoke
└── CONTRIBUTING.md           # How agents (human or AI) should scope work and open PRs
```

## How this project runs

1. **Stage 1 — Research.** Every candidate tool gets its own evaluation PR against `evaluations/TEMPLATE.md`. No integration code yet.
2. **Review.** Every PR is reviewed against the checklist in `CONTRIBUTING.md` before merge — checking sourcing, license risk, and compliance flags.
3. **Stage 2 — Build.** Once a module's tool choice is approved, an implementation PR opens against `modules/<name>/`, spec'd from the approved evaluation.
4. **Stage 3 — Integrate & harden.** Wire modules into `services/control-plane`, add tests, CI, data-retention/deletion controls. The control plane is already running for the five available modules.

## Run locally

Quick start (two terminals):

```bash
# Terminal 1 — Go control plane (http://localhost:8080)
make demo-api

# Terminal 2 — Next.js UI (http://localhost:3000)
make demo-ui
```

Then open [http://localhost:3000](http://localhost:3000). The UI expects the API at `http://localhost:8080`; override with `NEXT_PUBLIC_API_BASE_URL`.

For a one-shot smoke test (API must be running):

```bash
make smoke
```

For the full extraction `ok` path, set `EXTRACTION_CRAWL4AI_PYTHON` to a Python venv with Crawl4AI installed:

```bash
cd modules/extraction
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
playwright install chromium
cd ../..
EXTRACTION_CRAWL4AI_PYTHON="$PWD/modules/extraction/.venv/bin/python" make demo-api
```

For full social + domain runs, raise `HTTP_WRITE_TIMEOUT` and see the env matrix in [`docs/status/platform-v1.md`](docs/status/platform-v1.md). Detailed API + env docs are in [`services/control-plane/README.md`](services/control-plane/README.md) and UI docs in [`ui/web-console/README.md`](ui/web-console/README.md).

## Compliance

This platform processes personal data (names, emails, phone numbers). Read `docs/compliance.md` before opening any PR — some categories of OSINT tooling (bulk breach-checking, non-consensual social scraping, LinkedIn scraping) are explicitly out of scope or require a documented legal basis before use.

## License

Code in this repository is MIT-licensed (see `LICENSE`). This license covers the platform's own code only — it does not grant any rights over personal data processed by the platform, and does not override the license terms of any third-party tool referenced or integrated (several candidates use AGPL-3.0 or GPL-3.0; see the research doc for per-tool license notes).
