# CRM-ready stage policy

**Status:** implemented in `services/control-plane` and `ui/web-console`
**Date:** 2026-07-19

## Pipeline stages

```text
raw → enriched → validated → crm_ready
```

A lead advances through the first three stages automatically as modules are
run (`computeStage` in `services/control-plane/internal/runner/runner.go`).
`crm_ready` is a deliberate promotion gate: it is never set by a module run,
only by an explicit promotion request once the lead is ready for export.

## Promotion rules to `crm_ready`

ALL of the following must hold.

1. **Permission reference** — `permission_ref` is non-empty.
2. **Identity contact** — at least one of `email` or `phone` is non-empty.
3. **Contact validation** —
   - If `email` is non-empty: `email_validate.status == "ok"`.
   - Else (email empty, phone non-empty): `phone_validate.status == "ok"`.
   - When both are present, email is the required channel; phone is a bonus.
4. **Company context** —
   - `company` or `domain` is non-empty, AND
   - at least one of:
     - `company_enrich.status` in `["ok", "partial"]`
     - `extraction.status == "ok"`
     - `domain_intel.status == "ok"`
5. **Risk ceiling** — `risk_level` is not `"high"`.
   - Missing or `"unknown"` risk is allowed but the readiness report includes a
     warning.
6. **No required-channel error** — if a channel is required by rule 3, the last
   run of that channel (`email_validate` or `phone_validate`) must not have
   `status == "error"`.

## Explicit non-requirements

The following are **not** required for `crm_ready`:

- `social_footprint` ok
- `company_enrich` ok (`partial` is sufficient when domain/company is present)
- `deliverable == "yes"` (SMTP probe is disabled in `email-validate`, so
  `deliverable` is often `"unknown"` while `status` is `"ok"`)

## Demotion

- A lead can be manually demoted from `crm_ready` to `validated` (or any earlier
  stage) via the demote endpoint.
- Auto-demotion is **out of scope** for this PR and is listed as a follow-up.

## Export stub

`GET /api/leads/{id}/export` (or `POST`) returns a deterministic JSON payload
without calling any external CRM.

- If the lead stage is `crm_ready`: `200 OK` with `format: "crm_stub_v1"` and a
  summary of safe CRM fields plus the readiness checks that passed.
- If the lead is not `crm_ready`: `409 Conflict` with the same readiness report
  shape and a `checks` array showing which rules passed or failed.

## Examples

| Lead fixture | Expected `crm_ready` |
|---|---|
| email + `email_validate` ok + domain/company + `company_enrich` partial + `permission_ref` | can promote |
| email only, `email_validate` never run or not ok | cannot — missing required contact validation |
| domain-only + `company_enrich` partial, no email/phone | cannot — no identity contact |
| email ok + company context, but `risk_level == "high"` | cannot — risk ceiling blocked |
| missing `permission_ref` | cannot |

## Endpoints

- `GET /api/leads/{id}/readiness` — readiness report, idempotent.
- `POST /api/leads/{id}/promote` body `{ "target": "crm_ready" }` — promote if
  rules pass; otherwise `409` with checks.
- `POST /api/leads/{id}/demote` body `{ "target": "validated" }` — manual
  demotion to an allow-listed earlier stage.
- `GET /api/leads/{id}/export` — CRM export stub (`200` or `409`).

## Audit trail

Every promote/demote/export attempt creates an `AuditEvent` with:

- `module`: `"pipeline"`
- `status`: `"ok"`, `"error"`, or `"skipped"`
- `legal_basis`: the lead's `permission_ref`-derived legal basis
- `raw_stderr_json`: a sanitized JSON summary of the checks and stage transition

## Residuals

- Auto-demote when a re-run invalidates a previously promoted lead.
- Bulk promote endpoint (`POST /api/leads/promote`).
- Real HubSpot/Salesforce connectors (out of scope by design).
