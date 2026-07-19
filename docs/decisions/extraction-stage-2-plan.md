# Extraction Module — Stage 2 Planning Brief

**Author:** Devin (planning pass only)
**Date:** 2026-07-19
**Status:** Pass 1 — planning brief. No implementation. No dependencies installed.
**Canonical repo:** `https://github.com/Devin-gits/osint-lead-platform`

---

## Discovery summary

The following was verified from the actual repository before writing this plan.

### Existing extraction module (`modules/extraction/`)

A Stage 2 implementation already exists at `modules/extraction/`. Files present:

```
modules/extraction/
  extraction.go          — Extractor type, Input/Fields/Result/AuditRecord structs, Extract() method
  extraction_test.go     — unit tests: default backend, env override, missing URL, confidence
  crawl4ai.go            — crawl4aiRunner: subprocess invocation of wrapper/crawl4ai_extract.py
  crawl4ai_test.go       — fake wrapper tests: explicit wrapper, missing python, timeout
  firecrawl.go           — firecrawlRunner: HTTP adapter to POST /v1/scrape, field extraction
  firecrawl_test.go      — httptest server tests: missing key, success, API error
  ratelimit.go           — in-process rate limiter with exponential backoff
  cmd/extraction/main.go — CLI: stdin JSON in, augmented JSON out, audit on stderr
  cmd/extraction/main_test.go — end-to-end CLI test with fake wrapper
  wrapper/crawl4ai_extract.py — Python wrapper: AsyncWebCrawler, LLM-free extraction
  go.mod                 — github.com/Moyeil-73/osint-lead-platform/modules/extraction, go 1.22.5
  requirements.txt       — crawl4ai==0.9.2
  Makefile               — build, test, test-short, pydeps, vet, clean
  README.md              — full I/O contract, config, backends, guardrails, license table
```

**Current I/O contract** (from README and source):
- Input: `{"url":"..."}` on stdin (or `--url` flag). Optional: email, name, company, domain, schema.
- Output: same record + `extraction` key with status, url, final_url, source_tool, confidence, fields (company_name, emails, phones, addresses, social_links, contact_urls, description, title), raw_markdown (bounded 100KB), metadata, error, checked_at.
- Audit (stderr): one JSON line per run: tool, url, checked_at, status, legal_basis. Subject is URL only.
- Exit 0 for all operational outcomes; exit 1 only for unreadable input.
- Statuses: ok, partial, skipped, error.

**Current backends**:
- Crawl4AI (default): Go calls `wrapper/crawl4ai_extract.py` as subprocess. Apache-2.0 + attribution. Single page, LLM-free, rate-limited.
- Firecrawl (optional): Go HTTP adapter to hosted API. Requires `FIRECRAWL_API_KEY`. AGPL / hosted service.

**Current test coverage**: unit tests for Extractor, crawl4ai runner (fake wrapper), firecrawl runner (httptest), CLI e2e, rate limiter. Live tests gated by `EXTRACTION_LIVE=1`.

### Control-plane integration state

- `models.go`: `ModuleExtraction = "extraction"` defined.
- `registry.go`: extraction is now registered with `DevStatus: "available"`, description reflecting URL-based page extraction, and `MinInputField: "url"`.
- `runner.go`: extraction is wired via an in-process `extraction.Extractor` and produces real `ok`/`partial`/`skipped`/`error` results; `computeStage` maps extraction (with status "ok") to `StageEnriched`.

**Status update**: The registry has been updated to match the actual module contract (`url` input, page extraction). The gap above is closed.

### Existing module conventions (from email-validate, domain-intel)

