# OSINT Lead Console

A Next.js 15 App Router control plane for the OSINT Lead Enrichment Platform.

## Stack

- Next.js 15 (App Router)
- React 19
- TypeScript (strict mode)
- Tailwind CSS 4
- TanStack Query (server state)
- Zustand (ephemeral UI state)
- lucide-react (icons)

## Development

Requires Node.js 20+ and `npm`.

The UI talks to the real control-plane API. Run both in separate terminals:

```bash
# Terminal 1 — Go control plane (with extraction wrapper discoverable)
cd services/control-plane
# Optional: set wrapper path explicitly if auto-locate does not find it
# export EXTRACTION_CRAWL4AI_WRAPPER=../../modules/extraction/wrapper/crawl4ai_extract.py
# Optional: install Crawl4AI in the active Python env for live ok results
# pip install -r ../../modules/extraction/requirements.txt
go run ./cmd/server
```

```bash
# Terminal 2 — Next.js UI
cd ui/web-console
npm install
npm run dev
```

Open [http://localhost:3000](http://localhost:3000). The root route (`/`) redirects to `/command-center`. The API is expected at `http://localhost:8080` by default; set `NEXT_PUBLIC_API_BASE_URL` to override. For the one-command loopback demo and current operator path, see [the local smoke runbook](../../docs/runbooks/local-dev-smoke.md).

Never run `npm run build` while `npm run dev` uses the same `.next` directory; it causes `ENOENT` `app-build-manifest` 500 errors.

Bulk module actions return `202` and show an active-run banner on `/leads`; the UI polls its run status, refreshes the leads list, and dismisses the banner when the run reaches a terminal state.

### Manual smoke test

With the control-plane running on `http://localhost:8080`:

```bash
# 1. Verify all modules are available
curl -s http://localhost:8080/api/modules | jq '.data[] | {name, dev_status}'

# 2. Create a lead with domain, company, and permission_ref
LEAD=$(curl -s -X POST http://localhost:8080/api/leads \
  -H 'Content-Type: application/json' \
  -d '{"domain":"example.com","company":"Example","permission_ref":"UI-SMOKE-1"}' | jq -r '.data.id')

# 3. Run company-enrich
curl -s -X POST "http://localhost:8080/api/leads/$LEAD/run" \
  -H 'Content-Type: application/json' \
  -d '{"modules":["company-enrich"]}' | jq '.data.company_enrich | {status, fields: .fields | {domain, name, website}}'

# 4. Open the lead in the UI, click the Company tab, and confirm status is "ok"

# 5. Promote to CRM-ready and export
# Run Email validate and Company enrich, then use the CRM readiness card:
#   - Promote to CRM-ready
#   - Export stub (downloads a JSON file)
#   - Demote back to validated
```

A domain-only lead returns `partial` with an empty company name; this is the
honest, expected behaviour. `partial` does not advance the lead to `enriched`.
Promotion to `crm_ready` is explicit and only succeeds when all readiness checks
pass.

## Sidebar breakpoints

| Viewport | Behavior |
|----------|----------|
| `>= 1280px` | Fixed expanded sidebar (14rem) with icons + labels. |
| `1024px–1279px` | Icon rail (4rem), labels hidden. |
| `< 1024px` | No persistent sidebar; top-bar menu opens a drawer overlay. |

Keyboard: `Esc` closes the mobile drawer; focus is trapped inside and restored to the menu trigger.

## Build & quality gates

```bash
npm run build      # production build
npm run typecheck  # tsc --noEmit
npm run lint       # next lint
```

## Architecture notes

- All UI code lives under `ui/web-console/**`.
- This PR intentionally does not touch `modules/**`, Go code, evaluations, or API contracts.
- The app runs in Node server mode; no `output: 'export'` target in v1.
- API client, types, and TanStack Query hooks live in `lib/api/**`.
- `APIHealthIndicator` uses `GET /api/leads?page_size=1` to test whether the control-plane API is reachable. It does **not** claim database, runner, or module health.
- Leads list supports filters, stage funnel, permission-ref warnings, multi-select bulk runs, and live pagination.
- Lead detail shows module result tabs, per-module run actions, an expandable audit panel,
  and a CRM readiness card with promote/demote/export actions.
  The Domain tab renders DNS, SSL/TLS, HTTP, WHOIS and theHarvester cards.
  The Social tab renders per-handle status and claimed/available platform chips.
  The Extraction tab renders extracted fields, provenance, and collapsible raw markdown (UI-truncated).
  The Company tab renders firmographics from `company-enrich` (identity, description, size/maturity, headquarters, industry/tech stack, social links, sources) and collapsible raw JSON.
- The leads list bulk-actions bar dynamically offers every `available` module returned by `/api/modules`.
- `/runs/[id]` shows a single pipeline run and links back to its leads.
- `social-footprint` requires the Maigret Python wrapper on the API host; long Maigret/theHarvester
  runs may need `MODULE_TIMEOUT` and the control-plane HTTP write timeout raised — see
  `services/control-plane/README.md`.
- `extraction` requires a public `url` and `permission_ref` on the lead. It uses Crawl4AI by default
  (Python wrapper in `modules/extraction/wrapper/crawl4ai_extract.py`) or optional Firecrawl when
  configured. Missing Crawl4AI yields a structured `error`/`skipped` result in the UI, not a fake success.
- Design tokens are defined in `lib/theme/tokens.ts` and mirrored as CSS custom properties in `app/globals.css`; no one-off colors in pages.
