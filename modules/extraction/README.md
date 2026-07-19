# modules/extraction

Extract structured, low-risk lead fields from a landing page or business website.

This module is part of the OSINT lead platform Validate stage. It takes a lead
record containing a `url` and a `permission_ref`, fetches the page using one of two
swappable backends, and returns a normalized extraction result plus a
structured audit line on stderr.

Per the Stage 2 decision (`docs/decisions/extraction-stage-2-plan.md`):

- **Primary backend:** [Crawl4AI](https://github.com/unclecode/crawl4ai) — local
  Python crawler, Apache-2.0 + attribution clause. This is the default.
- **Secondary backend:** [Firecrawl](https://www.firecrawl.dev/) hosted API —
  optional adapter only, enabled when `EXTRACTION_BACKEND=firecrawl` and
  `FIRECRAWL_API_KEY` is set. No Firecrawl server code is vendored.
- **Control-plane integration:** the module is wired as an in-process Go
  library from `services/control-plane` (Pass 2B). The control-plane runner
  passes `url` and `permission_ref` from the lead and stores the `extraction`
  result + audit event.

## Install

```bash
cd modules/extraction
python3 -m venv .venv
source .venv/bin/activate
make pydeps   # pip install -r requirements.txt (Crawl4AI)
# Optional: if Crawl4AI cannot launch the browser, install Playwright browsers
# playwright install chromium
make build    # go build -o bin/extraction ./cmd/extraction
```

Requirements:

- Go 1.22.5
- Python 3.10+ (for the Crawl4AI backend)
- Crawl4AI 0.9.2 (`requirements.txt`)
- Playwright browsers are sometimes required by Crawl4AI 0.9.2 depending on the
  crawler configuration. If the wrapper reports a browser/launcher error, run
  `playwright install chromium` in the same Python environment.

Firecrawl requires only an API key; no Python dependency.

## Quick start

```bash
export EXTRACTION_BACKEND=crawl4ai
echo '{"url":"https://example.com","permission_ref":"CAMP-2026-Q3-001"}' | ./bin/extraction
```

`permission_ref` is **mandatory**. A missing `permission_ref` produces a
`skipped` result and a clear audit message.

## End-to-end demo (via control-plane)

From the repository root:

```bash
cd modules/extraction
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
# Only if Crawl4AI complains about a missing browser:
# playwright install chromium
cd ../../services/control-plane
go run ./cmd/server
```

Then in another terminal:

```bash
# Create a lead with a public URL and permission reference
LEAD=$(curl -s -X POST http://localhost:8080/api/leads \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com","permission_ref":"DEMO-1"}' | jq -r '.data.id')

# Run extraction
curl -s -X POST "http://localhost:8080/api/leads/${LEAD}/run" \
  -H 'Content-Type: application/json' \
  -d '{"modules":["extraction"]}' | jq '.data.extraction | {status, source_tool, confidence, fields}'

# Check the audit event
curl -s "http://localhost:8080/api/leads/${LEAD}" | jq '.data.audit_events[0] | {module, status, legal_basis, subject}'
```

Expected results:

- If Crawl4AI is installed and the page loads: `status: "ok"` with extracted fields.
- If Crawl4AI is not installed: `status: "error"` with a structured message such
  as `crawl4ai is not installed`. This is an honest API result, not a failure of
  the control-plane.
- If `permission_ref` is missing: `status: "skipped"` with `reason: "missing permission_ref"`.

## I/O contract

### Input

A single JSON object on stdin, or `--url` (optionally with `--backend` and
`--timeout`).

```json
{
  "url": "https://example.com",
  "permission_ref": "CAMP-2026-Q3-001",
  "email": "support@example.com",
  "name": "Jane Doe",
  "company": "Example Inc",
  "domain": "example.com"
}
```

| Field | Required | Description |
|---|---|---|
| `url` | yes | Exact public HTTP(S) URL to extract from. |
| `permission_ref` | yes | Privacy-safe reference tying the extraction to an approved customer/data-processing basis. |
| `email` | no | Passed through for context; never logged to audit. |
| `name` | no | Passed through for context; never logged to audit. |
| `company` | no | Passed through for context; never logged to audit. |
| `domain` | no | Passed through for context; never logged to audit. |

### Output (stdout)

The same lead record with an `extraction` object added:

```json
{
  "url": "https://example.com",
  "permission_ref": "CAMP-2026-Q3-001",
  "extraction": {
    "status": "ok",
    "url": "https://example.com",
    "final_url": "https://example.com/",
    "source_tool": "unclecode/crawl4ai@v0.9.2 (CLI subprocess)",
    "confidence": 0.286,
    "fields": {
      "company_name": "Example Inc",
      "emails": ["support@example.com"],
      "phones": [],
      "addresses": [],
      "social_links": ["https://www.linkedin.com/company/example"],
      "contact_urls": ["https://example.com/contact"],
      "description": "Example Inc makes widgets.",
      "title": "Example Inc - Home"
    },
    "raw_markdown": "...page markdown truncated to 100 KB...",
    "provenance": [
      {
        "field": "company_name",
        "value": "Example Inc",
        "source_url": "https://example.com/",
        "method": "crawl4ai",
        "timestamp": "2026-07-19T12:00:00Z"
      }
    ],
    "metadata": {
      "backend": "crawl4ai",
      "legal_basis": "GDPR Art.6(1)(f) legitimate-interest",
      "permission_ref": "CAMP-2026-Q3-001",
      "http_status": 200,
      "truncated": false,
      "raw_bytes": 1234,
      "duration_ms": 1234,
      "limits_applied": "max_body=2MB,max_markdown=100KB,timeout=45s,max_redirects=5"
    },
    "error": "",
    "checked_at": "2026-07-19T12:00:00Z"
  }
}
```

Statuses:

- `ok` — extraction succeeded and produced fields.
- `partial` — the page was fetched but few/no structured fields were found.
- `skipped` — missing URL, missing `permission_ref`, URL rejected by SSRF policy, or missing Firecrawl API key (operational skip).
- `error` — crawl/API failure, timeout, missing binary, or unparseable output.

`raw_markdown` is bounded to 100 KB to keep storage and audit logs sane.
`provenance` records one entry per extracted value, linking it to the source URL,
extraction method, and timestamp.

### Audit (stderr)

One JSON line per run:

```json
{
  "module": "extraction",
  "tool": "unclecode/crawl4ai@v0.9.2 (CLI subprocess)",
  "tool_version": "crawl4ai==0.9.2",
  "timestamp": "2026-07-19T12:00:00Z",
  "legal_basis": "GDPR Art.6(1)(f) legitimate-interest",
  "permission_ref": "CAMP-2026-Q3-001",
  "request_url": "https://example.com?utm_source=[redacted]",
  "final_url": "https://example.com/",
  "status": "ok",
  "duration_ms": 1234,
  "limits": "max_body=2MB,max_markdown=100KB,timeout=45s,max_redirects=5"
}
```

The audit subject is the URL only. Query parameter values are redacted to
`[redacted]`. Userinfo (credentials) is stripped. Emails, names, companies, API
keys, raw page bodies, and credentials are never written to stderr.

### Exit codes

- `0` — a well-formed lead was read and an `extraction` record was emitted,
  including soft operational failures (`skipped`/`error`).
- `1` — unreadable/invalid input JSON, or no URL was provided.

## CLI

```bash
./bin/extraction --url https://example.com
./bin/extraction --url https://example.com --backend firecrawl --timeout 60s
```

When run from a terminal, `--url` can supply the URL if stdin is not used.
At least one of `--url` or a `url` field in stdin JSON must be provided. The
`permission_ref` must come from stdin JSON; there is no `--permission-ref` flag
by design (to avoid credentials/policy leakage through shell history).

## Configuration

| Env var | Default | Meaning |
|---|---|---|
| `EXTRACTION_BACKEND` | `crawl4ai` | Backend to use: `crawl4ai` or `firecrawl`. |
| `EXTRACTION_TIMEOUT` | `45s` | Per-extraction timeout (Go duration). |
| `EXTRACTION_MIN_INTERVAL` | `2s` | Minimum delay enforced between consecutive `Extract` calls on a reused `Extractor`. |
| `EXTRACTION_CRAWL4AI_PYTHON` | `python3` | Python interpreter for the Crawl4AI wrapper. Point this at a venv Python (e.g., `modules/extraction/.venv/bin/python`) to get the real `ok` extraction path. |
| `EXTRACTION_CRAWL4AI_WRAPPER` | auto-locate | Path to `wrapper/crawl4ai_extract.py`. |
| `FIRECRAWL_API_KEY` | *(none)* | Bearer token for the Firecrawl hosted API. Required only for `firecrawl` backend. |
| `FIRECRAWL_BASE_URL` | `https://api.firecrawl.dev/v1` | Firecrawl API base URL (allows future self-host). |

## Backends

### Crawl4AI (default)

Go orchestrator runs `wrapper/crawl4ai_extract.py` as a subprocess. The wrapper
imports Crawl4AI locally and uses `AsyncWebCrawler` to fetch the URL. Extraction
is LLM-free: regexes and `html.parser` for title, meta description, emails,
phones, social links, and contact URLs.

- Single-page only (no recursion).
- No proxy/Tor flags.
- Bounded markdown payload.
- Missing Crawl4AI install → structured `error`, exit 0.
- The wrapper also validates the target URL for SSRF in defence in depth.

### Firecrawl (optional adapter)

Set `EXTRACTION_BACKEND=firecrawl` and `FIRECRAWL_API_KEY`. The Go adapter calls
`POST /v1/scrape` and normalizes the markdown/links into the same `fields`
shape. If the key is missing, the result is `skipped` with a clear message.

The Firecrawl HTTP client sets `CheckRedirect` to reject any redirect target that
does not pass the same SSRF policy applied to the input URL.

Firecrawl is an optional Stage 2 spike adapter; the primary and maintained path
is Crawl4AI.

## Security / SSRF controls

Before any backend is invoked, the Go orchestrator validates the input URL:

- **Scheme:** `http` and `https` only.
- **Userinfo:** URLs containing `user:pass@...` are rejected. Credentials in URLs
  leak through logs, shell history, process lists, proxies, and audit trails.
- **IP-literal hostnames:** rejected by default. Public websites should use DNS
  names.
- **DNS resolution:** every resolved IP is checked against the forbidden list.
- **Forbidden IPs:** loopback, link-local, RFC1918, CGNAT (`100.64.0.0/10`),
  unique-local IPv6, multicast, unspecified, and cloud metadata
  (`169.254.169.254`).
- **Ports:** only 80 and 443 by default.
- **Redirects:** validated by the Firecrawl adapter `CheckRedirect`.

These controls are defence-in-depth: the Crawl4AI Python wrapper repeats the
same checks before fetching. Residual risk remains because the subprocess crawler
may follow server-side redirects; the Go layer cannot intercept every hop inside
Python. This residual limitation is documented in `docs/decisions/extraction-stage-2-plan.md`.

## LinkedIn link harvesting vs LinkedIn scraping

`modules/extraction` may collect a LinkedIn URL **only when it already appears on
the permissioned page** (e.g., in a `social_links` array). It does **not** target
LinkedIn pages, perform LinkedIn search, or scrape LinkedIn profiles. LinkedIn
profile/page scraping as a target is forbidden per `docs/compliance.md` and is
out of scope for this module.

## Guardrails / compliance

- **One URL per call**, no site-wide recursive crawl.
- **No LLM extraction by default**; no LLM API key required for Crawl4AI.
- **No PII in audit logs**; subject is URL only, with query values redacted.
- **Rate-limited** per-process via `EXTRACTION_MIN_INTERVAL`.
- **Legal basis:** `GDPR Art.6(1)(f) legitimate-interest` on every audit line,
  every result, and every provenance record.
- **`permission_ref` mandatory** for every extraction.

## License table

| Component | License | Notes |
|---|---|---|
| `modules/extraction` Go code | MIT | This repo's module code. |
| `wrapper/crawl4ai_extract.py` | MIT | This repo's wrapper script. |
| Crawl4AI (local Python) | Apache-2.0 + attribution | Invoked only as a subprocess CLI; no source imported or vendored. See `LICENSES.md` and `https://github.com/unclecode/crawl4ai/blob/main/LICENSE`. |
| Firecrawl (hosted API adapter) | Hosted service / partner terms | Only a thin HTTP client in Go; no server code vendored. |
| Go standard library | BSD-style (Go license) | Used for URL parsing, HTTP, DNS, JSON, etc. |

Explicitly **not** used in this module: Photon, Crawlab, `linkedin_scraper`,
`linkedin2username`, LinkedInt, or any LinkedIn scraping library. See
`LICENSES.md` for the full rejection list.

## Test

```bash
make test-short   # unit tests only, no network
make test         # full tests including any live-gated network tests
make vet
make build
```

Or directly:

```bash
go test ./...
go test ./... -short
go vet ./...
go build -o bin/extraction ./cmd/extraction
```

All unit and security tests are network-free. Live Crawl4AI / Firecrawl tests
are skipped under `-short` and additionally gated by `EXTRACTION_LIVE=1` so CI
runs `go test ./...` without needing real network credentials.

### Manual security checks

```bash
echo '{"url":"https://example.com","permission_ref":"T-1"}' | ./bin/extraction
echo '{"url":"http://127.0.0.1/","permission_ref":"T-1"}' | ./bin/extraction
echo '{"url":"http://169.254.169.254/","permission_ref":"T-1"}' | ./bin/extraction
echo '{"url":"https://user:pass@example.com/","permission_ref":"T-1"}' | ./bin/extraction
```

Expected:

- Missing `permission_ref` → `skipped`.
- Private / metadata / credentialed URLs → `skipped`.
- `stderr` audit contains `legal_basis`, `permission_ref`, and `limits` and
  never contains raw HTML, PII, or credentials.
