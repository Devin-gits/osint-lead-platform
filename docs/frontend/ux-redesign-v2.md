# Enterprise UX Redesign v2 — Planning Brief

> **Scope:** planning and design system only. No production code changes in this PR.
> **Affected path:** `docs/frontend/ux-redesign-v2.md`.
> **Source of truth:** `ui/web-console/**`, `docs/frontend/api-contracts.md`, `docs/status/platform-v1.md`, `docs/compliance.md`, `services/control-plane/internal/models/models.go`, `services/control-plane/internal/http/handlers.go`, `services/control-plane/internal/http/server.go`.

---

## 1. Problem statement

The current `ui/web-console` is functionally wired to the live control-plane API, but the experience does not read as an enterprise lead-operations console. The Leads screenshot and the current `AppShell`/`Sidebar`/`TopBar` layout expose four categories of problems.

### 1.1 Visual problems

- **Generic “dark cyber dashboard” aesthetic.** The current palette (`#050816` background + bright cyan `#2dd4ff` primary) and the oversized sidebar give the product a hobby-project feel rather than a trustworthy operations tool.
- **Poor use of desktop width.** The sidebar is fixed at `w-60` on desktop (`Sidebar.tsx`), leaving a large empty column when the user is just navigating. Content is constrained to `max-w-7xl` with generous page padding, so high-density tables and grids waste the available real estate.
- **Weak visual hierarchy.** Filters, stage funnel chips, the create button, and the leads table all compete at the same visual weight. There is no clear “what do I do next?” signal.
- **No systematic state styling.** Loading uses ad-hoc skeleton blocks, empty states are just icon + text, and offline/API-health problems are shown only as a small badge in the top bar.
- **Card overuse.** Every subsection is wrapped in `Card`, which flattens the page into a stack of equal boxes with no elevation or density distinction.

### 1.2 UX problems

- **Empty state does not onboard.** On first visit, `/leads` shows filters, stage counters, and a generic “No leads yet — Create a lead” empty state. It does not explain *why* the operator should create a lead, what a permission reference is, or which modules are available.
- **Create Lead is a single long dialog.** All fields (name, email, phone, company, domain, permission_ref) appear with equal weight (`app/leads/page.tsx`). `permission_ref` is the compliance gate, yet it looks like any other optional field.
- **Module actions are hidden.** To run a module on a lead, the operator must open the lead detail, then notice the small “Run module” card on the right. Bulk actions require selecting rows, which is discoverable but not guided.
- **Statuses are not explained.** `unknown` and `skipped` are not failures, but the UI renders both with warning/danger styling, risking misinterpretation.
- **No run feedback.** `POST /api/pipelines/run` returns `202 { accepted, run_id }` and `RunBatch` is synchronous today (`services/control-plane/internal/runner/runner.go`). The UI navigates to `/runs/{run_id}`, but the Runs detail page shows only final status — there is no progress or per-lead result timeline.

### 1.3 Information architecture problems

- **No home / command center.** The product redirects `/` to `/leads` (`app/page.tsx`). There is no place to see overall queue health, latest runs, or compliance readiness at a glance.
- **Audit and Runs are under-utilized.** The global `/api/audit` endpoint exists, but there is no `/audit` screen. Runs are a list/table with no per-lead outcome summary.
- **Modules and Compliance feel isolated.** They are reference pages, not connected to lead workflows.
- **Settings contains only stubs.** API keys, CRM, SSO, and retention are placeholders with no connection to real configuration or API reachability.

### 1.4 Observability problems

- **Audit trail is lead-local.** `AuditLogPanel` is only rendered inside the lead detail. A compliance reviewer who wants to review all activity for a module or time window has no screen.
- **Run detail lacks a timeline.** `GET /api/runs/{id}` returns `PipelineRun` with `audit_event_ids`, but the UI shows only a summary grid and a list of lead IDs (`app/runs/[id]/page.tsx`). It does not surface per-module completion order or per-lead status.
- **No API reachability surface area.** `TopBar` shows a small “reachable / offline” badge, but it does not explain *what* is failing or provide a retry action.
- **No offline/unknown state system.** If `GET /api/leads` fails, the page shows an inline error but no persistent “retry” or “working offline” affordance.

---

## 2. Product principles

1. **Trustworthy lead-operations console, not a generic cyber dashboard.**
   The interface should feel like a compliance-aware operations tool: clean, dense, legible, and predictable. Decorative gradients and “hacker” tropes are out. Clear labels, consistent spacing, and readable tables are in.

2. **Decision-first module results.**
   A lead row or detail should immediately answer: “Is this lead safe to move forward?” Module results are presented as decision summaries first; raw JSON and implementation details are available one click away.

3. **Compliance evidence is visible but progressively disclosed.**
   `permission_ref`, `legal_basis`, and audit events are surfaced at every decision point. The full audit payload (`raw_stderr_json`) is available on demand, not dumped by default.

4. **Never fabricate real-time status or logs.**
   The backend does not stream logs. The frontend must not show fake progress bars, simulated step-by-step logs, or pretend async work is happening. Status is derived only from `PipelineRun.status`, `AuditEvent` records, and final module result keys.

5. **Accessibility and keyboard navigation are product requirements.**
   The console must be operable without a mouse: sidebar toggle, command palette-style navigation, table row focus, drawer traps, and visible focus rings. Motion must respect `prefers-reduced-motion`.

---

## 3. Personas and jobs

### 3.1 Ops analyst

- **Primary job:** Ingest leads, run available modules, and triage the queue.
- **Required evidence:** Lead identity, stage, risk level, module status chips, and which modules are wired/available.
- **High-risk mistakes to prevent:**
  - Bulk-running modules on leads without a `permission_ref`.
  - Treating `unknown` status as a hard failure.
  - Exporting or sharing `raw_stderr_json` without review.
  - Running long Maigret/theHarvester jobs without raising `HTTP_WRITE_TIMEOUT`.

### 3.2 Sales operations reviewer

- **Primary job:** Decide which leads are ready for the CRM hand-off.
- **Required evidence:** Validation summary (email/phone/domain/social), risk level, permission reference, and the final stage.
- **High-risk mistakes to prevent:**
  - Marking a lead CRM-ready when `permission_ref` is missing.
  - Ignoring `skipped` modules that were skipped because of missing input, not success.
  - Treating `risk_score` as a defined composite score when today it is only stored if present.

### 3.3 Compliance reviewer

- **Primary job:** Confirm that every enrichment run has a legal basis, permission reference, and bounded tool scope.
- **Required evidence:** `AuditEvent` records (`tool`, `checked_at`, `status`, `legal_basis`, `subject`, `raw_stderr_json`), run records, and module docs.
- **High-risk mistakes to prevent:**
  - Missing runs that used excluded tools.
  - Not noticing that `social-footprint` only derives public handles and redacts raw identifiers.
  - Confusing synchronous `RunBatch` with future async jobs and expecting real-time logs.

---

## 4. Revised information architecture

The sidebar is reorganized into three labeled groups. The new top-level entry is **Command Center**, which replaces the silent redirect to `/leads`.

### 4.1 Sidebar groups

```
Workspace
  Command Center        /command-center
  Leads                 /leads
  Runs                  /runs
  Audit Log             /audit

Operations
  Modules               /modules
  Compliance            /compliance

Administration
  Settings              /settings
```

### 4.2 Page responsibilities

