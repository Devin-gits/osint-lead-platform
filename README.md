# OSINT Lead Platform

A platform to **enrich and validate leads generated from ads and business websites**, where explicit permission to process that data has been obtained. Built collaboratively by human + AI agents, with each stage gated by research and review before any code lands.

## Status: Stage 2 — Build modules (in progress)

Stage 1 research is complete. Tool choices and adoption decisions live in `docs/decisions/stage-1-decision.md`; the full survey is in `docs/research/osint-tooling-research.md`. Implementation PRs now target `modules/<name>/` for approved modules and must follow the module contract in `docs/architecture.md` / `docs/codemap/01-module-contract.md`.

## Pipeline

```
Ad / Website  →  Raw lead (name, email, phone, company, domain)
             →  Enrichment   (fill missing fields)
             →  Validation   (is it real, deliverable, low-risk?)
             →  CRM-ready record
```

| Stage | Module (planned) | Candidate tools |
|---|---|---|
| Ingest | `modules/domain-intel` | web-check, theHarvester |
| Enrich | `modules/company-enrich` | discolike-cli, local-enrichment-tool, waterfall pattern + paid APIs |
| Enrich | `modules/extraction` | Firecrawl, Crawl4AI, browser-use, Scrapy |
| Validate | `modules/email-validate` | AfterShip email-verifier, holehe |
| Validate | `modules/phone-validate` | PhoneInfoga |
| Validate | `modules/social-footprint` | Maigret, Sherlock, Social-Analyzer |

Full rationale for every candidate tool: [`docs/research/osint-tooling-research.md`](docs/research/osint-tooling-research.md).

## Repo structure

```
osint-lead-platform/
├── README.md
├── docs/
│   ├── research/            # Tool surveys and deep-dive notes
│   ├── compliance.md        # GDPR / ToS risk notes — read before building any module
│   └── architecture.md      # Pipeline & module contracts (fleshed out entering Stage 2)
├── modules/                  # Empty until Stage 2 — one folder per capability
├── evaluations/              # Stage 1 output: one scorecard per tool evaluated
│   └── TEMPLATE.md
├── .github/
│   ├── PULL_REQUEST_TEMPLATE.md
│   └── ISSUE_TEMPLATE/tool-evaluation.md
└── CONTRIBUTING.md           # How agents (human or AI) should scope work and open PRs
```

## How this project runs

1. **Stage 1 — Research.** Every candidate tool gets its own evaluation PR against `evaluations/TEMPLATE.md`. No integration code yet.
2. **Review.** Every PR is reviewed against the checklist in `CONTRIBUTING.md` before merge — checking sourcing, license risk, and compliance flags.
3. **Stage 2 — Build.** Once a module's tool choice is approved, an implementation PR opens against `modules/<name>/`, spec'd from the approved evaluation.
4. **Stage 3 — Integrate & harden.** Wire modules into the pipeline, add tests, add data-retention/deletion controls.

## Compliance

This platform processes personal data (names, emails, phone numbers). Read `docs/compliance.md` before opening any PR — some categories of OSINT tooling (bulk breach-checking, non-consensual social scraping, LinkedIn scraping) are explicitly out of scope or require a documented legal basis before use.

## License

Code in this repository is MIT-licensed (see `LICENSE`). This license covers the platform's own code only — it does not grant any rights over personal data processed by the platform, and does not override the license terms of any third-party tool referenced or integrated (several candidates use AGPL-3.0 or GPL-3.0; see the research doc for per-tool license notes).
