# Company Enrichment Module — Stage 2 Planning Brief

**Author:** Devin (planning pass only)  
**Date:** 2026-07-19  
**Status:** Pass 3B module implementation and Pass 3C control-plane wiring complete. Review amendments applied: P0 slim (`domain`/`name`/`website`), local provider always-on, optional DiscoLike Go HTTP adapter, no LinkedIn/LLM/Crunchbase, GitHub org lookup gated, no invented company names from domain roots.  
**Canonical repo:** `https://github.com/Devin-gits/osint-lead-platform`

---

## Discovery summary

This section records what was verified before writing this plan.

### Existing `company-enrich` state

- `models.go`: `ModuleCompanyEnrich = "company-enrich"` is defined.
- `registry.go`: `company-enrich` is `DevStatus: "planned"`, `MinInputField: "company"`, `NamespacedKey: "company_enrich`.
- `runner.go`: `company-enrich` falls through to the default stub, returning `status: "skipped"` with reason `"not wired in control-plane v1"`.
- `docs/status/platform-v1.md`: `company-enrich` is listed as planned in the module matrix and backlog.
- `docs/decisions/stage-1-decision.md` `company-enrich` section: requires a second-opinion evaluation of `local-enrichment-tool` before any build; `waterfall-gtm` is reference-only for its cascade/early-stop pattern.

### Stage 1 second-opinion evaluations completed

| Evaluation | Recommendation |
|---|---|
| [`evaluations/local-enrichment-tool.md`](../evaluations/local-enrichment-tool.md) | **Reject** — one-day TypeScript/Node demo, requires paid Anthropic key, uses LinkedIn public-search scraping, LLM-synthesizes structured data. Not adoptable. |
| [`evaluations/discolike-cli.md`](../evaluations/discolike-cli.md) | **Optional paid adapter only** — clean MIT Python client for the DiscoLike B2B API, but requires paid subscription/key. Not an OSS core. |
| [`evaluations/waterfall-gtm.md`](../evaluations/waterfall-gtm.md) (existing) | **Reference only** — useful cascade/early-stop/merge pattern, but paid-only and unmaintained. |

### Existing module patterns to follow

- **Go library + CLI:** `modules/extraction` defines `Input`, `Fields`, `Result`, `Metadata`, `AuditRecord` structs, an `Extractor` type, and a `cmd/extraction/main.go` CLI that reads stdin and writes stdout + stderr audit. The same shape should apply to `modules/company-enrich`.
- **Namespaced result key:** `company_enrich` (already in registry).
- **Status semantics:** `ok`, `partial`, `unknown`, `skipped`, `error` (extraction added `partial` and `error` successfully).
- **Audit:** one JSON line per invocation with `module`, `tool`, `tool_version`, `timestamp`, `legal_basis`, `permission_ref`, `subject`, `status`, `duration_ms`, `limits`, `error`.
- **Permission reference:** extraction enforces `permission_ref` as mandatory. `company-enrich` should do the same.
- **Subprocess boundary:** non-MIT tools (theHarvester GPL, Maigret, Crawl4AI Apache-2.0) are invoked as subprocesses. The same boundary should apply to any Python/Node adapter.

---

## 1. Decision and recommended approach

### Recommendation

**GO — implement `modules/company-enrich` as a Go library with a `Provider` interface.**

- **Primary default provider:** `local` — deterministic, no API key, combines data already on the lead (`extraction`, `domain_intel`) with public, non-scraping sources (GitHub public org API, website metadata already extracted). It is allowed to return `partial` or `unknown` for fields it cannot fill honestly.
- **Optional paid adapters:** `discolike` (DiscoLike business profile + tech vendors), and later `clearbit`, `apollo`, `pdl` — each behind an API key and a `COMPANY_ENRICH_*_API_KEY` env var. Never the default, never required.
- **Cost-control pattern:** a `WaterfallEnricher` (inspired by `waterfall-gtm`'s `WaterfallProcessor` pattern) that calls providers in order and short-circuits once `required_fields` are satisfied, so paid adapters are billed only for missing data.
- **Explicit rejects:** LinkedIn scraping, LLM synthesis of company facts, Crunchbase page scraping, breach/people contact expansion. These remain out of scope per `docs/compliance.md`.

### Why this approach

| Option | Verdict |
|---|---|
| A) Provider interface + deterministic local provider + optional paid adapters | **Selected** — respects the open-core MIT path, keeps paid APIs optional, and gives a clean seam for future sources. |
| B) Wrap a specific OSS tool | **Not viable** — `local-enrichment-tool` is rejected; no other evaluated OSS tool provides a free, maintained, ToS-clean B2B firmographic core. |
| C) Defer module | **Not selected** — the gap is real (page extraction ≠ registry firmographics), the interface design is tractable, and the risk is acceptable if the default provider stays deterministic and no-key. |

