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
- Consume the live `services/control-plane` API at `NEXT_PUBLIC_API_BASE_URL` (default `http://localhost:8080`); no mock Next.js route handlers or `seed.ts` product path.
- Ship PR-by-PR, one concern per PR, with build/type/lint gates.

### Non-goals

- No LinkedIn scraping, reverse-image, GHunt-style tooling, or breach/leak signal surfaced to sales.
- No auth/SSO integration in v1 (only local role stubs).
- No real secrets storage or CRM connector wiring.
- No extraction/company-enrich wiring (deferred to a future PR).

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
- **API status banner:** shows live connectivity to `NEXT_PUBLIC_API_BASE_URL`.

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
| `Toast` | User-facing feedback (e.g. "Run started — view /runs/{run_id}") |
| `Tooltip` | Icon-only buttons and metadata hints |
| `Skeleton` | Loading placeholders for cards/tables |
| `EmptyState` | No results / module not run |
| `PageHeader` | Title + breadcrumb + actions |
| `StatusChip` | `ok | unknown | skipped | pending | not_run` |
| `PipelineStepper` | Funnel: raw → enriched → validated → crm_ready |
| `AuditLogPanel` | Collapsible timeline of `AuditEvent`; expands `raw_stderr_json` |
| `EnvironmentBanner` | Shows live API connectivity and sandbox/production badge |

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
  id: string; // UUID assigned by services/control-plane
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

// The v1 registry currently marks modules as "available" or "planned";
// "in_development" is reserved for future use.

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
- `DomainIntelResult` and `SocialFootprintResult` contain the real nested structures returned by the live modules (DNS/SSL/HTTP/WHOIS, theHarvester, handle results, platform signals). `PhoneValidateResult` remains a flat summary. All module results live under their namespaced key on `LeadRecord`.

---

## 6. Live API contract (`services/control-plane`)

The UI consumes the Go API running at `NEXT_PUBLIC_API_BASE_URL`. Base response envelope:

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
| `GET` | `/api/modules` | — | `{ data: ModuleInfo[] }` | Real statuses from `services/control-plane/internal/registry` |
| `GET` | `/api/modules/[name]` | — | `{ data: ModuleDetail }` | `docs` is Markdown from the registry |
| `GET` | `/api/audit` | `?module=&status=&page=&page_size=` | `{ data: AuditEvent[], meta }` | Always includes `legal_basis` |
| `GET` | `/api/runs` | — | `{ data: PipelineRun[] }` | Live run timeline |
| `GET` | `/api/runs/[id]` | — | `{ data: PipelineRun }` | Single run detail |
| `GET` | `/api/compliance/summary` | — | `{ data: { hard_rules, risk_table, checklist, exclusions } }` | Static governance content from `docs/compliance.md` |
| `POST` | `/api/leads/[id]/run` | `{ modules[], permission_ref?, legal_basis? }` | `{ data: LeadRecord }` | Runs modules on a single lead |
| `POST` | `/api/pipelines/run` | `{ lead_ids[], modules[], permission_ref?, legal_basis? }` | `202 { data: { accepted: true, run_id } }` | Batch run; poll `GET /api/runs/[id]` |

### API client

`lib/api/client.ts` fetches from `NEXT_PUBLIC_API_BASE_URL` (default `http://localhost:8080`). There is no local Next.js `/api/*` route handler fallback or `lib/mocks/seed.ts` product path. The UI requires the control-plane server to be running.

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
|   |  API status: connected to :8080              |
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
| | available      | | available      |              |
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

## 8. v1 status: completed vs backlog

### Completed (merged)

- UI scaffold, design tokens, app shell, sidebar, dark theme.
- Live API client/hooks wired to `services/control-plane` at `NEXT_PUBLIC_API_BASE_URL`.
- `/leads` list with filters, stage funnel, permission-ref warnings, multi-select bulk runs, and live pagination.
- `/leads/[id]` detail with raw fields, module result tabs, per-module run actions, structured Domain/Social panels, and `AuditLogPanel`.
- `/modules` grid grouped by availability and `/modules/[name]` detail.
- `/runs` and `/runs/[id]` timeline + detail.
- `/compliance` static governance view.
- Bulk-actions bar derives available modules from `/api/modules`.
- Control-plane CI workflow (`.github/workflows/control-plane.yml`) and timeout env hardening (`HTTP_READ_TIMEOUT`, `HTTP_WRITE_TIMEOUT`, per-module runner timeout policy).

### Backlog / deferred

- **crm_ready stage policy:** the stage machine advances to `validated` but has no explicit `crm_ready` transition yet.
- **Risk scoring from domain/social signals:** `risk_score` is not computed beyond the existing `low | medium | high | unknown` labels.
- **company-enrich wiring:** `company-enrich` is `planned` in the registry and not executable. `extraction` is now `available`.
- **Async long-run jobs:** long Maigret/theHarvester runs currently run inside the synchronous HTTP request. If HTTP write timeouts become a bottleneck, batch runs may move to an async worker.

### One-module-at-a-time rule for future modules

