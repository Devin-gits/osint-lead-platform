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

```bash
cd ui/web-console
npm install
npm run dev
```

Open [http://localhost:3000](http://localhost:3000).

## Build & quality gates

```bash
npm run build      # production build (do not enable output: 'export' in v1)
npm run typecheck  # tsc --noEmit
npm run lint       # next lint
```

## Architecture notes

- All UI code lives under `ui/web-console/**`.
- This PR intentionally does not touch `modules/**`, Go code, or evaluations.
- The app runs in Node server mode by default so Next.js App Router API routes can be added in PR2.
- A client-side seed fallback will be added in `lib/api/client.ts` for resilience, but `output: 'export'` is not a v1 target.
- Every mock-data screen shows the persistent `MockDataBanner`.
- Design tokens are defined in `lib/theme/tokens.ts` and mirrored as CSS custom properties in `app/globals.css`; no one-off colors in pages.
