# Frontend Control Plane — Megaplan

> **Scope:** build the UI/UX control plane for the OSINT Lead Enrichment Platform.
> **Repository:** `Moyeil-73/osint-lead-platform` (default branch `main`).
> **Target directory:** `ui/web-console/**` and, for planning/docs only, `docs/frontend/**`.
> **Hard constraint:** no `modules/**`, Go code, evaluations, orchestration backend, or real pipeline runner is created or modified in frontend PRs.

---

## 1. Goals & non-goals

### Goals

- Give ops/sales a dark, compact, enterprise-grade web console to:
  - watch leads move through `raw → enriched → validated → crm_ready`;
  - inspect per-module results and the full compliance audit trail;
  - manage module status/configuration placeholders until real orchestration exists.
- Surface only compliant, non-sensitive data in sales views.
- Provide a mock API layer (Next.js App Router route handlers) that mirrors the planned backend contract, plus a client-side seed fallback so the app can run in static-export mode if ever needed.
- Ship PR-by-PR, one concern per PR, with build/type/lint gates.

### Non-goals

- No real orchestration backend, no SpiderFoot glue, no custom Go orchestrator.
- No live module execution from the UI.
- No LinkedIn scraping, reverse-image, GHunt-style tooling, or breach/leak signal surfaced to sales.
- No auth/SSO integration in PRs 1–4 (only local role stubs).
- No real secrets storage or CRM connector wiring.

---

## 2. User personas

| Persona | Primary need | Typical screens |
|---------|--------------|-----------------|
| **Ops analyst** | See which leads are valid/risky, why, and rerun a module. | `/leads`, `/leads/[id]`, `/modules/[name]` |
| **Sales ops** | Export/schedule CRM-ready leads and trust the validation summary. | `/leads` (filters), `/runs`, `/compliance` (read-only) |
| **Compliance reviewer** | Confirm legal basis, permission refs, redaction, and exclusions. | `/leads/[id]` audit log, `/compliance`, `/settings` role checks |

---

## 3. IA / sitemap

```
/                         → redirect to /leads
/leads                    → lead list + filters + stage funnel
/leads/[id]               → lead detail (raw, enriched, module tabs, audit panel)
/modules                  → module grid (6 modules)
/modules/[name]           → module detail (overview | config | health | docs)
/runs                     → pipeline runs timeline
/runs/[id]                → run detail (related leads + audit subset)
/compliance               → hard rules, risk table, pre-run checklist
/settings                 → env, CRM, SSO, API keys, role selector stubs
```

### App shell

- **Left sidebar:** collapsible on mobile; links: Leads, Modules, Runs, Compliance, Settings.
- **Top bar:** product name “OSINT Lead Console”, environment badge (`Sandbox | Production stub`), user avatar stub, search stub.
- **Content area:** max-width container, dark theme only for v1.
- **Footer strip:** legal-basis reminder + link to `/compliance`.
- **Mock data banner:** persistent, non-dismissible: `Mock data — backend not wired yet`.

---

## 4. Design system & tokens

### Palette

All UI uses tokens from `lib/theme/tokens.ts`; no hard-coded one-off colors in pages.

```ts
export const tokens = {
  color: {
    background: '#050816',
    surface: {
      DEFAULT: '#0b1224',
      elevated: '#0f172a',
    },
    border: 'rgba(45,212,255,0.12)',
    primary: '#2dd4ff',
    secondary: '#6366f1',
    success: '#34d399',
    warning: '#fbbf24',
    danger: '#f97373',
    muted: '#94a3b8',
    text: {
      primary: '#f8fafc',
      secondary: '#cbd5e1',
      meta: '#94a3b8',
    },
    focusGlow: '0 0 0 2px rgba(45,212,255,0.25)',
  },
  font: {
    sans: ['Inter', 'ui-sans-serif', 'system-ui', 'sans-serif'],
    bodySize: '14px',
    metaSize: '12px',
    heading: {
      weight: 700,
      tracking: '-0.025em',
    },
  },
  spacing: {
    density: 'compact',
    pagePad: '1.5rem',
    cardPad: '1rem',
  },
  motion: {
    fast: '150ms',
    medium: '200ms',
    easing: 'cubic-bezier(0.4, 0, 0.2, 1)',
    reducedMotion: 'prefers-reduced-motion',
  },
  radii: {
    card: '0.5rem',
    button: '0.375rem',
    badge: '9999px',
  },
};
```

