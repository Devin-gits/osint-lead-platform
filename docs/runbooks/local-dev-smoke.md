# Local development smoke runbook

This runbook verifies the control-plane and UI locally without any paid API keys.
It uses the in-memory store, the deterministic `company-enrich/local` provider,
and `example.com` so no live OSINT scraping is required.

## Prerequisites

```bash
# Go 1.22.5+ (pin exact via go.mod)
go version

# Node 20+ (pin exact via .nvmrc or package.json when provided)
cd ui/web-console && node --version

# jq
jq --version
```

## 1. Build everything locally

```bash
make test-go   # module tests + control-plane tests
make test-ui   # typecheck, lint, build
```

Or run the pieces individually:

```bash
for m in modules/*/; do
  [ -f "$m/go.mod" ] || continue
  (cd "$m" && go test -short ./... && go test ./... && go vet ./... && go build ./...)
done
cd services/control-plane && go test ./... && go vet ./... && go build ./...
cd ui/web-console && npm run typecheck && npm run lint && npm run build
```

## 2. Start the control-plane (memory store)

```bash
make demo-api
# or
cd services/control-plane && go run ./cmd/server
```

Wait until you see `store: memory (set DATABASE_URL for postgres)`. The API is
ready at `http://localhost:8080`.

## 3. Start the web console

In a second terminal:

```bash
make demo-ui
# or
cd ui/web-console && NEXT_PUBLIC_API_BASE_URL=http://localhost:8080 npm run dev
```

Open `http://localhost:3000` (or the port printed by `next dev`).

## 4. Run operator API smoke

In a third terminal:

```bash
make smoke-api
# or
./scripts/smoke-api.sh
```

Expected output:

```text
==> Smoke target: http://localhost:8080
==> company-enrich module is available
==> domain-intel module is available
==> email-validate module is available
==> extraction module is available
==> phone-validate module is available
==> social-footprint module is available
==> company-enrich status: ok
{ ... "name": "Example" ... }
==> company-enrich status: partial
{ ... "name": "" ... }
==> company-enrich status: skipped, reason: missing permission_ref
==> smoke-api passed
```

### CRM-ready smoke

```bash
make smoke-crm
# or
./scripts/smoke-crm-ready.sh
```

### Risk-score v2 smoke

```bash
make smoke-risk
# or
./scripts/smoke-risk.sh
```

Expected output:

```text
==> Smoke target: http://localhost:8080
==> lead <id>
==> promoting before validation (expect 409)
==> checking readiness before validation
==> running email-validate and company-enrich
==> checking readiness after validation
==> promoting to crm_ready
==> lead stage is crm_ready
==> exporting lead
{ "format": "crm_stub_v1", ... }
==> demoting lead to validated
==> smoke-crm-ready passed
```

Risk-score expected output:

```text
==> Smoke target: http://localhost:8080
==> no-signal lead <id>
==> no-signal risk level: unknown
==> lead <id>
==> risk before validation (expect low with unvalidated-contact score)
before: low 10
==> running email-validate and company-enrich
==> risk after validation (expect low numeric score)
after: low <score>
==> factors present
>=2
==> smoke-risk passed
```

## 5. Manual UI checks

1. Open `http://localhost:3000/modules` — `company-enrich` should show
   `available`.
2. Create a lead with:
   - `domain`: `example.com`
   - `company`: `Example`
   - `permission_ref`: `SMOKE-UI-1`
3. Open the lead detail page.
4. Click the **Company** tab → **Run company enrich**.
   - Expect status `ok`, `name: Example`, `website: https://example.com`,
     `stage` updated to `enriched`.
5. Create a second lead with only:
   - `domain`: `example.com`
   - `permission_ref`: `SMOKE-UI-2`
6. Run **Company enrich** on the second lead.
   - Expect status `partial`, company name section shows honest copy
     (“Company name could not be derived from this input”), `stage` stays `raw`.
7. Create a third lead without `permission_ref` and with `domain` only.
   - The **Run company enrich** button should be disabled with the tooltip
     `needs permission ref`.
8. Promote a validated lead:
   - Create a lead with `email`, `company`, `domain`, and `permission_ref`.
   - Run **Email validate** and **Company enrich**.
   - The **CRM readiness** card should show all checks passing.
   - Click **Promote to CRM-ready** → stage chip updates to `crm ready`.
   - Click **Export stub** → a JSON file downloads with `format: crm_stub_v1`.
   - Click **Demote** → stage returns to `validated` and **Export stub** becomes
     unavailable.
9. Risk score:
   - Open a lead with no modules run → risk card shows `—` with `unknown` badge.
   - Run **Email validate** and **Company enrich** → risk card shows `low` and a
     numeric score; expanding factors shows `contact_validated` and `company_context_ok`.
10. The `EnvironmentBanner` at the top should say `Live API` when the
   control-plane is reachable; it should never say `mock data`.

## Expected behaviour notes

- A domain-only lead returns `partial` **by design**. The local provider does not
  invent a company name from the domain root.
- `partial` does **not** advance the lead to `enriched`. Only `ok` advances the
  stage.
- No `DISCOLIKE_API_KEY`, `FIRECRAWL_API_KEY`, or LinkedIn/LLM credentials are
  required for this smoke path.

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `control-plane not reachable` | control-plane not started | `make demo-api` |
| `extraction module is not available` | extraction Python venv not set | optional; this runbook does not require extraction |
| `company-enrich status: skipped, reason: missing permission_ref` | lead missing `permission_ref` | add a `permission_ref` value |
| `UI shows no Company tab` | UI not built / API not reachable | check `NEXT_PUBLIC_API_BASE_URL` and rebuild |
| `npm run build` fails | Node version mismatch | use Node 20 LTS |
| `go test ./...` hangs | network-dependent test blocked | use `go test -short ./...` or run in an environment with outbound DNS |