### Problem statement

`extraction` gives low-risk page fields (title, description, emails, phones, contact URLs, social links) from a permissioned URL, but it does **not** produce registry-style firmographics: legal name, founded year, HQ country/city, industry codes, employee count band, tech stack, funding. Sales workflows need this context to score and route leads. `company-enrich` fills that gap without re-running the extraction module and without scraping LinkedIn or synthesizing data via LLM.

---

## 2. In-scope outputs (v1 proposal)

The `company_enrich` namespaced result should expose a conservative set of firmographic fields, all sourced deterministically or from paid adapters with clear attribution:

```json
{
  "company_enrich": {
    "status": "ok",
    "source_tool": "company-enrich/local+discolike",
    "confidence": 0.6,
    "domain": "example.com",
    "name": "Example Corp",
    "legal_name": "Example Corporation, Inc.",
    "website": "https://example.com",
    "description": "Enterprise widget platform.",
    "founded": 2015,
    "employee_count": 250,
    "employee_count_range": "201-500",
    "headquarters": {
      "city": "San Francisco",
      "state": "CA",
      "country": "US",
      "address": "123 Mission St"
    },
    "industry": ["Software", "B2B SaaS"],
    "social_links": {
      "linkedin": "https://linkedin.com/company/example",
      "twitter": "https://twitter.com/example",
      "github": "https://github.com/example"
    },
    "tech_stack": ["React", "Node.js", "AWS"],
    "sources": ["extraction", "github_public_api", "discolike/profile"],
    "metadata": {
      "backend": "local+discolike",
      "legal_basis": "GDPR Art.6(1)(f) legitimate-interest",
      "permission_ref": "CAMP-2026-Q3-001",
      "duration_ms": 1240,
      "limits_applied": "max_providers=3,timeout=30s,required_fields=name,industry"
    },
    "error": "",
    "checked_at": "2026-07-19T12:00:00Z"
  }
}
```

### v1 field priority

| Priority | Field | Likely source in v1 |
|---|---|---|
| P0 | `domain`, `name`, `website`, `description` | `extraction`/`domain_intel`/website |
| P0 | `headquarters.country`, `headquarters.city` | Paid adapter or website contact page (if already extracted) |
| P1 | `industry`, `employee_count_range`, `founded` | Paid adapter or public GitHub org metadata |
| P1 | `legal_name`, `employee_count` | Paid adapter only |
| P2 | `tech_stack` | Paid adapter (`discolike vendors`) or website tech detection |
| P2 | `social_links` | Already from `extraction`; normalized here |
| Out | `funding`, `leadership`, `job_postings`, `ceo`, `founders` | Too PII-heavy, too unreliable, or requires LinkedIn/LLM. Defer. |

### Status semantics

- `ok` — at least one provider returned enough data to satisfy the P0 fields.
- `partial` — some data returned but P0 fields incomplete.
- `skipped` — no `company`/`domain` input, no `permission_ref`, or no providers configured.
- `error` — all configured providers failed.
- `unknown` — a provider was attempted but returned no usable data (preserved per-provider if needed).

---

## 3. Out of scope

- LinkedIn scraping or profile-field extraction.
- Crunchbase public-page scraping.
- LLM synthesis of company facts (no Anthropic/OpenAI dependency).
- People/contact discovery (CEO, founders, individual emails) — this bleeds into personal data and is not required for firmographic enrichment.
- Funding, leadership changes, job postings in v1.
- Bulk market discovery (`discolike discover`, `count`) — `company-enrich` enriches one lead's company, not generates lead lists.
- Breach/leak data.
- Company-wide graph crawling.
- CRM/HubSpot sync.
- Auth/SSO, orchestration, queues.

---

## 4. Proposed module I/O contract

### Input

A single JSON object on stdin, or CLI flags `--domain`, `--company`, `--url`.

```json
{
  "domain": "example.com",
  "company": "Example Corp",
  "url": "https://example.com",
  "permission_ref": "CAMP-2026-Q3-001",
  "source_id": "ad-campaign-summer-2026",
  "extraction": { "status": "ok", "fields": { "company_name": "..." } },
  "domain_intel": { "status": "ok", "whois": { ... } },
  "required_fields": ["name", "industry"]
}
```

