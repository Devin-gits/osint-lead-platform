# modules/extraction

Extract structured, low-risk lead fields from a landing page or business website.

This module is part of the OSINT lead platform Validate stage. It takes a lead
record containing a `url`, fetches the page using one of two swappable backends,
and returns a normalized extraction result plus a structured audit line on
stderr.

Per the Stage 1 decision (`docs/decisions/stage-1-decision.md` § extraction):

- **Primary backend:** [Crawl4AI](https://github.com/unclecode/crawl4ai) — local
  Python crawler, Apache-2.0 + attribution clause. This is the default.
- **Secondary backend:** [Firecrawl](https://www.firecrawl.dev/) hosted API —
  optional adapter only, enabled when `EXTRACTION_BACKEND=firecrawl` and
  `FIRECRAWL_API_KEY` is set. No Firecrawl server code is vendored.

## Install

```bash
cd modules/extraction
make pydeps   # pip install -r requirements.txt (Crawl4AI)
make build    # go build -o bin/extraction ./cmd/extraction
```

Requirements:

- Go 1.22.5
- Python 3.10+ (for the Crawl4AI backend)
- Crawl4AI 0.9.2 (`requirements.txt`)

Firecrawl requires only an API key; no Python dependency.

## Quick start

```bash
export EXTRACTION_BACKEND=crawl4ai
echo '{"url":"https://example.com"}' | ./bin/extraction
```

## I/O contract

### Input

A single JSON object on stdin, or `--url` (optionally with `--backend` and
`--timeout`).

```json
{
  "url": "https://example.com",
  "email": "support@example.com",
  "name": "Jane Doe",
  "company": "Example Inc",
  "domain": "example.com"
}
```

Only `url` is required. The other fields are preserved in the output for
context but are never required to crawl.

### Output (stdout)

The same lead record with an `extraction` object added:

```json
{
  "url": "https://example.com",
  "company": "Example Inc",
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
    "metadata": {
      "backend": "crawl4ai",
      "legal_basis": "GDPR Art.6(1)(f) legitimate-interest",
      "http_status": 200,
      "truncated": false,
      "raw_bytes": 1234
    },
    "error": "",
    "checked_at": "2026-07-16T12:00:00Z"
  }
}
```

Statuses:

- `ok` — extraction succeeded and produced fields.
- `partial` — the page was fetched but few/no structured fields were found.
- `skipped` — missing URL or missing Firecrawl API key (operational skip).
- `error` — crawl/API failure, timeout, missing binary, or unparseable output.

`raw_markdown` is bounded to 100 KB to keep storage and audit logs sane.

### Audit (stderr)

One JSON line per run:

```json
{"tool":"unclecode/crawl4ai@v0.9.2 (CLI subprocess)","url":"https://example.com","checked_at":"2026-07-16T12:00:00Z","status":"ok","legal_basis":"GDPR Art.6(1)(f) legitimate-interest"}
```

The audit subject is the URL only. Emails, names, companies, API keys, and page
bodies are never written to stderr.

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
At least one of `--url` or a `url` field in stdin JSON must be provided.

## Configuration

| Env var | Default | Meaning |
|---|---|---|
| `EXTRACTION_BACKEND` | `crawl4ai` | Backend to use: `crawl4ai` or `firecrawl`. |
| `EXTRACTION_TIMEOUT` | `45s` | Per-extraction timeout (Go duration). |
| `EXTRACTION_MIN_INTERVAL` | `2s` | Minimum delay enforced between consecutive `Extract` calls on a reused `Extractor`. |
| `EXTRACTION_CRAWL4AI_PYTHON` | `python3` | Python interpreter for the Crawl4AI wrapper. |
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

### Firecrawl (optional adapter)

Set `EXTRACTION_BACKEND=firecrawl` and `FIRECRAWL_API_KEY`. The Go adapter calls
`POST /v1/scrape` and normalizes the markdown/links into the same `fields`
shape. If the key is missing, the result is `skipped` with a clear message.

Firecrawl is an optional Stage 2 spike adapter; the primary and maintained path
is Crawl4AI.

## Guardrails / compliance

- **One URL per call**, no site-wide recursive crawl.
- **No LLM extraction by default**; no LLM API key required for Crawl4AI.
- **No PII in audit logs**; subject is URL only.
- **Rate-limited** per-process via `EXTRACTION_MIN_INTERVAL`.
- **Legal basis:** `GDPR Art.6(1)(f) legitimate-interest` on every audit line and
  in result metadata.

## License table

| Component | License | Notes |
|---|---|---|
| `modules/extraction` Go code | MIT | This repo's module code. |
| Crawl4AI (local Python) | Apache-2.0 + attribution | Invoked only as a subprocess CLI; no source imported or vendored. See [Crawl4AI license](https://github.com/unclecode/crawl4ai/blob/main/LICENSE). |
| Firecrawl (hosted API adapter) | AGPL / hosted service | Only a thin HTTP client in Go; no server code vendored. |

## Test

```bash
make test-short   # unit tests only, no network
make test         # full tests including any live-gated network tests
make vet
make build
```

Live Crawl4AI / Firecrawl tests are skipped under `-short` and additionally
gated by `EXTRACTION_LIVE=1` so CI runs `go test ./...` without needing real
network credentials.
