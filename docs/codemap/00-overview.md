# Overview

## Purpose

Enrich and validate **leads from ads and business websites** where the business has granted permission to process that data. Output: CRM-ready records with enrichment fields, validation signals, and audit logs.

**License:** MIT for platform code only. Does not cover personal data or third-party tool licenses.

## Pipeline

```
Ad / Website
  → Raw lead: name, email, phone, company, domain, source_id, permission_ref
  → Enrichment (fill gaps): domain-intel, [company-enrich], [extraction]
  → Validation (score + flags): email-validate, phone-validate, social-footprint
  → CRM-ready: enriched fields + validation signals + audit log
```

There is **no orchestrator binary yet**. Modules are independent Go CLIs composed with shell pipes (or a future DAG runner). Stage 1 rejected SpiderFoot and deferred custom orchestration until modules exist.

## Stage status (as of 2026-07)

| Stage | Status |
|-------|--------|
| Stage 1 — Research | **Closed.** 12/12 priority tools evaluated; decision doc merged. |
| Stage 2 — Build modules | **In progress.** 4 modules implemented; `extraction` approved not built; `company-enrich` blocked. |
| Stage 3 — Integrate & harden | **Not started.** Storage, scoring, retention, pipeline runner TBD. |

## Implemented modules

| Module | Pipeline stage | Result key | Primary engine |
|--------|----------------|------------|----------------|
| `modules/email-validate` | Validate | `email_validate` | AfterShip email-verifier v1.4.1 |
| `modules/domain-intel` | Ingest | `domain_intel` | web-check-lite (Go stdlib) + theHarvester CLI |
| `modules/phone-validate` | Validate | `phone_validate` | nyaruka/phonenumbers + optional numverify |
| `modules/social-footprint` | Validate | `social_footprint` | Maigret 0.6.2 (Python lib via wrapper) |

## Design pillars

- **One capability per module**, one static Go binary, no daemon.
- **Partial lead in → same lead out** with one namespaced key added.
- **Never block the pipeline:** failures are `status: "unknown"` (or `"skipped"`), exit 0.
- **Audit every call** on stderr with tool version + legal basis.
- **Compliance in code** where risk is high (source allowlists, rate limits, scope caps).
- **License firewalls:** GPL tools only as external subprocesses (or avoided entirely).

## Open architecture questions (do not invent answers in code)

From `docs/architecture.md`:

- Orchestration: lightweight custom runner vs wait for more evidence
- Storage / retention policy
- Composite score vs separate validation flags
- AI-agent / MCP layer placement

## Key decision doc

`docs/decisions/stage-1-decision.md` — gate for what may be built under `modules/`.