| Field | Required | Description |
|---|---|---|
| `domain` | preferred | Company domain; the canonical lookup key. |
| `company` | fallback | Human-readable company name if domain is missing. |
| `url` | optional | Permissioned URL; can be used to derive domain and pre-populate website/description. |
| `permission_ref` | **yes** | Privacy-safe reference for audit/compliance. |
| `source_id` | no | Campaign/source reference for provenance. |
| `extraction` | no | Existing `extraction` result to reuse (avoid re-fetching). |
| `domain_intel` | no | Existing `domain_intel` result to reuse. |
| `required_fields` | no | Ordered list of fields the waterfall should try to satisfy before calling paid adapters. |

### Output

The same record with a `company_enrich` key added:

```json
{
  "domain": "example.com",
  "company": "Example Corp",
  "company_enrich": { ... }
}
```

### Audit record (stderr)

One JSON line per provider invocation plus one top-level module line:

```json
{
  "module": "company-enrich",
  "tool": "local",
  "tool_version": "company-enrich/local@v1",
  "timestamp": "2026-07-19T12:00:00Z",
  "legal_basis": "GDPR Art.6(1)(f) legitimate-interest",
  "permission_ref": "CAMP-2026-Q3-001",
  "subject": { "domain": "example.com" },
  "status": "partial",
  "duration_ms": 340,
  "limits": "timeout=30s,required_fields=name,industry",
  "error": ""
}
```

For paid adapters, the subject stays `domain`; no personal names or emails are logged.

---

## 5. Recommended architecture

### Module layout

```
modules/company-enrich/
  companyenrich.go        # Enricher type, Input, Fields, Result, Metadata, AuditRecord, Enrich()
  providers.go            # Provider interface
  local.go                # Local provider: extraction/domain_intel reuse, GitHub public API, website heuristics
  discolike.go            # Optional DiscoLike HTTP adapter
  waterfall.go            # Ordered provider runner with early-stop and merge logic
  cmd/company-enrich/
    main.go               # CLI: stdin JSON, stdout augmented, stderr audit
  *_test.go               # Unit and fixture tests
  go.mod
  README.md
```

### `Provider` interface (sketch)

```go
type Provider interface {
    Name() string
    Enrich(ctx context.Context, in Input) (ProviderResult, error)
}

type ProviderResult struct {
    Status     string   // ok, partial, unknown, error
    Fields     Fields
    SourceTool string
    Error      string
}
```

### `Enricher` responsibilities

- Validate input (domain or company; `permission_ref` required).
- Normalize domain from URL/company if needed.
- Run providers in order:
  1. `local` (always, no key)
  2. `discolike` (only if `DISCOLIKE_API_KEY` set and `required_fields` not yet satisfied)
  3. Future adapters (Clearbit, Apollo, PDL) behind their own keys.
- Merge results with later providers filling gaps (or overriding with higher-confidence data if configured).
- Stop early once `required_fields` are satisfied.
- Compute top-level `status` and `confidence`.
- Emit one audit record per provider plus a top-level module audit record.
- Exit 0 for all operational outcomes; exit 1 only for unreadable input.

### Provider-specific notes

#### `local` provider

- Reuse `extraction.fields.company_name` if present.
- Reuse `extraction.fields.description`.
- If `domain_intel.whois.registrant_organization` or `domain_intel.web_check.ssl.subject` contain an organization name, use as candidate `legal_name`.
- Optional: derive GitHub org from domain (`https://github.com/<domain-root>`) and call public `GET /orgs/{org}` with unauthenticated rate limits (60/hr). Use `blog`, `location`, `name`, `public_repos` only. Degrade gracefully on 404/403.
- No scraping of LinkedIn/Crunchbase/job boards.
- Expected to return `partial` for most fields; this is honest.

#### `discolike` adapter

- Re-implement as a small Go HTTP client (do not import `discolike-cli` into the Go module).
- Endpoint: `GET /profile?domain=...` and `GET /vendors?domain=...`.
- Env: `DISCOLIKE_API_KEY`, `DISCOLIKE_BASE_URL` (default to DiscoLike API), `DISCOLIKE_TIMEOUT`.
- If key is missing, return `skipped` with reason `"DISCOLIKE_API_KEY not set"`.
- Map response into `Fields`.
- Cache per-run to avoid duplicate calls.