1. **Go Validator/Analyzer type**: constructed with `NewValidator(timeout)` / `NewAnalyzer(timeout)`, reusable across calls.
2. **CLI under `cmd/<name>/`**: reads one JSON lead on stdin, writes augmented record on stdout, audit line(s) on stderr. Exit 0 for all operational outcomes.
3. **Namespaced key**: result stored under e.g. `email_validate`, `domain_intel`, `extraction`.
4. **Graceful degradation**: all failures produce structured result with status "unknown"/"error"/"skipped", never crash the pipeline.
5. **Audit on stderr**: one JSON line per tool per call, always emitted, carrying tool, checked_at, subject, status, legal_basis.
6. **Legal basis**: `GDPR Art.6(1)(f) legitimate-interest` on every audit record.
7. **Tests**: unit tests (no network), integration tests (live network, gated by `-short` or env vars), CLI e2e test.
8. **Subprocess boundary for non-MIT tools**: theHarvester (GPL-2.0) invoked as subprocess only, never imported. Same pattern used for Crawl4AI (Apache-2.0) via Python wrapper.

### Documentation constraints

- `docs/compliance.md`: "No non-consensual personal surveillance", "Respect third-party ToS", rate-limit breach-checking, log legal basis, define retention.
- `docs/decisions/stage-1-decision.md` extraction section: Crawl4AI primary (self-hosted, Apache-2.0), Firecrawl secondary (hosted, AGPL tension with EU data residency), do not make Firecrawl permanent without self-host fallback.
- `docs/architecture.md`: module contract (partial record in, namespaced key out, graceful degradation, audit).

---

## 1. Decision and narrow v1 scope

### Tool decisions

- **Crawl4AI** is the self-hosted primary candidate (Apache-2.0 + attribution clause). Already implemented as `wrapper/crawl4ai_extract.py` invoked via Go subprocess.
- **Firecrawl** is deferred as a future optional adapter. Already implemented as a Go HTTP client in `firecrawl.go`, gated behind `EXTRACTION_BACKEND=firecrawl` + `FIRECRAWL_API_KEY`. Not a default or permanent dependency. Any future adoption requires a documented self-host fallback and explicit legal/commercial review of AGPL-3.0 and data-residency implications.
- **Photon** (`s0md3v/Photon`) is reference-only and explicitly deferred. GPL-3.0. Must not be imported, copied, vendored, embedded, or called by the default pipeline. Its broad OSINT extraction behavior (email harvesting, account discovery, endpoint enumeration) exceeds the narrow v1 scope. Recorded here only as a rejected/deferred reference for future human review.
- **Crawlab** (`crawlab-team/crawlab`) is reference-only and explicitly deferred. It is a crawler-management/orchestration platform, not an extraction engine. Custom orchestration is explicitly out of scope until real module evidence justifies it. Recorded here only as a future operational reference after several proven modules exist.

### v1 extraction module goal

> One permissioned public HTTP(S) URL in, one bounded and provenance-preserving structured page extraction result out.

### Explicit non-goals

- Recursive internet reconnaissance
- General-purpose website discovery
- Subdomain discovery
- Bulk crawling
- Authenticated crawling
- Form submission
- Session/cookie replay
- CAPTCHA bypass
- Private or login-gated content
- LinkedIn scraping
- Reverse-image/deep-account discovery
- Breach/leak data
- AI inference of unknown personal or company data
- Orchestration, queues, workers, UI, or control-plane wiring beyond a future clearly bounded integration point
- Proxy rotation or anti-bot evasion
- Browser fingerprinting work

---

## 2. Proposed module I/O contract

The existing implementation already defines a contract. This section proposes enhancements for the planning review, adding `permission_ref`, `source_id`, `mode`, and provenance fields that the current implementation lacks.

### Input

```json
{
  "url": "https://example.com/landing-page",
  "permission_ref": "CAMP-2026-Q3-001",
  "source_id": "ad-campaign-summer-2026",
  "mode": "page",
  "allowed_fields": ["email", "phone", "company_name", "domain"]
}
```

| Field | Required | Description |
|---|---|---|
| `url` | **yes** | The single HTTP(S) URL to extract from. Must be a URL the business has explicit permission to process. |
| `permission_ref` | **yes** | Privacy-safe reference tying this extraction to an approved target. Carried through to audit. |
| `source_id` | no | Campaign, ad, or landing-page reference for provenance. |
| `mode` | no | Extraction mode. Enum: `page` (default — extract from the given URL only) or `contact_page` (future: follow one same-origin /contact link). |
| `allowed_fields` | no | If set, only extract these field types. Default: all safe fields. |

