# OSINT Lead Platform — Control Plane API

A real Go HTTP control plane that runs the existing domain modules and persists
leads, audit events, and pipeline runs.

## Scope

- Lives in `services/control-plane/**`.
- Wired modules in v1:
  - `email-validate` (in-process, AfterShip/email-verifier)
  - `phone-validate` (in-process, libphonenumber; optional numverify)
- Stubbed / not wired in v1:
  - `domain-intel`, `social-footprint`, `extraction`, `company-enrich`
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
| `MODULE_TIMEOUT` | `10s` | Per-module timeout |

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
# 1. Create a lead
LEAD=$(curl -s -X POST http://localhost:8080/api/leads \
  -H 'Content-Type: application/json' \
  -d '{"email":"support@github.com","company":"GitHub"}' | jq -r '.data.id')

# 2. Run email-validate
curl -s -X POST "http://localhost:8080/api/leads/$LEAD/run" \
  -H 'Content-Type: application/json' \
  -d '{"modules":["email-validate"]}' | jq '.data.results.email_validate'

# 3. Get the lead with audit events
curl -s "http://localhost:8080/api/leads/$LEAD" | jq '.data.audit_events'
```

With SMTP disabled, `deliverable` is typically `"unknown"` while `status` is
`"ok"` and `has_mx_records`/`syntax_valid` are true for real domains.

## Notes

- No mock Next.js APIs; the UI will consume this real API in the next PR.
- No `modules/`, `ui/web-console/`, evaluation, or CI changes in this PR.
- Audit events include `legal_basis` on every line and never store raw phone
  numbers (the `phone-validate` module redacts them before returning).
