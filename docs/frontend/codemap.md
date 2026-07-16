# Frontend Codemap

> **Project area:** `ui/web-console/` — Next.js 15 App Router control plane.
> **Planning docs:** `docs/frontend/megaplan.md` + `docs/frontend/api-contracts.md`.

---

## 1. Top-level layout

```
ui/web-console/
├── README.md
├── package.json
├── tsconfig.json
├── next.config.ts
├── tailwind.config.ts
├── postcss.config.mjs
├── eslint.config.mjs
├── public/
│   └── (favicon, logo placeholder)
├── app/
│   ├── layout.tsx              # root layout: fonts, providers, shell
│   ├── page.tsx                # redirect to /leads
│   ├── globals.css             # Tailwind + design token CSS variables
│   ├── leads/
│   │   ├── page.tsx            # leads list
│   │   └── [id]/
│   │       └── page.tsx        # lead detail
│   ├── modules/
│   │   ├── page.tsx            # module grid
│   │   └── [name]/
│   │       └── page.tsx        # module detail
│   ├── runs/
│   │   ├── page.tsx            # runs timeline
│   │   └── [id]/
│   │       └── page.tsx        # run detail (optional)
│   ├── compliance/
│   │   └── page.tsx            # compliance page
│   ├── settings/
│   │   └── page.tsx            # settings stubs
│   └── api/
│       ├── leads/
│       │   ├── route.ts
│       │   └── [id]/
│       │       └── route.ts
│       ├── modules/
│       │   ├── route.ts
│       │   └── [name]/
│       │       └── route.ts
│       ├── audit/
│       │   └── route.ts
│       ├── runs/
│       │   └── route.ts
│       ├── compliance/
│       │   └── summary/
│       │       └── route.ts
│       └── pipelines/
│           └── run/
│               └── route.ts
├── components/
│   ├── layout/
│   │   ├── Sidebar.tsx
│   │   ├── TopBar.tsx
│   │   ├── Footer.tsx
│   │   ├── AppShell.tsx
│   │   └── MockDataBanner.tsx
│   ├── ui/                     # design-system primitives
│   │   ├── Button.tsx
│   │   ├── IconButton.tsx
│   │   ├── Input.tsx
│   │   ├── Select.tsx
│   │   ├── Textarea.tsx
│   │   ├── Card.tsx
│   │   ├── Badge.tsx
│   │   ├── Table.tsx
│   │   ├── Tabs.tsx
│   │   ├── Dialog.tsx
│   │   ├── Toast.tsx
│   │   ├── Tooltip.tsx
│   │   ├── Skeleton.tsx
│   │   ├── EmptyState.tsx
│   │   ├── PageHeader.tsx
│   │   ├── StatusChip.tsx
│   │   ├── PipelineStepper.tsx
│   │   └── AuditLogPanel.tsx
│   ├── leads/                  # page-specific components
│   │   ├── LeadFilters.tsx
│   │   ├── StageFunnel.tsx
│   │   ├── LeadTable.tsx
│   │   ├── LeadDetailTabs.tsx
│   │   ├── RawLeadCard.tsx
│   │   ├── EnrichedCard.tsx
│   │   ├── EmailResultCard.tsx
│   │   ├── PhoneResultCard.tsx
│   │   ├── DomainResultCard.tsx
│   │   └── SocialResultCard.tsx
│   ├── modules/
│   │   ├── ModuleGrid.tsx
│   │   ├── ModuleDetailTabs.tsx
│   │   ├── ModuleDocsPanel.tsx
│   │   ├── ModuleConfigPanel.tsx
│   │   └── ModuleHealthPanel.tsx
│   ├── runs/
│   │   ├── RunTimeline.tsx
│   │   └── RunDetail.tsx
│   ├── compliance/
│   │   ├── HardRules.tsx
│   │   ├── RiskTable.tsx
│   │   ├── PreRunChecklist.tsx
│   │   └── ExclusionsCallout.tsx
│   └── settings/
│       ├── EnvironmentSetting.tsx
│       ├── RoleSelector.tsx
│       ├── CrmConnectorStub.tsx
│       ├── SsoOidStub.tsx
│       ├── ApiKeysVaultStub.tsx
│       └── RetentionPolicyStub.tsx
├── lib/
│   ├── theme/
│   │   └── tokens.ts           # design tokens (colors, typography, motion)
│   ├── api/
│   │   ├── types.ts            # domain types (frozen in api-contracts.md)
│   │   ├── client.ts           # fetch-first API client with seed fallback
│   │   └── hooks.ts            # TanStack Query hooks
│   ├── mocks/
│   │   └── seed.ts             # mock dataset + in-memory filtering helpers
│   ├── store/
│   │   └── ui.ts               # Zustand store for ephemeral UI state (role, sidebar)
│   ├── utils/
│   │   ├── cn.ts               # Tailwind class merge
│   │   ├── risk.ts             # risk-level helpers (tested)
│   │   └── stage.ts            # pipeline-stage helpers (tested)
│   └── validators/
│       └── search.ts           # query-param validation helpers
├── content/
│   └── docs/
│       └── email-validate.md   # static docs copy for module detail
└── __tests__/
    ├── risk.test.ts
    └── stage.test.ts
```