Existing fields (`email`, `name`, `company`, `domain`) are preserved for pass-through context but are never required for the extraction itself.

### Output

The output adds an `extraction` key to the lead record:

```json
{
  "url": "https://example.com/landing-page",
  "permission_ref": "CAMP-2026-Q3-001",
  "extraction": {
    "status": "ok",
    "url": "https://example.com/landing-page",
    "final_url": "https://example.com/landing-page/",
    "source_tool": "unclecode/crawl4ai@v0.9.2 (CLI subprocess)",
    "tool_version": "crawl4ai==0.9.2",
    "confidence": 0.429,

    "fields": {
      "title": "Example Corp - Summer Campaign",
      "description": "Example Corp provides enterprise widgets.",
      "company_name": "Example Corp",
      "emails": ["sales@example.com"],
      "phones": ["+1-555-123-4567"],
      "addresses": [],
      "domain": "example.com",
      "social_links": ["https://linkedin.com/company/example"],
      "contact_urls": ["https://example.com/contact"]
    },

    "provenance": [
      {
        "field": "emails",
        "value": "sales@example.com",
        "source_url": "https://example.com/landing-page/",
        "extraction_method": "regex",
        "timestamp": "2026-07-19T14:00:00Z"
      },
      {
        "field": "company_name",
        "value": "Example Corp",
        "source_url": "https://example.com/landing-page/",
        "extraction_method": "html_title_heuristic",
        "timestamp": "2026-07-19T14:00:00Z"
      }
    ],

    "raw_markdown": "...page markdown truncated to 100 KB...",

    "metadata": {
      "backend": "crawl4ai",
      "legal_basis": "GDPR Art.6(1)(f) legitimate-interest",
      "permission_ref": "CAMP-2026-Q3-001",
      "http_status": 200,
      "truncated": false,
      "raw_bytes": 12345,
      "duration_ms": 2340,
      "limits_applied": "max_body=2MB, max_markdown=100KB, timeout=45s, max_redirects=5"
    },

    "warnings": [],
    "error": "",
    "checked_at": "2026-07-19T14:00:00Z"
  }
}
```

### Concept separation

1. **Raw retrieved content**: `raw_markdown` (bounded to 100 KB). Never logged to stderr. Subject to retention policy.
2. **Normalized candidate fields**: `fields` object. These are publicly displayed candidate data with provenance, not verified truth. Speculative, inferred, or LLM-generated values are explicitly prohibited.
3. **Provenance**: `provenance` array. Every candidate field should trace back to a source URL and extraction method.
4. **Audit metadata**: `metadata` object + stderr audit record. Legal basis, permission ref, duration, limits.

### Statuses

- `ok` — extraction succeeded and produced fields.
- `partial` — the page was fetched but few/no structured fields were found.
- `skipped` — missing URL, missing permission_ref, or operational skip (e.g. missing backend).
- `error` — crawl failure, timeout, SSRF rejection, missing binary, or unparseable output.

---

## 3. Security and target controls

### URL and SSRF controls