### Components (under `components/ui/`)

| Component | Purpose |
|-----------|---------|
| `Button` | Primary/secondary/ghost actions; focus ring with cyan glow |
| `IconButton` | Toolbar/compact actions (aria-label required) |
| `Input` | Form controls on dark surfaces |
| `Select` | Single filters and settings |
| `Textarea` | Audit raw JSON expand, docs preview |
| `Card` | `surface` bg + 1px cyan border |
| `Badge` / `Tag` | Stage, risk, module status, legal basis chips |
| `Table` | Compact data tables; sticky headers |
| `Tabs` | Lead detail, module detail tab panels |
| `Modal` / `Dialog` | Confirmation shells, audit expand |
| `Toast` | User-facing feedback (e.g. “Orchestrator not wired”) |
| `Tooltip` | Icon-only buttons and metadata hints |
| `Skeleton` | Loading placeholders for cards/tables |
| `EmptyState` | No results / module not run |
| `PageHeader` | Title + breadcrumb + actions |
| `StatusChip` | `ok | unknown | skipped | pending | not_run` |
| `PipelineStepper` | Funnel: raw → enriched → validated → crm_ready |
| `AuditLogPanel` | Collapsible timeline of `AuditEvent`; expands `raw_stderr_json` |
| `MockDataBanner` | Persistent non-dismissible mock-data notice |

---

## 5. Domain model / exact TypeScript types

> Source of truth: `lib/api/types.ts`.
> Additional namespaced sub-fields may be added later by the real backend without changing these keys, because modules write only under their namespace.

