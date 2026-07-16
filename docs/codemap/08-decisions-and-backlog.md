# Decisions and backlog

Source of truth: `docs/decisions/stage-1-decision.md` (2026-07-13).

## Stage 1 summary

| Module / area | Decision | Confidence | Code status |
|---------------|----------|------------|-------------|
| email-validate | Move to Stage 2 — AfterShip | High | **Built** (#14) |
| domain-intel | Move to Stage 2 (time-boxed) — web-check + theHarvester subprocess | High | **Built** (#15) |
| phone-validate | Move to Stage 2 (time-boxed) — local scanner + optional numverify | Medium | **Built** (#16) |
| social-footprint | Move to Stage 2 — Maigret after handle source | Medium-High | **Built** (#17) |
| extraction | Move to Stage 2 — Crawl4AI primary, Firecrawl secondary | High | **Not built** |
| Orchestration | Reject for now | — | Deferred |
| AI-agent / MCP | Needs second-opinion (OpenOSINT MCP pattern only) | — | Deferred |
| company-enrich | Needs second-opinion (local-enrichment-tool) | — | **Blocked** |
| Risk/breach (h8mail) | Reject — no pipeline row | — | No module |

## Time-box follow-ups (still open)

1. **domain-intel / theHarvester:** re-assess whether keyless-only output is useful enough or demote to optional/manual.
2. **phone-validate:** PhoneInfoga archive risk already mitigated by not depending on it; keep numverify swappable.
3. **Orchestration:** revisit now that ≥2 modules exist — simple sequential/DAG runner is unblocked by decision text.

## Unblocked but not started

- `modules/extraction/` — Crawl4AI first; Firecrawl adapter optional; watch AGPL/data-residency.

## Blocked

- `modules/company-enrich/` — evaluate **local-enrichment-tool** before any code; do not ship paid-API-only waterfall as default. waterfall-gtm is pattern reference only (2/5).

## Evaluations present (`evaluations/`)

crawl4ai, email-verifier-aftership, firecrawl, h8mail, holehe, maigret, openosint, phoneinfoga, spiderfoot, theharvester, waterfall-gtm, web-check + TEMPLATE.

## Suggested pipeline compose order (manual)

```
domain-intel → email-validate → phone-validate → social-footprint
```

social-footprint benefits from prior `domain_intel` on the same JSON record for secondary handles.

## Doc debt

- Root `README.md` and `CONTRIBUTING.md` still describe Stage 1-only / empty modules.
- `docs/architecture.md` open questions partially outdated (modules now exist).

Prefer codemap + decision doc + code over README for agent work until those are refreshed.