| Control | Requirement |
|---|---|
| Scheme | Permit only `http` and `https`. Reject all others (`ftp`, `file`, `data`, `javascript`, etc.). |
| Hostname | Reject empty, missing, or IP-literal-only hostnames unless explicitly approved. |
| Loopback | Reject `localhost`, `127.0.0.0/8`, `::1`. |
| Link-local | Reject `169.254.0.0/16`, `fe80::/10`. |
| Private ranges | Reject `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`. |
| Carrier-grade NAT | Reject `100.64.0.0/10`. |
| Unique-local IPv6 | Reject `fc00::/7`. |
| Multicast | Reject `224.0.0.0/4`, `ff00::/8`. |
| Unspecified | Reject `0.0.0.0`, `::`. |
| Cloud metadata | Reject `169.254.169.254`, AWS/GCP/Azure metadata endpoints. |
| DNS resolution | Resolve hostname and validate every resolved IP against the above. |
| DNS rebinding | Revalidate resolved IP at actual connection time (not just at resolution). |
| Redirect validation | Reject redirects to prohibited hosts/IPs. Every redirect hop must pass the same validation. |
| Max redirects | Maximum 5 redirects per extraction. |
| URL credentials | Reject URLs containing `user:password@host`. |
| Query-string redaction | Redact query strings in all log/audit output. Preserve only `?param=<redacted>` structure. |
| Non-default ports | Allow 80 and 443 only by default. Non-standard ports require explicit configuration. |

### Crawl boundaries

| Boundary | Limit |
|---|---|
| URLs per invocation | 1 (strictly single-page in v1) |
| Recursive crawl | Not permitted in v1 |
| Sitemap processing | Not permitted in v1 |
| Max request duration | 45s (configurable via `EXTRACTION_TIMEOUT`) |
| Max response body | 2 MB |
| Max extracted-text size | 100 KB (`MaxRawMarkdown`) |
| Max HTML/DOM processing | 5 MB (reject responses larger than this before parsing) |
| Max concurrent extractions | 1 per process (single-threaded rate limiter enforces this) |
| Content-type allow-list | `text/html`, `application/xhtml+xml`. Reject binary, PDF, image, etc. |
| File downloads | Default reject. No file downloads in v1. |
| JS-triggered navigation | Must not leave the approved URL boundary |
| Same-origin for `contact_page` mode | Future: strictly same-origin only |

### Browser/session controls

- No authenticated sessions
- No injected cookies containing credentials
- No form submissions
- No account creation
- No CAPTCHA handling or bypass
- No proxy rotation or anti-bot evasion
- No browser fingerprinting work
- No automated retries that bypass a website's access controls

### ToS / permission policy

| Aspect | Enforcement |
|---|---|
| `permission_ref` required | **Technical**: the Go `Extract()` method should reject requests with empty `permission_ref`. The current implementation does not enforce this — this is a gap to address. |
| `permission_ref` in audit | **Technical**: carried through to every audit record and result metadata. |
| Target ToS awareness | **Organisational policy**: the platform operator is responsible for ensuring they have permission to process each target URL. The module cannot technically verify ToS compliance for arbitrary websites. |
| ToS documentation | **Technical**: the README and this plan document the policy. The module logs the permission reference so an operator can trace every extraction to an approved target. |
| `robots.txt` | **Organisational policy**: the module should respect `robots.txt` by default. Crawl4AI's `AsyncWebCrawler` has configuration for this. The Go wrapper should set this default. |

---

## 4. Recommended architecture

### Options evaluated (not implemented)

1. **Go Validator/library plus Go CLI** — the existing pattern for email-validate, domain-intel. Would require rewriting Crawl4AI in Go or embedding a Go HTML/HTTP client. Loses Crawl4AI's browser-based rendering.

2. **Python Crawl4AI subprocess with JSON stdin/stdout** — the current implementation. Go orchestrator invokes `wrapper/crawl4ai_extract.py` as a subprocess, receives JSON on stdout. Clean license boundary. Already built and tested.

3. **Localhost Python worker service** — a long-running Python process with a narrow internal API. Better for high-throughput but adds operational surface (daemon to deploy, health-check). Premature for current volumes.

4. **Future Firecrawl adapter** — already implemented as a Go HTTP client in `firecrawl.go`. Behind the same interface, optional, not default.

### Recommendation: Option 2 (current) — Python subprocess

The existing architecture is the correct choice for v1:

| Component | Responsibility |
|---|---|
| **Go `Extractor` type** (`extraction.go`) | Orchestrates the extraction: validates input, selects backend, enforces rate limits, truncates raw content, computes confidence, emits audit record. Never performs network I/O directly for the crawl. |
| **Go CLI** (`cmd/extraction/main.go`) | Reads one lead as JSON on stdin, invokes `Extractor.Extract()`, writes augmented lead to stdout, audit to stderr. Exit 0 for all operational outcomes. |
| **Python wrapper** (`wrapper/crawl4ai_extract.py`) | The ONLY file that imports Crawl4AI. Accepts `--url` and `--timeout`. Fetches single page via `AsyncWebCrawler`. Extracts fields via LLM-free heuristics (regex, html.parser). Prints one JSON object on stdout. Exit 0 even on internal errors. |
| **Go Firecrawl adapter** (`firecrawl.go`) | Thin HTTP client to Firecrawl's `POST /v1/scrape`. Behind the same `runner` interface. Optional, not default. |

### Architecture details

| Aspect | Design |
|---|---|
| JSON schema/versioning | The `Result` struct in `extraction.go` defines the schema. Versioning is implicit via `source_tool` string. Future breaking changes would require a `schema_version` field. |
| Timeout ownership | Go `Extractor` owns the context deadline. Python wrapper receives `--timeout` as a page-timeout hint. If the Go context expires, the subprocess is killed via `exec.CommandContext`. |
| Cancellation propagation | Go context cancellation kills the subprocess. The Python wrapper does not need its own cancellation — process death is sufficient. |
| stdout | Reserved for machine JSON only. One JSON object per invocation. |
| stderr | Reserved for audit and diagnostic records only. One JSON line per invocation from the Go layer. Python wrapper's stderr is captured but not forwarded to the pipeline's audit path. |
| Exit codes | 0 = well-formed lead processed (including operational errors). 1 = unreadable input / no URL. The Go CLI maps these. |
| Malformed worker response | Go `parseCrawl4AIOutput` returns `Result{Status: "error"}` with a descriptive error. Never panics. |
| Process isolation | Python runs as a subprocess. No shared memory, no long-lived connection. Each extraction is a fresh process invocation. |
| Python version | 3.10+ (Crawl4AI requirement). Pinned in README and verified at runtime. |
| Dependency lockfile | `requirements.txt` pins `crawl4ai==0.9.2`. A future `requirements.lock` or `uv.lock` would provide transitive dependency pinning. |
| Container/venv | Current: system Python with `pip install -r requirements.txt`. Future: a `venv` or Docker container for isolation. Not required for v1 pre-production. |
| Resource limits | Rate limiter (`ratelimit.go`) enforces `EXTRACTION_MIN_INTERVAL` (default 2s) with exponential backoff on errors. Max concurrent = 1 per `Extractor` instance. |
| MIT compatibility | Go code is MIT. Crawl4AI (Apache-2.0) is invoked as a subprocess only — no source imported or vendored. The Go binary never contains Crawl4AI code. Attribution clause satisfied via README license table. |

### Why not Firecrawl, Photon, or Crawlab in v1

- **Firecrawl**: AGPL-3.0 license tension with EU data-residency posture. Hosted-only anti-bot features. No self-host fallback documented. Already implemented as an optional adapter — not removed, but not the default or primary path.
- **Photon**: GPL-3.0. Overbroad OSINT behavior (email harvesting, account discovery, endpoint enumeration). Exceeds the narrow v1 scope. Must not be imported, copied, or called.
- **Crawlab**: Not an extraction engine. It is a crawler-management/orchestration platform. Custom orchestration is out of scope until real module evidence justifies it.

---

## 5. Licensing and supply-chain gate

