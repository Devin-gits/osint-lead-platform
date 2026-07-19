# LinkedIn Enrichment Module — Stage 2 Planning Brief

**Author:** Devin (planning pass only)
**Date:** 2026-07-19
**Status:** Pass 1 — planning brief. No implementation. No dependencies installed.
**Canonical repo:** `https://github.com/Devin-gits/osint-lead-platform`

---

## Discovery summary

The following was verified from the actual repository and upstream sources before writing this plan.

### Existing compliance and architecture context

- `docs/compliance.md` classifies **LinkedIn scraping as High risk** and explicitly excludes it from production: "Do not deploy scrapers against platforms whose ToS explicitly prohibits automated access at the scale we'd need (this currently rules out LinkedIn profile scraping as a production data source — reference implementations may be studied, not deployed)."
- `.windsurf/workflows/compliance-check.md` lists "LinkedIn profile scraping" as a hard rejection in the default path.
- `docs/research/osint-tooling-research.md` notes: "LinkedIn scrapers (several in Section 6's neighborhood) violate LinkedIn's ToS outright and carry real legal precedent (hiQ v. LinkedIn and later rulings cut both ways) — treat as 'reference only, do not deploy' unless you get explicit written legal sign-off."
- `docs/architecture.md` places enrichment in the `Enrichment` stage, alongside `domain-intel`, `company-enrich`, and `extraction`. It does not currently list `linkedin-enrich`.
- Existing modules (`email-validate`, `domain-intel`, `social-footprint`, `extraction`) follow a common contract: partial lead JSON on stdin, namespaced key added to stdout, audit line on stderr, graceful degradation, exit 0 for operational outcomes.

### Existing extraction module (`modules/extraction/`)

A Stage 2 implementation already exists at `modules/extraction/`. It extracts low-risk business contact fields from an **approved public web page URL**, with strict SSRF controls, bounded raw content, and provenance tracking. Its social-link extraction may capture a LinkedIn URL if it appears on the page, but it does not perform LinkedIn-specific enrichment, search, or profile scraping.

### Existing social-footprint module (`modules/social-footprint/`)

Handles `Maigret` and `Sherlock` with a curated platform allow-list. LinkedIn is **not** in the curated list. The module does profile-scraping or username discovery beyond a `claimed/available/unknown` status on public platforms.

### Control-plane registry gap

`services/control-plane/internal/registry/registry.go` does not register `linkedin-enrich` as a module. `services/control-plane/internal/models/models.go` does not define a `ModuleLinkedInEnrich` constant. Any future implementation would require these additions, but **this planning PR does not change them**.

### Product requirement reconciliation

The request is to plan a module that can, in the future, help obtain a LinkedIn profile link for a person given an email or username. This capability is **not achievable safely or legally through public scraping**, because:

1. LinkedIn does **not** expose a public API for reverse email-to-profile lookup. The Marketing API, Compliance API, and Recruiting API have strict use-case and partner restrictions and generally prohibit this pattern for lead enrichment.
2. Public HTML scraping for profile discovery by email or username violates LinkedIn's ToS and carries documented legal precedent (hiQ Labs v. LinkedIn and follow-on cases).
3. Third-party scrapers (`linkedin_scraper`, `linkedin2username`, `LinkedInt`) all require authentication, bypass mechanisms, or enumeration against LinkedIn's non-public surfaces.

Therefore, this plan documents the product requirement **only through defensible access methods**. The table below maps the user's "find by email or username" requirement to versioned, gated capabilities:

| Version | Input | Capability | Access method | Gate |
|---|---|---|---|---|
| v1 | `url` — exact public LinkedIn profile/company URL supplied by customer | Extract allowed public fields from the supplied URL | Option B (public page, exact URL) — if legally approved | ToS/legal review |
| v1.1 / v2 | `linkedin_handle` — exact username supplied by customer | Construct canonical URL (`https://www.linkedin.com/in/{handle}/`); optionally validate via public page or official API | Option B or Option A | ToS/legal review for validation; no search/discovery |
| v2 | `email` | Resolve email to LinkedIn profile URL | **Only** via approved B2B data partner or official LinkedIn API with documented consent/legal basis | Data-partner ADR, DP/LIA, customer contract |