```ts
// lib/api/types.ts

export type PipelineStage =
  | "raw"
  | "enriched"
  | "validated"
  | "crm_ready";

export type RiskLevel = "low" | "medium" | "high" | "unknown";

export type ModuleStatus = "ok" | "unknown" | "skipped" | "pending" | "not_run";

/** Raw ingest fields — never overwritten by modules */
export interface RawLead {
  id: string; // UI/mock id; real backend may use UUID
  name?: string;
  email?: string;
  phone?: string; // prefer E.164
  company?: string;
  domain?: string;
  source_id?: string; // ad campaign / website id
  permission_ref?: string; // proof of business permission
  created_at: string; // RFC3339
  updated_at: string;
}

export interface EmailValidateResult {
  status: "ok" | "unknown" | "skipped";
  deliverable: "yes" | "no" | "unknown";
  syntax_valid?: boolean;
  has_mx_records?: boolean;
  is_disposable?: boolean;
  is_role_account?: boolean;
  is_free_provider?: boolean;
  checked_at?: string;
  source_tool?: string; // e.g. "AfterShip/email-verifier@v1.4.1"
  error?: string;
}

export interface DomainIntelResult {
  status: ModuleStatus;
  web_check?: { status: ModuleStatus; summary?: string };
  harvester?: { status: ModuleStatus; emails_found?: number };
  checked_at?: string;
  source_tool?: string;
  error?: string;
}

export interface PhoneValidateResult {
  status: ModuleStatus;
  e164?: string;
  region?: string;
  line_type?: string;
  carrier?: string;
  risk_flags?: string[];
  checked_at?: string;
  source_tool?: string;
  error?: string;
}

export interface SocialFootprintResult {
  status: ModuleStatus;
  matches?: Array<{
    platform: string;
    handle: string;
    confidence: number; // 0–1
    url?: string;
  }>;
  summary?: string;
  checked_at?: string;
  source_tool?: string;
  error?: string;
}

export interface LeadRecord extends RawLead {
  stage: PipelineStage;
  risk_level: RiskLevel;
  risk_score?: number; // 0–100 optional composite for UI only
  email_validate?: EmailValidateResult;
  domain_intel?: DomainIntelResult;
  phone_validate?: PhoneValidateResult;
  social_footprint?: SocialFootprintResult;
  audit_events: AuditEvent[];
}

/** Lead list projection. The detail endpoint returns the full `LeadRecord`. */
export type LeadSummary = Omit<LeadRecord, 'audit_events'>;

export interface AuditEvent {
  id: string;
  module:
    | "email-validate"
    | "domain-intel"
    | "phone-validate"
    | "social-footprint"
    | "extraction"
    | "company-enrich"
    | "pipeline";
  tool: string; // name@version
  checked_at: string; // RFC3339
  status: "ok" | "unknown" | "skipped";
  legal_basis: string; // "GDPR Art.6(1)(f) legitimate-interest"
  subject?: {
    email?: string;
    domain?: string;
    phone_redacted?: string;
    handle?: string;
  };
  raw_stderr_json?: string;
}

export type ModuleDevStatus =
  | "available"
  | "in_development"
  | "planned"
  | "not_configured";

export interface ModuleInfo {
  name:
    | "email-validate"
    | "domain-intel"
    | "phone-validate"
    | "social-footprint"
    | "extraction"
    | "company-enrich";
  display_name: string;
  category: "ingest" | "enrich" | "validate";
  dev_status: ModuleDevStatus;
  namespaced_key: string;
  backing_tools: string[];
  description: string;
  min_input_field: string;
  risk_level_note: string;
  last_run_at?: string;
  error_rate_24h?: number;
  config_schema?: Array<{
    key: string;
    label: string;
    type: "string" | "secret" | "number" | "boolean";
    required: boolean;
    placeholder?: string;
  }>;
}

/** Module detail response includes the full `ModuleInfo` plus optional documentation. */
export interface ModuleDetail extends ModuleInfo {
  docs?: string;
}

export interface PipelineRun {
  id: string;
  type: "single" | "batch";
  started_at: string;
  finished_at?: string;
  status: "running" | "completed" | "failed" | "partial";
  lead_ids: string[];
  modules_executed: string[];
  audit_event_ids: string[];
  legal_basis: string;
  permission_refs: string[];
}
```

### Type notes for implementation

- `permission_ref` is optional on `RawLead` / `LeadRecord`; the UI renders a warning chip when it is missing.
- `DomainIntelResult`, `PhoneValidateResult`, and `SocialFootprintResult` are thin UI projections for v1 mock data. Future real module output may include extra namespaced sub-fields; these will be appended under the same namespaced key without breaking the top-level contract.

---

## 6. Mock API contract (Next.js App Router route handlers)

All routes live under `ui/web-console/app/api/`. Base response envelope:

```ts
{ data: T, meta?: { page, page_size, total } }
```

Error envelope:

```ts
{ error: { code: string, message: string } }
```

| Method | Path | Query / Body | Response | Notes |
|--------|------|--------------|----------|-------|
| `GET` | `/api/leads` | `?stage=&risk=&module_status=&q=&page=&page_size=` | `{ data: LeadSummary[], meta }` | Full-text search over name/email/company/domain; `module_status` matches any namespaced module status (missing key = `not_run`) |
| `GET` | `/api/leads/[id]` | — | `{ data: LeadRecord }` | Includes `audit_events` |
| `GET` | `/api/modules` | — | `{ data: ModuleInfo[] }` | Seed statuses per PR3 |
| `GET` | `/api/modules/[name]` | — | `{ data: ModuleDetail }` | `docs` is static markdown for `email-validate`, placeholder for others |
| `GET` | `/api/audit` | `?module=&lead_id=&page=&page_size=` | `{ data: AuditEvent[], meta }` | Always includes `legal_basis` |
| `GET` | `/api/runs` | — | `{ data: PipelineRun[] }` | Timeline of mock runs |
| `GET` | `/api/compliance/summary` | — | `{ data: { hard_rules, risk_table, checklist, exclusions } }` | Structured JSON from `docs/compliance.md` |
| `POST` | `/api/pipelines/run` | `{ lead_ids[], modules[], permission_ref, legal_basis? }` | `501` or `{ data: { accepted: true, run_id } }` | Stub only; UI toasts “Orchestrator not wired” |

