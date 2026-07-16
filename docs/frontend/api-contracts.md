# Frontend API Contracts (v1 — Mock)

> **Status:** mock-only contract for the web console.
> All routes are implemented as Next.js App Router route handlers in `ui/web-console/app/api/**`.
> A client-side fallback to `lib/mocks/seed.ts` is required so the app can also run under `output: 'export'`.
> **Scope:** these contracts live entirely in `ui/web-console/` and `docs/frontend/`; no `modules/**`, Go code, or real orchestrator is created or modified.

---

## 1. Conventions

### Base response envelope

Every successful response returns `data` and optional `meta`:

```ts
interface ApiResponse<T> {
  data: T;
  meta?: {
    page?: number;
    page_size?: number;
    total?: number;
  };
}
```

### Error response envelope

```ts
interface ApiErrorResponse {
  error: {
    code: string;
    message: string;
  };
}
```

HTTP status codes:

| Code | When |
|------|------|
| `200` | Success (GET/POST) |
| `400` | Bad request / missing required query or body field |
| `404` | Resource not found (`lead`, `module`, `run`) |
| `501` | `POST /api/pipelines/run` — orchestrator not wired yet |
| `500` | Unexpected mock handler error |

### Pagination

Paginated list endpoints accept:

- `page` (number, default `1`)
- `page_size` (number, default `25`, max `100`)

They return `meta.page`, `meta.page_size`, `meta.total`.

---

## 2. Domain types

These types are canonical for the frontend. The real backend is expected to emit the same JSON shapes; additional namespaced sub-fields may be added under existing module keys without changing the top-level keys.