| Component | Proposed role | License | Current decision | Verification required before implementation |
|---|---|---|---|---|
| `modules/extraction` Go code | Core orchestrator, CLI | MIT | **Active** — already implemented | N/A (own code) |
| Crawl4AI (`crawl4ai==0.9.2`) | Primary extraction engine | Apache-2.0 + attribution clause | **Active** — subprocess only | Re-verify upstream license at implementation time. Verify attribution clause obligations. Audit transitive Python dependencies (playwright, httpx, beautifulsoup4, etc.) for GPL contamination. |
| Crawl4AI transitive Python deps | Runtime dependencies of crawl4ai | Various (unknown until audited) | **Pending audit** | Run `pip show crawl4ai` + `pip-licenses` or equivalent to inventory all transitive deps and their licenses before any production deployment. No GPL dep may be required. |
| Firecrawl (hosted API) | Optional secondary adapter | AGPL-3.0 (server) / hosted service | **Deferred** — implemented but not default | Requires documented self-host fallback and legal/commercial review before becoming a permanent dependency. |
| Photon (`s0md3v/Photon`) | Reference only | GPL-3.0 | **Rejected for v1** | Must not be imported, copied, vendored, embedded, or called. Future reconsideration requires separate license review, narrow scope, and explicit compliance approval. |
| Crawlab (`crawlab-team/crawlab`) | Reference only | BSD-3-Clause | **Rejected for v1** | Not an extraction dependency. Orchestration is out of scope. |
| `net/url`, `net/http`, `net`, `os/exec` (Go stdlib) | URL parsing, HTTP, DNS, subprocess | BSD-style (Go) | **Active** | N/A (stdlib) |
| `encoding/json` (Go stdlib) | JSON I/O | BSD-style (Go) | **Active** | N/A (stdlib) |
| Future: `net/netip` or IP-validation pkg | SSRF IP validation | TBD | **Pending** | Verify license before adoption. Prefer stdlib `net.IP` / `net/netip`. |

### Rules

- Verify current upstream license and release/version at implementation time.
- Record required attribution/NOTICE obligations (Crawl4AI attribution clause).
- No GPL component may be imported, copied, vendored, or linked into the MIT Go core.
- Photon is not a v1 subprocess either.
- No dependency is adopted merely because it is popular.
- Each dependency needs a license, maintenance, security, and scope review before adoption.

---

## 6. Audit, privacy, and retention

### Stderr audit record contract

Every extraction invocation must emit one JSON line on stderr:

```json
{
  "module": "extraction",
  "tool": "unclecode/crawl4ai@v0.9.2 (CLI subprocess)",
  "tool_version": "crawl4ai==0.9.2",
  "timestamp": "2026-07-19T14:00:00Z",
  "legal_basis": "GDPR Art.6(1)(f) legitimate interest",
  "permission_ref": "CAMP-2026-Q3-001",
  "request_url": "https://example.com/landing-page?utm_source=<redacted>",
  "final_url": "https://example.com/landing-page/",
  "status": "ok",
  "duration_ms": 2340,
  "limits": "max_body=2MB, max_markdown=100KB, timeout=45s, max_redirects=5"
}
```

### Current gap

The existing `AuditRecord` struct in `extraction.go` carries `Tool`, `URL`, `CheckedAt`, `Status`, `LegalBasis`, `Error` — but is missing `permission_ref`, `duration_ms`, `limits`, and `tool_version`. The URL is not sanitized (query strings are included). These gaps should be addressed when the module is enhanced.

### URL sanitization rules

- Query strings: redact all parameter values. Preserve parameter names for debugging. E.g. `?utm_source=<redacted>&id=<redacted>`.
- Fragments: strip entirely.
- Credentials in URL: reject the request (see SSRF controls), never log.

### Fields allowed in stderr

- module, tool, tool_version, timestamp, legal_basis, permission_ref
- request_url (sanitized), final_url (sanitized)
- status, duration_ms, limits, error (operational error description only)

### Fields forbidden in stderr

- Raw page content (HTML, markdown, text)
- Extracted email addresses, phone numbers, company names
- API keys, tokens, credentials
- Full unsanitized query strings
- Cookie values, session identifiers
- Any PII from the extracted page

### Raw page text in logs

**Default: no.** Full raw page text must never be logged to stderr or any system log. The `raw_markdown` field in the result is stored in the lead record only, subject to the retention policy below.

### PII redaction in logs

