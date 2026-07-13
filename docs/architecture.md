# Architecture (draft — fleshed out entering Stage 2)

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

## Module contract (proposed)

Every module in `modules/<name>/` should expose a single well-defined function/interface:

- **Input:** a partial lead record (JSON) — only the fields the module needs.
- **Output:** the same record with new fields added under a namespaced key (e.g. `email_validate: {status, deliverable, is_disposable, checked_at, source_tool}`), never overwriting raw ingested fields.
- **Failure mode:** must degrade gracefully (timeout, tool unavailable) and mark the field as `unknown` rather than blocking the pipeline.
- **Audit:** every module call logs which underlying tool/API was used, when, and the legal-basis tag from `docs/compliance.md`.

## Open questions for Stage 2 (do not answer yet — track here as they come up)

- Orchestration: adopt SpiderFoot as the backbone vs. build a lightweight custom orchestrator?
- Storage: what database/warehouse holds enriched lead records, and what's the retention policy per `docs/compliance.md`?
- Scoring: single composite "lead quality score" vs. separate validation flags surfaced individually to sales/marketing?
- Where does the AI-agent layer sit — orchestrating module calls, or only used for unstructured extraction (parsing scraped landing-page text into structured fields)?
