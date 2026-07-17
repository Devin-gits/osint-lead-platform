# Frontend Wikimap

> A concept map of the web console: screens, domain entities, API endpoints, components, and data flows.
> Companion to `megaplan.md` and `api-contracts.md`.

---

## 1. Concept graph

```
                         +------------------+
                         |   User (ops /    |
                         |  sales / compliance)
                         +--------+---------+
                                  |
          +-----------------------+-----------------------+
          |                       |                       |
+---------v---------+   +---------v---------+   +---------v---------+
|   Leads           |   |   Modules         |   |   Compliance      |
|   /leads          |   |   /modules        |   |   /compliance     |
|   /leads/[id]     |   |   /modules/[name]  |   |                   |
+---------+---------+   +---------+---------+   +---------+---------+
          |                       |                       |
          |        +--------------+                       |
          |        |                                      |
          |  +-----v------+                                 |
          |  |   Runs     |                                 |
          |  |   /runs    |                                 |
          |  |   /runs/[id]                                  |
          |  +-----+------+                                 |
          |        |                                      |
          +--------v--------------------------------------v
                   |
         +---------v---------+
         |   Settings        |
         |   /settings       |
         +---------+---------+
                   |
                   v
+----------------------------------+
|  App Shell (Sidebar + TopBar +   |
|  Footer + APIStatusBanner)       |
+----------------------------------+
```

---

## 2. Domain entities → files

| Entity | Source file | UI consumers |
|--------|-------------|--------------|
| `RawLead` | `lib/api/types.ts` | `LeadTable`, `RawLeadCard`, `LeadDetailTabs` |
| `LeadRecord` | `lib/api/types.ts` | everywhere in `/leads` |
| `EmailValidateResult` | `lib/api/types.ts` | `EmailResultCard` |
| `PhoneValidateResult` | `lib/api/types.ts` | `PhoneResultCard` |
| `DomainIntelResult` | `lib/api/types.ts` | `DomainResultCard` |
| `SocialFootprintResult` | `lib/api/types.ts` | `SocialResultCard` |
| `AuditEvent` | `lib/api/types.ts` | `AuditLogPanel`, `/compliance` |
| `ModuleInfo` | `lib/api/types.ts` | `ModuleGrid`, `ModuleDetailTabs` |
| `PipelineRun` | `lib/api/types.ts` | `RunTimeline`, `RunDetail` |

---

## 3. Screen → API endpoint → hook

| Screen | Control-plane endpoint | TanStack Query hook | Key components |
|--------|------------------------|---------------------|----------------|
| `/leads` | `GET /api/leads` | `useLeads` | `LeadFilters`, `StageFunnel`, `LeadTable` |
| `/leads/[id]` | `GET /api/leads/{id}` | `useLead` | `LeadDetailTabs`, `AuditLogPanel` |
| `/modules` | `GET /api/modules` | `useModules` | `ModuleGrid` |
| `/modules/[name]` | `GET /api/modules/{name}` | `useModule` | `ModuleDetailTabs`, `ModuleDocsPanel` |
| `/runs` | `GET /api/runs` | `useRuns` | `RunTimeline` |
| `/runs/[id]` | `GET /api/runs/{id}` | `useRun(id)` | `RunDetail` |
| `/compliance` | `GET /api/compliance/summary` | `useComplianceSummary` | `HardRules`, `RiskTable`, `PreRunChecklist`, `ExclusionsCallout` |
| `/settings` | none (local state) | none | `RoleSelector`, `EnvironmentSetting`, stub components |

---

## 4. Data flow

```
User action (filter, click, tab change)
        │
        ▼
  React component
        │
        ▼
  TanStack Query hook (lib/api/hooks.ts)
        │
        ▼
  API client (lib/api/client.ts)
        │
        ▼
fetch(`${NEXT_PUBLIC_API_BASE_URL}/api/*`)
        │
        ▼
services/control-plane API
        │
        ▼
Postgres (or in-memory store)
```

---

## 5. State map

| State | Location | Persisted? | Purpose |
|-------|----------|------------|---------|
| Server data | TanStack Query | In-memory cache | Leads, modules, runs, audit, compliance |
| UI role | Zustand (`lib/store/ui.ts`) | localStorage | `sales | admin | risk` stub |
| Sidebar open | Zustand | localStorage | Mobile/desktop sidebar toggle |
| Environment badge | Zustand | localStorage | `Sandbox | Production` stub; actual target URL comes from `NEXT_PUBLIC_API_BASE_URL` |
| Pre-run checklist | Zustand | session only | Interactive compliance checkboxes |
| Filters | URL search params + Zustand | URL | Shareable filtered list views |

---

## 6. Key design constraints (compliance & product)

- Every lead screen shows `permission_ref` and `legal_basis`.
- Social footprint shows only `platform`, `handle`, `status` (claimed/available/unknown), and `confidence`.
- No breach/leak signals in sales views; role stub exists for future admin/risk gating.
- No LinkedIn scraping, reverse-image, or GHunt-style modules exposed.
- API status banner reflects live connectivity to `NEXT_PUBLIC_API_BASE_URL`.
- `POST /api/pipelines/run` returns `202` with a `run_id`; UI links to `/runs/{run_id}` and may poll for completion.

---

## 7. File-to-concept index

- **App shell** → `components/layout/*`
- **Design tokens** → `lib/theme/tokens.ts`
- **API contracts** → `lib/api/types.ts` and `lib/api/client.ts`
- **Leads domain** → `app/leads/**`, `components/leads/**`
- **Modules domain** → `app/modules/**`, `components/modules/**`
- **Runs domain** → `app/runs/**`, `components/runs/**`
- **Compliance domain** → `app/compliance/**`, `components/compliance/**`
- **Settings domain** → `app/settings/**`, `components/settings/**`
- **Pure helpers** → `lib/utils/risk.ts`, `lib/utils/stage.ts`

---

*This wikimap is a planning artifact. Update it as routes, entities, or state management evolve.*
