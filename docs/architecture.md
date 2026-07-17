# Architecture

## Pipeline overview

```
                ┌───────────────────┐
Ad / Website →  │  Ingest           │  raw lead: name, email, phone, company, domain, source_id, permission_ref
                └─────────┬─────────┘
                          ▼
                ┌───────────────────┐
                │  Enrichment       │  domain-intel, company-enrich, extraction
                │  (fill gaps)      │
                └─────────┬─────────┘
                          ▼
                ┌───────────────────┐
                │  Validation       │  email-validate, phone-validate, social-footprint
                │  (score + flags)  │
                └─────────┬─────────┘
                          ▼
                ┌───────────────────┐
                │  CRM-ready record │  enriched fields + validation_score + risk_flags + audit log
                └───────────────────┘
```

The `services/control-plane` Go HTTP API is the current orchestrator. It persists leads, runs, audit events, and modules to Postgres (or an in-memory fallback) and invokes the module libraries defined in `modules/<name>/`.

The current stage machine: `raw → enriched` (domain/extraction/company when `ok`) → `validated` (email/phone/social when `ok`). `crm_ready` is reserved and not assigned yet. Risk is computed from `email_validate` and `phone_validate` heuristics only (`runner.computeRisk`).

## Module contract

Every module in `modules/<name>/` should expose a single well-defined constructor/function:

- **Input:** a partial lead record (JSON) — only the fields the module needs.
- **Output:** the same record with new fields added under a namespaced key (e.g. `email_validate: {status, deliverable, is_disposable, checked_at, source_tool}`), never overwriting raw ingested fields.
- **Failure mode:** must degrade gracefully (timeout, tool unavailable) and mark the field as `unknown` rather than blocking the pipeline.
- **Audit:** every module call logs which underlying tool/API was used, when, and the legal-basis tag from `docs/compliance.md`.

See `services/control-plane/internal/runner/runner.go` for the live wiring of `email-validate`, `phone-validate`, `domain-intel`, and `social-footprint`.

## Open questions for v2 and beyond

- Async worker: should long Maigret/theHarvester batch runs move out of the synchronous HTTP request path?
- Storage: enforce retention windows and deletion in Postgres.
- Scoring: define a composite risk/quality score from module signals or keep individual validation flags.
- AI-agent layer: where does it sit — orchestrating module calls, or only parsing unstructured landing-page text into structured fields?
