# OSINT Lead Platform — Control Plane API

A real Go HTTP control plane that runs the existing domain modules and persists
leads, audit events, and pipeline runs.

## Scope

- Lives in `services/control-plane/**`.
- Wired modules in v1:
  - `email-validate` (in-process, AfterShip/email-verifier)
  - `phone-validate` (in-process, libphonenumber; optional numverify)
  - `domain-intel` (in-process Go web-check reimplementation + optional theHarvester subprocess)
  - `social-footprint` (handle derivation, Maigret Python wrapper subprocess, curated platform allow-list)
- Stubbed / not wired in v1:
  - `extraction`, `company-enrich`
- Uses Postgres for persistence; falls back to an in-memory store when
  `DATABASE_URL` is unset (useful for local tests).

## Requirements

- Go 1.22.5+
- Postgres 14+ (or run with `DATABASE_URL` unset for the in-memory fallback)

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | — | Postgres URL, e.g. `postgres://user:pass@localhost/osint?sslmode=disable` |
| `PORT` | `8080` | HTTP port |
| `CORS_ORIGIN` | `http://localhost:3000` | UI dev server origin |
| `MODULE_TIMEOUT` | `90s` | Shared timeout for email/phone and floor for domain-intel (60s). Social-footprint ignores this and uses its own 90s per-handle default. |
| `HTTP_READ_TIMEOUT` | `30s` | HTTP server read timeout |
| `HTTP_WRITE_TIMEOUT` | `180s` | HTTP server write timeout. Must exceed the longest expected module run (e.g., Maigret multi-handle). |
| `DOMAIN_INTEL_HARVESTER_BIN` | `theHarvester` (on PATH) | Override the theHarvester executable used by `domain-intel` |
| `SOCIAL_FOOTPRINT_BACKEND` | `maigret` | Backend selector: `maigret`, `sherlock`, `both`, or `osintgram` |
| `SOCIAL_FOOTPRINT_PYTHON` | `python3` (on PATH) | Python interpreter used to run the wrapper |
| `SOCIAL_FOOTPRINT_WRAPPER` | `wrapper/maigret_check.py` (auto-located) | Path to the Maigret wrapper script |
| `SOCIAL_FOOTPRINT_TIMEOUT` | `90s` | Per-handle subprocess timeout |
| `SOCIAL_FOOTPRINT_MIN_INTERVAL` | `5s` (`15s` for Osintgram) | Minimum spacing between consecutive per-lead `Check` calls |

## Run locally

```bash
cd services/control-plane

# With Postgres
export DATABASE_URL='postgres://osint:osint@localhost:5432/osint?sslmode=disable'
go run ./cmd/server

# With in-memory store (no DATABASE_URL)
go run ./cmd/server
```

## Test

```bash
go test ./...
```

For Postgres-backed tests, set `TEST_DATABASE_URL`.

## Recommended timeout profiles

For email/phone-only runs the defaults are fine. For `domain-intel` and
`social-footprint` (Maigret can check up to 3 handles × 90s each plus rate
limits), raise the HTTP write timeout so the server does not close the
connection before the runner finishes:

```bash
# Fast local machine with Maigret wrapper installed
export HTTP_WRITE_TIMEOUT=300s

# Slower network / many handles
export HTTP_WRITE_TIMEOUT=600s
```

`MODULE_TIMEOUT` still controls email/phone and acts as a 60s floor for
domain-intel; `social-footprint` ignores it and uses `SOCIAL_FOOTPRINT_TIMEOUT`
(default `90s`) for each handle.

## API

All endpoints return JSON envelopes:

```json
{ "data": <T>, "meta": { "page": 1, "page_size": 25, "total": 0 } }
{ "error": { "code": "not_found", "message": "..." } }
```

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/leads` | Create a lead |
| `GET`  | `/api/leads` | List leads (stage, risk, module_status, q, page, page_size) |
| `GET`  | `/api/leads/{id}` | Get a lead with hydrated audit events |
| `POST` | `/api/leads/{id}/run` | Run modules on one lead |
| `GET`  | `/api/modules` | List modules |
| `GET`  | `/api/modules/{name}` | Module detail |
| `GET`  | `/api/audit` | List audit events |
| `GET`  | `/api/runs` | List pipeline runs |
| `GET`  | `/api/runs/{id}` | Get a pipeline run |
| `GET`  | `/api/compliance/summary` | Compliance summary |
| `POST` | `/api/pipelines/run` | Batch run over `lead_ids` |

### Manual smoke test

```bash
# 1. Create a lead with an email and domain
LEAD=$(curl -s -X POST http://localhost:8080/api/leads \
  -H 'Content-Type: application/json' \
  -d '{"email":"support@github.com","company":"GitHub","domain":"github.com","permission_ref":"p-001"}' | jq -r '.data.id')

# 2. Run email-validate
curl -s -X POST "http://localhost:8080/api/leads/$LEAD/run" \
  -H 'Content-Type: application/json' \
  -d '{"modules":["email-validate"]}' | jq '.data.email_validate'

# 3. Run domain-intel (optionally set DOMAIN_INTEL_HARVESTER_BIN for theHarvester)
curl -s -X POST "http://localhost:8080/api/leads/$LEAD/run" \
  -H 'Content-Type: application/json' \
  -d '{"modules":["domain-intel"]}' | jq '.data.domain_intel'

# 4. Run social-footprint (uses email + any enriched domain_intel.harvester)
curl -s -X POST "http://localhost:8080/api/leads/$LEAD/run" \
  -H 'Content-Type: application/json' \
  -d '{"modules":["social-footprint"]}' | jq '.data.social_footprint'

# 5. Social-footprint on a lead with no usable handle stays skipped without crashing
HANDLELESS=$(curl -s -X POST http://localhost:8080/api/leads \
  -H 'Content-Type: application/json' \
  -d '{"company":"NoHandle"}' | jq -r '.data.id')
curl -s -X POST "http://localhost:8080/api/leads/$HANDLELESS/run" \
  -H 'Content-Type: application/json' \
  -d '{"modules":["social-footprint"]}' | jq '.data | {stage, social_footprint}'

# 6. Get the lead with audit events (raw_stderr_json and legal_basis)
curl -s "http://localhost:8080/api/leads/$LEAD" | jq '.data.audit_events'
```

With SMTP disabled, `deliverable` is typically `"unknown"` while `status` is
`"ok"` and `has_mx_records`/`syntax_valid` are true for real domains.

### Response shape notes

- Lead records expose module results as top-level keys (`email_validate`,
  `phone_validate`, `domain_intel`, `social_footprint`). The internal `results`
  map is not part of the public JSON.
- `risk_level` is one of `low`, `medium`, `high`, or `unknown`.
- Stage advances only when an executed module reports `status: "ok"`. Skipped or
  unknown modules do not move the lead forward (e.g., running `domain-intel` on a
  lead with no domain stays `raw`).
- Audit events use `raw_stderr_json` and include `legal_basis` on every line.

## Notes

- No mock Next.js APIs; the UI consumes this real API.
- No `modules/` code changes (`domain-intel` and `social-footprint` libraries are used as-is via `go.mod` replace).
- No `ui/web-console/`, evaluation, or CI changes in this PR.
- Audit events include `legal_basis` on every line and never store raw phone
  numbers (the `phone-validate` module redacts them before returning).