| Page | Responsibility |
|------|----------------|
| `/command-center` | Queue health, recent runs, compliance readiness, quick-create lead, one-click actions. |
| `/leads` | Operational queue: filter, sort, bulk run, inspect. |
| `/leads/create` | Two-step create flow: lead + permission, then module selection. |
| `/leads/{id}` | Lead detail with Overview, Checks, Activity & Audit, Raw Record tabs. |
| `/runs` | Pipeline run history with status, lead count, modules, and error summary. |
| `/runs/{id}` | Run detail with progress timeline, per-lead outcomes, and linked audit events. |
| `/audit` | Global audit explorer: filter by module, status, lead, time range. |
| `/modules` | Module catalog grouped by availability; link to docs. |
| `/modules/{name}` | Module docs, input schema, risk note, backing tools, config. |
| `/compliance` | Hard rules, risk table, pre-run checklist, exclusions. |
| `/settings` | API base URL / health, role selector, connector stubs, retention policy stub. |

### 4.3 URL changes from current UI

| Current | v2 |
|---------|-----|
| `/` -> `/leads` | `/` -> `/command-center` |
| `/leads` (dialog create) | `/leads` + `/leads/create` |
| no `/audit` | new `/audit` using `GET /api/audit` |

---

## 5. Page-by-page wireframes and interaction notes

Wireframes use a 12-column desktop grid reference. `|` separates panes; `[ ]` indicates interactive elements. All wireframes assume the new **ResponsiveSidebar**.

### 5.1 Command Center (`/command-center`)

> **Sequencing note:** The wireframes below show the fully data-backed Command Center envisioned once Leads, Runs, and Audit Explorer are implemented and their API data requirements are verified. **PR1 implements only a static workspace-home foundation**: product explanation, workflow explainer, a “Create lead” link, navigation shortcuts, and the API reachability indicator. Live operational metrics, trends, recent activity, lead/run-derived actions, compliance calculations, and module-health claims are explicitly excluded from PR1 and deferred to a later, separately scoped frontend PR.

Desktop (sidebar expanded):

```
+-----------------------------------------------------------------------------------+
| ≡ | OSINT Lead Console        | API reachable | Role: Ops analyst | Settings  | ? |
+-----------------------------------------------------------------------------------+
| W |                                                                                 |
| o |  Command Center                                              [+ Create lead]  |
| r |                                                                                 |
| k |  ┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐   |
|   |  │ Leads ready         │  │ Last run            │  │ Compliance          │   |
|   |  │ 24                  │  │ #run-8a2f completed │  │ Guidance available        │   |
|   |  │ +3 today            │  │ 5 leads, 2 modules  │  │ Review checklist    │   |
|   |  │ [View queue ->]      │  │ [View run ->]        │  │ [Open audit ->]      │   |
|   |  └─────────────────────┘  └─────────────────────┘  └─────────────────────┘   |
|   |                                                                                 |
|   |  Quick actions                                                                   |
|   |  [Create lead]  [Run email-validate on 3 raw]  [Review skipped social]         |
|   |                                                                                 |
|   |  Recent activity                                                                 |
|   |  ┌─────────────────────────────────────────────────────────────────────────┐   |
|   |  │ Time        │ Lead / Run        │ Action              │ Outcome         │   |
|   |  │ 09:41       │ lead-7c1          │ email-validate      │ ok, low risk    │   |
|   |  │ 09:38       │ run-8a2f          │ batch domain-intel  │ completed       │   |
|   |  │ 09:20       │ lead-9d4          │ social-footprint    │ skipped         │   |
|   |  └─────────────────────────────────────────────────────────────────────────┘   |
|   |                                                                                 |
+-----------------------------------------------------------------------------------+
```

Mobile:

```
+--------------------------------+
| ≡ OSINT Lead Console     [+    |
+--------------------------------+
| Command Center           [+    |
|                                |
| Leads ready              24    |
| +3 today                       |
| [View queue ->]                 |
|                                |
| Last run                       |
| #run-8a2f completed            |
| [View run ->]                   |
|                                |
| Compliance                     |
| Guidance available                   |
| [Open audit ->]                 |
|                                |
| Quick actions                  |
| [Create lead]                  |
| [Run email-validate]           |
|                                |
| Recent activity                |
| 09:41 lead-7c1 email ok        |
| 09:38 run-8a2f completed         |
| ...                            |
+--------------------------------+
```

**Interaction notes:**
- The “Recent activity” table is a client-side composition of the latest `GET /api/leads` and `GET /api/runs` results; it does not require a new endpoint.
- Quick-action buttons are context-aware: e.g., “Run email-validate on 3 raw” is disabled if no raw leads exist.
- “Review compliance checklist” links to `/compliance`.

### 5.2 Leads — first-time empty state (`/leads`, no leads)

Desktop:

```
+-----------------------------------------------------------------------------------+
| ≡ | OSINT Lead Console                                              | Settings  |
+-----------------------------------------------------------------------------------+
| W |                                                                                 |
| o |  Leads                                                                          |
| r |                                                                                 |
| k |  ┌─────────────────────────────────────────────────────────────────────────┐   |
|   |  │                                                                         │   |
|   |  │  [Icon: inbox]                                                          │   |
|   |  │  Your lead queue is empty                                               │   |
|   |  │                                                                         │   |
|   |  │  Start by creating a lead. You will need a permission reference         │   |
|   |  │  (e.g., a campaign or privacy-policy ID) before running any module.     │   |
|   |  │                                                                         │   |
|   |  │  [+ Create lead]   [View modules]                                       │   |
|   |  │                                                                         │   |
|   |  │  Tip: email-validate and phone-validate are available now.               │   |
|   |  │       domain-intel and social-footprint may need extra timeout config. │   |
|   |  │                                                                         │   |
|   |  └─────────────────────────────────────────────────────────────────────────┘   |
|   |                                                                                 |
+-----------------------------------------------------------------------------------+
```

**Interaction notes:**
- Filters, stage funnel, and bulk-action bar are **not** rendered when the queue is empty.
- The empty state includes a primary CTA, a secondary link to modules, and a short compliance tip.
- The illustration area uses a neutral icon, not a mascot.

### 5.3 Leads — populated operational queue (`/leads`)

Desktop:

```
+-----------------------------------------------------------------------------------+
| ≡ | OSINT Lead Console                                              | Settings  |
+-----------------------------------------------------------------------------------+
| W |                                                                                 |
| o |  Leads                                           [+ Create lead] [Run batch ▼]  |
| r |                                                                                 |
| k |  ┌─────────────────────────────────────────────────────────────────────────┐   |
|   |  │ Stage: [All ▼]  Risk: [All ▼]  Status: [All ▼]  [Search...      ] [search] │   |
|   |  └─────────────────────────────────────────────────────────────────────────┘   |
|   |                                                                                 |
|   |  raw 3 │ enriched 8 │ validated 12 │ crm_ready 0 │ unknown risk 1              |
|   |  ────────────────────────────────────────────────────────────────────────       |
|   |                                                                                 |
|   |  3 selected  [Clear]  [Run email-validate] [Run domain-intel]                  |
|   |                                                                                 |
|   |  ┌─────────────────────────────────────────────────────────────────────────┐   |
|   |  │ □ │ Contact          │ Company │ Readiness │ Risk │ Stage    │ Perm │    │   |
|   |  ├───┼──────────────────┼─────────┼───────────┼──────┼──────────┼──────┼────┤   │
|   |  │ ■ │ Jane Doe         │ GitHub  │ ██░░░░    │ low  │ validated│  OK   │ ->  │   │
|   |  │   │ support@github.c │         │ email ok  │      │          │      │    │   │
|   |  ├───┼──────────────────┼─────────┼───────────┼──────┼──────────┼──────┼────┤   │
|   |  │ ■ │ Acme, John       │ Acme    │ ████░░    │ unk  │ enriched │  MISSING   │ ->  │   │
|   |  │   │ john@acme.com    │         │ domain ok │      │          │      │    │   │
|   |  ├───┼──────────────────┼───────────┼─────────┼──────┼──────────┼──────┼────┤   │
|   |  │ □ │ ...              │ ...     │ ...       │ ...  │ ...      │ ...  │ ->  │   │
|   |  └─────────────────────────────────────────────────────────────────────────┘   │
|   |                                                                                 |
|   |  Showing 1-25 of 24                                                              |
|   |                                                                                 |
+-----------------------------------------------------------------------------------+
```

