# OSINT Lead Platform — v1 Status

This doc describes what works today in the `main` branch, how to run it, and what is intentionally out of scope or still open. It is the answer to "what can I actually run right now?"

---

## What works today

### Wired modules (available in `services/control-plane`)

| Module | Status | Minimum input | Backing tools | Notes |
|---|---|---|---|---|
| `email-validate` | available | `email` | AfterShip/email-verifier | Syntax/MX/disposable/role/free checks. SMTP probe disabled, so `deliverable` is often `unknown` while `status` is `ok`. |
| `phone-validate` | available | `phone` | libphonenumber | Offline parse; optional numverify carrier lookup. Phone numbers are redacted in audit logs. |
| `domain-intel` | available | `domain` | Go web-check reimplementation + optional theHarvester | DNS/TLS/HTTP/WHOIS + optional theHarvester subprocess for hosts/emails. |
| `social-footprint` | available | `email` (uses `domain_intel.harvester` if present) | Maigret Python wrapper | Derives up to 3 handles and checks a curated platform allow-list. Degrades to `unknown`/`skipped` if Python/wrapper missing. |
| `extraction` | available | `url` + `permission_ref` | Crawl4AI 0.9.2 Python wrapper (default); optional Firecrawl | Fetches a public page and extracts low-risk fields (title, description, company name, emails, phones, social links, contact URLs). Requires a public `url`; `permission_ref` is mandatory. |
| `company-enrich` | available | `domain`/`company`/`url` + `permission_ref` | `company-enrich/local` (deterministic); optional DiscoLike adapter via `DISCOLIKE_API_KEY` | Enriches company firmographics. Control-plane wired in Pass 3C; UI tab is Pass 3D. |

### UI screens (`ui/web-console`)

- `/` redirects to `/leads`.
- `/leads` — list, filters (stage, risk, module_status, q), stage funnel, multi-select bulk actions for every `available` module, live pagination.
- `/leads/[id]` — raw card with `url` field, module result panels (Email/Phone/Domain/Social/Extraction), expandable audit panel with `legal_basis` and `raw_stderr_json`, per-module "Run anyway" actions.
- `/modules` — grouped by `available` / `in_development` / `planned`.
- `/modules/[name]` — module docs from the registry.
- `/runs` and `/runs/[id]` — pipeline run timeline and detail.
- `/compliance` — hard rules, risk table, checklist, exclusions.
- `/settings` — environment badge, role selector, CRM/SSO/API/retention stubs.

### CI

- `.github/workflows/ui.yml` — `typecheck`, `lint`, `build` for `ui/web-console`.
- `.github/workflows/control-plane.yml` — `go vet ./...`, `go test ./...`, `go test -short ./...`, `go build ./...` for `services/control-plane`.
- `.github/workflows/extraction-ci.yml` — `go test ./...`, `go build ./...` for `modules/extraction`.

---

## How to run

The fastest path is the root `Makefile`:

```bash
# Terminal 1 — control-plane API (http://localhost:8080)
make demo-api

# Terminal 2 — Next.js UI (http://localhost:3000)
cd ui/web-console && npm install   # once
make demo-ui
```

