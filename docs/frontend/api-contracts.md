# Frontend API Contracts (v1 — Live)

> **Status:** live contract implemented by `services/control-plane`.
> The Next.js web console talks directly to the Go API at `NEXT_PUBLIC_API_BASE_URL` (default `http://localhost:8080`).
> There are no mock Next.js route handlers and no `lib/mocks/seed.ts` product path.
> **Scope:** these contracts describe `services/control-plane/internal/http/**` and the payloads consumed by `ui/web-console/lib/api/**`; `modules/**` supply the underlying libraries.

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
| `201` | `POST /api/leads` created |
| `202` | `POST /api/pipelines/run` accepted — poll `GET /api/runs/{id}` |
| `400` | Bad request / missing required query or body field |
| `404` | Resource not found (`lead`, `module`, `run`) |
| `500` | Unexpected server error |

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
  checked_at?: string;
  source_tools?: string[];
  web_check?: {
    status: ModuleStatus;
    resolvable?: boolean;
    dns?: {
      a?: string[];
      aaaa?: string[];
      cname?: string[];
      mx?: string[];
      ns?: string[];
      txt?: string[];
    };
    ssl?: {
      subject?: string;
      issuer?: string;
      valid?: boolean;
      not_before?: string;
      not_after?: string;
      days_until_expiry?: number;
      protocol?: string;
      sans?: string[];
    };
    http?: {
      status_code?: number;
      server?: string;
      headers?: Record<string, string>;
    };
    whois?: {
      registrar?: string;
      created_date?: string;
      domain_age_days?: number;
    };
    source_tool?: string;
    checked_at?: string;
  };
  harvester?: {
    status: ModuleStatus;
    emails?: string[];
    hosts?: unknown[];
    ips?: string[];
    sources?: string[];
    error?: string;
    source_tool?: string;
    checked_at?: string;
  };
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

export interface PlatformSignal {
  platform: string;
  status: "claimed" | "available" | "unknown";
  url?: string;
  http_status?: number;
}

export interface HandleResult {
  handle: string;
  origin: string;
  status: ModuleStatus;
  platforms: PlatformSignal[];
  claimed_count: number;
  checked_at: string;
  source_tool: string;
  error?: string;
}

export interface SocialFootprintResult {
  status: ModuleStatus;
  reason?: string;
  handles_checked?: string[];
  handles?: HandleResult[];
  active_signals?: number;
  confidence?: number;
  metadata?: Record<string, unknown>;
  rate_limit_note?: string;
  checked_at?: string;
  source_tool?: string;
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

### Module result types as namespaced keys

`LeadRecord` exposes each module result as a top-level namespaced key (`email_validate`, `phone_validate`, `domain_intel`, `social_footprint`). The backend returns these flattened onto the lead object; the internal `results` map is not part of the public JSON. Each module writes only under its own namespace and never overwrites raw ingested fields.

`DomainIntelResult` and `SocialFootprintResult` include the real nested sub-structures emitted by `modules/domain-intel` and `modules/social-footprint` (DNS/SSL/HTTP/WHOIS, theHarvester, handle results, platform signals, etc.). The UI renders the structured fields and keeps a collapsible raw JSON view for power users.

---

## 3. Endpoint definitions

Path parameters use Go 1.22 `http.ServeMux` brace syntax (`{id}`, `{name}`).

All lead responses are produced by `leadToJSON`: module result keys are flattened to the top level (e.g. `email_validate`, `domain_intel`) and the internal `Results` map is never exposed. `audit_events` are included only on detail endpoints.

### `POST /api/leads`

Create a lead. Body fields match `RawLead`.

**Request body**

```json
{
  "name": "Jane Doe",
  "email": "support@github.com",
  "phone": "+14155551212",
  "company": "GitHub",
  "domain": "github.com",
  "source_id": "cmp-001",
  "permission_ref": "perm-2026-001"
}
```

**Response `201`**

Returns the created `LeadRecord` (module keys flattened to the top level; `audit_events` not included).

```json
{
  "data": {
    "id": "lead-001",
    "stage": "raw",
    "risk_level": "unknown",
    "name": "Jane Doe",
    "email": "support@github.com",
    "company": "GitHub",
    "domain": "github.com",
    "source_id": "cmp-001",
    "permission_ref": "perm-2026-001",
    "created_at": "2026-07-13T13:00:00Z",
    "updated_at": "2026-07-13T13:00:00Z"
  }
}
```

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

### `GET /api/leads/{id}`

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

### `POST /api/leads/{id}/run`

Run one or more modules on a single lead. Returns the full `LeadRecord` with `audit_events` hydrated after the run.

**Request body**

```json
{
  "modules": ["email-validate", "domain-intel"],
  "permission_ref": "perm-2026-001",
  "legal_basis": "GDPR Art.6(1)(f) legitimate-interest"
}
```

`modules` is required. `permission_ref` and `legal_basis` are optional; they default to the lead's stored `permission_ref` and GDPR Art.6(1)(f) legitimate interest.

**Response `200`**

```json
{
  "data": {
    "id": "lead-001",
    "stage": "validated",
    "risk_level": "low",
    "email_validate": { "status": "ok", ... },
    "domain_intel": { "status": "ok", ... },
    "audit_events": [ ... ]
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

### `GET /api/modules/{name}`

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
| `status` | string | Filter by audit status (`ok`, `unknown`, `skipped`) |
| `page` | number | Default `1` |
| `page_size` | number | Default `25`, max `100` |

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

### `GET /api/runs/{id}`

Return a single `PipelineRun` by ID.

**Response `200`**

```json
{
  "data": {
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
}
```

### `GET /api/compliance/summary`

Return structured compliance content derived from `docs/compliance.md`.

**Known limitation:** this endpoint returns static governance content (`hard_rules`, `risk_table`, `checklist`, `exclusions`). It does not yet return numeric stats, counts, or per-run compliance scoring.

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

Batch run endpoint. Accepts a list of lead IDs and modules, creates a `PipelineRun`, executes the modules against each lead sequentially in the same request, and returns `202 Accepted` with the run ID. The run is updated to `completed` or `partial` before the response is sent, so the UI may immediately navigate to `/runs/{run_id}` or poll `GET /api/runs/{id}` as a future-proof pattern.

**Request body**

```ts
interface PipelineRunRequest {
  lead_ids: string[];
  modules: string[];
  permission_ref?: string; // falls back to each lead's stored permission_ref
  legal_basis?: string;    // defaults to "GDPR Art.6(1)(f) legitimate-interest"
}
```

**Response `202`**

```json
{
  "data": {
    "accepted": true,
    "run_id": "run-001"
  }
}
```

The UI toasts a link to `/runs/{run_id}` and may continue polling the run detail.

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

### No local mock fallback

The UI requires a running control-plane API. There is no `lib/mocks/seed.ts` product path and no Next.js `/api/*` route handler fallback. Set `NEXT_PUBLIC_API_BASE_URL` to point at the Go API (default `http://localhost:8080`).

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
- `permission_ref` is optional on `RawLead` / `LeadRecord`; the UI shows a warning chip when it is missing and the control-plane falls back to the stored lead `permission_ref` if the request omits one.
- `risk_level` is one of `low`, `medium`, `high`, or `unknown`.

---

## 6. Versioning

- **v1.0** — Live API contract implemented by `services/control-plane` and consumed by `ui/web-console`.
- Future breaking changes will be versioned through a path prefix (`/api/v2/...`) or endpoint negotiation in `lib/api/client.ts`.