**Interaction notes:**
- The “Readiness” column uses `LeadReadinessCell`: a compact stacked bar of wired modules (email/phone/domain/social) colored by status.
- “Risk” and “Stage” are sortable via the table headers.
- Missing `permission_ref` is shown as a warning badge in the Perm column, not hidden.
- Row click navigates to `/leads/{id}`; checkbox click toggles selection without navigation.
- “Run batch” primary button opens a small dropdown of available modules when rows are selected; if nothing is selected, it is disabled with a tooltip.

### 5.4 Create Lead (`/leads/create`) — two steps

**Step 1: Lead & permission**

```
+-----------------------------------------------------------------------------------+
| ≡ | OSINT Lead Console                                              | Settings  |
+-----------------------------------------------------------------------------------+
| W |                                                                                 |
| o |  Create lead — Step 1 of 2                                      [Save draft]  |
| r |                                                                                 |
| k |  Lead information                                                                 |
|   |  Name            [                                ]                             |
|   |  Email *         [                                ]                             |
|   |  Phone           [                                ]                             |
|   |  Company         [                                ]                             |
|   |  Domain          [                                ]                             |
|   |                                                                                 |
|   |  Source ID       [                                ]                             |
|   |                                                                                 |
|   |  Permission reference *                                                           |
|   |  [                                ]                                               |
|   |  Required. This links every audit event to the legal basis for this lead.       |
|   |  Examples: cmp-2026-q1, privacy-policy-v3, consent-12345                          |
|   |                                                                                 |
|   |  Legal basis         [GDPR Art.6(1)(f) legitimate-interest ▼]                   |
|   |                                                                                 |
|   |                            [Cancel]  [Continue ->]                               |
|   |                                                                                 |
+-----------------------------------------------------------------------------------+
```

**Step 2: Select modules**

```
+-----------------------------------------------------------------------------------+
| ≡ | OSINT Lead Console                                              | Settings  |
+-----------------------------------------------------------------------------------+
| W |                                                                                 |
| o |  Create lead — Step 2 of 2                                      [Back] [Create] │
| r |                                                                                 |
| k |  Available modules                                                                |
|   |  ┌─────────────────────────────────────────────────────────────────────────┐   |
|   |  │ [x] email-validate   Syntax, MX, disposable, role checks — fast          │   |
|   |  │ [ ] phone-validate   Carrier / line type — fast                           │   |
|   |  │ [ ] domain-intel     DNS / TLS / HTTP / WHOIS + optional theHarvester    │   |
|   |  │                    May take longer; configure HTTP_WRITE_TIMEOUT.              │   |
|   |  │ [ ] social-footprint Maigret handle lookup — rate-limited, no ETA           │   │
|   |  └─────────────────────────────────────────────────────────────────────────┘   │
|   |                                                                                 |
|   |  Durations vary. No ETA is available from the current API.                                                 |
|   |                                                                                 |
+-----------------------------------------------------------------------------------+
```

**Interaction notes:**
- Step 2 module list is driven by `GET /api/modules` filtered to `available`.
- Warnings are static guidance based on `docs/status/platform-v1.md` timeout recommendations; they are not fake estimates.
- The Step 1 form requires a non-empty `permission_ref` before the user can proceed to module selection or submit a lead. This is a frontend guard; authoritative enforcement is a separately scoped control-plane concern.
- On submit, the page calls `POST /api/leads` then `POST /api/leads/{id}/run` for the selected modules, then redirects to `/leads/{id}?tab=checks`.
- If only Step 1 is submitted (no modules), the user lands on `/leads/{id}?tab=overview`.

### 5.5 Lead Detail (`/leads/{id}`)

**Overview tab:**

```
+-----------------------------------------------------------------------------------+
| ≡ | OSINT Lead Console                                              | Settings  |
+-----------------------------------------------------------------------------------+
| W |                                                                                 |
| o |  ← Back to leads  Jane Doe · GitHub                               [Actions ▼]  (Export evidence — not available in v1)  │
| r |                                                                                 |
| k |  ┌────────────────────────────────────────┐  ┌────────────────────────────┐  │
|   |  │ Stage: validated                       │  │ Run module                 │  │
|   |  │ Risk: low                              │  │ [email-validate  ->]        │  │
|   |  │ Permission: cmp-2026-q1  OK             │  │ [phone-validate  ->]        │  │
|   |  │ Updated: 18 Jul 2026, 09:41            │  │ [domain-intel    ->] (slow) │  │
|   |  │                                        │  │ [social-footprint->] (slow) │  │
|   |  │ Email  support@github.com        ok    │  │                            │  │
|   |  │ Phone  —                         —     │  │                            │  │
|   |  │ Domain github.com                ok    │  │                            │  │
|   |  │ Social 2 handles checked         1 ok  │  │                            │  │
|   |  └────────────────────────────────────────┘  └────────────────────────────┘  │
|   |                                                                                 |
|   |  [Overview] [Checks] [Activity & Audit] [Raw record]                            │
|   |                                                                                 |
+-----------------------------------------------------------------------------------+
```

**Checks tab:**

```
+-----------------------------------------------------------------------------------+
| ... tabs ...                                                                      │
|                                                                                   │
|  Email validation                                                                 │
|  Status: ok │ Deliverable: unknown │ Syntax: yes │ MX: yes │ Disposable: no      │
|  Source: AfterShip/email-verifier@v1.4.1                                          │
|  [View raw JSON]                                                                  │
|                                                                                   │
|  Domain intel                                                                     │
|  Status: ok │ Resolvable: yes │ SSL: valid │ Harvester: not_run                  │
|  [View raw JSON]                                                                  │
|                                                                                   │
|  +-------------------+                                                            │
|  | ModuleRunDrawer   |                                                            │
|  | Running domain... |  (only while request in flight)                            │
|  +-------------------+                                                            │
+-----------------------------------------------------------------------------------+
```

**Activity & Audit tab:**

```
+-----------------------------------------------------------------------------------+
| ... tabs ...                                                                      │
|                                                                                   │
|  Timeline                                            [Expand all] [Export evidence — not available in v1]  │
|                                                                                   │
|  09:41  email-validate  ok   tool: AfterShip/email-verifier  [details ->]        │
|  09:40  lead-created    —    permission_ref: cmp-2026-q1                        │
|                                                                                   │
|  Clicking “details ->” opens AuditEventDetailDrawer with:                         │
|  - module, tool, status, legal_basis                                              │
|  - subject (email/domain/phone_redacted/handle)                                │
|  - `raw_stderr_json`, collapsed by default, labeled “Technical evidence — may contain sensitive operational data"                                                 │
+-----------------------------------------------------------------------------------+
```

**Raw record tab:**

