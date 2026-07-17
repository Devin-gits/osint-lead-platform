# OSINT Lead Platform — v1 Status

This doc describes what works today in the `main` branch, how to run it, and what is intentionally out of scope or still open. It is the answer to "what can I actually run right now?"

---

## What works today

### Wired modules (available in `services/control-plane`)

| Module | Status | Input | Notes |
|--------|--------|-------|-------|
| `email-validate` | available | `email` | AfterShip/email-verifier, SMTP probe disabled, syntax/MX/disposable/role/free checks. |
| `phone-validate` | available | `phone` | libphonenumber offline parse; optional numverify carrier lookup. Phone redacted in audit logs. |
| `domain-intel` | available | `domain` | Go reimplementation of web-check (DNS/TLS/HTTP/WHOIS) + optional theHarvester subprocess. |
| `social-footprint` | available | `email` (+ `domain_intel.harvester` if present) | Derives up to 3 handles and runs Maigret Python wrapper over curated platform allow-list. Degrades to `unknown`/`skipped` if Python/wrapper missing. |
| `extraction` | planned | — | Not wired. |
| `company-enrich` | planned | — | Not wired. |

### UI screens (`ui/web-console`)

- `/` redirects to `/leads`.
- `/leads` — list, filters (stage, risk, module_status, q), stage funnel, multi-select bulk actions for every `available` module, live pagination.
- `/leads/[id]` — raw card, module result panels (Email/Phone/Domain/Social), expandable audit panel with `legal_basis` and `raw_stderr_json`, per-module "Run anyway" actions.
- `/modules` — grouped by `available` / `in_development` / `planned`.
- `/modules/[name]` — module docs from the registry.
- `/runs` and `/runs/[id]` — pipeline run timeline and detail.
- `/compliance` — hard rules, risk table, checklist, exclusions.
- `/settings` — environment badge, role selector, CRM/SSO/API/retention stubs.

### CI

- `.github/workflows/ui.yml` — typecheck, lint, build for `ui/web-console`.
- `.github/workflows/control-plane.yml` — `go test ./...` (full suite, no `-short`) and `go build ./...` for `services/control-plane`.

---

## How to run

Two processes are required.

### 1. Control-plane API (`:8080`)

```bash
cd services/control-plane

# with in-memory store (quickest start)
go run ./cmd/server

# or with Postgres
export DATABASE_URL='postgres://osint:osint@localhost:5432/osint?sslmode=disable'
go run ./cmd/server
```

### 2. Web console (`:3000`)

```bash
cd ui/web-console
npm install
npm run dev
```

Open [http://localhost:3000](http://localhost:3000). The UI expects the API at `http://localhost:8080`; override with `NEXT_PUBLIC_API_BASE_URL`.

### Manual smoke test

```bash
# Create a lead
LEAD=$(curl -s -X POST http://localhost:8080/api/leads \
  -H 'Content-Type: application/json' \
  -d '{"email":"support@github.com","company":"GitHub","domain":"github.com","permission_ref":"p-001"}' | jq -r '.data.id')

# Run modules
curl -s -X POST "http://localhost:8080/api/leads/$LEAD/run" \
  -H 'Content-Type: application/json' \
  -d '{"modules":["email-validate","domain-intel","social-footprint"]}' | jq '.data'

# View lead + audit events
curl -s "http://localhost:8080/api/leads/$LEAD" | jq '.data.audit_events'
```

---

## Env matrix

### Email/phone only (fast, no external Python tools)

```bash
# services/control-plane defaults are sufficient
go run ./cmd/server
```

### Full social + domain (theHarvester / Maigret)

```bash
# services/control-plane
export HTTP_WRITE_TIMEOUT=300s        # or 600s on slower networks
export SOCIAL_FOOTPRINT_TIMEOUT=90s   # per handle; default is fine
export SOCIAL_FOOTPRINT_BACKEND=maigret
export SOCIAL_FOOTPRINT_PYTHON=python3
export SOCIAL_FOOTPRINT_WRAPPER=wrapper/maigret_check.py
export DOMAIN_INTEL_HARVESTER_BIN=theHarvester
go run ./cmd/server
```

Why: `social-footprint` runs up to 3 handles × 90s each plus rate limits, and `domain-intel` may invoke theHarvester. `HTTP_WRITE_TIMEOUT` must exceed the longest expected request or the server closes the connection before the runner finishes.

---

## Compliance posture

- Every module call logs an `AuditEvent` with `tool`, `checked_at`, `status`, `legal_basis` (`GDPR Art.6(1)(f) legitimate-interest`), `subject` (email/domain/phone_redacted/handle), and `raw_stderr_json`.
- Phone numbers are redacted by `phone-validate` before returning to the control plane.
- Social footprint derives public handles only; raw email/name never appears in social audit lines.
- Curated platform allow-list in `modules/social-footprint` keeps the check bounded and ToS-respectful.
- The compliance page and `/api/compliance/summary` expose the hard rules and exclusions.

---

## Explicit non-goals (not in v1)

- Bulk breach/leak signals in sales views.
- LinkedIn scraping.
- Reverse-image / deep account discovery (GHunt-style).
- Profile-field scraping beyond handle presence.
- Real auth/SSO or CRM connector wiring.
- `extraction` / `company-enrich` execution.

---

## Known limitations

- **Social top-level status** can be `"ok"` even when every individual handle is `"unknown"` because the runner only degrades the lead if the module itself errors; the UI panel renders per-handle status chips.
- **Multi-handle duration** can exceed the default HTTP write timeout; operators must raise `HTTP_WRITE_TIMEOUT`.
- **crm_ready stage** is not set by the current stage machine; leads stop at `validated`.
- **Risk scoring** is not computed from module signals; `risk_level` remains `unknown` unless a future policy assigns it.
- **Compliance summary** is static governance content (hard rules, risk table, checklist, exclusions). It does not yet return numeric stats or per-run scores.
- **Async long runs** are not supported; batch runs execute synchronously inside the HTTP request.

---

## Backlog for v2 and beyond

- `crm_ready` stage policy and CRM export trigger.
- Risk scoring derived from email/phone/domain/social signals.
- `extraction` and `company-enrich` wiring.
- Async worker for long-running Maigret/theHarvester batch jobs.
- Retention/deletion enforcement in the backend.