- **Exact public profile URL supplied by the customer** is the v1 boundary.
- **Exact LinkedIn username/handle supplied by the customer** is a future, lower-risk addition because it requires no search or inference — only canonical URL construction.
- **Email-based reverse lookup** is **not available from LinkedIn's public surfaces** and therefore requires a partner/API with explicit legal authorization.

Any other path (scraping, enumeration, credential automation, search bots) is rejected in this plan.

---

## 1. Decision and narrow v1 scope

### Tool decisions

| Repository | License (verified upstream) | Maintenance/activity | Needs LinkedIn credentials | Supports enumeration / bypass / proxies | Decision |
|---|---|---|---|---|---|
| `https://github.com/joeyism/linkedin_scraper` | README badge claims Apache-2.0, but repository `LICENSE` file is **GPL-3.0** (conflict). | Last push 2026-04-10; not archived, but built on Playwright auth sessions. | **Yes** — manual login or `LINKEDIN_EMAIL`/`LINKEDIN_PASSWORD`. | Scrapes full person/company/job/post profiles; supports session replay and `headless=False` browser automation. | **Reference only; not adopted.** License conflict, ToS violation, and overbroad data collection make it unsuitable for v1. |
| `https://github.com/initstring/linkedin2username` | MIT | Last push 2026-05-20; not archived. | **Yes** — LinkedIn username/password. | Generates username lists from company pages, employee enumeration, `--geoblast` to bypass 1,000-record search limit, `--proxy`, `--keywords` pivoting. | **Rejected.** Employee enumeration, username guessing, authenticated scraping, and proxy support are out of scope. |
| `https://github.com/vysecurity/LinkedInt` | **No `LICENSE` file found** in repository; archived. | Archived since 2023-03-06. | **Yes** — LinkedIn credentials + Hunter.io API key. | "LinkedIn Recon Tool"; search by keywords/company, email prefix generation, HTML report generation. | **Rejected.** Archived reconnaissance tool, credential/API-key workflow, no explicit license, out of scope. |

### v1 LinkedIn enrichment module goal

> One customer-supplied, exact LinkedIn public profile or company URL in; one bounded, provenance-preserving, permission-audited enrichment result out.

### Explicit v1 allowed input

```json
{
  "url": "https://www.linkedin.com/in/example-person/",
  "permission_ref": "CLIENT-2026-001",
  "purpose": "sales-lead-verification",
  "entity_type": "person",
  "allowed_fields": [
    "public_name",
    "headline",
    "current_company",
    "public_location",
    "public_profile_url"
  ],
  "customer_scope_ref": "ACME-CORP-2026-Q3"
}
```

| Field | Required | Description |
|---|---|---|
| `url` | **yes** | Exact canonical LinkedIn profile or company URL supplied by the customer. No discovery, search, or derivation. |
| `permission_ref` | **yes** | Privacy-safe reference tying this enrichment to an approved customer/data-processing basis. |
| `purpose` | **yes** | Approved enum. v1: `sales-lead-verification` only. |
| `entity_type` | **yes** | `person` or `company`. |
| `allowed_fields` | **yes** | Subset of safe, publicly displayed fields the customer is permitted to retrieve. |
| `customer_scope_ref` | **yes** | Customer or account identifier for rate-limiting and audit. |
| `retention_class` | no | Optional retention label (e.g., `short`, `standard`). Default: module policy. |

The URL must be an **exact public LinkedIn URL** (`https://www.linkedin.com/in/{vanity-name}/` or `https://www.linkedin.com/company/{company-name}/`). No shorteners, redirects, search URLs, or non-LinkedIn hosts are accepted.

### v1 permitted output