### Client fallback

`lib/api/client.ts` first tries `fetch('/api/...')` against the running Next.js server. If the route is unavailable (dev server not reachable), it imports `lib/mocks/seed.ts` and filters/sorts locally as a resilience fallback. `output: 'export'` is not enabled in PR1 and is not a production target in v1.

---

## 7. Wireframes (ASCII)

### App shell

```
+--------------------------------------------------+
| ≡ | OSINT Lead Console      [Sandbox]  🔍  👤   |
+---+----------------------------------------------+
|   |                                              |
| L |  Leads list                                  |
| E |  +------------+  +------------------------+  |
| A |  | filters    |  | table                  |  |
| D |  +------------+  +------------------------+  |
| S |                                              |
|   |  Mock data — backend not wired yet           |
+---+----------------------------------------------+
| footer: legal basis reminder  |  Compliance →   |
+--------------------------------------------------+
```

### Leads list (`/leads`)

```
+--------------------------------------------------+
| Leads                         [Schedule run ▼]   |
+--------------------------------------------------+
| Stage ▾ | Risk ▾ | Module ▾ | [🔍 search...]   |
+--------------------------------------------------+
| raw  | enriched | validated | crm_ready | total |
|  3   |    4     |    3      |     2     |  12   |
+--------------------------------------------------+
| □ | Name    | Email     | Stage   | Risk | Email | Phone | Domain | Social |
| □ | Acme…   | a@ac.me   | enriched| low  | ✓ ok  | —     | —      | —      |
| □ | …       |           |         |      |       |       |        |        |
+--------------------------------------------------+
```

### Lead detail (`/leads/[id]`)

```
+--------------------------------------------------+
| Jane Doe  | enriched  | low risk  | pid: cmp-42   |
| support@github.com  |  source: web-001           |
+--------------------------------------------------+
| Raw lead        |  Tabs: Email | Phone | Domain | Social |
| name, email…    |  +----------------------------+ |
| source_id       |  | deliverable: unknown       | |
| permission_ref  |  | syntax_valid: true         | |
|                 |  | has_mx_records: true       | |
| Enriched (stub) |  | is_role_account: true      | |
|                 |  +----------------------------+ |
+-----------------+----------------------------------+
| AuditLogPanel — expand raw_stderr_json             |
| • email-validate  ok  GDPR Art.6(1)(f) …           |
+--------------------------------------------------+
```

### Modules (`/modules`)

```
+--------------------------------------------------+
| Modules                                            |
+--------------------------------------------------+
| ┌----------------┐ ┌----------------┐ …            |
| | email-validate | | domain-intel   |              |
| | available      | | in_development |              |
| | Validate       | | Ingest         |              |
| └----------------┘ └----------------┘              |
+--------------------------------------------------+
```

### Module detail (`/modules/email-validate`)

```
+--------------------------------------------------+
| email-validate   [Validate]   available            |
+--------------------------------------------------+
| Overview | Configuration | Health | Documentation |
|                                                    |
| Description, tools, input field, last run, status |
| Config form (disabled placeholders)              |
| Health sparkline stub                            |
| Docs: I/O contract from README                   |
+--------------------------------------------------+
```

### Compliance (`/compliance`)

```
+--------------------------------------------------+
| Compliance                                         |
+--------------------------------------------------+
| Hard rules (5 alert cards)                         |
| Risk table (category / risk / notes)               |
| Pre-run checklist (interactive checkboxes)         |
| Exclusions callout: LinkedIn, reverse-image, breach  |
+--------------------------------------------------+
```

### Settings (`/settings`)