When wiring a new module in a future PR:
1. Add the module as a `go.mod` replace in `services/control-plane`, import it in `internal/runner/runner.go`, map its result and `AuditRecord`s, and add/adapt runner tests.
2. Update `internal/registry/registry.go` to `available` and document the module's behavior.
3. Update `ui/web-console/lib/api/types.ts` with the namespaced result type.
4. Add or extend the lead detail panel to render the new result.
5. Update `services/control-plane/README.md` and `ui/web-console/README.md` with env/runtime requirements.
6. Do not touch `modules/<name>/` source code unless the module contract itself needs changing.

---

## 9. Risks & open questions

| Risk / question | Impact | Mitigation / tracking |
|-----------------|--------|-----------------------|
| Real auth/SSO not designed | v2 | `RoleSelector` local-state stub exists; no backend gate |
| Storage/retention policy undefined | v2 | Settings retention stub; backend enforces retention |
| Risk scoring algorithm undefined | v2 | Keep `risk_score` optional and UI-only; derive from module flags in a future PR |
| Long Maigret/theHarvester runs block HTTP request | v1+ | Raise `HTTP_WRITE_TIMEOUT`; consider async batch worker if still limiting |
| TanStack Query v5 vs v4 | v1 | Pin `@tanstack/react-query@5.x` |
| Dependency license scanning | ongoing | Only MIT-compatible libraries; review before each PR |
| API contract drift between UI and control-plane | v1+ | `api-contracts.md` is the live freeze; versioned updates |
| Accessibility / keyboard nav | v1 | Built into component base; explicit a11y pass in future PR |

---

## 10. v1 acceptance criteria

- [ ] `npm install` and `npm run build` succeed with zero errors.
- [ ] `npm run typecheck` and `npm run lint` pass.
- [ ] App shell renders with the dark corporate palette (`#050816`, cyan accents).
- [ ] Sidebar navigation works across `/leads`, `/modules`, `/runs`, `/compliance`, `/settings`.
- [ ] Leads list fetches from `/api/leads`, supports filters, stage funnel, and bulk available-module actions.
- [ ] Lead detail fetches from `/api/leads/{id}`, displays raw fields, module result tabs with structured Domain/Social panels, and `AuditLogPanel` with `legal_basis` and `raw_stderr_json` visible.
- [ ] `/modules` groups modules by `available` / `in_development` / `planned` (the registry currently uses `available` and `planned`; `in_development` is reserved) and `/modules/{name}` renders docs from the registry.
- [ ] `/runs` lists live `PipelineRun`s and `/runs/{id}` shows run detail linked to leads.
- [ ] `/compliance` shows hard rules, risk table, checklist, and exclusions from `/api/compliance/summary`.
- [ ] `/settings` includes environment badge, role selector, and stub connectors.
- [ ] `POST /api/leads/{id}/run` and `POST /api/pipelines/run` call the control-plane API and return updated lead / accepted run.
- [ ] README and `api-contracts.md` describe the live `services/control-plane` base URL and `NEXT_PUBLIC_API_BASE_URL`.

---

## 11. Compliance UI mapping to `docs/compliance.md` hard rules

| Hard rule (from `docs/compliance.md`) | UI behavior |
|---------------------------------------|-------------|
| **1. No non-consensual personal surveillance** | Social footprint tab shows only `match/no-match + confidence` (`matches[]` with `platform`, `handle`, `confidence`, `url`). No screenshots, no follower/location dumps, no reverse-image UI. Reverse-image / GHunt excluded entirely. |
| **2. Respect third-party ToS** | LinkedIn scraping is not listed as an available module; exclusions callout names it. Modules list shows only approved Stage 2 modules. |
| **3. Rate-limit / document breach-checking** | No breach/leak signals surfaced to sales views. Settings role selector provides `admin | risk` stub, but the UI never renders breach/leak data in v1 because no such module exists. Pre-run checklist records legal basis and permission ref before a run. |
| **4. Log legal basis and source permission reference** | Every `LeadRecord` shows `permission_ref` chip. Every `AuditEvent` shows `legal_basis` and `checked_at`. Every `PipelineRun` shows `legal_basis` and `permission_refs`. Footer reminds users of legal basis. |
| **5. Data retention** | Settings page includes a retention-policy stub. The control-plane stores leads/runs/audit events; retention windows are enforced in a future backend pass. |

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
- **API transport:** The UI fetches the live `services/control-plane` API at `NEXT_PUBLIC_API_BASE_URL`. No Next.js `/api/*` route handlers, no `lib/mocks/seed.ts` product path.
- **Role stub:** Settings page contains a local-state `RoleSelector` (`sales | admin | risk`) that gates any future sensitive views; for v1 it primarily controls whether advanced risk cards are visible and is persisted only in Zustand.
- **Module wiring discipline:** future modules are integrated one at a time via `services/control-plane/internal/runner` and `internal/registry`, with `lib/api/types.ts` and the lead detail panels updated to match.

---

*Plan version: 1.0. Generated for human review before any UI code is written.*