```json
{
  "url": "https://www.linkedin.com/in/example-person/",
  "permission_ref": "CLIENT-2026-001",
  "linkedin_enrich": {
    "status": "ok",
    "source_url": "https://www.linkedin.com/in/example-person/",
    "canonical_url": "https://www.linkedin.com/in/example-person/",
    "source_access_mode": "approved_public_page",
    "entity_type": "person",

    "fields": {
      "public_name": {
        "value": "Jane Example",
        "classification": "observed_public_data",
        "verified": false
      },
      "headline": {
        "value": "Senior Engineer at Example Corp",
        "classification": "observed_public_data",
        "verified": false
      },
      "current_company": {
        "value": "Example Corp",
        "classification": "observed_public_data",
        "verified": false
      },
      "public_location": {
        "value": "London, United Kingdom",
        "classification": "observed_public_data",
        "verified": false
      },
      "public_profile_url": {
        "value": "https://www.linkedin.com/in/example-person/",
        "classification": "observed_public_data",
        "verified": false
      }
    },

    "provenance": [
      {
        "field": "public_name",
        "source_url": "https://www.linkedin.com/in/example-person/",
        "extraction_method": "approved_public_page",
        "observed_at": "2026-07-19T14:00:00Z",
        "classification": "observed_public_data",
        "not_verified": true
      }
    ],

    "metadata": {
      "backend": "approved_public_page",
      "legal_basis": "GDPR Art.6(1)(f) legitimate-interest",
      "permission_ref": "CLIENT-2026-001",
      "purpose": "sales-lead-verification",
      "customer_scope_ref": "ACME-CORP-2026-Q3",
      "http_status": 200,
      "truncated": false,
      "raw_bytes": 0,
      "duration_ms": 2340,
      "limits_applied": "max_body=2MB, max_markdown=0, timeout=30s, max_redirects=5"
    },

    "warnings": [],
    "error": "",
    "checked_at": "2026-07-19T14:00:00Z"
  }
}
```

### Field semantics

- Every value is labelled `observed_public_data` and `not_verified` (i.e., "we saw it on the page, we did not independently verify identity").
- No inferred or derived data.
- No raw HTML stored by default.
- No screenshots by default.
- No connection/follower/activity/post data.

### v1 prohibited collection

Explicitly prohibited in the default path:

- LinkedIn people/company search as an input method.
- Employee enumeration.
- Username prediction or candidate generation.
- Email-address guessing or reverse email lookup.
- Contact scraping beyond the supplied exact URL.
- Connections, followers, groups, endorsements, recommendations.
- Activity feeds, posts, reactions, comments.
- Education history, work-history expansion, interests, skills, or sensitive characteristics.
- Downloading resumes or documents.
- Screenshots by default.
- Direct messages, invitations, follows, likes, or any write action.
- Authentication bypass.
- CAPTCHA bypass.
- Proxy rotation.
- Anti-bot evasion.
- Session replay.
- Browser fingerprint-evasion.
- Bulk profile crawling.
- URL discovery through Google/Bing/LinkedIn search.
- Use of `linkedin2username`, LinkedInt, Photon, Crawlab, or similar reconnaissance/enumeration tools.

### Future v2 discovery scope (gated)

See the versioned capability table in the Discovery summary. In short:

- `email` → LinkedIn URL is **only** supported through an approved B2B data partner or official LinkedIn API with a documented legal basis.
- `linkedin_handle` → canonical URL construction is a future low-risk addition, but any validation against public LinkedIn pages still requires a ToS/legal review.

These are **not v1** and are documented only to align the product roadmap with the compliance posture.

---

## 2. Important security explanations

### Credentialed URLs are forbidden

The module must reject URLs such as:

```text
https://username:password@www.linkedin.com/in/example
```

Reasons:

- Credentials embedded in URLs leak through logs, shell history, process lists (`ps`), error messages, browser telemetry, proxy access logs, and audit trails.
- A credential in a URL does not prove account ownership or authorized automated use of LinkedIn.
- Future authenticated access, if ever approved, must use a dedicated secret-management design (e.g., environment variables, secret manager, short-lived tokens) with explicit customer consent and never URL credentials.
- The module must parse and reject any URL containing `userinfo` before any network request is made.

### Private / link-local / metadata IPs remain forbidden

This rule is SSRF prevention, not a limitation on legitimate public-site access.

The module must reject:

- `localhost`, `127.0.0.0/8`, `::1`
- RFC1918 private ranges: `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`
- Link-local: `169.254.0.0/16`, `fe80::/10`
- Carrier-grade NAT: `100.64.0.0/10`
- Unique-local IPv6: `fc00::/7`
- Multicast: `224.0.0.0/4`, `ff00::/8`
- Unspecified: `0.0.0.0`, `::`
- Cloud metadata endpoints: `169.254.169.254` and known AWS/GCP/Azure metadata hosts

Public LinkedIn URLs (`www.linkedin.com`, `linkedin.com`, `*.linkedin.com`) remain valid external targets. Redirects and DNS resolutions must be revalidated against the above list at every hop. This protects internal services, cloud credentials, databases, admin endpoints, and developer machines.

---

## 3. Proposed module I/O contract

### Input

```go
// Input is the per-call LinkedIn enrichment request.
type Input struct {
    URL            string   `json:"url"`                       // exact LinkedIn profile/company URL
    PermissionRef  string   `json:"permission_ref"`            // mandatory privacy-safe approval reference
    Purpose        string   `json:"purpose"`                   // enum: "sales-lead-verification"
    EntityType     string   `json:"entity_type"`               // "person" or "company"
    AllowedFields  []string `json:"allowed_fields"`            // subset of safe fields
    CustomerScope  string   `json:"customer_scope_ref"`        // customer/account identifier
    RetentionClass string   `json:"retention_class,omitempty"` // optional retention label
}
```

### Output

```go
// FieldValue is one observed public field with provenance metadata.
type FieldValue struct {
    Value          string `json:"value"`
    Classification string `json:"classification"` // "observed_public_data"
    Verified       bool   `json:"verified"`       // always false for v1
}

// ProvenanceRecord links a field to its source.
type ProvenanceRecord struct {
    Field            string `json:"field"`
    SourceURL        string `json:"source_url"`
    ExtractionMethod string `json:"extraction_method"`
    ObservedAt       string `json:"observed_at"`
    Classification   string `json:"classification"`
    NotVerified      bool   `json:"not_verified"`
}

// Metadata carries non-PII runtime and policy context.
type Metadata struct {
    Backend       string `json:"backend"`                      // "approved_public_page" or "official_api"
    LegalBasis    string `json:"legal_basis"`
    PermissionRef string `json:"permission_ref"`
    Purpose       string `json:"purpose"`
    CustomerScope string `json:"customer_scope_ref"`
    HTTPStatus    int    `json:"http_status,omitempty"`
    Truncated     bool   `json:"truncated,omitempty"`
    RawBytes      int    `json:"raw_bytes,omitempty"`
    DurationMs    int    `json:"duration_ms,omitempty"`
    LimitsApplied string `json:"limits_applied"`
}

// Result is the value stored under the lead's "linkedin_enrich" key.
type Result struct {
    Status           string                `json:"status"` // "ok" | "partial" | "skipped" | "error"
    SourceURL        string                `json:"source_url"`
    CanonicalURL     string                `json:"canonical_url"`
    SourceAccessMode string                `json:"source_access_mode"` // "approved_public_page" | "official_api"
    EntityType       string                `json:"entity_type"`
    Fields           map[string]FieldValue `json:"fields"`
    Provenance       []ProvenanceRecord    `json:"provenance"`
    Metadata         Metadata              `json:"metadata"`
    Warnings         []string              `json:"warnings,omitempty"`
    Error            string                `json:"error,omitempty"`
    CheckedAt        string                `json:"checked_at"`
}
```

### Audit (stderr)

One JSON line per invocation:

```json
{
  "module": "linkedin-enrich",
  "tool": "approved_public_page_extractor",
  "tool_version": "v0.0.0",
  "timestamp": "2026-07-19T14:00:00Z",
  "legal_basis": "GDPR Art.6(1)(f) legitimate interest",
  "permission_ref": "CLIENT-2026-001",
  "purpose": "sales-lead-verification",
  "customer_scope_ref": "ACME-CORP-2026-Q3",
  "source_url": "https://www.linkedin.com/in/example-person/",
  "entity_type": "person",
  "access_mode": "approved_public_page",
  "status": "ok",
  "duration_ms": 2340
}
```

Audit must **not** include:

- Profile content (names, headlines, locations, company names)
- Email addresses
- Phone numbers
- Access tokens or credentials
- Cookies or session data
- Screenshots or raw HTML
- Full unsanitized query strings

### Exit codes

- `0` — a well-formed lead was read and a `linkedin_enrich` record was emitted, including soft operational failures (`skipped`/`error`).
- `1` — unreadable/invalid input JSON, missing required fields, or rejected URL.

---

## 4. Access-method decision framework

### Option A — Official LinkedIn API / approved partner source

**Preferred path, if feasible.**

Requirements to document before any future implementation:

- Customer authorization and scope (which fields, for what purpose, for how long).
- LinkedIn partner program or approved B2B data provider contract.
- Data-minimisation defaults (only request fields in `allowed_fields`).
- Token storage via secret manager; no tokens in code, env, logs, or CLI args.
- Token revocation and rotation procedures.
- Audit logging of every API call.
- Retention and deletion policy aligned with the provider's terms and GDPR.
- API-version lifecycle plan (deprecation handling).
- Rate-limit compliance per customer and per API key.

This is the **only path** that can support email-to-LinkedIn-profile lookup in the future, and only if the partner/source contractually permits it and the legal basis is documented.

### Option B — Public-page, exact-URL extraction

**Possible future fallback only after explicit ToS/legal approval.**

Constraints:

- Input is an exact LinkedIn public profile or company URL supplied by the customer.
- No search, discovery, bulk crawl, account login, bypass, or evasion.
- HTTPS only; strict hostname allow-list; SSRF controls identical to `modules/extraction`.
- Field allow-list enforced; only visibly displayed fields collected.
- Raw HTML not stored by default; `raw_markdown` size is `0` or bounded and deleted immediately after parsing.
- Every value labelled `observed_public_data` and `not_verified`.

This is the **only scraping-adjacent path** the plan allows, and it is still subject to legal/ToS review before implementation.

### Option C — Authenticated browser/session automation

**Explicitly deferred.**

No implementation now. Requires a separate future ADR and legal/ToS approval before any proof-of-concept. If ever approved, it must:

- Use customer-owned account credentials stored through a secret manager.
- Never use URL credentials.
- Define account ownership, scope, retention, audit, rate limits, kill switch, deletion, and explicit customer approval.
- Be disabled by default and behind a feature flag.

**Recommendation:**

```text
No LinkedIn access implementation in the current stage.
Future implementation may begin only with Option A or a formally approved Option B.
Option C requires a separate approval gate.
```

---

## 5. Licensing and supply-chain gate

| Component | Proposed role | License (verified) | Decision | Verification required before implementation |
|---|---|---|---|---|
| `modules/linkedin-enrich` Go code | Core orchestrator, CLI | MIT (own code) | **Planned** | N/A |
| `joeyism/linkedin_scraper` | Evaluated candidate | README badge: Apache-2.0; `LICENSE` file: **GPL-3.0** (conflict) | **Rejected / reference only** | License conflict already disqualifies adoption. ToS violation and auth scraping make it unsuitable regardless of license. |
| `initstring/linkedin2username` | Evaluated candidate | MIT | **Rejected** | Employee enumeration, credential login, proxy support, search-limit bypass are out of scope. |
| `vysecurity/LinkedInt` | Evaluated candidate | No `LICENSE` file found; archived | **Rejected** | No explicit license, archived, reconnaissance tool. |
| Official LinkedIn API or B2B data partner | Future Option A | Commercial/partner terms | **Deferred** | Contract, data-processing agreement, and legal review required. |
| `net/url`, `net/http`, `net` (Go stdlib) | URL parsing, HTTP | BSD-style (Go) | **Planned** | N/A |
| `encoding/json` (Go stdlib) | JSON I/O | BSD-style (Go) | **Planned** | N/A |

### Rules

- No GPL component may be imported, copied, vendored, or linked into the MIT Go core.
- No scraping tool may be adopted as a dependency or subprocess.
- No dependency is adopted merely because it is popular.
- Each future dependency needs a license, maintenance, security, and scope review.

---

## 6. Compliance, data minimisation, and retention

### Legal basis and purpose limitation