```
+--------------------------------------------------+
| Settings                                           |
+--------------------------------------------------+
| Environment  [Sandbox ▾]                             |
| Role         [sales ▾]  (admin | risk)             |
| CRM connector     [stub]                           |
| SSO / OIDC        [stub]                           |
| API keys vault    [stub]                           |
| Retention policy  [stub]                           |
+--------------------------------------------------+
```

---

## 8. PR sequence & file paths

### Branch / PR naming

| PR | Branch | Title |
|----|--------|-------|
| 0 | `docs/frontend-megaplan` | `docs: frontend megaplan and API contracts` |
| 1 | `feature/ui-web-console-scaffold` | `feat(ui): web-console scaffold and design system` |
| 2 | `feature/ui-leads-mock-api` | `feat(ui): leads list and detail with mock API` |
| 3 | `feature/ui-modules-dashboard` | `feat(ui): modules dashboard` |
| 4 | `feature/ui-runs-compliance-settings` | `feat(ui): runs, compliance, settings` |

### PR0 — Planning docs only

Files:
- `docs/frontend/megaplan.md`
- `docs/frontend/api-contracts.md`
- `docs/frontend/codemap.md`

No UI code. Human review of the plan before PR1 starts.

### PR1 — Scaffold + design system + app shell

Files (under `ui/web-console/`):
- `package.json`
- `tsconfig.json`
- `next.config.ts`
- `tailwind.config.ts`
- `postcss.config.mjs`
- `eslint.config.mjs` (or `.eslintrc.json`)
- `README.md`
- `app/layout.tsx`
- `app/page.tsx` (redirect to `/leads`)
- `app/leads/page.tsx`
- `app/modules/page.tsx`
- `app/runs/page.tsx`
- `app/compliance/page.tsx`
- `app/settings/page.tsx`
- `components/layout/Sidebar.tsx`
- `components/layout/TopBar.tsx`
- `components/layout/Footer.tsx`
- `components/ui/Button.tsx`
- `components/ui/IconButton.tsx`
- `components/ui/Input.tsx`
- `components/ui/Select.tsx`
- `components/ui/Textarea.tsx`
- `components/ui/Card.tsx`
- `components/ui/Badge.tsx`
- `components/ui/Table.tsx`
- `components/ui/Tabs.tsx`
- `components/ui/Dialog.tsx`
- `components/ui/Toast.tsx` (or toast provider)
- `components/ui/Tooltip.tsx`
- `components/ui/Skeleton.tsx`
- `components/ui/EmptyState.tsx`
- `components/ui/PageHeader.tsx`
- `components/ui/StatusChip.tsx`
- `components/ui/PipelineStepper.tsx`
- `components/ui/AuditLogPanel.tsx`
- `components/ui/MockDataBanner.tsx`
- `lib/theme/tokens.ts`
- `lib/utils/cn.ts` (Tailwind class merge utility)
- `public/` (favicon, maybe logo placeholder)

### PR2 — Types + mock API + leads

Files:
- `lib/api/types.ts`
- `lib/mocks/seed.ts`
- `lib/api/client.ts` (fetch-first with seed fallback)
- `lib/api/hooks.ts` (TanStack Query hooks: `useLeads`, `useLead`)
- `app/api/leads/route.ts`
- `app/api/leads/[id]/route.ts`
- `app/api/audit/route.ts`
- `app/api/runs/route.ts`
- `app/api/compliance/summary/route.ts`
- `app/api/pipelines/run/route.ts`
- `app/leads/page.tsx` (list + filters + funnel)
- `app/leads/[id]/page.tsx` (detail)
- `components/leads/LeadFilters.tsx`
- `components/leads/StageFunnel.tsx`
- `components/leads/LeadTable.tsx`
- `components/leads/LeadDetailTabs.tsx`
- `components/leads/EmailResultCard.tsx`
- `components/leads/PhoneResultCard.tsx`
- `components/leads/DomainResultCard.tsx`
- `components/leads/SocialResultCard.tsx`
- `components/leads/RawLeadCard.tsx`
- `components/leads/EnrichedCard.tsx`