```
+-----------------------------------------------------------------------------------+
| ... tabs ...                                                                      │
|                                                                                   │
|  Raw lead record (read-only)                                                     │
|  ┌─────────────────────────────────────────────────────────────────────────┐     │
|  │ { id, name, email, ... }                                                │     │
|  │ (formatted JSON with copy button)                                       │     │
|  └─────────────────────────────────────────────────────────────────────────┘     │
+-----------------------------------------------------------------------------------+
```

**Interaction notes:**
- Tab state is persisted in the URL (`?tab=checks`).
- The “Run module” list is the `ModuleRunDrawer` trigger set. Clicking a module opens a confirmation drawer showing the selected module, legal basis, and a duration hint (e.g. “Fast”, “May take longer”, or “Rate-limited / potentially slow; no ETA available”).
- There is no export action in v2. Evidence export is not available until authentication, authorization, export audit logging, and retention/deletion enforcement are in place.
- `raw_stderr_json` is rendered inside `AuditEventDetailDrawer`, not inline.

### 5.6 Runs (`/runs`)

```
+-----------------------------------------------------------------------------------+
| ≡ | OSINT Lead Console                                              | Settings  |
+-----------------------------------------------------------------------------------+
| W |                                                                                 |
| o |  Runs                                                                           │
| r |                                                                                 │
| k |  ┌─────────────────────────────────────────────────────────────────────────┐   │
|   |  │ Run ID       │ Status    │ Started         │ Leads │ Modules │ Error   │   │
|   |  ├──────────────┼───────────┼─────────────────┼───────┼─────────┼─────────┤   │
|   |  │ run-8a2f     │ completed │ 18 Jul 09:30    │ 5     │ email…  │ —       │   │
|   |  │ run-3c91     │ partial   │ 18 Jul 09:15    │ 12    │ domain… │ 1 lead… │   │
|   |  │ run-1e04     │ completed │ 17 Jul 18:22    │ 1     │ social… │ —       │   │
|   |  └─────────────────────────────────────────────────────────────────────────┘   │
|   |                                                                                 |
+-----------------------------------------------------------------------------------+
```

**Interaction notes:**
- The list is `GET /api/runs`.
- Status colors: `completed` success, `partial` warning, `failed` danger, `running` primary (rare because `RunBatch` is synchronous, but possible if a future async worker is added).
- Row click navigates to `/runs/{id}`.

### 5.7 Run Detail (`/runs/{id}`)

```
+-----------------------------------------------------------------------------------+
| ≡ | OSINT Lead Console                                              | Settings  |
+-----------------------------------------------------------------------------------+
| W |                                                                                 │
| o |  ← Back to runs  Run #run-8a2f                                  [Export evidence — not available in v1] │
| r |                                                                                 │
| k |  Status: completed   Type: batch   Started: 18 Jul 09:30   Finished: 09:31   │
|   |  Legal basis: GDPR Art.6(1)(f) legitimate-interest                            │
|   |                                                                                 │
|   |  RunProgressTimeline                                                            │
|   |  queued -> running -> completed                                                   │
|   |  (no per-module streaming logs; final state only)                                 │
|   |                                                                                 │
|   |  Lead outcomes                                                                   │
|   |  ┌──────────────┬─────────────────┬─────────────┬─────────────┐                │
|   |  │ Lead         │ email-validate  │ domain-intel│ Result link │                │
|   |  ├──────────────┼─────────────────┼─────────────┼─────────────┤                │
|   |  │ lead-7c1     │ ok              │ ok          │ [View ->]    │                │
|   |  │ lead-9d4     │ ok              │ skipped     │ [View ->]    │                │
|   |  └──────────────┴─────────────────┴─────────────┴─────────────┘                │
|   |                                                                                 │
|   |  Audit events (from run.audit_event_ids)                                       │
|   |  [ActivityTimeline component with module/status/tool/checked_at]               │
|   |                                                                                 │
+-----------------------------------------------------------------------------------+
```

**Interaction notes:**
- `RunProgressTimeline` is built from `PipelineRun.status`, `started_at`, and `finished_at`. It does **not** synthesize fake step logs.
- The “Lead outcomes” table links each lead to `/leads/{id}?tab=checks`.
- Audit events are fetched from `GET /api/audit?module=&status=`. The v1 API does not support `run_id` filtering; the page may display events whose IDs match `run.audit_event_ids` from the events already loaded for the current audit page only.

### 5.8 Audit Log Explorer (`/audit`)

```
+-----------------------------------------------------------------------------------+
| ≡ | OSINT Lead Console                                              | Settings  |
+-----------------------------------------------------------------------------------+
| W |                                                                                 │
| o |  Audit Log                                                                      │
| r |                                                                                 │
| k |  Filters: Module [All ▼]  Status [All ▼]  Lead ID (filter loaded page) [     ]    │
|   |                                                                                 │
|   |  ┌─────────────────────────────────────────────────────────────────────────┐   │
|   |  │ Time │ Module       │ Lead     │ Status │ Tool              │ Basis   │   │
|   |  ├──────┼──────────────┼──────────┼────────┼───────────────────┼─────────┤   │
|   |  │09:41 │ email-validate│ lead-7c1 │ ok     │ AfterShip/...     │ GDPR…   │   │
|   |  │09:40 │ domain-intel│ lead-7c1 │ ok     │ web-check         │ GDPR…   │   │
|   |  │09:38 │ social-...  │ lead-1e04│ skipped│ maigret           │ GDPR…   │   │
|   |  └─────────────────────────────────────────────────────────────────────────┘   │
|   |                                                                                 │
|   |  Row click opens AuditEventDetailDrawer.                                        │
+-----------------------------------------------------------------------------------+
```

**Interaction notes:**
- Uses `GET /api/audit` with `module`, `status`, `page`, `page_size`.
- The v1 audit API does **not** support server-side filtering by `lead_id`, `run_id`, or date range. The Lead ID box is a client-side text filter over the currently loaded page only; it cannot search the entire audit log.
- The drawer exposes `subject` and `raw_stderr_json`.

### 5.9 Modules (`/modules`)

```
+-----------------------------------------------------------------------------------+
| ≡ | OSINT Lead Console                                              | Settings  |
+-----------------------------------------------------------------------------------+
| W |                                                                                 │
| o |  Modules                                                                        │
| r |                                                                                 │
| k |  Available now                                                                   │
|   |  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐                 │
|   |  │ email-validate  │ │ domain-intel    │ │ social-footprint│                 │
|   |  │ Validate        │ │ Ingest          │ │ Enrich          │                 │
|   |  │ [View docs ->]   │ │ [View docs ->]   │ │ [View docs ->]   │                 │
|   |  └─────────────────┘ └─────────────────┘ └─────────────────┘                 │
|   |                                                                                 │
|   |  Planned                                                                         │
|   |  ┌─────────────────┐                                                           │
|   |  │ extraction      │                                                           │
|   |  │ company-enrich  │                                                           │
|   |  └─────────────────┘                                                           │
|   |                                                                                 │
+-----------------------------------------------------------------------------------+
```

**Interaction notes:**
- Keep existing grouping but use denser cards and remove the “In development” group if empty (the registry currently has only `available` and `planned`).
- Each card links to `/modules/{name}`.

### 5.10 Module Detail (`/modules/{name}`)

