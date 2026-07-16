# modules/extraction

Validate-stage module: extract structured lead fields from a landing page or
business website URL.

## Decision basis

See `docs/decisions/stage-1-decision.md` § extraction:

- **Primary engine:** Crawl4AI (local Python), Apache-2.0 + attribution — default.
- **Secondary engine:** Firecrawl hosted API — optional adapter only.

## Layout

```
modules/extraction/
├── go.mod
├── Makefile
├── README.md
├── requirements.txt
├── extraction.go       # Extractor, backend selection, rate limit, audit
├── ratelimit.go        # in-process per-URL rate limiter
├── crawl4ai.go         # subprocess runner
├── firecrawl.go        # HTTP adapter
├── extraction_test.go
├── crawl4ai_test.go
├── firecrawl_test.go
├── cmd/extraction/main.go
├── cmd/extraction/main_test.go
├── wrapper/crawl4ai_extract.py
└── testdata/
```

## Public API

```go
package extraction

type Extractor struct{}
func NewExtractor(timeout, minInterval time.Duration, backend string) *Extractor
func (e *Extractor) Extract(ctx context.Context, in Input) (Result, AuditRecord)
```

Only `Input.URL` is required. Email/name/company/domain are preserved for
context but never required.

## Backends

| Env | Default | Options |
|---|---|---|
| `EXTRACTION_BACKEND` | `crawl4ai` | `crawl4ai`, `firecrawl` |

### Crawl4AI

- `wrapper/crawl4ai_extract.py` is the only file that imports Crawl4AI.
- Subprocess only (`python3 wrapper/crawl4ai_extract.py --url ...`).
- Single-page, LLM-free extraction (regex + html.parser).
- Missing Crawl4AI install → structured error, CLI exit 0.

### Firecrawl

- Thin Go HTTP client to `https://api.firecrawl.dev/v1/scrape`.
- Requires `FIRECRAWL_API_KEY`; missing key → `skipped`.
- Optional `FIRECRAWL_BASE_URL` for future self-host.

## Output schema (under lead["extraction"])

| Field | Meaning |
|---|---|
| `status` | `ok`/`partial`/`error`/`skipped` |
| `url` | requested URL |
| `final_url` | resolved URL |
| `source_tool` | backend identifier + version |
| `confidence` | 0.0–1.0 based on populated field categories |
| `fields` | `company_name`, `emails`, `phones`, `addresses`, `social_links`, `contact_urls`, `description`, `title` |
| `raw_markdown` | bounded page markdown (max 100 KB) |
| `metadata` | `backend`, `legal_basis`, `http_status`, `truncated`, `raw_bytes`, `error` |
| `error` | human-readable operational error |

## Audit

One JSON line on stderr per run:

```json
{"tool":"...","url":"https://...","checked_at":"...","status":"...","legal_basis":"GDPR Art.6(1)(f) legitimate-interest","error":"..."}
```

Subject is URL only; no PII, keys, or page bodies.

## Compliance notes

- Rate limit: `EXTRACTION_MIN_INTERVAL` (default 2s).
- Legal basis: `GDPR Art.6(1)(f) legitimate-interest`.
- Crawl4AI attribution required; stated in README license table.
- No recursive crawl, no proxy/Tor flags, no arbitrary JS eval beyond normal page load.

## Tests

```bash
cd modules/extraction && go test ./...
```

Offline tests use fake wrapper scripts and `httptest` for Firecrawl. Live tests
are skipped under `-short` and gated by `EXTRACTION_LIVE=1`.

## Do not

- Import Crawl4AI or Firecrawl server code into Go.
- Recursively crawl a whole site.
- Log raw page bodies, cookies, or API keys on stderr.
- Make Firecrawl required for the default path.
- Wire into `services/control-plane` or other modules in this PR.