### PR3 — Modules dashboard

Files:
- `app/api/modules/route.ts`
- `app/api/modules/[name]/route.ts`
- `app/modules/page.tsx`
- `app/modules/[name]/page.tsx`
- `content/docs/email-validate.md`
- `components/modules/ModuleGrid.tsx`
- `components/modules/ModuleDetailTabs.tsx`
- `components/modules/ModuleDocsPanel.tsx`
- `components/modules/ModuleConfigPanel.tsx`
- `components/modules/ModuleHealthPanel.tsx`

### PR4 — Runs, compliance, settings, polish

Files:
- `app/runs/page.tsx`
- `app/runs/[id]/page.tsx` (optional)
- `app/compliance/page.tsx`
- `app/settings/page.tsx`
- `components/runs/RunTimeline.tsx`
- `components/runs/RunDetail.tsx`
- `components/compliance/HardRules.tsx`
- `components/compliance/RiskTable.tsx`
- `components/compliance/PreRunChecklist.tsx`
- `components/compliance/ExclusionsCallout.tsx`
- `components/settings/EnvironmentSetting.tsx`
- `components/settings/RoleSelector.tsx`
- `components/settings/CrmConnectorStub.tsx`
- `components/settings/SsoOidStub.tsx`
- `components/settings/ApiKeysVaultStub.tsx`
- `lib/utils/risk.ts` (unit-tested helpers: risk label, score color)
- `lib/utils/stage.ts` (unit-tested helpers: stage label, step index)
- `__tests__/...` or `vitest`/`playwright` smoke test setup

---

## 9. Risks & open questions

| Risk / question | Impact | Mitigation / tracking |
|-----------------|--------|-----------------------|
| Real auth/SSO not designed | PR5+ | Add `RoleSelector` local-state stub now; no backend gate |
| Storage/retention policy undefined | PR5+ | Settings retention stub; no real data persistence in UI |
| Risk scoring algorithm undefined | PR2+ | Keep `risk_score` optional and UI-only; derive from flags |
| Static export vs server runtime | PR1+ | Route handlers + seed fallback covers both |
| TanStack Query v5 vs v4 | PR1+ | Pin `@tanstack/react-query@5.x` |
| Dependency license scanning | PR1+ | Only MIT-compatible libraries; review before each PR |
| Real orchestration API contract may drift | PR5+ | `api-contracts.md` is the freeze; versioned updates |
| Accessibility / keyboard nav | PR4+ | Built into component base; explicit a11y pass in PR4 |

---

## 10. Acceptance criteria per PR

### PR0

- [ ] `docs/frontend/megaplan.md` contains all 11 required sections.
- [ ] `docs/frontend/api-contracts.md` freezes TS types and endpoint shapes.
- [ ] `docs/frontend/codemap.md` maps planned files and conventions.
- [ ] No code or `modules/**` changes.

### PR1

- [ ] `npm install` and `npm run build` succeed with zero errors.
- [ ] `npm run typecheck` and `npm run lint` pass.
- [ ] App shell renders with the dark corporate palette (`#050816`, cyan accents).
- [ ] Sidebar navigation works across `/leads`, `/modules`, `/runs`, `/compliance`, `/settings`.
- [ ] `MockDataBanner` is visible and non-dismissible.
- [ ] All base UI components render in a style guide / placeholder page without runtime errors.
- [ ] `README.md` includes `npm i && npm run dev`, Node version, and architecture note.
- [ ] No touch of `modules/**`, Go files, evaluations.

### PR2

- [ ] `lib/api/types.ts` matches the exact types in this plan.
- [ ] Mock seed has ≥ 12 leads across all stages, ≥ 1 full `email_validate` matching the module README (`support@github.com` style), ≥ 1 `unknown` with error, ≥ 1 missing phone/domain, `source_id`/`permission_ref` on most leads.
- [ ] `/api/leads` and `/api/leads/[id]` return `{ data }` envelopes and consistent errors.
- [ ] TanStack Query hooks `useLeads` and `useLead` fetch and cache data.
- [ ] Leads list supports filters (stage, risk, module status, free-text) and shows stage funnel counts.
- [ ] Lead detail displays raw fields, module result tabs, and `AuditLogPanel` with `legal_basis` visible.
- [ ] Mock-data badge present.
- [ ] No `modules/**` changes.