```
+-----------------------------------------------------------------------------------+
| ≡ | OSINT Lead Console                                              | Settings  |
+-----------------------------------------------------------------------------------+
| W |                                                                                 │
| o |  ← Back to modules  email-validate                              [Run on lead] │
| r |                                                                                 │
| k |  Status: available   Category: validate   Min input: email                     │
|   |                                                                                 │
|   |  Description                                                                    │
|   |  Syntax, MX, disposable and role-account checks. SMTP probe disabled.          │
|   |                                                                                 │
|   |  Backing tools: AfterShip/email-verifier@v1.4.1                                │
|   |  Risk note: Low personal-data exposure.                                        │
|   |                                                                                 │
|   |  Configuration schema                                                           │
|   |  (table or empty state)                                                        │
|   |                                                                                 │
|   |  Documentation                                                                  │
|   |  (rendered module.docs markdown)                                               │
|   |                                                                                 │
+-----------------------------------------------------------------------------------+
```

**Interaction notes:**
- “Run on lead” opens a small dialog that lets the operator pick from recent leads (client-side list from `GET /api/leads` limited to 10) and then calls `POST /api/leads/{id}/run`.

### 5.11 Compliance (`/compliance`)

```
+-----------------------------------------------------------------------------------+
| ≡ | OSINT Lead Console                                              | Settings  |
+-----------------------------------------------------------------------------------+
| W |                                                                                 │
| o |  Compliance                                                                     │
| r |                                                                                 │
| k |  Hard rules                    │ Risk table                                    │
|   |  ┌──────────────────────────┐  │ ┌─────────────────────────────────────────┐ │
|   |  │ 1 No non-consensual...   │  │ │ Category           │ Risk  │ Notes      │ │
|   |  │ 2 Respect third-party... │  │ ├────────────────────┼───────┼────────────┤ │
|   |  │ ...                      │  │ │ Email verification │ Low   │ ...        │ │
|   |  └──────────────────────────┘  │ └─────────────────────────────────────────┘ │
|   |                                                                                 │
|   |  Pre-run checklist              Exclusions                                     │
|   |  [[x]] Permission ref recorded    - LinkedIn scraping                          │
|   |  [[x]] Legal basis confirmed      - Reverse-image / deep discovery              │
|   |  ...                            - Bulk breach/leak signals in sales views     │
|   |                                                                                 │
+-----------------------------------------------------------------------------------+
```

**Interaction notes:**
- Keep the content from `docs/compliance.md` and the current `app/compliance/page.tsx`.
- Add a “Review recent audit events” link to `/audit`.

### 5.12 Settings (`/settings`)

```
+-----------------------------------------------------------------------------------+
| ≡ | OSINT Lead Console                                              | Settings  |
+-----------------------------------------------------------------------------------+
| W |                                                                                 │
| o |  Settings                                                                       │
| r |                                                                                 │
| k |  Environment & API reachability                                                       │
|   |  API base URL: http://localhost:8080            [APIHealthIndicator: reachable]  │
|   |  [Refresh reachability]  [Edit base URL]                                             │
|   |                                                                                 │
|   |  Role: [Ops analyst ▼]                                                         │
|   |                                                                                 │
|   |  Connectors (stubs)                                                            │
|   |  CRM      [Configure ->]  (not wired)                                          │
|   |  SSO      [Configure ->]  (not wired)                                            │
|   |  API keys [Configure ->]  (not wired)                                            │
|   |  Retention [Configure ->]  (not wired)                                           │
|   |                                                                                 │
+-----------------------------------------------------------------------------------+
```

**Interaction notes:**
- `APIHealthIndicator` calls `GET /api/leads?page_size=1` and reports whether the API is reachable (not overall system health) plus latency.
- Base URL editing is local-only and requires a page reload; it does not persist to backend.
- All connector cards remain stubs and are labeled “not wired”.

---

## 6. Logs and run observability

### 6.1 What the current API can truthfully show

- `GET /api/runs` and `GET /api/runs/{id}` return a `PipelineRun` with:
  - `id`, `type`, `status` (`running` | `completed` | `failed` | `partial`),
  - `started_at`, `finished_at`,
  - `lead_ids`, `modules_executed`, `audit_event_ids`,
  - `legal_basis`, `permission_refs`, `error`.
- `GET /api/audit` returns persisted `AuditEvent` records with:
  - `module`, `tool`, `checked_at`, `status`, `legal_basis`, `subject`, `raw_stderr_json`.
- `GET /api/leads/{id}` returns final module result keys flattened by `leadToJSON` (`services/control-plane/internal/http/handlers.go`).

### 6.2 What the current API cannot truthfully show

- **Real-time per-module streaming logs.** `RunBatch` and `RunSingle` are synchronous. The backend does not emit SSE/WebSocket events or per-step status updates while a module is executing.
- **Partial completion inside a single-lead run.** `POST /api/leads/{id}/run` returns after all requested modules finish; intermediate status is not available.
- **ETA or progress percentage.** The frontend has no deterministic signal of how long Maigret/theHarvester will take.

### 6.3 Proposed frontend behaviour for synchronous requests and polling

1. **Single-lead module run (`POST /api/leads/{id}/run`)**
   - Show a local `pending` state on the button/module row.
   - Block only the module(s) being run, not the whole page.
   - On success (HTTP 200), refresh `useLead(id)` and show a toast with the final status.
   - On error, show the error from the response body and keep the prior state.
   - Do not render a fake log stream.

2. **Batch pipeline run (`POST /api/pipelines/run`)**
   - On `202 Accepted`, immediately navigate to `/runs/{run_id}`.
   - Because `RunBatch` is synchronous today, the run will usually already be `completed` or `partial` on arrival. The page should still poll `GET /api/runs/{id}` every 5 seconds for up to 30 seconds, stopping when status is terminal.
   - Polling interval must be backed off and cancelled on unmount.

3. **Run detail timeline**
   - `RunProgressTimeline` renders three states: `queued`, `running`, `completed`/`partial`/`failed`.
   - If `status === "running"` and `finished_at` is missing, show an indeterminate progress indicator and the copy “Waiting for synchronous run to finish… No live logs are available.”
   - The lead-outcomes table uses final module result keys from each lead.

4. **Audit log**
   - The global `/audit` page uses `GET /api/audit` with `module`, `status`, `page`, `page_size`.
   - Server-side filtering by `lead_id`, `run_id`, or date range is **not** available in v1; any such filtering in the UI is local to the loaded page.

### 6.4 Future API proposal (documented, not implemented)

A separate future control-plane proposal could add:

- `POST /api/jobs` returning a `job_id`.
- `GET /api/jobs/{id}` with `status`, `progress`, `events[]`.
- Optional SSE endpoint `GET /api/jobs/{id}/stream` for terminal/emitted events.
- `PipelineRun` extended with an optional `job_id`.
- Audit query expansion: `GET /api/audit?lead_id=&run_id=&from=&to=&module=&status=&page=&page_size=` so the Audit Explorer can search by lead/run/date without client-side filtering limitations.

This redesign does **not** implement or depend on that proposal. Frontend code must continue to work with the current synchronous API.

### 6.5 Technical-evidence (`raw_stderr_json`) disclosure rules

- `raw_stderr_json` is **collapsed by default** in every UI surface.
- The disclosure control is labeled **“Technical evidence — may contain sensitive operational data”**.
- It is shown only inside `AuditEventDetailDrawer` or an equivalent detail panel; it is **never rendered in table cells, summary cards, or command-center metrics**.
- There is **no bulk export** of `raw_stderr_json` in v2.
- The UI must preserve the existing phone-redaction behaviour: redacted phone identifiers must not be exposed in the audit list or detail.

---

## 7. Visual design system v2

### 7.1 Responsive sidebar states

| Viewport | Sidebar behaviour |
|----------|-----------------|
| `>= 1280px` (xl) | Expanded, 14rem (`56`), icons + labels, fixed. |
| `1024px–1279px` (lg) | Collapsed icon-only rail, 4rem (`16`), tooltips on focus/hover. |
| `< 1024px` | Hidden; hamburger toggle opens a drawer overlay with a close button and backdrop. |