Then open [http://localhost:3000](http://localhost:3000). The UI expects the API at `http://localhost:8080`; override with `NEXT_PUBLIC_API_BASE_URL`.

### One-shot extraction smoke test

With the API running:

```bash
make smoke
```

This runs `scripts/smoke-extraction.sh`, which creates a lead, runs `extraction`, and prints the result and audit event. It exits 0 for `ok`, `partial`, `skipped`, or a structured `error` (e.g., Crawl4AI not installed). Use `make smoke-ok` to require `ok`/`partial`.

### Full extraction `ok` path

To see `extraction.status = ok`, install Crawl4AI in a venv and point the control-plane at it:

```bash
cd modules/extraction
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
# Optional: only if Crawl4AI reports a missing browser
playwright install chromium
cd ../../services/control-plane
EXTRACTION_CRAWL4AI_PYTHON="$PWD/../../modules/extraction/.venv/bin/python" go run ./cmd/server
```

Or from the repo root with the Makefile:

```bash
make install-extraction-venv
EXTRACTION_CRAWL4AI_PYTHON="$PWD/modules/extraction/.venv/bin/python" make demo-api-ok
```

Then `make smoke` (or `make smoke-ok`) will report `ok` on public pages such as `https://example.com`.

### Manual smoke test

```bash
# Create a lead with a URL and permission_ref
LEAD=$(curl -s -X POST http://localhost:8080/api/leads \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com","permission_ref":"p-001"}' | jq -r '.data.id')

# Run extraction
curl -s -X POST "http://localhost:8080/api/leads/$LEAD/run" \
  -H 'Content-Type: application/json' \
  -d '{"modules":["extraction"]}' | jq '.data.extraction'

# View lead + audit events
curl -s "http://localhost:8080/api/leads/$LEAD" | jq '.data.audit_events[0]'
```

Missing `permission_ref`:

```bash
NOPERM=$(curl -s -X POST http://localhost:8080/api/leads \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com"}' | jq -r '.data.id')
curl -s -X POST "http://localhost:8080/api/leads/$NOPERM/run" \
  -H 'Content-Type: application/json' \
  -d '{"modules":["extraction"]}' | jq '.data.extraction'
# -> { "status": "skipped", "reason": "missing permission_ref" }
```

---

## Env matrix

### Email/phone only (fast, no external Python tools)

```bash
# services/control-plane defaults are sufficient
go run ./cmd/server
```

### Full extraction `ok` path

```bash
export EXTRACTION_CRAWL4AI_PYTHON="$PWD/modules/extraction/.venv/bin/python"
export EXTRACTION_TIMEOUT=45s
cd services/control-plane && go run ./cmd/server
```

Crawl4AI must be installed in that Python environment (`modules/extraction/requirements.txt`). Playwright Chromium may also be required depending on the environment.

### Full social + domain (theHarvester / Maigret)

```bash
export HTTP_WRITE_TIMEOUT=300s        # or 600s on slower networks
export SOCIAL_FOOTPRINT_TIMEOUT=90s   # per handle; default is fine
export SOCIAL_FOOTPRINT_BACKEND=maigret
export SOCIAL_FOOTPRINT_PYTHON=python3
export SOCIAL_FOOTPRINT_WRAPPER=wrapper/maigret_check.py
export DOMAIN_INTEL_HARVESTER_BIN=theHarvester
cd services/control-plane && go run ./cmd/server
```

Why: `social-footprint` runs up to 3 handles × 90s each plus rate limits, and `domain-intel` may invoke theHarvester. `HTTP_WRITE_TIMEOUT` must exceed the longest expected request or the server closes the connection before the runner finishes.

---

## Compliance posture

- Every module call logs an `AuditEvent` with `tool`, `checked_at`, `status`, `legal_basis` (`GDPR Art.6(1)(f) legitimate-interest`), `subject` (email/domain/phone_redacted/handle/url), and `raw_stderr_json`.
- `permission_ref` is **mandatory** for `extraction`. Missing `permission_ref` produces a `skipped` result and audit line.
- Phone numbers are redacted by `phone-validate` before returning to the control plane.
- Social footprint derives public handles only; raw email/name never appears in social audit lines.
- Extraction audit subject is the sanitized URL only; query values are redacted and userinfo stripped.
- Curated platform allow-list in `modules/social-footprint` keeps the check bounded and ToS-respectful.
- The compliance page and `/api/compliance/summary` expose the hard rules and exclusions.

---

## Explicit non-goals (not in v1)

- Bulk breach/leak signals in sales views.
- LinkedIn scraping or profile-field extraction beyond public links found on a page.
- Reverse-image / deep account discovery (GHunt-style).
- Real auth/SSO or CRM connector wiring.

---

## Known limitations

- **Social top-level status** can be `"ok"` even when every individual handle is `"unknown"` because the runner only degrades the lead if the module itself errors; the UI panel renders per-handle status chips.
- **Multi-handle duration** can exceed the default HTTP write timeout; operators must raise `HTTP_WRITE_TIMEOUT`.
- **crm_ready stage** is not set by the current stage machine; leads stop at `validated`.
- **Risk is computed from email + phone only** via `runner.computeRisk()`; `domain_intel` and `social_footprint` do not affect `risk_level` today. `risk_score` is stored if present but is not a composite algorithm field.
- **Compliance summary** is static governance content (hard rules, risk table, checklist, exclusions). It does not yet return numeric stats or per-run scores.
- **Async long runs** are not supported; batch runs execute synchronously inside the HTTP request.

---

## Backlog for v2 and beyond

- `crm_ready` stage policy and CRM export trigger.
- Extend risk scoring to include `domain_intel` and `social_footprint` signals; define a `risk_score` algorithm if needed.
- `company-enrich`: available in control-plane; UI Company tab is Pass 3D.
- Async worker for long-running Maigret/theHarvester batch jobs.
- Retention/deletion enforcement in the backend.
