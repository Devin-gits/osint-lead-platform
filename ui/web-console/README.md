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

Open [http://localhost:3000](http://localhost:3000). The API is expected at
`http://localhost:8080` by default; set `NEXT_PUBLIC_API_BASE_URL` to override.

## Build & quality gates

```bash
npm run build      # production build
npm run typecheck  # tsc --noEmit
npm run lint       # next lint
```

## Architecture notes

- All UI code lives under `ui/web-console/**`.
- This PR intentionally does not touch `modules/**`, Go code, or evaluations.
- The app runs in Node server mode; no `output: 'export'` target in v1.
- API client, types, and TanStack Query hooks live in `lib/api/**`.
- The `EnvironmentBanner` and top-bar badge reflect live API connectivity.
- Design tokens are defined in `lib/theme/tokens.ts` and mirrored as CSS custom properties in `app/globals.css`; no one-off colors in pages.