- Keyboard: `Esc` closes the mobile drawer; focus is trapped inside while open.
- Active item uses a 2px left accent bar + subtle background tint.
- Group labels are visible only in expanded mode.

### 7.2 Content max widths / 12-column layout

- Page canvas: full width, with a consistent `px-4 sm:px-6 lg:px-8` horizontal padding.
- Content max-widths by page:
  - Command Center: full-width 12-col grid, max `max-w-[1600px]`.
  - Leads list: full-width table, no inner max-width.
  - Lead detail: 8-col primary + 4-col secondary at `lg`, stacked at `md`.
  - Runs/Audit: full-width table.
  - Settings/Compliance: `max-w-5xl` for readability.
- Grid gap: `1rem` (`gap-4`) default, `1.5rem` (`gap-6`) for card grids.

### 7.3 Typography scale

| Token | Size | Usage |
|-------|------|-------|
| `text-xs` | 12px / 0.75rem | Meta labels, timestamps, status chips, table headers. |
| `text-sm` | 14px / 0.875rem | Body, buttons, inputs, table cells, badge text. |
| `text-base` | 16px / 1rem | Page titles, card titles, emphasis. |
| `text-lg` | 18px / 1.125rem | Command Center metric values. |
| `text-xl` | 20px / 1.25rem | Empty-state headline. |
| `font-medium` | — | Labels, buttons, active nav item. |
| `font-semibold` | — | Headings, metric values. |

Font family remains Inter (`var(--font-inter)`). No all-caps labels except for style-guide section headers.

### 7.4 Semantic color tokens and status usage

Base tokens (existing, refined):

```
--background:        #050816   (page background)
--surface:           #0b1224   (cards, sidebar)
--surface-elevated:  #0f172a   (hover, drawers, code blocks)
--foreground:        #f8fafc   (primary text)
--foreground-secondary: #cbd5e1 (secondary text)
--foreground-muted:  #94a3b8   (disabled, placeholders)
--primary:           #2dd4ff   (interactive accent)
--secondary:         #6366f1   (secondary actions)
--success:           #34d399   (ok, completed, low risk)
--warning:           #fbbf24   (unknown, partial, missing permission)
--danger:            #f97373   (error, high risk, exclusion)
--muted:             #94a3b8   (not_run, planned, disabled)
--border:            rgba(45, 212, 255, 0.12)
```

Status mapping:

| Status | Token | Usage |
|--------|-------|-------|
| `ok` / `completed` / `low` | `success` | Positive result. |
| `unknown` / `partial` / `medium` / missing permission_ref | `warning` | Needs attention, not a failure. |
| `skipped` / `planned` / `not_run` | `muted` | Neutral, informational. |
| `failed` / `high` / excluded | `danger` | Blocking or risky. |
| `running` / `pending` | `primary` | In-progress. |

### 7.5 Table density and card rules

- **Table rows:** 44px min-height (`h-11`).
- **Table headers:** 36px, sticky on scroll, uppercase `text-xs` with `font-medium` and `foreground-muted`.
- **Cards:**
  - Default: `bg-surface`, `border border-border`, `rounded-lg`, `p-4`.
  - Elevated (drawers, dropdowns): `bg-surface-elevated`, `shadow-lg`.
  - Flat lists inside cards use internal dividers only when necessary.
- **Spacing scale:** page section gaps `24px` (`space-y-6`), card internal gaps `16px` (`space-y-4`).
- **Borders:** 1px `border` token; do not stack multiple nested cards with identical borders.

### 7.6 Empty, loading, error, unknown, skipped, and offline states

| State | Component | Behaviour |
|-------|-----------|-----------|
| Empty | `EmptyWorkspaceState` | Centered in page (not inside a tiny card), icon + headline + one-line explanation + primary CTA + secondary link. |
| Loading | `Skeleton` | Use `aria-busy="true"` and `aria-describedby` pointing to a visually hidden “Loading” label. Avoid shimmer animation if reduced motion is preferred. |
| Error | `ErrorState` | Inline banner with message, `Retry` button, and (if available) error code. Persistent at top of page. |
| Unknown | `StatusChip` + tooltip | Label “unknown — check did not return a clear signal”. Orange badge. |
| Skipped | `StatusChip` + tooltip | Label “skipped — missing input or not applicable”. Gray badge. |
| Offline | `APIHealthIndicator` + `ErrorState` | Top-level banner: “Cannot reach control-plane API. [Retry]”. Disable destructive actions. |

### 7.7 Accessibility rules

- **Contrast:** text on `background` and `surface` must meet WCAG AA (4.5:1 for normal text, 3:1 for large text). `foreground-muted` on `surface` is allowed only for non-essential meta text.
- **Focus:** all interactive elements receive a 2px `primary` focus ring with `outline-offset-2`. Focus rings are never suppressed.
- **Icon labels:** every `IconButton` must have an explicit `label` prop rendered as `aria-label`.
- **Reduced motion:** honor `prefers-reduced-motion` by disabling transform transitions and animations; instant state changes only.
- **Keyboard controls:**
  - `Tab` moves through all focusables in logical order.
  - `Esc` closes drawers, dialogs, and mobile sidebar.
  - Data tables: each row is focusable; `Enter` navigates to detail; checkbox has its own focus target.
  - Skip link to main content on first `Tab`.
- **Screen reader:** page titles update per route; live region announces async toast messages.

---

## 8. Component inventory

### 8.1 Existing components to retain

Most primitives remain. Refinements are styling/density only.

- `Button`, `IconButton` — add `aria-busy` when pending.
- `Input`, `Select`, `Textarea` — keep API, tighten spacing.
- `Card` — keep, but discourage nesting.
- `Badge` — keep variants, add `muted`.
- `Table`, `TableHead`, `TableBody`, `TableRow`, `TableHeader`, `TableCell` — keep.
- `Tabs` — keep, add URL persistence.
- `Dialog` — keep; ensure focus trap.
- `Skeleton` — keep; add reduced-motion variant.
- `EmptyState` — keep as base; new `EmptyWorkspaceState` wraps it for page-level empty states.
- `Toast`, `Tooltip` — keep.
- `PageHeader` — keep; add support for actions and breadcrumb.
- `StatusChip` — keep; refine label tooltips.
- `PipelineStepper` — keep; use only in lead detail Overview.
- `EnvironmentBanner` — refactor into `APIHealthIndicator`.

### 8.2 Existing components to refactor

| Component | Current issues | Refactor target |
|-----------|----------------|-----------------|
| `AppShell` | Sidebar always 15rem, content narrow. | Three-column flex layout: sidebar rail/collapsed/exp, main scroll area. |
| `Sidebar` | No groups, no icon-only state, mobile drawer simple. | `ResponsiveSidebar` with Workspace/Operations/Admin groups, rail mode, active accent bar. |
| `TopBar` | Search stub, weak orientation. | Show current page title + breadcrumb, move API reachability to `APIHealthIndicator` in Settings or a subtle inline indicator. |
| `Footer` | Optional; keep minimal copyright/help links. | No major change. |
| `leads/page.tsx` | Empty state buried, filters always shown. | New `LeadsPage` with `EmptyWorkspaceState`, filter bar hidden when empty, `LeadReadinessCell`. |
| `leads/[id]/page.tsx` | Long vertical stack, audit below fold. | Tabbed layout: Overview, Checks, Activity & Audit, Raw record; use `ModuleRunDrawer`. |
| `runs/page.tsx` | No error column summary, no status count. | Add status badges and error snippet. |
| `runs/[id]/page.tsx` | No timeline, no per-lead outcomes. | Add `RunProgressTimeline` and lead-outcomes table. |
| `modules/page.tsx` | Empty “In development” group possible. | Hide empty groups; use denser cards. |
| `settings/page.tsx` | Stubs not connected. | Add `APIHealthIndicator`, label every stub as “not wired”. |
| `AuditLogPanel` | Only local to lead detail. | Extract shared timeline row; use in lead detail and `/audit`. |