---

## 2. Naming & placement conventions

| Rule | Example |
|------|---------|
| App Router pages live under `app/<route>/page.tsx` | `app/leads/page.tsx` |
| API routes live under `app/api/<route>/route.ts` | `app/api/leads/route.ts` |
| Page-specific components under `components/<route>/` | `components/leads/LeadTable.tsx` |
| Shared UI primitives under `components/ui/` | `components/ui/Button.tsx` |
| Domain logic under `lib/` | `lib/api/client.ts`, `lib/theme/tokens.ts` |
| Static content under `content/` | `content/docs/email-validate.md` |
| Utility helpers are pure and unit-tested | `lib/utils/risk.ts` |

---

## 3. Design token usage

- All colors, spacing, typography, and motion values come from `lib/theme/tokens.ts`.
- Tailwind config maps token values to CSS custom properties so components can use `bg-surface`, `border-primary/12`, `text-meta`, etc.
- No hard-coded hex values in page or component files.
- Dark theme only for v1; tokens are structured to support a future light theme.

---

## 4. State management

- **Server state:** TanStack Query via `lib/api/hooks.ts`.
- **Ephemeral UI state:** Zustand in `lib/store/ui.ts` (sidebar collapse, role, environment, checklist state).
- **No global mutable lead store** — all lead data is fetched or derived from mock seed.

---

## 5. Mock data strategy

- `lib/mocks/seed.ts` exports the canonical mock dataset and filtering/sorting functions.
- `app/api/*` route handlers consume `seed.ts`.
- `lib/api/client.ts` attempts `fetch('/api/...')` first, then falls back to `seed.ts` if the route is unavailable (e.g. static export).
- All mock routes return `{ data, meta? }` envelopes.
- Every screen using mock data shows `<MockDataBanner />`.

---

## 6. Compliance & security conventions

- Legal basis (`GDPR Art.6(1)(f) legitimate-interest`) and `permission_ref` are rendered on every lead, run, and audit event.
- Phone audit subjects use `phone_redacted` (`+14*******86`).
- Social footprint audit subjects use `handle` only.
- Sales views never show breach/leak signals; no reverse-image or LinkedIn modules exist in the UI.
- Role selector in Settings (`sales | admin | risk`) is local state only and gates any future sensitive views.

---

## 7. Testing approach

- `lib/utils/risk.ts` and `lib/utils/stage.ts` are pure and unit-tested.
- Components use semantic HTML, `aria-label` on icon buttons, and visible focus rings.
- A single Playwright/RTL smoke test checks navigation and the mock-data banner after PR4.
- `npm run build`, `npm run lint`, and `npm run typecheck` are required to pass in every PR.

---

## 8. Dependency rules

- Next.js 15 (App Router), TypeScript strict, Tailwind CSS.
- `@tanstack/react-query` for server state.
- `zustand` for ephemeral UI state.
- `lucide-react` or `@heroicons/react` for icons (MIT-compatible).
- No GPL/AGPL libraries imported as dependencies.
- All dependency versions pinned.

---

## 9. Boundaries

- This frontend does **not** touch `modules/`, Go code, CI, or evaluations.
- It does **not** implement a real orchestrator; `POST /api/pipelines/run` returns `501` or a stub accepted response.
- It does **not** store real secrets; all settings are stubs.

---

*This codemap is a living planning document. Update it when the real backend contract or route structure changes.*