```ts
export type PipelineStage =
  | "raw"
  | "enriched"
  | "validated"
  | "crm_ready";

export type RiskLevel = "low" | "medium" | "high" | "unknown";

export type ModuleStatus = "ok" | "unknown" | "skipped" | "pending" | "not_run";

export interface RawLead {
  id: string;
  name?: string;
  email?: string;
  phone?: string;
  company?: string;
  domain?: string;
  source_id?: string;
  permission_ref?: string;
  created_at: string;
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
  source_tool?: string;
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
    confidence: number;
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
  risk_score?: number;
  email_validate?: EmailValidateResult;
  domain_intel?: DomainIntelResult;
  phone_validate?: PhoneValidateResult;
  social_footprint?: SocialFootprintResult;
  audit_events: AuditEvent[];
}

/** Lead list projection: no audit_events; detail endpoint returns full LeadRecord. */
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
  tool: string;
  checked_at: string;
  status: "ok" | "unknown" | "skipped";
  legal_basis: string;
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

/** Module detail response includes the full ModuleInfo plus optional static docs. */
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

### Module result types as UI projections

`DomainIntelResult`, `PhoneValidateResult`, and `SocialFootprintResult` are intentionally thin UI projections for v1 mock data. The real backend may return additional namespaced sub-fields (e.g. `domain_intel.web_check.dns`, `phone_validate.local`, `social_footprint.handles[]`). Such additions are append-only and must live under the existing namespaced key; the frontend will display the richer fields when wiring real data in a later PR.

---

## 3. Endpoint definitions

### `GET /api/leads`

List leads with optional filters and pagination.

**Query parameters**

| Name | Type | Description |
|------|------|-------------|
| `stage` | `PipelineStage` | Filter by pipeline stage |
| `risk` | `RiskLevel` | Filter by risk level |
| `module_status` | `ModuleStatus` | Lead matches if any of `email_validate.status`, `phone_validate.status`, `domain_intel.status`, or `social_footprint.status` equals the value; a missing key is treated as `not_run` for filtering only |
| `q` | string | Free-text search over `name`, `email`, `company`, `domain` |
| `page` | number | Default `1` |
| `page_size` | number | Default `25`, max `100` |

Response type: `ApiResponse<LeadSummary[]>`. `audit_events` is not included in list responses.

**Response `200`**

```json
{
  "data": [
    {
      "id": "lead-001",
      "stage": "validated",
      "risk_level": "low",
      "name": "Jane Doe",
      "email": "support@github.com",
      "company": "GitHub",
      "source_id": "cmp-001",
      "permission_ref": "perm-2026-001",
      "updated_at": "2026-07-13T14:00:00Z",
      "email_validate": { "status": "ok", "deliverable": "unknown" }
    }
  ],
  "meta": { "page": 1, "page_size": 25, "total": 12 }
}
```

### `GET /api/leads/[id]`

Return a single `LeadRecord` with `audit_events` hydrated.

**Response `200`**

```json
{
  "data": {
    "id": "lead-001",
    "stage": "validated",
    "risk_level": "low",
    "name": "Jane Doe",
    "email": "support@github.com",
    "company": "GitHub",
    "domain": "github.com",
    "source_id": "cmp-001",
    "permission_ref": "perm-2026-001",
    "created_at": "2026-07-13T13:00:00Z",
    "updated_at": "2026-07-13T14:00:00Z",
    "email_validate": {
      "status": "ok",
      "deliverable": "unknown",
      "syntax_valid": true,
      "has_mx_records": true,
      "is_disposable": false,
      "is_role_account": true,
      "is_free_provider": false,
      "checked_at": "2026-07-13T13:45:46Z",
      "source_tool": "AfterShip/email-verifier@v1.4.1"
    },
    "audit_events": [
      {
        "id": "evt-001",
        "module": "email-validate",
        "tool": "AfterShip/email-verifier@v1.4.1",
        "checked_at": "2026-07-13T13:45:46Z",
        "status": "ok",
        "legal_basis": "GDPR Art.6(1)(f) legitimate-interest",
        "subject": { "email": "support@github.com" },
        "raw_stderr_json": "{\"tool\":\"AfterShip/email-verifier@v1.4.1\",...}"
      }
    ]
  }
}
```

### `GET /api/modules`

Return all modules with dev status and metadata.

**Response `200`**

```json
{
  "data": [
    {
      "name": "email-validate",
      "display_name": "Email Validate",
      "category": "validate",
      "dev_status": "available",
      "namespaced_key": "email_validate",
      "backing_tools": ["AfterShip/email-verifier@v1.4.1"],
      "description": "Syntax, MX, disposable, role, and free-provider checks for an email address.",
      "min_input_field": "email",
      "risk_level_note": "Low — no email sent, no third-party PII disclosure.",
      "last_run_at": "2026-07-13T14:00:00Z",
      "error_rate_24h": 0.0
    }
  ]
}
```

### `GET /api/modules/[name]`

Return a single `ModuleDetail` (`ModuleInfo` plus optional documentation).

**Response `200`**

```json
{
  "data": {
    "name": "email-validate",
    "display_name": "Email Validate",
    "category": "validate",
    "dev_status": "available",
    "namespaced_key": "email_validate",
    "backing_tools": ["AfterShip/email-verifier@v1.4.1"],
    "description": "...",
    "min_input_field": "email",
    "risk_level_note": "Low",
    "docs": "# Email Validate\n\nSyntax/MX/disposable/role/free checks..."
  }
}
```

`docs` is a Markdown string. For `email-validate` it contains the key I/O sections from the module README; for other modules it is a short placeholder.

Response type: `ApiResponse<ModuleDetail>`.

### `GET /api/audit`

Paginated `AuditEvent` list. Useful for global audit log views.

**Query parameters**

| Name | Type | Description |
|------|------|-------------|
| `module` | string | Filter by module name |
| `lead_id` | string | Filter by lead (mock only; real backend may resolve from `subject`) |
| `page` | number | Default `1` |
| `page_size` | number | Default `50` |

**Response `200`**

```json
{
  "data": [ /* AuditEvent[] */ ],
  "meta": { "page": 1, "page_size": 50, "total": 24 }
}
```

### `GET /api/runs`

Return `PipelineRun` timeline.

**Response `200`**

```json
{
  "data": [
    {
      "id": "run-001",
      "type": "batch",
      "started_at": "2026-07-13T10:00:00Z",
      "finished_at": "2026-07-13T10:05:00Z",
      "status": "completed",
      "lead_ids": ["lead-001", "lead-002"],
      "modules_executed": ["email-validate", "domain-intel"],
      "audit_event_ids": ["evt-001", "evt-002"],
      "legal_basis": "GDPR Art.6(1)(f) legitimate-interest",
      "permission_refs": ["perm-2026-001", "perm-2026-002"]
    }
  ]
}
```

### `GET /api/compliance/summary`

Return structured compliance content derived from `docs/compliance.md`.

**Response `200`**

```json
{
  "data": {
    "hard_rules": [
      { "id": 1, "title": "No non-consensual personal surveillance", "summary": "..." },
      { "id": 2, "title": "Respect third-party ToS", "summary": "..." },
      { "id": 3, "title": "Rate-limit and document breach-checking", "summary": "..." },
      { "id": 4, "title": "Log the legal basis and source permission reference", "summary": "..." },
      { "id": 5, "title": "Data retention", "summary": "..." }
    ],
    "risk_table": [
      { "category": "Email verification", "risk_level": "Low", "notes": "..." }
    ],
    "checklist": [
      { "id": "permission", "label": "Permission ref recorded for every lead" },
      { "id": "legal_basis", "label": "Legal basis confirmed before enrichment" },
      { "id": "rate_limit", "label": "Rate limits respected for third-party lookups" },
      { "id": "retention", "label": "Retention window defined for this run" },
      { "id": "review", "label": "Results reviewed before CRM export" }
    ],
    "exclusions": [
      "LinkedIn scraping",
      "Reverse-image / deep account discovery",
      "Bulk breach/leak signals in sales views"
    ]
  }
}
```

### `POST /api/pipelines/run`

Stub endpoint. Accepts a run request body and returns either a mock accepted job or `501`.

**Request body**

```ts
interface PipelineRunRequest {
  lead_ids: string[];
  modules: string[];
  permission_ref: string;
  legal_basis?: string; // defaults to "GDPR Art.6(1)(f) legitimate-interest"
}
```

**Response `501`** (default)

```json
{
  "error": {
    "code": "ORCHESTRATOR_NOT_WIRED",
    "message": "Pipeline orchestration is not implemented yet. Use manual module runs instead."
  }
}
```

**Response `202`** (optional alternate mock behavior)

```json
{
  "data": {
    "accepted": true,
    "run_id": "run-mock-001",
    "status": "pending"
  }
}
```

The UI must always toast: `Orchestrator not wired`.

---

## 4. Client implementation notes

### `lib/api/client.ts`

All data fetching goes through a thin client:

```ts
export async function fetchLeads(params?: LeadSearchParams): Promise<ApiResponse<LeadSummary[]>>;
export async function fetchLead(id: string): Promise<ApiResponse<LeadRecord>>;
export async function fetchModules(): Promise<ApiResponse<ModuleInfo[]>>;
export async function fetchModule(name: string): Promise<ApiResponse<ModuleDetail>>;
export async function fetchAudit(params?: AuditSearchParams): Promise<ApiResponse<AuditEvent[]>>;
export async function fetchRuns(): Promise<ApiResponse<PipelineRun[]>>;
export async function fetchComplianceSummary(): Promise<ApiResponse<ComplianceSummary>>;
export async function runPipeline(body: PipelineRunRequest): Promise<ApiResponse<{ accepted: true; run_id: string }>>;
```

### Fallback to static seed

The default mode is a Node server (`next dev` / `next start`) so `/api/*` routes work. If `fetch` fails because the dev server is unreachable, the client imports `lib/mocks/seed.ts` and applies the same filtering/sorting logic locally as a resilience fallback. `output: 'export'` is not enabled in PR1 and is not a production target in v1.

### TanStack Query hooks

```ts
export function useLeads(params?: LeadSearchParams): UseQueryResult<ApiResponse<LeadSummary[]>, Error>;
export function useLead(id: string): UseQueryResult<ApiResponse<LeadRecord>, Error>;
export function useModules(): UseQueryResult<ApiResponse<ModuleInfo[]>, Error>;
export function useModule(name: string): UseQueryResult<ApiResponse<ModuleDetail>, Error>;
export function useAudit(params?: AuditSearchParams): UseQueryResult<ApiResponse<AuditEvent[]>, Error>;
export function useRuns(): UseQueryResult<ApiResponse<PipelineRun[]>, Error>;
export function useComplianceSummary(): UseQueryResult<ApiResponse<ComplianceSummary>, Error>;
```

### Type safety

- TypeScript `strict` mode enabled.
- No `any` without an explanatory comment.
- All API shapes are frozen in this document; additions must be append-only and preserve the `{ data, meta? }` / `{ error }` envelopes.

---

## 5. Compliance & security notes

- No PII is logged to the browser console in production builds.
- Phone numbers in audit subjects are always returned as `phone_redacted` (e.g. `+14*******86`).
- Social footprint audit subjects use `handle` only; raw email/name never appears in `social-footprint` audit lines.
- Every `AuditEvent` and `PipelineRun` includes `legal_basis`.
- `permission_ref` is optional on `RawLead` / `LeadRecord`; the UI shows a warning chip when it is missing.
- The UI never sends mock data to a real backend; all network calls are local to the Next.js dev server.

---

## 6. Versioning

- **v1.0** — Mock API for frontend PRs 1–4.
- Future real backend wiring will be versioned as `v2` or through endpoint negotiation in `lib/api/client.ts`.