### 8.3 New components needed

| Component | Location | Responsibility |
|-----------|----------|----------------|
| `CommandCenter` | `app/command-center/page.tsx` + `components/command-center/` | Landing dashboard: metrics, quick actions, recent activity. |
| `OperationalMetric` | `components/ui/OperationalMetric.tsx` | Big number + trend + link for Command Center cards. |
| `LeadQuickCreate` | `components/leads/LeadQuickCreate.tsx` | Two-step create form used by `/leads/create` and Command Center quick action. |
| `LeadReadinessCell` | `components/leads/LeadReadinessCell.tsx` | Mini stacked progress of wired modules in table rows. |
| `ModuleRunDrawer` | `components/modules/ModuleRunDrawer.tsx` | Drawer for running a module: shows module, legal basis, duration warning, confirm/cancel. |
| `RunProgressTimeline` | `components/runs/RunProgressTimeline.tsx` | Non-streaming timeline from `PipelineRun` status and timestamps. |
| `ActivityTimeline` | `components/ui/ActivityTimeline.tsx` | Shared timeline of audit events / system events. |
| `AuditExplorer` | `app/audit/page.tsx` + `components/audit/AuditExplorer.tsx` | Global audit log table with filters. |
| `AuditEventDetailDrawer` | `components/audit/AuditEventDetailDrawer.tsx` | Drawer showing full `AuditEvent` including subject and raw JSON. |
| `EmptyWorkspaceState` | `components/ui/EmptyWorkspaceState.tsx` | Page-level empty state with CTA and guidance. |
| `PermissionReferenceField` | `components/leads/PermissionReferenceField.tsx` | Input with help text, validation hint, and required badge. |
| `APIHealthIndicator` | `components/ui/APIHealthIndicator.tsx` | API reachability check with status, latency, and retry action. |
| `ResponsiveSidebar` | `components/layout/ResponsiveSidebar.tsx` | Refactored sidebar with rail/expanded/drawer states. |

### 8.4 Proposed directory reorganization

```
components/
  layout/
    AppShell.tsx
    ResponsiveSidebar.tsx
    TopBar.tsx
    Footer.tsx
  ui/               # design-system primitives
    Button.tsx
    IconButton.tsx
    Input.tsx
    Select.tsx
    Textarea.tsx
    Card.tsx
    Badge.tsx
    Table.tsx
    Tabs.tsx
    Dialog.tsx
    Toast.tsx
    Tooltip.tsx
    Skeleton.tsx
    EmptyState.tsx
    EmptyWorkspaceState.tsx
    PageHeader.tsx
    StatusChip.tsx
    PipelineStepper.tsx
    ActivityTimeline.tsx
    OperationalMetric.tsx
    APIHealthIndicator.tsx
  leads/
    LeadQuickCreate.tsx
    LeadReadinessCell.tsx
    PermissionReferenceField.tsx
  lead-detail/
    ModuleRunDrawer.tsx
    LeadOverviewTab.tsx
    LeadChecksTab.tsx
    LeadAuditTab.tsx
    LeadRawRecordTab.tsx
  runs/
    RunProgressTimeline.tsx
  audit/
    AuditExplorer.tsx
    AuditEventDetailDrawer.tsx
  command-center/
    CommandCenter.tsx
    CommandCenterMetrics.tsx
    CommandCenterActivity.tsx
  settings/
    (existing stubs)
```

---

## 9. Implementation PR plan

Keep each PR small, reviewable, and accompanied by desktop + mobile screenshots. All PRs are frontend-only and must pass `npm run typecheck && npm run lint && npm run build`.

### PR1 — Shell & design system / Command Center foundation

**Title:** `feat(ui): responsive shell and command-center foundation`

**Allowed paths:**

```
ui/web-console/app/layout.tsx
ui/web-console/app/page.tsx
ui/web-console/app/command-center/page.tsx
ui/web-console/app/style-guide/page.tsx
ui/web-console/app/globals.css
ui/web-console/components/layout/AppShell.tsx
ui/web-console/components/layout/TopBar.tsx
ui/web-console/components/layout/Footer.tsx
ui/web-console/components/layout/ResponsiveSidebar.tsx  (new)
ui/web-console/components/ui/EmptyWorkspaceState.tsx     (new)
ui/web-console/components/ui/APIHealthIndicator.tsx        (new)
ui/web-console/components/ui/OperationalMetric.tsx         (new)
ui/web-console/lib/theme/tokens.ts
```

**What it does:**
- Introduces `ResponsiveSidebar` with rail/expanded/drawer states and grouped navigation.
- Refactors `AppShell` to use the new sidebar and full-width main area.
- Updates `TopBar` to show page title/breadcrumb and moves API reachability to `APIHealthIndicator`.
- Adds new primitive components (`EmptyWorkspaceState`, `APIHealthIndicator`, `OperationalMetric`).
- Updates `style-guide` with all new components and states.
- Changes `/` to redirect to `/command-center`.
- Adds `/command-center` as a restrained foundation page: title, workflow explainer, links to Leads/Runs/Modules/Compliance, primary “Create lead” link, and `APIHealthIndicator`.

**PR1 intentionally does not show:** lead counts, trends, compliance scores, risk scores, module health claims, runtime estimates, or any data from Leads/Runs/Audit pages.

**Out of scope:** leads, lead detail, runs, audit explorer, create flow, modules, compliance, settings.

### PR2 — Leads & create lead

**Title:** `feat(ui): leads queue, readiness column, and two-step create lead`

**Allowed paths:**

```
ui/web-console/app/leads/page.tsx
ui/web-console/app/leads/create/page.tsx                  (new)
ui/web-console/components/leads/LeadQuickCreate.tsx       (new)
ui/web-console/components/leads/LeadReadinessCell.tsx     (new)
ui/web-console/components/leads/PermissionReferenceField.tsx (new)
ui/web-console/lib/api/hooks.ts                           (read-only usage; no API changes)
ui/web-console/lib/api/types.ts                           (read-only usage; no API changes)
```

**What it does:**
- Redesigns `/leads` list with hidden filters on empty state, `LeadReadinessCell`, and improved bulk actions.
- Moves create flow from dialog to `/leads/create` with two steps.
- Highlights `permission_ref` as required; Step 1 must be non-empty before module selection is enabled. This is a frontend guard only; backend enforcement is a separate control-plane concern.

**Out of scope:** lead detail.

### PR3 — Lead detail & audit

**Title:** `feat(ui): tabbed lead detail with module run drawer and audit timeline`

**Allowed paths:**

```
ui/web-console/app/leads/[id]/page.tsx
ui/web-console/components/lead-detail/LeadOverviewTab.tsx  (new)
ui/web-console/components/lead-detail/LeadChecksTab.tsx   (new)
ui/web-console/components/lead-detail/LeadAuditTab.tsx    (new)
ui/web-console/components/lead-detail/LeadRawRecordTab.tsx (new)
ui/web-console/components/modules/ModuleRunDrawer.tsx      (new)
ui/web-console/components/ui/ActivityTimeline.tsx        (new)
ui/web-console/components/audit/AuditEventDetailDrawer.tsx    (new)
ui/web-console/components/ui/AuditLogPanel.tsx             (refactor)
```