### PR3

- [ ] `/modules` grid shows 6 modules with seed dev statuses.
- [ ] `/modules/email-validate` Documentation tab renders static markdown content.
- [ ] Configuration and Health tabs are disabled/placeholder.
- [ ] Module detail route handles unknown module names gracefully.
- [ ] No `modules/**` changes.

### PR4

- [ ] `/runs` timeline lists `PipelineRun`s; click shows related leads and audit subset.
- [ ] `/compliance` shows hard rules, risk table, interactive checklist, and explicit exclusions.
- [ ] `/settings` includes environment badge, role selector, and stub connectors.
- [ ] Keyboard navigation and focus rings pass manual a11y check.
- [ ] Responsive sidebar collapse works on mobile widths.
- [ ] `npm run build` still passes.

---

## 11. Compliance UI mapping to `docs/compliance.md` hard rules

| Hard rule (from `docs/compliance.md`) | UI behavior |
|---------------------------------------|-------------|
| **1. No non-consensual personal surveillance** | Social footprint tab shows only `match/no-match + confidence` (`matches[]` with `platform`, `handle`, `confidence`, `url`). No screenshots, no follower/location dumps, no reverse-image UI. Reverse-image / GHunt excluded entirely. |
| **2. Respect third-party ToS** | LinkedIn scraping is not listed as an available module; exclusions callout names it. Modules list shows only approved Stage 2 modules. |
| **3. Rate-limit / document breach-checking** | No breach/leak signals surfaced to sales views. Settings role selector provides `admin | risk` stub, but the UI never renders breach/leak data in PRs 1–4 because no such module exists. Pre-run checklist records legal basis and permission ref before stub run. |
| **4. Log legal basis and source permission reference** | Every `LeadRecord` shows `permission_ref` chip. Every `AuditEvent` shows `legal_basis` and `checked_at`. Every `PipelineRun` shows `legal_basis` and `permission_refs`. Footer reminds users of legal basis. |
| **5. Data retention** | Settings page includes a retention-policy stub. Runs view and compliance text note that retention windows are enforced by future backend; UI does not persist enriched lead data beyond the mock session. |

### Per-category risk table UI

Rendered from `/api/compliance/summary` as a read-only table matching `docs/compliance.md`:

| Category | Risk level | Notes |
|---|---|---|
| Website/domain intel | Low | Public DNS/WHOIS/tech-stack data about a business's own domain. |
| Email verification | Low | Syntax/MX/SMTP checks, no data sent to third parties about the person. |
| Email → account checks | Medium | Rate-limit, use selectively. (Not default pipeline in UI.) |
| Breach/leak checking | Medium-High | Internal risk signal only; hidden from sales views. |
| Phone OSINT | Low-Medium | Carrier/line-type lookups; scam-score lookups rate-limited. |
| Social footprint | Medium | Match/no-match + confidence only. |
| Reverse-image / deep discovery | High | Excluded from default pipeline. |
| LinkedIn scraping | High | Excluded from production. |
| Web scraping infra | Low | Risk depends on target ToS. |

---

## 12. Decisions from planning clarifications

- **Package manager:** `npm` + Next.js 15 (App Router, strict TypeScript, Tailwind).
- **Mock data transport:** Next.js App Router route handlers with a client-side fallback to `lib/mocks/seed.ts` for static-export compatibility.
- **Role stub:** Settings page contains a local-state `RoleSelector` (`sales | admin | risk`) that gates any future sensitive views; for PRs 1–4 it primarily controls whether advanced risk cards are visible and is persisted only in Zustand.

---

*Plan version: 1.0. Generated for human review before any UI code is written.*
