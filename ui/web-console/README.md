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
# Terminal 1 — Go control plane
cd services/control-plane
go run ./cmd/server
```

```bash
# Terminal 2 — Next.js UI
cd ui/web-console
npm install
npm run dev
```

Open [http://localhost:3000](http://localhost:3000). The root route (`/`) redirects to `/command-center`. The API is expected at `http://localhost:8080` by default; set `NEXT_PUBLIC_API_BASE_URL` to override.

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
- Lead detail shows module result tabs, per-module run actions, and an expandable audit panel.
  The Domain tab renders DNS, SSL/TLS, HTTP, WHOIS and theHarvester cards.
  The Social tab renders per-handle status and claimed/available platform chips.
- The leads list bulk-actions bar dynamically offers every `available` module returned by `/api/modules`.
- `/runs/[id]` shows a single pipeline run and links back to its leads.
- `social-footprint` requires the Maigret Python wrapper on the API host; long Maigret/theHarvester
  runs may need `MODULE_TIMEOUT` and the control-plane HTTP write timeout raised — see
  `services/control-plane/README.md`.
- Design tokens are defined in `lib/theme/tokens.ts` and mirrored as CSS custom properties in `app/globals.css`; no one-off colors in pages.