**What it does:**
- Replaces the single long lead detail page with tabbed navigation.
- Adds `ModuleRunDrawer` for module execution.
- Extracts `ActivityTimeline` and `AuditEventDetailDrawer`, refactors `AuditLogPanel` to reuse them.

**Out of scope:** global audit explorer, runs.

### PR4 — Runs & audit explorer

**Title:** `feat(ui): run progress timeline and global audit log explorer`

**Allowed paths:**

```
ui/web-console/app/runs/page.tsx
ui/web-console/app/runs/[id]/page.tsx
ui/web-console/app/audit/page.tsx                         (new)
ui/web-console/components/runs/RunProgressTimeline.tsx   (new)
ui/web-console/components/audit/AuditExplorer.tsx         (new)
ui/web-console/components/audit/AuditEventDetailDrawer.tsx
ui/web-console/components/ui/ActivityTimeline.tsx
```

**What it does:**
- Redesigns `/runs` list and detail with `RunProgressTimeline` and per-lead outcome table.
- Adds `/audit` global explorer using `GET /api/audit`.
- Reuses `ActivityTimeline` and `AuditEventDetailDrawer`.

**Out of scope:** control-plane changes.

### PR5 — Optional backend observability proposal

**Title:** `docs(control-plane): proposal for async job observability API`

**Allowed paths:**

```
docs/frontend/ux-redesign-v2.md                           (append async API proposal)
docs/frontend/api-contracts.md                          (only if adding a clearly-versioned v2 proposal section)
```

**What it does:**
- Documents a future async job/SSE proposal separately from the v2 UI redesign.
- Does **not** change `services/control-plane` code or package files.

---

## 10. Acceptance criteria

### 10.1 UX criteria

- [ ] `/` redirects to `/command-center`, which shows queue health, recent runs, and a primary “Create lead” action.
- [ ] `/leads` empty state hides filters and guides the user to create a lead and view modules.
- [ ] `/leads` populated state shows `LeadReadinessCell` and missing-permission warnings.
- [ ] `/leads/create` is a two-step form with `permission_ref` emphasized in step 1 and module selection in step 2.
- [ ] `/leads/{id}` uses tabs: Overview, Checks, Activity & Audit, Raw record.
- [ ] Running a module opens `ModuleRunDrawer` and blocks only the requested module.
- [ ] `/runs/{id}` shows `RunProgressTimeline` and a per-lead outcome table.
- [ ] `/audit` provides module/status filters and opens `AuditEventDetailDrawer`.
- [ ] `/settings` surfaces `APIHealthIndicator` and labels all connector stubs as “not wired”.
- [ ] Every implementation PR includes screenshots at 1440px desktop and 375px mobile widths.

### 10.2 Accessibility criteria

- [ ] All interactive elements have visible focus rings.
- [ ] Sidebar toggle, drawer close, and dialog close work with `Esc`.
- [ ] Icon-only buttons have explicit `aria-label` text.
- [ ] Tables are keyboard-navigable: row focus + `Enter` to open detail.
- [ ] `prefers-reduced-motion` disables slide/scale animations.
- [ ] Empty, loading, and error states use appropriate ARIA live regions or `aria-busy`.
- [ ] Color contrast meets WCAG AA for all text and interactive states.

### 10.3 Responsive criteria

- [ ] Sidebar expands to 14rem at `xl`, collapses to 4rem rail at `lg`, and becomes a drawer below `lg`.
- [ ] Command Center metrics stack 1-col on mobile, 3-col on desktop.
- [ ] Leads table switches to a card list below `md` if horizontal scrolling is not acceptable; otherwise the table horizontally scrolls.
- [ ] Lead detail 8+4 grid stacks to 1-col below `lg`.
- [ ] Create lead form is single-column on mobile, two-column helper text on desktop.
- [ ] No layout breakage from 320px to 2560px widths.

### 10.4 Test / quality criteria

- [ ] `npm run typecheck` passes.
- [ ] `npm run lint` passes.
- [ ] `npm run build` passes.
- [ ] No new runtime dependencies.
- [ ] Existing control-plane tests (`go test ./...`) remain green; no `services/**` changes in frontend PRs.
- [ ] New components have at least smoke-level rendering in `app/style-guide/page.tsx` or equivalent.
- [ ] No fake streaming logs or simulated progress values are introduced.

---

## 11. Before / after user-flow comparison

### Current flow (pain points)

1. User opens app -> `/leads`.
2. Empty queue, but filters and stage funnel dominate the page.
3. User clicks “Create lead” -> long dialog with all fields equal.
4. User saves lead -> lands on detail page with everything stacked vertically.
5. To run a module, user scrolls to “Run module” card and clicks a button.
6. Module runs synchronously; toast appears; user must scroll back to results.
7. Audit trail is below the fold, no global view.
8. Runs are a separate list with no detail timeline.

### Redesigned flow

1. User opens app -> `/command-center` sees queue health + “Create lead” CTA.
2. Click “Create lead” -> `/leads/create` step 1 highlights `permission_ref`.
3. Step 2 lets user pick available modules with honest duration hints (no fixed ETA).
4. On create, user lands on `/leads/{id}?tab=checks`.
5. Each module result is decision-first; raw JSON is one click away.
6. Running a module opens a drawer with confirmation and legal basis.
7. Activity & Audit tab shows the full timeline; `/audit` provides global view.
8. Batch runs navigate to `/runs/{id}` with a timeline and per-lead outcomes.

---

## 12. Not in this redesign

The following items are explicitly excluded from the v2 UI redesign and are documented as future work or backend concerns:

- **CRM/evidence export wiring.** No real CRM connector, evidence export endpoint, or export audit trail exists; no export action is added in v2.
- **Real auth / SSO.** Settings stubs remain; no login flow.
- **Async job streaming / SSE.** Proposed separately in PR5; UI must work without it.
- **New control-plane API endpoints.** The redesign consumes existing v1 endpoints only.
- **Risk-level interpretation.** The UI renders `risk_level` returned by the current control-plane API. v2 must not imply that a composite or policy-derived risk score exists. `unknown` is a valid outcome and must be shown honestly.
- **`extraction` and `company-enrich` modules.** These are `planned` in the registry; UI shows them in the Planned group.
- **Bulk breach/leak signals, LinkedIn scraping, reverse-image discovery.** Excluded per `docs/compliance.md`; UI must not add UI for them.
- **Email/phone validation logic changes.** No changes to `modules/**` or `services/**`.
- **Dependency additions.** No new npm packages; use existing Tailwind + React + TanStack Query stack.

---

## 13. Appendix: current routes and components inventory

For reference, these are the existing files that inform this plan:

- Routes: `ui/web-console/app/{leads,leads/[id],modules,modules/[name],runs,runs/[id],compliance,settings,style-guide}/page.tsx`
- Layout: `ui/web-console/app/layout.tsx`, `ui/web-console/components/layout/{AppShell,Sidebar,TopBar,Footer}.tsx`
- Primitives: `ui/web-console/components/ui/*.tsx`
- API layer: `ui/web-console/lib/api/{client.ts,hooks.ts,types.ts}`
- Theme: `ui/web-console/lib/theme/tokens.ts`, `ui/web-console/app/globals.css`

This plan does not modify any of those files in the planning PR; implementation PRs will change them according to the sections above.