---

## 6. Control-plane integration sketch

No code changes in this pass. Future Pass 3C would:

1. Add `modules/company-enrich` as a Go module and `go.mod` replace in `services/control-plane`.
2. Import `companyenrich` in `services/control-plane/internal/runner/runner.go`.
3. Add `runCompanyEnrich` method analogous to `runExtraction`:
   - Require `permission_ref` (from lead or run request) → `skipped` if missing.
   - Use lead's `domain`, `company`, `url`, `extraction`, `domain_intel` to build input.
   - Call `companyenrich.Enricher.Enrich()`.
   - Convert `AuditRecord` into `models.AuditEvent` with `Subject.Domain`.
   - Store result under `lead.Results["company_enrich"]`.
4. Update `computeStage` so that `company_enrich` status `ok` advances to `StageEnriched` (alongside `extraction`/`domain_intel`).
5. Update `internal/registry/registry.go` `DevStatus` to `available` and `BackingTools`/`Description` to match the plan.

### Registry changes (future)

```go
{
  Name:          models.ModuleCompanyEnrich,
  DisplayName:   "Company Enrichment",
  Category:      "enrich",
  DevStatus:     "available",
  NamespacedKey: "company_enrich",
  BackingTools:  []string{"local (deterministic)", "discolike (optional paid adapter)"},
  Description:   "Enrich company firmographics from deterministic public sources and optional paid B2B data adapters.",
  MinInputField: "domain",
  RiskLevelNote: "Low-Medium: company-level data only; paid adapters require API key and are optional.",
}
```

---

## 7. UI integration sketch

No UI code in this pass. Future Pass 3D would:

1. Add `company_enrich` to `lib/api/types.ts` `CompanyEnrichResult` interface matching the `Fields` output.
2. Add a **Company** tab on `/leads/[id]` showing:
   - `name`, `legal_name`, `domain`, `website`
   - `description`
   - `headquarters` (city/state/country)
   - `founded`, `employee_count`, `employee_count_range`
   - `industry` chips
   - `tech_stack` chips
   - `social_links` links
   - `sources` chips
   - `confidence` progress bar
   - `metadata` collapsible (legal basis, permission ref, duration)
3. Add `company-enrich` to module selection in `CreateLeadFlow` and bulk actions; gating: enabled when `domain` or `company` or `url` is present.
4. Add `company-enrich` status filter options on `/leads` and `/audit` (status already supports `partial`/`error`).
5. Run CTA: "Run company enrichment" with disabled reason when `domain`/`company`/`url` missing or no `permission_ref`.

---

## 8. License and supply-chain gate

| Component | Proposed role | License | Decision |
|---|---|---|---|
| `modules/company-enrich` Go code | Core orchestrator, CLI | MIT (own code) | Active |
| `local` provider | Deterministic aggregator | MIT (own code) | Active |
| GitHub public API | Optional org metadata | API ToS / public data | Active for unauthenticated low-volume calls |
| `discolike` adapter | Optional paid adapter | Go adapter = own MIT; data = DiscoLike commercial terms | Optional, behind `DISCOLIKE_API_KEY` |
| `local-enrichment-tool` | — | MIT but unsuitable | **Rejected** |
| `waterfall-gtm` | Design pattern reference only | MIT | Reference only |

### Rules

- No GPL/AGPL component imported or vendored into the Go core.
- Paid adapters are optional and must fail gracefully when their key is missing.
- Every adapter's `SourceTool` string must identify the upstream data source for audit.
- No LinkedIn, Crunchbase, or job-board scraping in the default `local` provider.

---

## 9. Compliance and `permission_ref` rules

- **Legal basis:** `GDPR Art.6(1)(f) legitimate-interest` on every audit record, matching extraction.
- **`permission_ref`:** Mandatory. If missing, return `skipped` with reason `"missing permission_ref"`.
- **Subject:** Audit `Subject.Domain` (or `Subject.Company` if no domain). No personal names or emails in audit.
- **Data minimization:** v1 restricts output to company-level firmographics. People/contact enrichment (CEO, founders, job postings) is out of scope.
- **Honest confidence:** `local` provider must not invent fields. `partial`/`unknown` are valid outcomes.
- **Paid adapter consent:** Using a paid adapter is an operator choice; the module logs which provider produced each field.

---

## 10. Test plan

Tests are designed only; no tests are written in this planning pass.

### Unit / fixture tests