- Emails extracted from pages: never logged in stderr audit records. The audit subject is the URL only (matching the existing convention).
- Phones: never logged.
- Company names: never logged in audit (they appear in the result, not the audit trail).

### Retention questions (to be resolved before production)

| Data | Current retention | Decision needed |
|---|---|---|
| `raw_markdown` (in lead record) | Indefinite (in-memory store) | Define a retention window. Raw content should have a shorter retention than normalized fields. |
| Extracted `fields` (in lead record) | Indefinite | Define retention aligned with lead lifecycle. |
| Stderr audit records | Indefinite (log stream) | Define log retention window. Audit records should be retained longer than raw content for compliance traceability. |
| Candidate provenance records | Not yet stored separately | Decide whether provenance is embedded in the lead record or stored separately with its own retention. |

### Traceability

An operator can trace every extracted field to an approved target URL and permission reference via:
1. The `permission_ref` in the extraction result metadata.
2. The `provenance` array in the result, linking each field to a source URL.
3. The stderr audit record, which logs the permission_ref, request URL, and status.
4. The control-plane's `AuditEvent` record, which stores the full stderr JSON as `raw_stderr_json`.

---

## 7. Test strategy

Tests designed only — no tests written in this planning pass.

### Unit / fixture tests

| Test | Purpose |
|---|---|
| URL scheme validation | Reject `ftp://`, `file://`, `data:`, `javascript:`, allow `http://`, `https://` |
| Hostname validation | Reject empty, whitespace-only, IP-literal-only (configurable) |
| IPv4 private-address rejection | Reject 10.x, 172.16-31.x, 192.168.x, 127.x, 169.254.x, 100.64-127.x, 0.0.0.0 |
| IPv6 private-address rejection | Reject ::1, fe80::/10, fc00::/7, ff00::/8, :: |
| Cloud metadata rejection | Reject 169.254.169.254 and known cloud metadata endpoints |
| DNS resolution result validation | After resolving, validate every IP against the private-address list |
| Redirect validation | Reject redirects to prohibited hosts/IPs; enforce max redirect count |
| Query-string redaction | Verify parameter values are redacted, parameter names preserved |
| Content-type filtering | Accept text/html, application/xhtml+xml; reject application/pdf, image/*, etc. |
| Byte-limit handling | Verify raw_markdown truncation at MaxRawMarkdown, response body rejection at max body size |
| Result normalization | Verify Fields struct is populated correctly from wrapper output |
| Provenance preservation | Verify provenance array links fields to source URLs |
| Audit output content | Verify audit record contains required fields, no PII |
| Audit PII redaction | Verify no emails, phones, page content in stderr |
| Malformed JSON worker output | Verify graceful degradation to `Result{Status: "error"}` |
| Worker timeout/non-zero exit | Verify context cancellation kills subprocess, result is "error" |
| Worker unavailable | Verify missing python/wrapper produces "error" with install hint |
| Missing permission_ref | Verify request is rejected (proposed enhancement) |
| Confidence calculation | Verify confidence score matches field population (existing tests) |
| Rate limiter | Verify min interval, backoff, reset (existing tests) |

### Integration tests

| Test | Purpose |
|---|---|
| Controlled local fixture server | HTTP test server returning known HTML. Verify extraction of title, emails, phones from fixture content. |
| Public-looking allowed target | Fixture server on 127.0.0.1 with a public-looking domain in the request. Verify extraction succeeds. |
| Disallowed private target | Attempt extraction against a private IP. Verify SSRF rejection. |
| No arbitrary external website in CI | All integration tests use local fixture servers only. |

Network-dependent tests must be guarded:

```go
if testing.Short() {
    t.Skip("skipping network-dependent test in short mode")
}
```

CI must run full:

```bash
go test ./...
```

CI must **never** be configured with `-short`.

### Security tests

| Test | Purpose |
|---|---|
| Redirect to private IP | Start extraction against public-looking URL that redirects to 127.0.0.1. Verify rejection. |
| DNS rebinding simulation | Test that IP is revalidated at connection time, not just at resolution. Strategy: use a test DNS server that returns different IPs on successive queries, or mock the resolver. |
| Metadata endpoint rejection | Attempt extraction against 169.254.169.254. Verify rejection. |
| Oversized response | Fixture server returns a 10MB response. Verify rejection at max body size. |
| Compression/decompression bomb | Fixture server returns a small gzipped response that decompresses to gigabytes. Policy: set a decompressed-size limit or disable automatic decompression. |
| Slow response timeout | Fixture server that sends data very slowly (1 byte/second). Verify timeout. |
| Unsupported protocol | Attempt `ftp://`, `file://`, `gopher://`. Verify rejection. |
| URL with credentials | Attempt `https://user:pass@example.com`. Verify rejection. |

---

## 8. Future Pass-2 PR boundary

### Smallest viable implementation PR

The future implementation PR should be scoped to:

```
modules/extraction/**
docs/decisions/extraction-stage-2-plan.md (this file — may be updated)
```

Changes elsewhere are allowed only if a concrete, documented interface blocker requires them (e.g. updating the registry entry in the control-plane to match the actual module).

### Future PR must include

- [ ] SSRF validation layer (URL scheme, hostname, IP address checks)
- [ ] `permission_ref` enforcement (reject if empty)
- [ ] Provenance tracking for extracted fields
- [ ] Query-string redaction in audit records
- [ ] Enhanced audit record with duration_ms, limits, permission_ref
- [ ] Content-type validation
- [ ] Response body size limit enforcement
- [ ] Updated unit tests for all new security controls
- [ ] Fixture-based integration tests (local HTTP test server)
- [ ] SSRF security tests
- [ ] Updated README documenting new controls
- [ ] License inventory verification for Crawl4AI transitive deps

### Explicitly deferred from the implementation PR

- Firecrawl adapter enhancements
- Photon integration (rejected)
- Crawlab integration (rejected)
- Orchestration/queue system
- Control-plane execution wiring (unless necessary to demonstrate one bounded local run — the registry entry update is the minimum)
- UI changes
- Authenticated crawling
- Multi-page recursive crawling (`contact_page` mode deferred)
- LLM enrichment
- New sales-facing data views
- Docker containerization of the Python wrapper
- `robots.txt` enforcement (organisational policy, not technical in v1)

---

## 9. Go / no-go checklist

### Must be true before Pass 2

- [ ] Canonical repository confirmed (`Devin-gits/osint-lead-platform`)
- [ ] Crawl4AI license/version reviewed from upstream (verify Apache-2.0 + attribution clause still applies to 0.9.2)
- [ ] Python dependency license inventory complete (run `pip-licenses` on installed crawl4ai and all transitive deps)
- [ ] No Photon code/dependency/subprocess path proposed
- [ ] No Crawlab code/dependency/platform path proposed
- [ ] Go/Python boundary chosen (subprocess — already chosen and implemented)
- [ ] SSRF design reviewed and testable (URL validation, IP checks, redirect validation — must have concrete test fixtures)
- [ ] Permission reference is mandatory (enforce in `Extract()`)
- [ ] Audit event schema approved (enhanced with permission_ref, duration_ms, limits, sanitized URLs)
- [ ] Raw-content retention policy decided (default retention window for `raw_markdown`)
- [ ] Response, redirect, time, and concurrency limits approved (2MB body, 5 redirects, 45s timeout, 1 concurrent)
- [ ] Fixture-based test approach approved (local HTTP test server, no arbitrary external websites in CI)
- [ ] No control-plane or UI scope creep required (registry entry update is the minimum allowed change outside `modules/extraction/`)

### Stop conditions

Do not start Pass 2 if:

- License verification is incomplete (Crawl4AI or its transitive deps)
- SSRF controls are vague or untestable
- Raw page-content retention is unresolved
- The module would crawl an arbitrary target without a permission reference
- Proposed scope includes Photon, Crawlab, Firecrawl hosted calls, or orchestration
- The repository identity is unclear