- **Purpose:** sales-lead verification only.
- **Legal basis:** `GDPR Art.6(1)(f) legitimate interest` (or another documented basis where required).
- **Mandatory `permission_ref`:** every request must carry an approved reference.
- **Target-ToS awareness:** the operator is responsible for ensuring they have permission/contractual right to process the target URL. The module logs `permission_ref` and `customer_scope_ref` for traceability.

### Data minimisation

- Per-request `allowed_fields` subset.
- No collection beyond what is visibly displayed and allowed.
- No inference, scoring, or expansion from observed data.
- Every value labelled `observed_public_data` and `not_verified`.

### Retention defaults (proposed)

| Data | Default retention | Notes |
|---|---|---|
| Raw HTML / page content | **Do not store** | Parsed and discarded immediately. |
| Screenshots | **Do not create/store by default** | Future opt-in requires separate approval. |
| Observed `fields` | Configurable, default 90 days or lead lifecycle | Aligned with customer contract and lead status. |
| Provenance records | Same as observed fields | Embedded in result or stored separately. |
| Audit events | 1 year or longer per compliance policy | No profile content, no PII. |

### Deletion and re-run

- Deleting a lead must delete its `linkedin_enrich` result and provenance.
- Re-run must overwrite previous result and provenance.
- Audit logs are immutable and retained per policy.

### Sensitive traits

- No scoring based on sensitive or inferred traits (gender, ethnicity, religion, political opinion, health, etc.).
- No use for automated decision-making that produces legal or similarly significant effects.

### Human review

- Low-confidence or ambiguous results must be flagged for human review.
- The module must not auto-accept observed public data as verified truth.

---

## 7. Security design requirements

### URL and hostname controls

| Control | Requirement |
|---|---|
| Scheme | `https` only. Reject `http`, `ftp`, `file`, `data`, `javascript`, etc. |
| Hostname allow-list | `www.linkedin.com`, `linkedin.com`, `*.linkedin.com` only (future: `*.licdn.com` for CDN assets if needed). |
| Sub-path allow-list | `/in/{vanity}` for person, `/company/{name}` for company. Reject search, feed, messaging, jobs, etc. |
| Userinfo | Reject any URL containing `user:password@`. |
| Query strings | Strip or redact. Do not forward unknown params. |
| Fragments | Strip. |
| Private IPs | Reject as defined in §2. |
| Redirect validation | Validate every hop against hostname allow-list and IP deny-list. Max 5 redirects. |
| DNS rebinding | Revalidate resolved IP at connection time. |

### Request boundaries

| Boundary | Limit |
|---|---|
| URLs per invocation | 1 |
| Max request duration | 30s (configurable) |
| Max response body | 2 MB |
| Raw content stored | 0 by default |
| Concurrent requests per process | 1 |
| Rate limit per customer | configurable, default 1 request per 10s |

### Secret handling

- No tokens, credentials, or session data in CLI args, env, logs, results, or audit.
- Future official API tokens stored in secret manager and injected at runtime.
- Kill switch / feature flag for any future access method.

### Content handling

- Content-type allow-list: `text/html`.
- No JavaScript execution in v1.
- No browser automation in v1.
- No CAPTCHA handling.
- No proxy rotation.
- No anti-bot evasion.

---

## 8. Test strategy proposal

Tests designed only — no tests written in this planning pass.

### Unit / fixture tests

| Test | Purpose |
|---|---|
| URL scheme validation | Accept `https://`; reject `http://`, `ftp://`, `file://`, `data:`, `javascript:` |
| Hostname allow-list | Accept `www.linkedin.com/in/example`; reject `evil-linkedin.com`, `linkedin.com.evil.com` |
| Userinfo rejection | Reject `https://user:pass@www.linkedin.com/in/example` |
| Path validation | Accept `/in/{vanity}` and `/company/{name}`; reject `/search`, `/feed`, `/messaging` |
| Private IP rejection | Reject localhost, 127.x, 10.x, 192.168.x, 172.16-31.x, etc. |
| Cloud metadata rejection | Reject `169.254.169.254` and known metadata endpoints |
| Redirect validation | Reject redirects to non-LinkedIn or private hosts |
| Query-string stripping | Verify query strings are not forwarded or logged |
| Field allow-list enforcement | Only return fields in `allowed_fields` |
| Provenance preservation | Verify every field has source URL, method, classification, `not_verified` |
| Audit PII redaction | Verify no names, emails, phones, HTML, tokens in stderr |
| Missing `permission_ref` | Reject request with empty `permission_ref` |
| Missing `purpose` / `entity_type` | Reject request with unsupported values |