| Test | Purpose |
|---|---|
| Missing `permission_ref` | Returns `skipped` with reason. |
| Missing domain/company/url | Returns `skipped` with reason. |
| Local provider with extraction fields | Reuses `company_name`, `description`, `social_links`. |
| Local provider with domain_intel WHOIS | Derives `legal_name` from registrant org. |
| Local provider with GitHub 404 | Degrades to `partial`, no panic. |
| Discolike adapter missing key | Returns `skipped` with reason. |
| Discolike adapter fixture | Maps `profile` JSON into `Fields`; emits audit. |
| Waterfall early-stop | `required_fields=["name"]` stops after `local` if name present; `discolike` not called. |
| Waterfall paid fallback | `local` returns partial, `discolike` fills industry; merged result is `ok`. |
| Merge conflict | Later provider does not overwrite earlier provider unless higher confidence. |
| Confidence calculation | More sources + more populated fields = higher confidence, capped at 1.0. |
| PII not in audit | Verify audit subject is domain, not names/emails. |

### Integration tests

- Fixture HTTP server for GitHub org endpoint and DiscoLike profile endpoint.
- No live paid API calls in CI.
- Network tests guarded with `if testing.Short() { t.Skip(...) }`.

### CI

- `go test ./...` in `modules/company-enrich`.
- `go vet ./...`.
- No API keys in CI.

---

## 11. Future implementation slices

### Pass 3B — Module implementation only

- Create `modules/company-enrich/`:
  - `Input`, `Fields`, `Result`, `Metadata`, `AuditRecord` structs.
  - `Provider` interface.
  - `local` provider.
  - `discolike` adapter (fixture-tested, key-gated).
  - `WaterfallEnricher`.
  - `cmd/company-enrich/main.go`.
  - `README.md` with I/O contract and env vars.
  - Unit tests.
- Do **not** touch `services/control-plane` or `ui/web-console`.

### Pass 3C — Control-plane wire

- `services/control-plane/internal/runner/runner.go`: `runCompanyEnrich`.
- `services/control-plane/internal/registry/registry.go`: update `DevStatus` and docs.
- `services/control-plane/README.md`: env var table.
- Add `COMPANY_ENRICH_*` env vars.
- Runner tests.

### Pass 3D — UI

- `lib/api/types.ts`: `CompanyEnrichResult`.
- `app/leads/[id]/page.tsx`: Company tab/panel.
- `components/leads/CreateLeadFlow.tsx`: module selection gating.
- `app/leads/page.tsx` and `app/audit/page.tsx`: filter options.
- UI typecheck, lint, build.

---

## 12. Go / no-go checklist

### Must be true before Pass 3B

- [ ] This plan is approved by a human reviewer.
- [ ] No `local-enrichment-tool` code is adopted (rejected in evaluation).
- [ ] `discolike-cli` is used only as an optional adapter design reference, not as a hard dependency.
- [ ] `local` provider stays no-key and deterministic; no LinkedIn scraping or LLM synthesis.
- [ ] I/O contract and audit shape match existing module conventions.
- [ ] `permission_ref` is mandatory.
- [ ] Test strategy reviewed (fixtures, `testing.Short()` gates, no paid API keys in CI).

### Stop conditions

Do not start Pass 3B if:

- The human reviewer has not approved this plan.
- A paid API is proposed as the default or only provider.
- LinkedIn scraping, Crunchbase scraping, or LLM synthesis of facts is added back in.
- Module code would be written in this pass.

---

## 13. Sources

- `modules/extraction/extraction.go` — module pattern, `Input`/`Result`/`AuditRecord` shapes, `LegalBasis` constant.
- `services/control-plane/internal/models/models.go` — `Lead`, `RawLead`, `AuditEvent`, `Subject`, `ModuleInfo` shapes.
- `services/control-plane/internal/registry/registry.go` — current `company-enrich` registry entry.
- `services/control-plane/internal/runner/runner.go` — default stub for unwired modules, extraction wiring pattern.
- `docs/decisions/stage-1-decision.md` — original `company-enrich` decision requiring second-opinion evaluation of `local-enrichment-tool`.
- `evaluations/waterfall-gtm.md` — waterfall pattern reference.
- `evaluations/local-enrichment-tool.md` — second-opinion evaluation rejecting the tool.
- `evaluations/discolike-cli.md` — second-opinion evaluation recommending optional adapter use only.
- `docs/compliance.md` — LinkedIn scraping out of scope, GDPR Art. 6 logging, data minimization.