### Integration tests

| Test | Purpose |
|---|---|
| Controlled local fixture server | Return a synthetic LinkedIn-like HTML page. Verify extraction of allowed fields only. |
| Disallowed private target | Attempt request against `https://127.0.0.1:8080/`. Verify rejection. |
| No live LinkedIn calls in CI | All integration tests use local fixture servers only. |

Network-dependent tests must be guarded:

```go
if testing.Short() {
    t.Skip("skipping network-dependent test in short mode")
}
```

CI must run full `go test ./...` (not `-short`).

### Security tests

| Test | Purpose |
|---|---|
| Redirect to private IP | Fixture redirects to `127.0.0.1` — reject. |
| DNS rebinding | Mock resolver returns public then private IP — reject at connection time. |
| Oversized response | Fixture returns 10MB — reject. |
| Credentialed URL | `https://user:pass@...` — reject before network. |
| Non-LinkedIn hostname | `https://phishing-linkedin.example/in/foo` — reject. |

---

## 9. Future implementation PR boundary

### Smallest viable implementation PR

The future Pass 2 implementation PR should be scoped to:

```
modules/linkedin-enrich/**
docs/decisions/linkedin-enrich-stage-2-plan.md (this file — may be updated)
```

Allowed external changes:
- `services/control-plane/internal/models/models.go` — add `ModuleLinkedInEnrich` constant.
- `services/control-plane/internal/registry/registry.go` — register module metadata (if needed for a local smoke test).

No other files may change.

### Future PR must include

- [ ] URL hostname/path allow-list
- [ ] `permission_ref`, `purpose`, `entity_type`, `allowed_fields`, `customer_scope_ref` validation
- [ ] Field allow-list enforcement
- [ ] Provenance tracking for every field
- [ ] Audit record with no PII/profile content
- [ ] Private-IP/redirect/credentialed-URL rejection
- [ ] Query-string stripping/redaction
- [ ] Unit tests for all validation
- [ ] Fixture-based integration tests (local server)
- [ ] Updated README documenting the narrow scope
- [ ] Compliance table update in `docs/compliance.md` (if the module is approved)

### Explicitly deferred from the implementation PR

- Email-based profile discovery (requires Option A partner/API gate)
- Username/handle discovery via search (requires Option B legal/ToS gate)
- Authenticated browser automation (Option C gate)
- LinkedIn search, employee enumeration, username generation
- `linkedin_scraper`, `linkedin2username`, `LinkedInt` integration
- Raw HTML storage
- Screenshots
- Control-plane execution wiring beyond minimal registry entry
- UI changes
- Proxy/anti-bot/CAPTCHA handling

---

## 10. Go / no-go gate

### Must be true before Pass 2

- [ ] Current LinkedIn ToS/legal review completed and documented
- [ ] Official API or approved B2B data partner feasibility assessed (for any email/username feature)
- [ ] Data Protection / legitimate-interest assessment approved
- [ ] Exact field allow-list approved
- [ ] Permission and customer scope model approved
- [ ] Retention and deletion policy approved
- [ ] SSRF and URL controls shared/reused from `modules/extraction`
- [ ] Audit schema approved
- [ ] No repository/tool with enumeration, guessing, proxy rotation, login bypass, or surveillance behavior is proposed
- [ ] License and supply-chain review completed for any adopted dependency
- [ ] No code, credentials, browser automation, or network scraping has been added in this planning PR

### Stop conditions

Do not start Pass 2 if:

- ToS/legal review is incomplete.
- A scraping, enumeration, or credential-based tool is proposed.
- Raw page-content retention is unresolved.
- The module would perform LinkedIn search/discovery without a separate approved Option A/B/C gate.
- The repository identity is unclear.
